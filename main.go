package main

import (
	"log"
	"net/http"

	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	h "maragu.dev/gomponents/html"
)

func page() g.Node {
	return c.HTML5(c.HTML5Props{
		Title:    "Scrapbook",
		Language: "en",
		Head: []g.Node{
			h.Script(h.Src("https://unpkg.com/konva@9/konva.min.js")),
			g.El("style", g.Raw(`
				* { box-sizing: border-box; margin: 0; padding: 0; }
				body { font-family: sans-serif; height: 100vh; display: flex; flex-direction: column; }
				header { padding: 0.75em 1em; background: #333; color: #fff; font-size: 1.1em; }
				header span { font-size: 0.8em; opacity: 0.6; margin-left: 1em; }
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

				window.addEventListener('resize', function() {
					stage.width(canvasEl.offsetWidth);
					stage.height(canvasEl.offsetHeight);
				});

				function addImageToCanvas(blob) {
					var url = URL.createObjectURL(blob);
					var imageObj = new Image();
					imageObj.onload = function() {
						var maxSize = 300;
						var scale = Math.min(1, maxSize / Math.max(imageObj.width, imageObj.height));
						var w = imageObj.width * scale;
						var h = imageObj.height * scale;

						var x = Math.random() * Math.max(0, stage.width() - w);
						var y = Math.random() * Math.max(0, stage.height() - h);
						var rot = (Math.random() * 30) - 15;

						var img = new Konva.Image({
							x: x,
							y: y,
							image: imageObj,
							width: w,
							height: h,
							rotation: rot,
							shadowColor: 'black',
							shadowBlur: 12,
							shadowOpacity: 0.35,
							shadowOffsetX: 2,
							shadowOffsetY: 4,
							draggable: true,
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
					};
					imageObj.src = url;
				}

				// Modal
				var modal = document.getElementById('upload-modal');
				function closeModal() { modal.classList.remove('open'); }

				document.getElementById('open-upload-modal').addEventListener('click', function() {
					modal.classList.add('open');
				});
				document.getElementById('modal-close').addEventListener('click', closeModal);
				modal.addEventListener('click', function(e) {
					if (e.target === modal) closeModal();
				});

				// File chooser
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

				// Drop zone
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
						if (files[i].type.startsWith('image/')) {
							addImageToCanvas(files[i]);
						}
					}
					if (files.length) closeModal();
				});

				// Global paste
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

	addr := ":8080"
	log.Printf("listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
