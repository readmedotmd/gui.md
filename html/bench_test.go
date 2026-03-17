package html

import (
	"io"
	"strings"
	"testing"

	gui "github.com/readmedotmd/gui.md"
)

// ============================================================================
// Tree construction helpers
// ============================================================================

func flatTree(n int) gui.Node {
	children := make([]gui.Node, n)
	for i := range children {
		children[i] = gui.Text("item")
	}
	return gui.El("div", gui.Props{"class": "container"}, children...)
}

func deepTree(depth int) gui.Node {
	node := gui.Node(gui.Text("leaf"))
	for i := 0; i < depth; i++ {
		node = gui.El("div", gui.Props{"class": "level"}, node)
	}
	return node
}

func wideTree(w, c int) gui.Node {
	children := make([]gui.Node, w)
	for i := range children {
		gc := make([]gui.Node, c)
		for j := range gc {
			gc[j] = gui.Text("cell")
		}
		children[i] = gui.El("div", gui.Props{"class": "row"}, gc...)
	}
	return gui.El("div", gui.Props{"class": "grid"}, children...)
}

func propsHeavyTree(attrCount int) gui.Node {
	props := gui.Props{}
	for i := 0; i < attrCount; i++ {
		props[gui.Textf("data-attr-%d", i).Content] = gui.Textf("value-%d", i).Content
	}
	return gui.El("div", props, gui.Text("content"))
}

func realisticPage(cards int) gui.Node {
	navLinks := make([]gui.Node, 5)
	for i := range navLinks {
		navLinks[i] = gui.El("a", gui.Props{"href": "#", "class": "nav-link"}, gui.Textf("Link %d", i))
	}
	nav := gui.El("nav", gui.Props{"class": "navbar"}, navLinks...)

	cardNodes := make([]gui.Node, cards)
	for i := range cardNodes {
		cardNodes[i] = gui.El("div", gui.Props{"class": "card"},
			gui.El("h2", nil, gui.Textf("Card %d", i)),
			gui.El("p", nil, gui.Text("Lorem ipsum dolor sit amet, consectetur adipiscing elit.")),
			gui.El("button", gui.Props{"class": "btn"}, gui.Text("Click me")),
		)
	}
	main := gui.El("main", gui.Props{"class": "content"}, cardNodes...)
	footer := gui.El("footer", gui.Props{"class": "footer"}, gui.Text("Built with gui.md"))

	return gui.El("html", nil,
		gui.El("head", nil,
			gui.El("title", nil, gui.Text("Benchmark Page")),
			gui.El("meta", gui.Props{"charset": "utf-8"}),
		),
		gui.El("body", nil, nav, main, footer),
	)
}

// ============================================================================
// Render benchmarks — RenderString (allocates result string)
// ============================================================================

