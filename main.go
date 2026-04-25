package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	h "maragu.dev/gomponents/html"
)

type boardItem struct {
	URL      string  `json:"url"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
}

type board struct {
	Items []boardItem `json:"items"`
}

var validID = regexp.MustCompile(`^[a-f0-9]{16}$`)

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func boardPath(id string) (string, bool) {
	if !validID.MatchString(id) {
		return "", false
	}
	return filepath.Join("data/boards", id+".json"), true
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "request too large", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "missing image field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	ct := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(ct, "image/") {
		http.Error(w, "file is not an image", http.StatusBadRequest)
		return
	}
	extByType := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}
	ext := extByType[ct]
	if ext == "" {
		ext = ".png"
	}

	b := make([]byte, 16)
	rand.Read(b)
	filename := hex.EncodeToString(b) + ext

	if err := os.MkdirAll("data/images", 0755); err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	dst, err := os.Create(filepath.Join("data/images", filename))
	if err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	dst.Write(buf[:n])
	io.Copy(dst, file)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": "/images/" + filename})
}

func handleCreateBoard(w http.ResponseWriter, r *http.Request) {
	var b board
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := os.MkdirAll("data/boards", 0755); err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	id := generateID()
	f, err := os.Create(filepath.Join("data/boards", id+".json"))
	if err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func handleUpdateBoard(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path, ok := boardPath(id)
	if !ok {
		http.Error(w, "invalid board id", http.StatusBadRequest)
		return
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "board not found", http.StatusNotFound)
		return
	}
	var b board
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(b)
	w.WriteHeader(http.StatusNoContent)
}

func handleLoadBoard(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path, ok := boardPath(id)
	if !ok {
		http.Error(w, "invalid board id", http.StatusBadRequest)
		return
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		http.Error(w, "board not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, f)
}

func page() g.Node {
	return c.HTML5(c.HTML5Props{
		Title:    "Scrapbook",
		Language: "en",
		Head: []g.Node{
			h.Script(h.Src("https://unpkg.com/konva@9/konva.min.js")),
			g.El("style", g.Raw(`
				* { box-sizing: border-box; margin: 0; padding: 0; }
				body { font-family: sans-serif; height: 100vh; display: flex; flex-direction: column; }
				header { padding: 0.75em 1em; background: #333; color: #fff; font-size: 1.1em; display: flex; align-items: center; gap: 1em; }
				header span { font-size: 0.8em; opacity: 0.6; }
				#load-btn, #save-btn {
					padding: 6px 18px;
					border: none;
					border-radius: 6px;
					font-size: 0.85em;
					font-weight: 700;
					letter-spacing: 0.05em;
					cursor: pointer;
					transition: background 0.15s, color 0.15s;
				}
				#load-btn { background: transparent; color: #aaa; border: 1px solid #555; }
				#load-btn:hover { background: #444; color: #fff; }
				#save-btn {
					margin-left: auto;
					background: #fff;
					color: #333;
				}
				#save-btn:hover { background: #eee; }
				#save-btn.saved { background: #27ae60; color: #fff; }
				#canvas-wrap {
					flex: 1;
					display: flex;
					overflow: hidden;
				}
				#canvas {
					flex: 1;
					background: #f5f0e8;
					background-image: radial-gradient(#ccc 1px, transparent 1px);
					background-size: 20px 20px;
				}
				#toolbar {
					width: 48px;
					background: #2a2a2a;
					display: flex;
					flex-direction: column;
					align-items: center;
					padding: 8px 0;
					gap: 4px;
					z-index: 10;
				}
				.tool-btn {
					width: 36px;
					height: 36px;
					border: none;
					border-radius: 6px;
					background: transparent;
					color: #aaa;
					cursor: pointer;
					display: flex;
					align-items: center;
					justify-content: center;
					transition: background 0.15s, color 0.15s;
				}
				.tool-btn:hover { background: #444; color: #fff; }
				#upload-modal {
					display: none;
					position: fixed;
					inset: 0;
					background: rgba(0,0,0,0.45);
					z-index: 100;
					align-items: center;
					justify-content: center;
				}
				#upload-modal.open { display: flex; }
				#upload-dialog {
					background: #fff;
					border-radius: 10px;
					padding: 24px;
					width: 360px;
					max-width: 90vw;
					display: flex;
					flex-direction: column;
					gap: 16px;
					box-shadow: 0 8px 32px rgba(0,0,0,0.3);
				}
				#upload-dialog h2 { font-size: 1.1em; color: #222; }
				#drop-zone {
					border: 2px dashed #ccc;
					border-radius: 8px;
					padding: 32px 16px;
					text-align: center;
					color: #999;
					font-size: 0.9em;
					cursor: default;
					transition: border-color 0.15s, background 0.15s;
					display: flex;
					flex-direction: column;
					align-items: center;
					gap: 10px;
				}
				#drop-zone.drag-over { border-color: #555; background: #f4f4f4; color: #555; }
				#drop-zone p { font-size: 0.8em; opacity: 0.75; }
				.modal-actions {
					display: flex;
					align-items: center;
					justify-content: space-between;
				}
				.choose-btn {
					padding: 8px 16px;
					background: #333;
					color: #fff;
					border: none;
					border-radius: 6px;
					cursor: pointer;
					font-size: 0.9em;
				}
				.choose-btn:hover { background: #555; }
				.close-btn {
					padding: 8px 14px;
					background: transparent;
					color: #666;
					border: 1px solid #ddd;
					border-radius: 6px;
					cursor: pointer;
					font-size: 0.9em;
				}
				.close-btn:hover { background: #f5f5f5; }
			`)),
		},
		Body: []g.Node{
			h.Header(
				g.Text("Scrapbook"),
				h.Span(g.Text("paste an image to add it")),
				h.Button(h.ID("load-btn"), g.Text("LOAD")),
				h.Button(h.ID("save-btn"), g.Text("SAVE")),
			),
			h.Div(h.ID("canvas-wrap"),
				h.Div(h.ID("canvas")),
				h.Div(h.ID("toolbar"),
					h.Button(
						h.ID("open-upload-modal"),
						g.Attr("class", "tool-btn"),
						g.Attr("title", "Upload image"),
						g.Raw(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>`),
					),
				),
			),
			h.Div(h.ID("upload-modal"),
				h.Div(h.ID("upload-dialog"),
					h.H2(g.Text("Add Image")),
					h.Div(h.ID("drop-zone"),
						g.Raw(`<svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="#ccc" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>`),
						h.P(g.Text("Drop an image here, or paste from clipboard")),
					),
					h.Div(g.Attr("class", "modal-actions"),
						h.Button(g.Attr("class", "choose-btn"), h.ID("choose-file-btn"), g.Text("Choose file…")),
						h.Input(h.Type("file"), h.ID("file-input"), g.Attr("accept", "image/*"), g.Attr("style", "display:none")),
						h.Button(g.Attr("class", "close-btn"), h.ID("modal-close"), g.Text("Close")),
					),
				),
			),
			g.El("script", g.Raw(`
				var canvasEl = document.getElementById('canvas');

				var stage = new Konva.Stage({
					container: 'canvas',
					width: canvasEl.offsetWidth,
					height: canvasEl.offsetHeight,
				});

				var layer = new Konva.Layer();
				stage.add(layer);

				var tr = new Konva.Transformer({
					keepRatio: true,
					enabledAnchors: ['top-left', 'top-right', 'bottom-left', 'bottom-right'],
					borderStroke: '#4a9eff',
					borderStrokeWidth: 2,
					borderDash: [6, 4],
					anchorFill: '#fff',
					anchorStroke: '#4a9eff',
					anchorStrokeWidth: 3,
					anchorCornerRadius: 11,
					anchorSize: 22,
					rotateAnchorOffset: 40,
					anchorStyleFunc: function(anchor) {
						if (anchor.hasName('rotater')) {
							anchor.sceneFunc(function(ctx, shape) {
								var s = shape.width();
								var h = s / 2;
								var r = h * 0.46;
								var lw = h * 0.23;
								var aw = lw * 1.5;

								ctx.beginPath();
								ctx.arc(h, h, h, 0, Math.PI * 2, false);
								ctx.closePath();
								ctx.setAttr('fillStyle', '#ff9f43');
								ctx.fill();

								var start = -Math.PI / 2 + 0.45;
								var end = start + Math.PI * 1.5;
								ctx.beginPath();
								ctx.arc(h, h, r, start, end, false);
								ctx.setAttr('strokeStyle', '#fff');
								ctx.setAttr('lineWidth', lw);
								ctx.setAttr('lineCap', 'round');
								ctx.stroke();

								var ex = h + Math.cos(end) * r;
								var ey = h + Math.sin(end) * r;
								var tx = -Math.sin(end);
								var ty = Math.cos(end);
								ctx.beginPath();
								ctx.moveTo(ex, ey);
								ctx.lineTo(ex - tx * aw - ty * aw * 0.55, ey - ty * aw + tx * aw * 0.55);
								ctx.lineTo(ex - tx * aw + ty * aw * 0.55, ey - ty * aw - tx * aw * 0.55);
								ctx.closePath();
								ctx.setAttr('fillStyle', '#fff');
								ctx.fill();
							});
							anchor.hitFunc(function(ctx, shape) {
								var s = shape.width();
								var h = s / 2;
								ctx.beginPath();
								ctx.arc(h, h, h, 0, Math.PI * 2, false);
								ctx.closePath();
								ctx.fillStrokeShape(shape);
							});
						}
					},
				});
				layer.add(tr);

				stage.on('mousedown', function(e) {
					if (e.target === stage) tr.nodes([]);
				});

				document.addEventListener('keydown', function(e) {
					if (e.key !== 'Delete' && e.key !== 'Backspace') return;
					var tag = document.activeElement.tagName;
					if (tag === 'INPUT' || tag === 'TEXTAREA') return;
					var nodes = tr.nodes();
					if (nodes.length) {
						nodes.forEach(function(n) { n.destroy(); });
						tr.nodes([]);
					}
				});

				window.addEventListener('resize', function() {
					stage.width(canvasEl.offsetWidth);
					stage.height(canvasEl.offsetHeight);
				});

				// --- Image helpers ---

				function addKonvaImage(imageObj, url, state) {
					var img = new Konva.Image({
						x: state.x,
						y: state.y,
						image: imageObj,
						width: state.width,
						height: state.height,
						rotation: state.rotation,
						shadowColor: 'black',
						shadowBlur: 12,
						shadowOpacity: 0.35,
						shadowOffsetX: 2,
						shadowOffsetY: 4,
						draggable: true,
					});
					img.setAttr('imageUrl', url);

					img.on('click tap', function() {
						img.moveToTop();
						tr.moveToTop();
						tr.nodes([img]);
					});
					img.on('mouseenter', function() {
						stage.container().style.cursor = 'grab';
					});
					img.on('mouseleave', function() {
						stage.container().style.cursor = 'default';
					});
					img.on('dragstart', function() {
						stage.container().style.cursor = 'grabbing';
					});
					img.on('dragend', function() {
						stage.container().style.cursor = 'grab';
					});

					layer.add(img);
				}

				function placeImage(url) {
					var imageObj = new Image();
					imageObj.onload = function() {
						var maxSize = 300;
						var scale = Math.min(1, maxSize / Math.max(imageObj.width, imageObj.height));
						var w = imageObj.width * scale;
						var h = imageObj.height * scale;
						addKonvaImage(imageObj, url, {
							x: Math.random() * Math.max(0, stage.width() - w),
							y: Math.random() * Math.max(0, stage.height() - h),
							width: w,
							height: h,
							rotation: (Math.random() * 30) - 15,
						});
					};
					imageObj.src = url;
				}

				function restoreImage(item) {
					var imageObj = new Image();
					imageObj.onload = function() {
						addKonvaImage(imageObj, item.url, item);
					};
					imageObj.src = item.url;
				}

				function addImageToCanvas(blob) {
					var ext = (blob.type || 'image/png').split('/')[1] || 'png';
					var fd = new FormData();
					fd.append('image', blob, 'image.' + ext);
					fetch('/api/images', { method: 'POST', body: fd })
						.then(function(r) { return r.json(); })
						.then(function(data) { placeImage(data.url); })
						.catch(function(err) { console.error('upload failed', err); });
				}

				// --- Board ID from URL ---

				function getBoardId() {
					var m = window.location.pathname.match(/^\/b\/([a-f0-9]{16})$/);
					return m ? m[1] : null;
				}

				// --- Save / Load ---

				function saveBoard() {
					var items = [];
					layer.getChildren().forEach(function(node) {
						if (node.getClassName() === 'Image') {
							items.push({
								url:      node.getAttr('imageUrl'),
								x:        node.x(),
								y:        node.y(),
								width:    node.width(),
								height:   node.height(),
								rotation: node.rotation(),
							});
						}
					});

					var id = getBoardId();
					var url = id ? '/api/board/' + id : '/api/board';
					var btn = document.getElementById('save-btn');

					fetch(url, {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ items: items }),
					})
					.then(function(r) {
						if (!id) return r.json().then(function(data) {
							history.pushState({}, '', '/b/' + data.id);
						});
					})
					.then(function() {
						btn.classList.add('saved');
						setTimeout(function() { btn.classList.remove('saved'); }, 1200);
					})
					.catch(function(err) { console.error('save failed', err); });
				}

				function loadBoard() {
					var id = getBoardId();
					if (!id) return;
					fetch('/api/board/' + id)
						.then(function(r) {
							if (!r.ok) throw new Error('not found');
							return r.json();
						})
						.then(function(data) {
							tr.nodes([]);
							layer.getChildren().forEach(function(node) {
								if (node.getClassName() === 'Image') node.destroy();
							});
							data.items.forEach(function(item) { restoreImage(item); });
						})
						.catch(function(err) { console.error('load failed', err); });
				}

				document.getElementById('save-btn').addEventListener('click', saveBoard);
				document.getElementById('load-btn').addEventListener('click', loadBoard);

				// Auto-load when arriving at a board URL
				if (getBoardId()) loadBoard();

				// --- Modal ---

				var modal = document.getElementById('upload-modal');
				function closeModal() { modal.classList.remove('open'); }

				document.getElementById('open-upload-modal').addEventListener('click', function() {
					modal.classList.add('open');
				});
				document.getElementById('modal-close').addEventListener('click', closeModal);
				modal.addEventListener('click', function(e) {
					if (e.target === modal) closeModal();
				});

				var fileInput = document.getElementById('file-input');
				document.getElementById('choose-file-btn').addEventListener('click', function() {
					fileInput.click();
				});
				fileInput.addEventListener('change', function() {
					if (this.files && this.files[0]) {
						addImageToCanvas(this.files[0]);
						closeModal();
						this.value = '';
					}
				});

				var dropZone = document.getElementById('drop-zone');
				dropZone.addEventListener('dragover', function(e) {
					e.preventDefault();
					this.classList.add('drag-over');
				});
				dropZone.addEventListener('dragleave', function() {
					this.classList.remove('drag-over');
				});
				dropZone.addEventListener('drop', function(e) {
					e.preventDefault();
					this.classList.remove('drag-over');
					var files = e.dataTransfer.files;
					for (var i = 0; i < files.length; i++) {
						if (files[i].type.startsWith('image/')) addImageToCanvas(files[i]);
					}
					if (files.length) closeModal();
				});

				document.addEventListener('paste', function(e) {
					var items = e.clipboardData.items;
					for (var i = 0; i < items.length; i++) {
						if (items[i].type.startsWith('image/')) {
							addImageToCanvas(items[i].getAsFile());
							if (modal.classList.contains('open')) closeModal();
						}
					}
				});
			`)),
		},
	})
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		page().Render(w)
	})
	http.HandleFunc("GET /b/{id}", func(w http.ResponseWriter, r *http.Request) {
		page().Render(w)
	})
	http.HandleFunc("/api/images", handleUpload)
	http.HandleFunc("POST /api/board", handleCreateBoard)
	http.HandleFunc("POST /api/board/{id}", handleUpdateBoard)
	http.HandleFunc("GET /api/board/{id}", handleLoadBoard)
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("data/images"))))

	addr := ":8080"
	log.Printf("listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
