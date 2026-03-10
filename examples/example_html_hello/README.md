# html_hello

Minimal HTML page rendered with NanoGUI. Builds a simple `<html>` document with a heading and paragraph.

## Run

**CLI** — prints HTML to stdout:

```bash
go run .
```

**Browser (WASM)** — renders the page in a browser:

```bash
./run.sh
# Open http://localhost:9090
```

## Output

```html
<html><head><title>Hello NanoGUI</title></head><body><h1>Hello from NanoGUI!</h1><p>A Go port of NanoJSX.</p></body></html>
```

## File Structure

| File | Build constraint | Purpose |
|------|-----------------|---------|
| `page.go` | _(none)_ | `renderPage() string` — shared render logic |
| `main_cli.go` | `//go:build !js` | CLI entry point — `fmt.Println(renderPage())` |
| `main_wasm.go` | `//go:build js && wasm` | WASM entry point — writes HTML into `document` |
| `index.html` | — | Browser shell that loads `wasm_exec.js` + `main.wasm` |

## Concepts Demonstrated

- Curried element builders (`h.Html()`, `h.Head()`, `h.Body()`, `h.H1()`, `h.P()`)
- `gui.Text()` for text nodes
- `html.Renderer.RenderString()` for string output
- Build-tag separation for CLI vs WASM targets
