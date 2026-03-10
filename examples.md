# Examples

Runnable examples demonstrating different NanoGUI backends and patterns.

| Example | Backend | Interactive | Description |
|---------|---------|-------------|-------------|
| [html_hello](../example_html_hello/) | HTML | No | Minimal HTML page — renders `<html>` with a heading and paragraph |
| [html_page](../example_html_page/) | HTML | No | Functional components, stateful components, and the generic store |
| [dom_counter](../example_dom_counter/) | DOM (WASM) | Yes | Interactive counter with live DOM rendering and click handlers |
| [dom_app](../example_dom_app/) | DOM (WASM) | Yes | Multi-page SPA with hash router, functional + stateful components, form inputs, and CSS styling |
| [web_input](../example_web_input/) | HTML (HTTP) | Yes | HTTP server with a form — re-renders on POST with live preview |

## Running Examples

Each example is its own Go module. Run from within its directory.

### HTML examples (CLI)

Print rendered HTML to stdout:

```bash
cd gui/example_html_hello && go run .
cd gui/example_html_page && go run .
```

### HTML/DOM examples (Browser via WebAssembly)

Each WASM example has its own `run.sh`. Run from within the example directory:

```bash
cd gui/example_html_hello && ./run.sh
cd gui/example_html_page && ./run.sh
cd gui/example_dom_counter && ./run.sh
cd gui/example_dom_app && ./run.sh
# Open http://localhost:9090
```

The run script:
1. Copies `wasm_exec.js` from your Go installation into the example directory
2. Cross-compiles the example to `main.wasm` (`GOOS=js GOARCH=wasm`)
3. Starts a local HTTP server on port 9090

### Web server example

```bash
cd gui/example_web_input && ./run.sh
# Open http://localhost:9090
```

---

## How WASM Support Works

The HTML examples (`html_hello`, `html_page`) support both CLI and browser targets from the same package using Go build tags:

```
example_html_hello/
├── page.go          # Shared renderPage() function (no build tag)
├── main_cli.go      # //go:build !js — prints to stdout
├── main_wasm.go     # //go:build js && wasm — writes to document
└── index.html       # Browser shell that loads wasm_exec.js + main.wasm
```

- **`page.go`** contains all the rendering logic in a `renderPage() string` function, with no build constraints. Both entry points call it.
- **`main_cli.go`** has `//go:build !js` and simply prints the result with `fmt.Println`.
- **`main_wasm.go`** has `//go:build js && wasm` and uses `syscall/js` to call `document.open()`, `document.write(html)`, `document.close()`, replacing the entire browser document with the rendered output.
- **`index.html`** is a minimal shell that loads Go's `wasm_exec.js` support file and the compiled `main.wasm` binary.

Each example has its own `go.mod`. The `go.work` file at the monorepo root handles local module resolution.
