package gui

import (
	"testing"
)

// ============================================================================
// Tree construction helpers — shared across benchmarks
// ============================================================================

// flatTree builds a flat div with n text children.
func flatTree(n int) Node {
	children := make([]Node, n)
	for i := range children {
		children[i] = Text("item")
	}
	return El("div", Props{"class": "container"}, children...)
}

// deepTree builds a chain of nested divs, depth levels deep, with a text leaf.
func deepTree(depth int) Node {
	node := Node(Text("leaf"))
	for i := 0; i < depth; i++ {
		node = El("div", Props{"class": "level"}, node)
	}
	return node
}

// wideTree builds a div with w children, each containing c grandchildren.
func wideTree(w, c int) Node {
	children := make([]Node, w)
	for i := range children {
		gc := make([]Node, c)
		for j := range gc {
			gc[j] = Text("cell")
		}
		children[i] = El("div", Props{"class": "row"}, gc...)
	}
	return El("div", Props{"class": "grid"}, children...)
}

// realisticPage builds a page-like tree: nav, main with cards, footer.
func realisticPage(cards int) Node {
	navLinks := make([]Node, 5)
	for i := range navLinks {
		navLinks[i] = El("a", Props{"href": "#", "class": "nav-link"}, Textf("Link %d", i))
	}
	nav := El("nav", Props{"class": "navbar"}, navLinks...)

	cardNodes := make([]Node, cards)
	for i := range cardNodes {
		cardNodes[i] = El("div", Props{"class": "card"},
			El("h2", nil, Textf("Card %d", i)),
			El("p", nil, Text("Lorem ipsum dolor sit amet, consectetur adipiscing elit.")),
			El("button", Props{"class": "btn", "onclick": func() {}}, Text("Click me")),
		)
	}
	main := El("main", Props{"class": "content"}, cardNodes...)

	footer := El("footer", Props{"class": "footer"}, Text("Built with gui.md"))

	return El("div", Props{"class": "app"},
		El("style", nil, Text("body { margin: 0; }")),
		nav, main, footer,
	)
}

// ============================================================================
// Node construction benchmarks
// ============================================================================

func BenchmarkTag_Simple(b *testing.B) {
	for b.Loop() {
		Div(Class("container"))(Text("hello"))
	}
}

func BenchmarkTag_WithAttrs(b *testing.B) {
	for b.Loop() {
		Div(Class("container"), Id("main"), Style("color: red"))(
			Text("hello"),
		)
	}
}

func BenchmarkEl_Simple(b *testing.B) {
	for b.Loop() {
		El("div", Props{"class": "container"}, Text("hello"))
	}
}

func BenchmarkTag_NestedElements(b *testing.B) {
	for b.Loop() {
		Div(Class("app"))(
			Nav(Class("nav"))(
				A(Href("#"))(Text("Home")),
				A(Href("#about"))(Text("About")),
			),
			Main()(
				H1()(Text("Title")),
				P()(Text("Content")),
			),
		)
	}
}

func BenchmarkText(b *testing.B) {
	for b.Loop() {
		Text("hello world")
	}
}

func BenchmarkTextf(b *testing.B) {
	for b.Loop() {
		Textf("hello %s, count=%d", "world", 42)
	}
}

func BenchmarkFrag(b *testing.B) {
	a := Text("a")
	bb := Text("b")
	c := Text("c")
	b.ResetTimer()
	for b.Loop() {
		Frag(a, bb, c)
	}
}

// ============================================================================
// Diff engine benchmarks
// ============================================================================

func BenchmarkDiff_Identical_Small(b *testing.B) {
	tree := flatTree(10)
	b.ResetTimer()
	for b.Loop() {
		Diff(tree, tree)
	}
}

func BenchmarkDiff_Identical_Medium(b *testing.B) {
	tree := flatTree(100)
	b.ResetTimer()
	for b.Loop() {
		Diff(tree, tree)
	}
}

func BenchmarkDiff_Identical_Large(b *testing.B) {
	tree := flatTree(1000)
	b.ResetTimer()
	for b.Loop() {
		Diff(tree, tree)
	}
}

