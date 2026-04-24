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
			g.El("style", g.Raw(`
				* { box-sizing: border-box; margin: 0; padding: 0; }
				body { font-family: sans-serif; height: 100vh; display: flex; flex-direction: column; }
				header { padding: 0.75em 1em; background: #333; color: #fff; font-size: 1.1em; }
				header span { font-size: 0.8em; opacity: 0.6; margin-left: 1em; }
				#canvas {
					flex: 1;
					position: relative;
					overflow: hidden;
					background: #f5f0e8;
					background-image: radial-gradient(#ccc 1px, transparent 1px);
					background-size: 20px 20px;
				}
			`)),
		},
		Body: []g.Node{
			h.Header(
				g.Text("Scrapbook"),
				h.Span(g.Text("paste an image to add it")),
			),
			h.Div(h.ID("canvas")),
			g.El("script", g.Raw(`
				document.addEventListener('paste', function(e) {
					var items = e.clipboardData.items;
					for (var i = 0; i < items.length; i++) {
						if (items[i].type.startsWith('image/')) {
							var blob = items[i].getAsFile();
							var url = URL.createObjectURL(blob);
							var canvas = document.getElementById('canvas');
							var img = document.createElement('img');
							img.src = url;
							img.style.position = 'absolute';
							img.style.maxWidth = '300px';
							img.style.maxHeight = '300px';
							img.style.cursor = 'grab';
							img.style.userSelect = 'none';
							img.onload = function() {
								var x = Math.random() * Math.max(0, canvas.offsetWidth - this.offsetWidth);
								var y = Math.random() * Math.max(0, canvas.offsetHeight - this.offsetHeight);
								var rot = (Math.random() * 30) - 15;
								this.style.left = x + 'px';
								this.style.top = y + 'px';
								this.style.transform = 'rotate(' + rot + 'deg)';
								this.style.boxShadow = '2px 4px 12px rgba(0,0,0,0.35)';
							};
							canvas.appendChild(img);
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
