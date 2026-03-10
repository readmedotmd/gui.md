package html_test

import (
	gui "github.com/readmedotmd/gui.md"
	"github.com/readmedotmd/gui.md/html"
	"strings"
	"testing"
)

// Compile-time interface satisfaction check.
// If Renderer stops implementing gui.Renderer the build fails with a clear
// error rather than a confusing runtime panic.
var _ gui.Renderer = (*html.Renderer)(nil)

// newR is a test helper that avoids repeating html.New() everywhere.
func newR() *html.Renderer { return html.New() }

// ---- helpers ----------------------------------------------------------------

func renderStr(t *testing.T, node gui.Node) string {
	t.Helper()
	return newR().RenderString(node)
}

func assertEq(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("\ngot:  %s\nwant: %s", got, want)
	}
}

// ---- tests ------------------------------------------------------------------

// 1. Empty element produces opening and closing tags.
func TestEmptyDiv(t *testing.T) {
	assertEq(t, renderStr(t, gui.El("div", nil)), "<div></div>")
}

// 2. Element with text child.
func TestParagraphWithText(t *testing.T) {
	assertEq(t, renderStr(t, gui.El("p", nil, gui.Text("hello"))), "<p>hello</p>")
}

// 3. Props are rendered as sorted attributes.
func TestSortedAttrs(t *testing.T) {
	node := gui.El("a", gui.Props{"href": "/", "class": "link"})
	assertEq(t, renderStr(t, node), `<a class="link" href="/"></a>`)
}

// 4. Nested elements produce correct tree output.
func TestNestedElements(t *testing.T) {
	node := gui.El("div", nil,
		gui.El("h1", nil, gui.Text("Title")),
		gui.El("p", nil, gui.Text("Body")),
	)
	want := "<div><h1>Title</h1><p>Body</p></div>"
	assertEq(t, renderStr(t, node), want)
}

// 5. Fragment renders children inline without a wrapper element.
func TestFragment(t *testing.T) {
	node := gui.Frag(
		gui.El("span", nil, gui.Text("a")),
		gui.El("span", nil, gui.Text("b")),
	)
	assertEq(t, renderStr(t, node), "<span>a</span><span>b</span>")
}

// 6. Void elements are self-closing.
func TestVoidElementSelfClose(t *testing.T) {
	assertEq(t, renderStr(t, gui.El("br", nil)), "<br />")
	assertEq(t, renderStr(t, gui.El("hr", nil)), "<hr />")
	assertEq(t, renderStr(t, gui.El("img", gui.Props{"src": "logo.png"})), `<img src="logo.png" />`)
	assertEq(t, renderStr(t, gui.El("input", gui.Props{"type": "text"})), `<input type="text" />`)
}

// 7. Boolean true prop renders as bare attribute flag.
func TestBooleanTrueProp(t *testing.T) {
	node := gui.El("input", gui.Props{"disabled": true, "type": "text"})
	got := renderStr(t, node)
	// Both attributes must appear; order is deterministic (sorted).
	if !strings.Contains(got, "disabled") {
		t.Errorf("expected 'disabled' flag in %q", got)
	}
	if strings.Contains(got, `disabled="`) {
		t.Errorf("boolean 'disabled' must not have a value, got %q", got)
	}
	assertEq(t, got, `<input disabled type="text" />`)
}

// 8. Boolean false prop is omitted from output.
func TestBooleanFalsePropOmitted(t *testing.T) {
	node := gui.El("input", gui.Props{"disabled": false, "type": "checkbox"})
	got := renderStr(t, node)
	if strings.Contains(got, "disabled") {
		t.Errorf("false boolean prop 'disabled' must not appear in %q", got)
	}
	assertEq(t, got, `<input type="checkbox" />`)
}

// 9. HTML special characters in text content are escaped.
func TestTextEscaping(t *testing.T) {
	node := gui.El("p", nil, gui.Text("<script>alert(1)</script>"))
	want := "<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>"
	assertEq(t, renderStr(t, node), want)
}

// 10. HTML special characters in attribute values are escaped.
func TestAttrEscaping(t *testing.T) {
	node := gui.El("a", gui.Props{"href": `/?q=<foo>&bar="baz"`})
	got := renderStr(t, node)
	if strings.Contains(got, "<foo>") {
		t.Errorf("unescaped '<' in attribute: %q", got)
	}
	want := `<a href="/?q=&lt;foo&gt;&amp;bar=&#34;baz&#34;"></a>`
	assertEq(t, got, want)
}

// 11. Function-valued props (event handlers) are omitted from HTML output.
func TestEventHandlerOmitted(t *testing.T) {
	handler := func() {}
	node := gui.El("button", gui.Props{"onclick": handler, "class": "btn"})
	got := renderStr(t, node)
	if strings.Contains(got, "onclick") {
		t.Errorf("event handler prop must not appear in HTML: %q", got)
	}
	assertEq(t, got, `<button class="btn"></button>`)
}