func BenchmarkDiff_Identical_Deep(b *testing.B) {
	tree := deepTree(50)
	b.ResetTimer()
	for b.Loop() {
		Diff(tree, tree)
	}
}

func BenchmarkDiff_Identical_Wide(b *testing.B) {
	tree := wideTree(20, 10) // 20 rows x 10 cells = 200 leaf nodes
	b.ResetTimer()
	for b.Loop() {
		Diff(tree, tree)
	}
}

func BenchmarkDiff_Identical_Realistic(b *testing.B) {
	tree := realisticPage(20)
	b.ResetTimer()
	for b.Loop() {
		Diff(tree, tree)
	}
}

func BenchmarkDiff_TextChange_Single(b *testing.B) {
	old := El("div", nil, Text("old"))
	new := El("div", nil, Text("new"))
	b.ResetTimer()
	for b.Loop() {
		Diff(old, new)
	}
}

func BenchmarkDiff_TextChange_InLargeTree(b *testing.B) {
	// 100 children, only the last one changes.
	oldChildren := make([]Node, 100)
	newChildren := make([]Node, 100)
	for i := 0; i < 100; i++ {
		oldChildren[i] = Text("same")
		newChildren[i] = Text("same")
	}
	oldChildren[99] = Text("old-value")
	newChildren[99] = Text("new-value")
	old := El("div", nil, oldChildren...)
	new := El("div", nil, newChildren...)
	b.ResetTimer()
	for b.Loop() {
		Diff(old, new)
	}
}

func BenchmarkDiff_PropsChange(b *testing.B) {
	old := El("div", Props{"class": "old", "id": "x", "data-v": "1"})
	new := El("div", Props{"class": "new", "id": "x", "data-v": "2"})
	b.ResetTimer()
	for b.Loop() {
		Diff(old, new)
	}
}

func BenchmarkDiff_PropsChange_WithHandlers(b *testing.B) {
	h1 := func(Event) {}
	h2 := func(Event) {}
	old := El("div", Props{"class": "x", "onclick": h1})
	new := El("div", Props{"class": "x", "onclick": h2})
	b.ResetTimer()
	for b.Loop() {
		Diff(old, new)
	}
}

func BenchmarkDiff_ChildAppend(b *testing.B) {
	old := El("div", nil, Text("a"), Text("b"))
	new := El("div", nil, Text("a"), Text("b"), Text("c"))
	b.ResetTimer()
	for b.Loop() {
		Diff(old, new)
	}
}

func BenchmarkDiff_ChildRemove(b *testing.B) {
	old := El("div", nil, Text("a"), Text("b"), Text("c"))
	new := El("div", nil, Text("a"), Text("b"))
	b.ResetTimer()
	for b.Loop() {
		Diff(old, new)
	}
}

func BenchmarkDiff_FullReplace(b *testing.B) {
	old := El("div", nil, Text("old"))
	new := El("span", nil, Text("new"))
	b.ResetTimer()
	for b.Loop() {
		Diff(old, new)
	}
}

func BenchmarkDiff_RealisticRerender(b *testing.B) {
	// Simulate a realistic re-render: same page structure, one card text changes.
	old := realisticPage(20)
	new := realisticPage(20)
	// Mutate one deep text node in the new tree.
	main := new.(*Element).Children[2].(*Element) // main element
	card := main.Children[5].(*Element)            // 6th card
	card.Children[0].(*Element).Children[0] = Text("Updated Card 5")
	b.ResetTimer()
	for b.Loop() {
		Diff(old, new)
	}
}

// ============================================================================
// Component resolution benchmarks
// ============================================================================

type benchCounter struct {
	BaseComponent[Props, int]
}

func (c *benchCounter) Render() Node {
	return El("span", nil, Textf("%d", c.State()))
}

type benchNested struct {
	BaseComponent[Props, struct{}]
}

func (c *benchNested) Render() Node {
	return El("div", nil,
		C(new(benchCounter), nil),
		C(new(benchCounter), nil),
		C(new(benchCounter), nil),
	)
}

