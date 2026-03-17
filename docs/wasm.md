# WebAssembly Guide

NanoGUI's HTML examples support both CLI and browser execution from the same Go package using build tags. This guide explains the architecture, how to add WASM support to your own examples, and how the run script works.

## Architecture

Each WASM-capable example has four files:

```
example_html_hello/
├── page.go          # Shared render logic (no build tag)
├── main_cli.go      # //go:build !js — CLI entry point
├── main_wasm.go     # //go:build js && wasm — WASM entry point
└── index.html       # Browser shell
```

### `page.go` — Shared Logic

Contains all the rendering code in a `renderPage() string` function. No build constraints — compiled for both targets.

```go
package main

import (
    "gui"
    h "gui/html"
)

func renderPage() string {
    page := h.Html()(
        h.Body()(
            h.H1()(gui.Text("Hello from NanoGUI!")),
        ),
    )
    r := h.New()
    return r.RenderString(page)
}
```

### `main_cli.go` — CLI Entry Point

Has `//go:build !js` so it is excluded when compiling for `GOOS=js`. Simply prints the rendered HTML to stdout.

```go
//go:build !js

package main

import "fmt"

func main() {
    fmt.Println(renderPage())
}
```

### `main_wasm.go` — WASM Entry Point

Has `//go:build js && wasm` so it is only compiled for the WASM target. Uses `syscall/js` to replace the entire browser document with the rendered HTML.

```go
//go:build js && wasm

package main

import "syscall/js"

func main() {
    html := renderPage()
    doc := js.Global().Get("document")
    doc.Call("open")
    doc.Call("write", html)
    doc.Call("close")
}
```

The `document.open()` / `document.write()` / `document.close()` pattern replaces the full document, including the `<html>` element. This means the `index.html` shell is completely replaced by the Go-rendered HTML once the WASM module loads.

### `index.html` — Browser Shell

A minimal HTML page that loads Go's `wasm_exec.js` runtime support and the compiled `main.wasm` binary:

```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>NanoGUI Example</title>
    <script src="wasm_exec.js"></script>
    <script>
        const go = new Go();
        WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
            .then(result => {
                if (document.readyState === "loading") {
                    document.addEventListener("DOMContentLoaded", () => go.run(result.instance));
                } else {
                    go.run(result.instance);
                }
            });
    </script>
</head>
<body>Loading...</body>
</html>
```

The `readyState` check handles the case where `DOMContentLoaded` has already fired by the time WASM compilation finishes. Without this guard, `document.getElementById("app")` may return null and crash the module.

The "Loading..." text is visible until the WASM module finishes loading and replaces the document.

## Build Tags

The build tags ensure clean separation:

| File | Normal build (`go build`) | WASM build (`GOOS=js GOARCH=wasm`) |
|------|--------------------------|-------------------------------------|
| `page.go` | Included | Included |
| `main_cli.go` (`!js`) | Included | **Excluded** |
| `main_wasm.go` (`js && wasm`) | **Excluded** | Included |

This means:
- `go build .` and `go test .` work normally from within each example directory — WASM files are excluded
- `GOOS=js GOARCH=wasm go build` compiles only `page.go` + `main_wasm.go`
- Both targets call the same `renderPage()` function

## Building Manually

```bash
# From within an example directory:
cd example_html_hello

# Build the WASM binary
GOOS=js GOARCH=wasm go build -o main.wasm .

# Copy the Go WASM support file (Go ≤1.23)
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .
# Go 1.24+ moved the file to lib/wasm/:
# cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" .

# Serve the directory (any HTTP server works)
python3 -m http.server 9090
```

Then open http://localhost:9090 in a browser.

## Run Script

Each WASM example has its own `run.sh` that automates all three steps:

```bash
cd example_html_hello && ./run.sh
```

What it does:

1. **Copies `wasm_exec.js`** from `$(go env GOROOT)/lib/wasm/wasm_exec.js` (or `misc/wasm/`) into the example directory
2. **Builds the WASM binary** with `GOOS=js GOARCH=wasm go build -o main.wasm .`
3. **Starts a local HTTP server** on port 9090 with `python3 -m http.server`