// 12. Functional component is resolved and rendered.
func TestFunctionalComponentResolved(t *testing.T) {
	greeting := func(props gui.Props, _ []gui.Node) gui.Node {
		name, _ := props["name"].(string)
		return gui.El("p", nil, gui.Textf("Hello, %s!", name))
	}
	node := gui.Comp(greeting, gui.Props{"name": "World"})
	assertEq(t, renderStr(t, node), "<p>Hello, World!</p>")
}

// 13. Stateful component is resolved and rendered.
type counterState struct{ Count int }

type counterComp struct {
	gui.BaseComponent[gui.Props, counterState]
}

func (c *counterComp) Render() gui.Node {
	return gui.El("span", nil, gui.Textf("Count: %d", c.State().Count))
}

func TestStatefulComponentResolved(t *testing.T) {
	c := &counterComp{}
	c.SetState(counterState{Count: 7})
	node := gui.Mount(c, nil)
	assertEq(t, renderStr(t, node), "<span>Count: 7</span>")
}

// 14. RenderString is a convenience wrapper that returns a string.
func TestRenderString(t *testing.T) {
	r := html.New()
	got := r.RenderString(gui.El("em", nil, gui.Text("italic")))
	assertEq(t, got, "<em>italic</em>")
}

// 15. Curried API: Div(Class("x"))(P()(gui.Text("hi"))) → correct HTML.
func TestCurriedAPI(t *testing.T) {
	node := gui.Div(gui.Class("x"))(
		gui.P()(gui.Text("hi")),
	)
	assertEq(t, renderStr(t, node), `<div class="x"><p>hi</p></div>`)
}

// 16. Nil node renders as empty string.
func TestNilNodeRendersEmpty(t *testing.T) {
	assertEq(t, renderStr(t, nil), "")
}

// 17. Textf formatted text nodes render correctly.
func TestTextfFormatted(t *testing.T) {
	node := gui.El("p", nil, gui.Textf("items: %d", 42))
	assertEq(t, renderStr(t, node), "<p>items: 42</p>")
}

// 18. Form with Input renders correct HTML.
func TestFormWithInput(t *testing.T) {
	node := gui.Form(gui.Action("/search"), gui.Method("POST"))(
		gui.Input(gui.Type("text"), gui.Name("q"))(),
	)
	want := `<form action="/search" method="POST"><input name="q" type="text" /></form>`
	assertEq(t, renderStr(t, node), want)
}

// 19. Deeply nested tree (5+ levels) renders without error.
func TestDeeplyNested(t *testing.T) {
	node := gui.Div()(
		gui.Section()(
			gui.Article()(
				gui.Header()(
					gui.H1()(
						gui.Text("deep title"),
					),
				),
				gui.Main()(
					gui.P()(gui.Text("deep body")),
				),
			),
		),
	)
	got := renderStr(t, node)
	want := "<div><section><article><header><h1>deep title</h1></header><main><p>deep body</p></main></article></section></div>"
	assertEq(t, got, want)
}

// ---- additional element / attr coverage ------------------------------------

// TestAllVoidElements verifies each tag in the void set self-closes.
func TestAllVoidElements(t *testing.T) {
	voids := []string{"area", "base", "br", "col", "embed", "hr", "img",
		"input", "link", "meta", "param", "source", "track", "wbr"}
	for _, tag := range voids {
		t.Run(tag, func(t *testing.T) {
			got := renderStr(t, gui.El(tag, nil))
			want := "<" + tag + " />"
			assertEq(t, got, want)
		})
	}
}