func BenchmarkResolve_FuncComponent(b *testing.B) {
	type P struct{ Name string }
	fn := func(p P, _ []Node) Node {
		return El("div", nil, Text(p.Name))
	}
	node := Comp(fn, P{Name: "test"})
	b.ResetTimer()
	for b.Loop() {
		Resolve(node)
	}
}

func BenchmarkResolve_StatefulComponent_Mount(b *testing.B) {
	for b.Loop() {
		c := &benchCounter{}
		node := Mount[Props](c, nil)
		Resolve(node)
	}
}

func BenchmarkResolve_ManagedComponent_C(b *testing.B) {
	for b.Loop() {
		node := C(new(benchCounter), nil)
		Resolve(node)
	}
}

func BenchmarkResolve_DeepNesting(b *testing.B) {
	// 10 levels of functional components wrapping each other.
	type P struct{}
	var build func(depth int) Node
	build = func(depth int) Node {
		if depth == 0 {
			return Text("leaf")
		}
		fn := func(_ P, _ []Node) Node { return El("div", nil, build(depth-1)) }
		return Comp(fn, P{})
	}
	node := build(10)
	b.ResetTimer()
	for b.Loop() {
		Resolve(node)
	}
}

func BenchmarkReconciler_FirstRender(b *testing.B) {
	node := El("div", nil,
		C(new(benchCounter), nil),
		C(new(benchCounter), nil),
		C(new(benchCounter), nil),
	)
	b.ResetTimer()
	for b.Loop() {
		r := NewReconciler()
		r.Resolve(node, nil)
	}
}

func BenchmarkReconciler_Rerender(b *testing.B) {
	node := El("div", nil,
		C(new(benchCounter), nil),
		C(new(benchCounter), nil),
		C(new(benchCounter), nil),
	)
	r := NewReconciler()
	r.Resolve(node, nil) // first render
	b.ResetTimer()
	for b.Loop() {
		r.Resolve(node, nil)
	}
}

func BenchmarkReconciler_ManyComponents(b *testing.B) {
	children := make([]Node, 50)
	for i := range children {
		children[i] = C(new(benchCounter), nil)
	}
	node := El("div", nil, children...)
	r := NewReconciler()
	r.Resolve(node, nil)
	b.ResetTimer()
	for b.Loop() {
		r.Resolve(node, nil)
	}
}

func BenchmarkReconciler_NestedComponents(b *testing.B) {
	node := El("div", nil,
		C(new(benchNested), nil),
		C(new(benchNested), nil),
	)
	r := NewReconciler()
	r.Resolve(node, nil)
	b.ResetTimer()
	for b.Loop() {
		r.Resolve(node, nil)
	}
}

func BenchmarkReconciler_WithTracking(b *testing.B) {
	children := make([]Node, 20)
	for i := range children {
		children[i] = C(new(benchCounter), nil)
	}
	node := El("div", nil, children...)
	r := NewReconciler()
	r.Resolve(node, func(Renderable) {})
	b.ResetTimer()
	for b.Loop() {
		r.Resolve(node, func(Renderable) {})
	}
}

// ============================================================================
// Store benchmarks
// ============================================================================

type benchState struct {
	Count int
	Name  string
	Items []string
}

func BenchmarkStore_Get(b *testing.B) {
	s := NewStore(benchState{Count: 42, Name: "test"})
	b.ResetTimer()
	for b.Loop() {
		s.Get()
	}
}

func BenchmarkStore_Set_NoSubscribers(b *testing.B) {
	s := NewStore(benchState{})
	b.ResetTimer()
	for b.Loop() {
		s.Set(benchState{Count: 1})
	}
}

func BenchmarkStore_Set_1Subscriber(b *testing.B) {
	s := NewStore(benchState{})
	s.Subscribe(func(_, _ benchState) {})
	b.ResetTimer()
	for b.Loop() {
		s.Set(benchState{Count: 1})
	}
}

func BenchmarkStore_Set_10Subscribers(b *testing.B) {
	s := NewStore(benchState{})
	for range 10 {
		s.Subscribe(func(_, _ benchState) {})
	}
	b.ResetTimer()
	for b.Loop() {
		s.Set(benchState{Count: 1})
	}
}