## Adding WASM Support to a New Example

1. **Create a new directory** `example_your_name/` with its own `go.mod`:
   ```
   module example_your_name

   go 1.23.6

   require gui v0.0.0
   ```
2. **Add the module to `go.work`** at the monorepo root
3. **Extract render logic** into `page.go` with a `renderPage() string` function
4. **Create `main_cli.go`** with `//go:build !js` and the existing `main()` that prints to stdout
5. **Create `main_wasm.go`** with `//go:build js && wasm` using the `syscall/js` pattern above
6. **Create `index.html`** using the shell template above
7. **Copy `run.sh`** from another WASM example
8. **Test both targets**:
   ```bash
   cd example_your_name
   go run .     # CLI
   ./run.sh     # Browser
   ```

## `wasm_exec.js`

This file is part of the Go standard distribution and provides the JavaScript runtime support for Go WASM binaries. It defines the `Go` class used to instantiate and run the WASM module. It is located at `$(go env GOROOT)/misc/wasm/wasm_exec.js`.

The run script copies it into the example directory at build time. It is `.gitignore`-able since it's always copied from the Go installation.

## DOM Renderer (`gui/dom`)

The `dom` package provides a live DOM renderer that creates real browser elements, wires event listeners, and supports incremental updates via the diff engine. It is build-tagged `js && wasm`.

### Basic Usage

```go
import "gui/dom"

container := js.Global().Get("document").Call("getElementById", "app")
renderer := dom.New(container)
defer renderer.Release()

renderer.Update(func() gui.Node {
    return html.Div()(gui.Text("Hello from DOM renderer!"))
})
```

### Event Handling

The DOM renderer automatically wires `on*` props as browser event listeners:

```go
html.On("click", func(e gui.Event) {
    fmt.Printf("Clicked at %d,%d\n", e.X, e.Y)
})
```

Event data extraction:
- **click/mouse events**: `clientX`, `clientY` → `Event.X`, `Event.Y`
- **keyboard events**: `key` → `Event.Key`
- **input/change events**: `target.value` → `Event.Value`

### Incremental Updates

Call `renderer.Update(rootFn)` after state changes. The renderer diffs the new tree against the previous one and applies minimal DOM patches:

```go
store.Subscribe(func(_, _ MyState) {
    renderer.Update(func() gui.Node {
        return buildUI(store.Get())
    })
})
```

### Memory Management

Call `renderer.Release()` when done to free all JS callback functions and prevent memory leaks.

## Router (`gui/dom`)

The `dom` package includes a full-featured router for single-page WASM applications. It supports:

- **Hash mode** (default) — uses `location.hash` and `hashchange` events (`/#/about`)
- **History API mode** — uses `pushState`/`popstate` for clean URLs (`/about`)
- **Route pattern matching** — named parameters (`:id`) and wildcards (`*path`)
- **Nested routes with layouts** — shared layout wrappers for route groups
- **Navigation guards** — global and per-route guards that can cancel navigation

### Basic Usage (Manual Matching)

The simplest usage works just like before — no route table, you match manually:

```go
router := dom.NewRouter()
defer router.Release()

router.Subscribe(func(route, prevRoute string) {
    fmt.Println("navigated from", prevRoute, "to", route)
})

router.Navigate("/about")
```

### Declarative Routes

Define routes with pattern matching and the router handles matching automatically:

```go
router := dom.NewRouter(
    dom.WithRoutes(
        dom.Route("/", homePage),
        dom.Route("/about", aboutPage),
        dom.Route("/user/:id", userPage),
        dom.Route("/files/*path", filesPage),
    ),
)
defer router.Release()

// In your render function:
page := router.Render() // returns matched page Node, or nil
```

Route handlers receive extracted parameters:

```go
func userPage(params gui.Params) gui.Node {
    id := params["id"] // "42" for /user/42
    return gui.Div()(gui.Textf("User %s", id))
}
```

### Nested Routes & Layouts

Use `RouteWithLayout` to wrap child routes in a shared layout. The layout receives the matched child as an "outlet":

