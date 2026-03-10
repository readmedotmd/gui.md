# HTML Renderer (`gui/html`)

The `html` package provides an HTML string renderer and a complete set of element builders and attribute helpers for the `gui` library.

## Renderer

`html.Renderer` implements `gui.Renderer`. It resolves components via `gui.Resolve` before traversing, so callers do not need to resolve manually.

```go
r := html.New()

// Write to any io.Writer
err := r.Render(node, os.Stdout)

// Or get a string directly
htmlStr := r.RenderString(node)
```

### Rendering Rules

- **Elements** render as `<tag attrs>children</tag>`.
- **Void elements** (`br`, `hr`, `img`, `input`, `meta`, `link`, `area`, `base`, `col`, `embed`, `param`, `source`, `track`, `wbr`) with no children render as self-closing: `<br />`.
- **Text content** is HTML-escaped (`<`, `>`, `&`, `"` are escaped).
- **Attribute values** are HTML-escaped and wrapped in double quotes.
- **Boolean attributes**: `true` renders as a bare flag (e.g. `disabled`); `false` is omitted entirely.
- **Function-valued props** (event handlers — both `func()` from `OnClick` and `func(gui.Event)` from `On`) are silently skipped in HTML output. They are wired up by interactive backends like the DOM renderer.
- **Props are sorted** alphabetically for deterministic output.
- **Fragments** render their children inline without a wrapper element.
- **nil nodes** produce no output.

## Elements

All elements use the curried `gui.Tag` pattern: `Element(attrs...)(children...)`.

### Block Elements

```go
html.Div     // <div>
html.Section // <section>
html.Article // <article>
html.Header  // <header>
html.Footer  // <footer>
html.Main    // <main>
html.Nav     // <nav>
html.Aside   // <aside>
```

### Headings

```go
html.H1  // <h1>
html.H2  // <h2>
html.H3  // <h3>
html.H4  // <h4>
html.H5  // <h5>
html.H6  // <h6>
```

### Inline Elements

```go
html.Span   // <span>
html.A      // <a>
html.Strong // <strong>
html.Em     // <em>
html.Code   // <code>
html.Pre    // <pre>
```

### Text Content

```go
html.P          // <p>
html.Blockquote // <blockquote>
html.Ul         // <ul>
html.Ol         // <ol>
html.Li         // <li>
```

### Form Elements

```go
html.Form     // <form>
html.Label    // <label>
html.Button   // <button>
html.Select   // <select>
html.Option   // <option>
html.Textarea // <textarea>
html.Input    // <input> (void)
```

### Void / Self-Closing Elements

```go
html.Br   // <br />
html.Hr   // <hr />
html.Img  // <img />
html.Meta // <meta />
html.Link // <link />
```

### Table Elements

```go
html.Table // <table>
html.Thead // <thead>
html.Tbody // <tbody>
html.Tr    // <tr>
html.Th    // <th>
html.Td    // <td>
```

### Document-Level Elements

```go
html.Html    // <html>
html.Head    // <head>
html.Body    // <body>
html.Title   // <title>
html.Script  // <script>
html.StyleEl // <style>
```

## Attribute Helpers

All attribute helpers return `gui.Attr` — a function that sets a prop on `gui.Props`.

| Helper | HTML attribute | Value type |
|--------|---------------|------------|
| `Class(v)` | `class` | `string` |
| `Id(v)` | `id` | `string` |
| `Style(v)` | `style` | `string` |
| `Href(v)` | `href` | `string` |
| `Src(v)` | `src` | `string` |
| `Alt(v)` | `alt` | `string` |
| `Type(v)` | `type` | `string` |
| `Name(v)` | `name` | `string` |
| `Value(v)` | `value` | `string` |
| `Placeholder(v)` | `placeholder` | `string` |
| `Action(v)` | `action` | `string` |
| `Method(v)` | `method` | `string` |
| `Disabled(v)` | `disabled` | `bool` |
| `Checked(v)` | `checked` | `bool` |
| `Data(key, val)` | `data-{key}` | `string` |
| `On(event, fn)` | `on{event}` | `func(gui.Event)` |
| `OnClick(fn)` | `onclick` | `func()` |

### `On(event, handler)` — Rich Event Handlers

Registers a `func(gui.Event)` handler in `Props` under the key `on{event}`. The handler receives event data (mouse coordinates, key name, input value, etc.). These are **omitted from HTML string output** but are wired up by the DOM renderer:

```go
html.On("click", func(e gui.Event) { fmt.Printf("clicked at %d,%d\n", e.X, e.Y) })
// Sets props["onclick"] = func(gui.Event){...}
```

### `OnClick(handler)` — Simple Click Handler

A convenience helper that registers a `func()` handler under the key `onclick`. For handlers that need event data, use `On("click", ...)` instead:

```go
html.OnClick(func() { fmt.Println("clicked") })
// Sets props["onclick"] = func(){...}
```

### `Data(key, value)` — Data Attributes

Sets a `data-*` attribute:

```go
html.Data("user-id", "42")
// Renders as: data-user-id="42"
```

### Escape Hatch

For attributes not covered by the typed helpers, use `gui.Attr_`:

```go
gui.Attr_("aria-label", "Close dialog")
gui.Attr_("role", "button")
```

## Usage Examples

### Basic Page

```go
page := html.Html()(
    html.Head()(
        html.Title()(gui.Text("My Page")),
        html.Meta(gui.Attr_("charset", "utf-8"))(),
    ),
    html.Body()(
        html.H1()(gui.Text("Welcome")),
        html.P()(gui.Text("Hello from NanoGUI.")),
    ),
)

r := html.New()
fmt.Println(r.RenderString(page))
```

### Form

```go
form := html.Form(html.Action("/search"), html.Method("POST"))(
    html.Label()(gui.Text("Query:")),
    html.Input(html.Type("text"), html.Name("q"), html.Placeholder("Search..."))(),
    html.Button(html.Type("submit"))(gui.Text("Go")),
)
// Renders: <form action="/search" method="POST"><label>Query:</label><input name="q" placeholder="Search..." type="text" /><button type="submit">Go</button></form>
```

### Table

```go
table := html.Table()(
    html.Thead()(
        html.Tr()(html.Th()(gui.Text("Name")), html.Th()(gui.Text("Age"))),
    ),
    html.Tbody()(
        html.Tr()(html.Td()(gui.Text("Alice")), html.Td()(gui.Text("30"))),
    ),
)
```

### Multiple Attributes

```go
link := html.A(html.Href("/about"), html.Class("nav-link"), html.Id("about-link"))(
    gui.Text("About"),
)
// Renders: <a class="nav-link" href="/about" id="about-link">About</a>
```

## Source Files

| File | Contents |
|------|----------|
| `html.go` | `Renderer` struct, `New()`, `Render()`, `RenderString()`, rendering logic, void element detection |
| `elements.go` | All element builders (`Div`, `H1`, `Form`, `Input`, etc.) |
| `attrs.go` | Package doc, attribute helpers (`Class`, `Href`, `On`, etc.) |