func BenchmarkStore_Set_100Subscribers(b *testing.B) {
	s := NewStore(benchState{})
	for range 100 {
		s.Subscribe(func(_, _ benchState) {})
	}
	b.ResetTimer()
	for b.Loop() {
		s.Set(benchState{Count: 1})
	}
}

func BenchmarkStore_Set_1000Subscribers(b *testing.B) {
	s := NewStore(benchState{})
	for range 1000 {
		s.Subscribe(func(_, _ benchState) {})
	}
	b.ResetTimer()
	for b.Loop() {
		s.Set(benchState{Count: 1})
	}
}

func BenchmarkStore_Update(b *testing.B) {
	s := NewStore(benchState{Count: 0})
	s.Subscribe(func(_, _ benchState) {})
	b.ResetTimer()
	for b.Loop() {
		s.Update(func(st benchState) benchState {
			st.Count++
			return st
		})
	}
}

func BenchmarkStore_Subscribe_Unsubscribe(b *testing.B) {
	s := NewStore(benchState{})
	b.ResetTimer()
	for b.Loop() {
		unsub := s.Subscribe(func(_, _ benchState) {})
		unsub()
	}
}

func BenchmarkStore_Get_Parallel(b *testing.B) {
	s := NewStore(benchState{Count: 42, Name: "test"})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Get()
		}
	})
}

func BenchmarkStore_Set_Parallel(b *testing.B) {
	s := NewStore(benchState{})
	s.Subscribe(func(_, _ benchState) {})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Set(benchState{Count: 1})
		}
	})
}

func BenchmarkStore_ReadWrite_Parallel(b *testing.B) {
	s := NewStore(benchState{Count: 0})
	s.Subscribe(func(_, _ benchState) {})
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				s.Set(benchState{Count: i})
			} else {
				s.Get()
			}
			i++
		}
	})
}

// ============================================================================
// Route matching benchmarks
// ============================================================================

func benchRouteHandler(_ Params) Node { return Text("page") }

func BenchmarkMatchRoute_Static_Small(b *testing.B) {
	routes := []RouteConfig{
		{Path: "/", Handler: benchRouteHandler},
		{Path: "/about", Handler: benchRouteHandler},
		{Path: "/contact", Handler: benchRouteHandler},
	}
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/contact")
	}
}

func BenchmarkMatchRoute_Static_Large(b *testing.B) {
	routes := make([]RouteConfig, 50)
	for i := range routes {
		routes[i] = RouteConfig{Path: Textf("/page%d", i).Content, Handler: benchRouteHandler}
	}
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/page49") // worst case: last route
	}
}

func BenchmarkMatchRoute_Param_Single(b *testing.B) {
	routes := []RouteConfig{
		{Path: "/user/:id", Handler: benchRouteHandler},
	}
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/user/42")
	}
}

func BenchmarkMatchRoute_Param_Multiple(b *testing.B) {
	routes := []RouteConfig{
		{Path: "/org/:orgID/repo/:repoID/issues/:issueID", Handler: benchRouteHandler},
	}
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/org/acme/repo/widget/issues/123")
	}
}

func BenchmarkMatchRoute_Wildcard(b *testing.B) {
	routes := []RouteConfig{
		{Path: "/files/*path", Handler: benchRouteHandler},
	}
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/files/docs/api/reference/v2/index.html")
	}
}

func BenchmarkMatchRoute_Nested_Shallow(b *testing.B) {
	routes := []RouteConfig{
		{
			Path: "/dashboard",
			Children: []RouteConfig{
				{Path: "", Handler: benchRouteHandler},
				{Path: "/settings", Handler: benchRouteHandler},
				{Path: "/profile", Handler: benchRouteHandler},
			},
		},
	}
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/dashboard/settings")
	}
}