```go
router := dom.NewRouter(
    dom.WithRoutes(
        dom.RouteWithLayout("/", appLayout,
            dom.Route("", homePage),          // matches /
            dom.Route("/about", aboutPage),   // matches /about
            dom.Route("/user/:id", userPage), // matches /user/42
        ),
    ),
)

func appLayout(outlet gui.Node) gui.Node {
    return gui.Div(gui.Class("app"))(
        NavBar(),
        outlet,    // matched child page renders here
        Footer(),
    )
}
```

Layouts can be nested. Each level wraps its children:

```go
dom.RouteWithLayout("/app", appLayout,
    dom.RouteWithLayout("/dashboard", dashLayout,
        dom.Route("", overview),
        dom.Route("/settings", settings),
    ),
)
// /app/dashboard/settings → appLayout(dashLayout(settings()))
```

### Navigation Guards

Guards run before navigation is committed. Return `false` to cancel.

**Global guards** (checked on every navigation):

```go
router := dom.NewRouter(
    dom.BeforeEach(func(from, to string) bool {
        if to == "/admin" && !isAuthenticated() {
            return false // cancel navigation
        }
        return true
    }),
    dom.WithRoutes(...),
)
```

**Per-route guards** (checked only for that route and its children):

```go
gui.RouteConfig{
    Path:    "/admin",
    Handler: adminPage,
    Guards:  []gui.RouteGuard{requireAuth},
}
```

Guards are inherited — a parent route's guards also protect its children.

### History API Mode

Use `WithHistoryMode()` for clean URLs without hash fragments. Requires server-side configuration to serve `index.html` for all routes.

```go
router := dom.NewRouter(
    dom.WithHistoryMode(),
    dom.WithRoutes(...),
)
// URLs: /about instead of /#/about
```

### API

| Method | Description |
|--------|-------------|
| `NewRouter(opts ...RouterOption) *Router` | Creates a router with optional configuration |
| `Route() string` | Returns the current path (e.g. `"/user/42"`) |
| `Navigate(path string)` | Navigates to path (guards are checked first) |
| `Params() gui.Params` | Returns extracted params for the current route |
| `Match() *gui.RouteMatch` | Returns the full match (params, handler, layouts, guards) |
| `Render() gui.Node` | Matches current path, runs handler, applies layouts — returns the page Node |
| `Subscribe(fn) func()` | Registers a route-change callback; returns unsubscribe |
| `Release()` | Removes event listener and frees JS callback |

**Options:**

| Option | Description |
|--------|-------------|
| `WithRoutes(routes ...gui.RouteConfig)` | Sets the declarative route table |
| `WithHistoryMode()` | Uses History API instead of hash fragments |
| `BeforeEach(guard)` | Registers a global navigation guard |

**Route builders:**

| Function | Description |
|----------|-------------|
| `dom.Route(path, handler, children...)` | Creates a route with a handler |
| `dom.RouteWithLayout(path, layout, children...)` | Creates a layout-only route |
| `dom.RouteWithGuards(path, handler, guards, children...)` | Creates a route with guards |

### Route Matching (Pure Go)

The route matching engine lives in the `gui` package (`route.go`) and is usable outside of WASM:

```go
routes := []gui.RouteConfig{
    {Path: "/user/:id", Handler: userPage},
    {Path: "/files/*path", Handler: filesPage},
}

m := gui.MatchRoute(routes, "/user/42")
// m.Params["id"] == "42"

node := gui.RenderMatch(m) // runs handler + layouts

ok := gui.CheckGuards(m, "/", "/user/42") // runs guard chain
```

### Wiring with a Store

Sync the router into your app store so a single `Subscribe` drives re-renders:

```go
store := gui.NewStore(AppState{Route: router.Route()})

router.Subscribe(func(route, _ string) {
    store.Set(AppState{Route: route})
})

store.Subscribe(func(_, _ AppState) { app.Render() })
```

See `example_dom_app/` for a complete example using declarative routes, params, layouts, and guards.

## Limitations

- The static WASM examples (`html_hello`, `html_page`) use `document.write()` and render once. For interactive apps, use the DOM renderer (`gui/dom`) instead.
- History API mode requires server-side configuration to serve `index.html` for all routes (standard SPA setup).
