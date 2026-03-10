# html_page

Demonstrates functional components, stateful components, and the generic store — all rendered as a static HTML page.

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

## File Structure

| File | Build constraint | Purpose |
|------|-----------------|---------|
| `page.go` | _(none)_ | Types, components, and `renderPage() string` |
| `main_cli.go` | `//go:build !js` | CLI entry point — `fmt.Println(renderPage())` |
| `main_wasm.go` | `//go:build js && wasm` | WASM entry point — writes HTML into `document` |
| `index.html` | — | Browser shell that loads `wasm_exec.js` + `main.wasm` |

## Concepts Demonstrated

- **Functional components** — `NavBar` takes props and children, returns a node tree
- **Stateful components** — `UserCard` embeds `gui.BaseComponent[CardState]` with typed state
- **`gui.Comp()`** — wraps a functional component for use in the tree
- **`gui.Mount()`** — wraps a stateful component, initializing props and children
- **`gui.NewStore()`** — creates a typed global store (`PageState` with theme and language)
- **Store-driven rendering** — store state is read and used to set element attributes (e.g. `h.Class(state.Theme)`)
- **Build-tag separation** for CLI vs WASM targets