func BenchmarkMatchRoute_Nested_Deep(b *testing.B) {
	routes := []RouteConfig{
		{
			Path: "/a",
			Children: []RouteConfig{
				{
					Path: "/b",
					Children: []RouteConfig{
						{
							Path: "/c",
							Children: []RouteConfig{
								{
									Path: "/d",
									Children: []RouteConfig{
										{Path: "/e", Handler: benchRouteHandler},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/a/b/c/d/e")
	}
}

func BenchmarkMatchRoute_WithLayouts(b *testing.B) {
	layout := func(outlet Node) Node { return El("div", nil, outlet) }
	routes := []RouteConfig{
		{
			Path:   "/app",
			Layout: layout,
			Children: []RouteConfig{
				{
					Path:   "/dashboard",
					Layout: layout,
					Children: []RouteConfig{
						{Path: "/settings", Handler: benchRouteHandler},
					},
				},
			},
		},
	}
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/app/dashboard/settings")
	}
}

func BenchmarkMatchRoute_WithGuards(b *testing.B) {
	allow := func(_, _ string) bool { return true }
	routes := []RouteConfig{
		{
			Path:   "/admin",
			Guards: []RouteGuard{allow, allow},
			Children: []RouteConfig{
				{Path: "/settings", Handler: benchRouteHandler, Guards: []RouteGuard{allow}},
			},
		},
	}
	b.ResetTimer()
	for b.Loop() {
		m := MatchRoute(routes, "/admin/settings")
		CheckGuards(m, "/", "/admin/settings")
	}
}

func BenchmarkMatchRoute_NoMatch(b *testing.B) {
	routes := make([]RouteConfig, 20)
	for i := range routes {
		routes[i] = RouteConfig{Path: Textf("/page%d", i).Content, Handler: benchRouteHandler}
	}
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/nonexistent")
	}
}

func BenchmarkMatchRoute_FirstMatchWins(b *testing.B) {
	routes := make([]RouteConfig, 50)
	for i := range routes {
		routes[i] = RouteConfig{Path: Textf("/page%d", i).Content, Handler: benchRouteHandler}
	}
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/page0") // best case: first route
	}
}

func BenchmarkRenderMatch_WithLayouts(b *testing.B) {
	layout1 := func(outlet Node) Node { return El("div", Props{"class": "outer"}, outlet) }
	layout2 := func(outlet Node) Node { return El("div", Props{"class": "inner"}, outlet) }
	m := &RouteMatch{
		Params:  Params{"id": "42"},
		Handler: func(p Params) Node { return Text(p["id"]) },
		Layouts: []func(Node) Node{layout1, layout2},
	}
	b.ResetTimer()
	for b.Loop() {
		RenderMatch(m)
	}
}

func BenchmarkCheckGuards(b *testing.B) {
	allow := func(_, _ string) bool { return true }
	m := &RouteMatch{
		Params:  Params{},
		Handler: benchRouteHandler,
		Guards:  []RouteGuard{allow, allow, allow, allow, allow},
	}
	b.ResetTimer()
	for b.Loop() {
		CheckGuards(m, "/", "/target")
	}
}

// ============================================================================
// Allocation-focused benchmarks — measure allocs/op
// ============================================================================

func BenchmarkAlloc_Diff_NoChange(b *testing.B) {
	tree := flatTree(50)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Diff(tree, tree)
	}
}

func BenchmarkAlloc_Diff_OneTextChange(b *testing.B) {
	old := El("div", nil, Text("old"))
	new := El("div", nil, Text("new"))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Diff(old, new)
	}
}

func BenchmarkAlloc_MatchRoute_Param(b *testing.B) {
	routes := []RouteConfig{
		{Path: "/user/:id", Handler: benchRouteHandler},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		MatchRoute(routes, "/user/42")
	}
}

func BenchmarkAlloc_Reconciler_Rerender(b *testing.B) {
	node := El("div", nil,
		C(new(benchCounter), nil),
		C(new(benchCounter), nil),
	)
	r := NewReconciler()
	r.Resolve(node, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		r.Resolve(node, nil)
	}
}

func BenchmarkAlloc_Store_Set(b *testing.B) {
	s := NewStore(benchState{})
	s.Subscribe(func(_, _ benchState) {})
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Set(benchState{Count: 1})
	}
}

func BenchmarkAlloc_TagBuilder(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Div(Class("x"), Id("y"))(Text("z"))
	}
}
