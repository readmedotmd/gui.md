# web_input

HTTP server demonstrating server-side rendering with NanoGUI. Serves a form input — on submit, re-renders the page with the input shown in a preview div.

## Run

```bash
go run .
# Open http://localhost:9090
```

Or use the convenience script:

```bash
./run.sh
```

## How It Works

1. `GET /` renders the page with the current store state (initially empty input)
2. User types in the text field and submits the form
3. `POST /` updates the store with the form value, then re-renders
4. The preview section shows the submitted text, its length, and word count

This is a traditional server-rendered form — no JavaScript, no WASM. Each submit is a full page reload with the updated state.

## Concepts Demonstrated

- **`gui.NewStore()`** — holds `FormState` with the current input value
- **`store.Set()`** — replaces state on form submission
- **`html.Renderer.RenderString()`** — renders the full page to an HTML string on each request
- **Server-side rendering** — the HTML renderer produces the response body directly
- **Form elements** — `h.Form()`, `h.Input()`, `h.Button()`, `h.Label()` with typed attributes
- **Conditional rendering** — preview text changes based on whether input is empty