// TestAttrHelpers verifies each gui.Attr* helper sets the correct prop key.
func TestAttrHelpers(t *testing.T) {
	tests := []struct {
		name string
		attr gui.Attr
		key  string
		want any
	}{
		{"Class", gui.Class("btn"), "class", "btn"},
		{"Id", gui.Id("main"), "id", "main"},
		{"Style", gui.Style("color:red"), "style", "color:red"},
		{"Href", gui.Href("/home"), "href", "/home"},
		{"Src", gui.Src("/img.png"), "src", "/img.png"},
		{"Alt", gui.Alt("logo"), "alt", "logo"},
		{"Type", gui.Type("submit"), "type", "submit"},
		{"Name", gui.Name("q"), "name", "q"},
		{"Value", gui.Value("42"), "value", "42"},
		{"Placeholder", gui.Placeholder("Search…"), "placeholder", "Search…"},
		{"Action", gui.Action("/api"), "action", "/api"},
		{"Method", gui.Method("GET"), "method", "GET"},
		{"Disabled true", gui.Disabled(true), "disabled", true},
		{"Disabled false", gui.Disabled(false), "disabled", false},
		{"Checked true", gui.Checked(true), "checked", true},
		{"Data", gui.Data("user-id", "7"), "data-user-id", "7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := gui.Props{}
			tt.attr(p)
			got, ok := p[tt.key]
			if !ok {
				t.Fatalf("key %q not set in props", tt.key)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOnEventHandler verifies On stores a func(gui.Event) under the correct key.
func TestOnEventHandler(t *testing.T) {
	var received gui.Event
	handler := func(e gui.Event) { received = e }
	p := gui.Props{}
	gui.On("click", handler)(p)

	fn, ok := p["onclick"]
	if !ok {
		t.Fatal("expected 'onclick' key in props")
	}
	f, ok := fn.(func(gui.Event))
	if !ok {
		t.Fatalf("expected func(gui.Event), got %T", fn)
	}
	f(gui.Event{Type: "click", X: 5, Y: 10})
	if received.Type != "click" {
		t.Errorf("expected event Type=%q, got %q", "click", received.Type)
	}
	if received.X != 5 || received.Y != 10 {
		t.Errorf("expected X=5 Y=10, got X=%d Y=%d", received.X, received.Y)
	}
}

// TestOnClickHelper verifies OnClick stores a func() under "onclick".
func TestOnClickHelper(t *testing.T) {
	called := false
	handler := func() { called = true }
	p := gui.Props{}
	gui.OnClick(handler)(p)

	fn, ok := p["onclick"]
	if !ok {
		t.Fatal("expected 'onclick' key in props")
	}
	f, ok := fn.(func())
	if !ok {
		t.Fatalf("expected func(), got %T", fn)
	}
	f()
	if !called {
		t.Error("handler was not called")
	}
}

// TestEventHandlerFuncEventOmitted verifies that func(gui.Event) props are
// omitted from HTML output, just like func() props.
func TestEventHandlerFuncEventOmitted(t *testing.T) {
	handler := func(gui.Event) {}
	node := gui.El("button", gui.Props{"onclick": handler, "class": "btn"})
	got := renderStr(t, node)
	if strings.Contains(got, "onclick") {
		t.Errorf("func(gui.Event) handler prop must not appear in HTML: %q", got)
	}
	assertEq(t, got, `<button class="btn"></button>`)
}

// TestRenderToWriter verifies the io.Writer path (Render) works correctly.
func TestRenderToWriter(t *testing.T) {
	var buf strings.Builder
	r := html.New()
	if err := r.Render(gui.El("div", nil, gui.Text("hello")), &buf); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	assertEq(t, buf.String(), "<div>hello</div>")
}

// TestFragmentAtRoot verifies a Fragment as the root node renders inline.
func TestFragmentAtRoot(t *testing.T) {
	node := gui.Frag(
		gui.El("h1", nil, gui.Text("one")),
		gui.El("h2", nil, gui.Text("two")),
		gui.El("h3", nil, gui.Text("three")),
	)
	assertEq(t, renderStr(t, node), "<h1>one</h1><h2>two</h2><h3>three</h3>")
}

// TestTableStructure verifies table-related elements render in order.
func TestTableStructure(t *testing.T) {
	node := gui.Table()(
		gui.Thead()(
			gui.Tr()(
				gui.Th()(gui.Text("Name")),
				gui.Th()(gui.Text("Age")),
			),
		),
		gui.Tbody()(
			gui.Tr()(
				gui.Td()(gui.Text("Alice")),
				gui.Td()(gui.Text("30")),
			),
		),
	)
	want := "<table><thead><tr><th>Name</th><th>Age</th></tr></thead>" +
		"<tbody><tr><td>Alice</td><td>30</td></tr></tbody></table>"
	assertEq(t, renderStr(t, node), want)
}

// TestMaxDepthReturnsError verifies that excessively deep trees return ErrMaxDepth.
func TestMaxDepthReturnsError(t *testing.T) {
	// Build a tree deeper than 512 levels.
	var node gui.Node = gui.Text("leaf")
	for i := 0; i < 600; i++ {
		node = gui.El("div", nil, node)
	}
	r := html.New()
	var buf strings.Builder
	err := r.Render(node, &buf)
	if err != html.ErrMaxDepth {
		t.Errorf("expected ErrMaxDepth, got: %v", err)
	}
}

// TestMaxDepthAllowsReasonableNesting verifies that normal trees render fine.
func TestMaxDepthAllowsReasonableNesting(t *testing.T) {
	var node gui.Node = gui.Text("leaf")
	for i := 0; i < 100; i++ {
		node = gui.El("div", nil, node)
	}
	r := html.New()
	got := r.RenderString(node)
	if !strings.Contains(got, "leaf") {
		t.Error("expected leaf text in output")
	}
}

// TestMultipleAttrsSameElement verifies multiple attrs compose correctly.
func TestMultipleAttrsSameElement(t *testing.T) {
	node := gui.A(gui.Href("/about"), gui.Class("nav-link"), gui.Id("about-link"))(
		gui.Text("About"),
	)
	got := renderStr(t, node)
	want := `<a class="nav-link" href="/about" id="about-link">About</a>`
	assertEq(t, got, want)
}