func BenchmarkRenderString_Text(b *testing.B) {
	r := New()
	node := gui.Text("Hello, World!")
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_SingleElement(b *testing.B) {
	r := New()
	node := gui.El("div", gui.Props{"class": "container", "id": "main"}, gui.Text("content"))
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_VoidElement(b *testing.B) {
	r := New()
	node := gui.El("img", gui.Props{"src": "photo.jpg", "alt": "Photo"})
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Flat10(b *testing.B) {
	r := New()
	node := flatTree(10)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Flat100(b *testing.B) {
	r := New()
	node := flatTree(100)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Flat1000(b *testing.B) {
	r := New()
	node := flatTree(1000)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Deep10(b *testing.B) {
	r := New()
	node := deepTree(10)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Deep50(b *testing.B) {
	r := New()
	node := deepTree(50)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Deep200(b *testing.B) {
	r := New()
	node := deepTree(200)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Wide20x10(b *testing.B) {
	r := New()
	node := wideTree(20, 10)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Props5(b *testing.B) {
	r := New()
	node := propsHeavyTree(5)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Props20(b *testing.B) {
	r := New()
	node := propsHeavyTree(20)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Props50(b *testing.B) {
	r := New()
	node := propsHeavyTree(50)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Realistic10(b *testing.B) {
	r := New()
	node := realisticPage(10)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Realistic50(b *testing.B) {
	r := New()
	node := realisticPage(50)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_Realistic100(b *testing.B) {
	r := New()
	node := realisticPage(100)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

// ============================================================================
// Render benchmarks — Render to io.Writer (no string allocation)
// ============================================================================

func BenchmarkRender_Discard_Realistic50(b *testing.B) {
	r := New()
	node := realisticPage(50)
	b.ResetTimer()
	for b.Loop() {
		r.Render(node, io.Discard)
	}
}

func BenchmarkRender_Discard_Flat1000(b *testing.B) {
	r := New()
	node := flatTree(1000)
	b.ResetTimer()
	for b.Loop() {
		r.Render(node, io.Discard)
	}
}

func BenchmarkRender_Buffer_Realistic50(b *testing.B) {
	r := New()
	node := realisticPage(50)
	var buf strings.Builder
	b.ResetTimer()
	for b.Loop() {
		buf.Reset()
		r.Render(node, &buf)
	}
}

// ============================================================================
// Component resolution + render (end-to-end)
// ============================================================================

type benchCard struct {
	gui.BaseComponent[gui.Props, struct{}]
}

func (c *benchCard) Render() gui.Node {
	return gui.El("div", gui.Props{"class": "card"},
		gui.El("h2", nil, gui.Text("Title")),
		gui.El("p", nil, gui.Text("Body")),
	)
}

func BenchmarkRenderString_WithComponents(b *testing.B) {
	r := New()
	children := make([]gui.Node, 20)
	for i := range children {
		children[i] = gui.C(new(benchCard), nil)
	}
	node := gui.El("div", nil, children...)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkRenderString_FuncComponents(b *testing.B) {
	r := New()
	type CardProps struct{ Title string }
	card := func(p CardProps, _ []gui.Node) gui.Node {
		return gui.El("div", gui.Props{"class": "card"},
			gui.El("h2", nil, gui.Text(p.Title)),
			gui.El("p", nil, gui.Text("Body")),
		)
	}
	children := make([]gui.Node, 20)
	for i := range children {
		children[i] = gui.Comp(card, CardProps{Title: gui.Textf("Card %d", i).Content})
	}
	node := gui.El("div", nil, children...)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

// ============================================================================
// Allocation-focused
// ============================================================================

func BenchmarkAlloc_RenderString_Small(b *testing.B) {
	r := New()
	node := gui.El("div", gui.Props{"class": "x"}, gui.Text("hello"))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkAlloc_RenderString_Realistic(b *testing.B) {
	r := New()
	node := realisticPage(10)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

func BenchmarkAlloc_Render_Discard(b *testing.B) {
	r := New()
	node := realisticPage(10)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		r.Render(node, io.Discard)
	}
}

// ============================================================================
// HTML escaping (exercises the stdhtml.EscapeString path)
// ============================================================================

func BenchmarkRenderString_HtmlEscaping(b *testing.B) {
	r := New()
	node := gui.El("div", nil,
		gui.Text(`<script>alert("xss")</script>`),
		gui.Text(`"quotes" & <angles> & 'apostrophes'`),
		gui.El("a", gui.Props{"href": `https://example.com?a=1&b=2&c="3"`}, gui.Text("link")),
	)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}

// ============================================================================
// Fragment rendering
// ============================================================================

func BenchmarkRenderString_Fragment(b *testing.B) {
	r := New()
	children := make([]gui.Node, 50)
	for i := range children {
		children[i] = gui.El("li", nil, gui.Textf("item %d", i))
	}
	node := gui.Frag(children...)
	b.ResetTimer()
	for b.Loop() {
		r.RenderString(node)
	}
}
