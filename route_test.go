package gui

import (
	"testing"
)

// helper to make a simple handler that returns a TextNode with the given label.
func handlerFor(label string) func(Params) Node {
	return func(p Params) Node { return Text(label) }
}

// --- parsePattern ---

func TestParsePatternEmpty(t *testing.T) {
	segs := parsePattern("/")
	if len(segs) != 0 {
		t.Fatalf("expected 0 segments, got %d", len(segs))
	}
}

func TestParsePatternLiteral(t *testing.T) {
	segs := parsePattern("/about")
	if len(segs) != 1 || segs[0].literal != "about" {
		t.Fatalf("expected [about], got %v", segs)
	}
}

func TestParsePatternParam(t *testing.T) {
	segs := parsePattern("/user/:id")
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
	if segs[0].literal != "user" {
		t.Fatalf("expected literal 'user', got %v", segs[0])
	}
	if segs[1].param != "id" {
		t.Fatalf("expected param 'id', got %v", segs[1])
	}
}

func TestParsePatternWildcard(t *testing.T) {
	segs := parsePattern("/files/*path")
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
	if segs[0].literal != "files" {
		t.Fatalf("expected literal 'files', got %v", segs[0])
	}
	if segs[1].wildcard != "path" {
		t.Fatalf("expected wildcard 'path', got %v", segs[1])
	}
}

func TestParsePatternMultipleParams(t *testing.T) {
	segs := parsePattern("/org/:orgID/repo/:repoID")
	if len(segs) != 4 {
		t.Fatalf("expected 4 segments, got %d", len(segs))
	}
	if segs[1].param != "orgID" || segs[3].param != "repoID" {
		t.Fatalf("unexpected segments: %v", segs)
	}
}

// --- MatchRoute basic ---

func TestMatchRouteExact(t *testing.T) {
	routes := []RouteConfig{
		{Path: "/", Handler: handlerFor("home")},
		{Path: "/about", Handler: handlerFor("about")},
	}

	m := MatchRoute(routes, "/")
	if m == nil {
		t.Fatal("expected match for /")
	}
	node := m.Handler(m.Params)
	if tn, ok := node.(*TextNode); !ok || tn.Content != "home" {
		t.Fatalf("expected home, got %v", node)
	}

	m = MatchRoute(routes, "/about")
	if m == nil {
		t.Fatal("expected match for /about")
	}
	node = m.Handler(m.Params)
	if tn, ok := node.(*TextNode); !ok || tn.Content != "about" {
		t.Fatalf("expected about, got %v", node)
	}
}

func TestMatchRouteNoMatch(t *testing.T) {
	routes := []RouteConfig{
		{Path: "/about", Handler: handlerFor("about")},
	}
	m := MatchRoute(routes, "/contact")
	if m != nil {
		t.Fatal("expected no match for /contact")
	}
}

// --- MatchRoute with params ---

func TestMatchRouteParams(t *testing.T) {
	routes := []RouteConfig{
		{Path: "/user/:id", Handler: handlerFor("user")},
	}

	m := MatchRoute(routes, "/user/42")
	if m == nil {
		t.Fatal("expected match")
	}
	if m.Params["id"] != "42" {
		t.Fatalf("expected id=42, got %v", m.Params)
	}
}

func TestMatchRouteMultipleParams(t *testing.T) {
	routes := []RouteConfig{
		{Path: "/org/:orgID/repo/:repoID", Handler: handlerFor("repo")},
	}

	m := MatchRoute(routes, "/org/acme/repo/widget")
	if m == nil {
		t.Fatal("expected match")
	}
	if m.Params["orgID"] != "acme" || m.Params["repoID"] != "widget" {
		t.Fatalf("expected orgID=acme, repoID=widget, got %v", m.Params)
	}
}

// --- MatchRoute with wildcard ---

func TestMatchRouteWildcard(t *testing.T) {
	routes := []RouteConfig{
		{Path: "/files/*path", Handler: handlerFor("files")},
	}

	m := MatchRoute(routes, "/files/docs/readme.md")
	if m == nil {
		t.Fatal("expected match")
	}
	if m.Params["path"] != "docs/readme.md" {
		t.Fatalf("expected path=docs/readme.md, got %v", m.Params["path"])
	}
}

func TestMatchRouteWildcardEmpty(t *testing.T) {
	routes := []RouteConfig{
		{Path: "/files/*path", Handler: handlerFor("files")},
	}

	m := MatchRoute(routes, "/files/")
	if m == nil {
		t.Fatal("expected match")
	}
	if m.Params["path"] != "" {
		t.Fatalf("expected empty path, got %q", m.Params["path"])
	}
}

// --- Nested routes ---

func TestMatchRouteNested(t *testing.T) {
	routes := []RouteConfig{
		{
			Path: "/dashboard",
			Children: []RouteConfig{
				{Path: "", Handler: handlerFor("overview")},
				{Path: "/settings", Handler: handlerFor("settings")},
				{Path: "/user/:id", Handler: handlerFor("user")},
			},
		},
	}

	m := MatchRoute(routes, "/dashboard")
	if m == nil {
		t.Fatal("expected match for /dashboard")
	}
	node := m.Handler(m.Params)
	if tn, ok := node.(*TextNode); !ok || tn.Content != "overview" {
		t.Fatalf("expected overview, got %v", node)
	}

	m = MatchRoute(routes, "/dashboard/settings")
	if m == nil {
		t.Fatal("expected match for /dashboard/settings")
	}
	node = m.Handler(m.Params)
	if tn, ok := node.(*TextNode); !ok || tn.Content != "settings" {
		t.Fatalf("expected settings, got %v", node)
	}

	m = MatchRoute(routes, "/dashboard/user/7")
	if m == nil {
		t.Fatal("expected match for /dashboard/user/7")
	}
	if m.Params["id"] != "7" {
		t.Fatalf("expected id=7, got %v", m.Params)
	}
}

func TestMatchRouteNestedNoMatch(t *testing.T) {
	routes := []RouteConfig{
		{
			Path: "/dashboard",
			Children: []RouteConfig{
				{Path: "/settings", Handler: handlerFor("settings")},
			},
		},
	}

	// /dashboard alone has no handler and no index child
	m := MatchRoute(routes, "/dashboard")
	if m != nil {
		t.Fatal("expected no match for /dashboard without index route")
	}
}

// --- Layouts ---

func TestMatchRouteLayouts(t *testing.T) {
	dashLayout := func(outlet Node) Node {
		return El("div", Props{"class": "dash"}, outlet)
	}

	routes := []RouteConfig{
		{
			Path:   "/dashboard",
			Layout: dashLayout,
			Children: []RouteConfig{
				{Path: "", Handler: handlerFor("overview")},
				{Path: "/settings", Handler: handlerFor("settings")},
			},
		},
	}

	m := MatchRoute(routes, "/dashboard/settings")
	if m == nil {
		t.Fatal("expected match")
	}
	if len(m.Layouts) != 1 {
		t.Fatalf("expected 1 layout, got %d", len(m.Layouts))
	}

	// RenderMatch should wrap the handler output in the layout.
	rendered := RenderMatch(m)
	el, ok := rendered.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", rendered)
	}
	if el.Tag != "div" {
		t.Fatalf("expected div, got %s", el.Tag)
	}
	if el.Props["class"] != "dash" {
		t.Fatalf("expected class=dash, got %v", el.Props["class"])
	}
	// Child should be the handler's output.
	if len(el.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(el.Children))
	}
	tn, ok := el.Children[0].(*TextNode)
	if !ok || tn.Content != "settings" {
		t.Fatalf("expected TextNode 'settings', got %v", el.Children[0])
	}
}

func TestMatchRouteNestedLayouts(t *testing.T) {
	outerLayout := func(outlet Node) Node {
		return El("div", Props{"class": "outer"}, outlet)
	}
	innerLayout := func(outlet Node) Node {
		return El("div", Props{"class": "inner"}, outlet)
	}

	routes := []RouteConfig{
		{
			Path:   "/app",
			Layout: outerLayout,
			Children: []RouteConfig{
				{
					Path:   "/panel",
					Layout: innerLayout,
					Children: []RouteConfig{
						{Path: "", Handler: handlerFor("content")},
					},
				},
			},
		},
	}

	m := MatchRoute(routes, "/app/panel")
	if m == nil {
		t.Fatal("expected match")
	}
	if len(m.Layouts) != 2 {
		t.Fatalf("expected 2 layouts, got %d", len(m.Layouts))
	}

	rendered := RenderMatch(m)
	// Should be outer(inner(content))
	outer, ok := rendered.(*Element)
	if !ok || outer.Props["class"] != "outer" {
		t.Fatalf("expected outer div, got %v", rendered)
	}
	inner, ok := outer.Children[0].(*Element)
	if !ok || inner.Props["class"] != "inner" {
		t.Fatalf("expected inner div, got %v", outer.Children[0])
	}
	tn, ok := inner.Children[0].(*TextNode)
	if !ok || tn.Content != "content" {
		t.Fatalf("expected content, got %v", inner.Children[0])
	}
}

// --- Guards ---

func TestCheckGuardsPass(t *testing.T) {
	routes := []RouteConfig{
		{
			Path:    "/admin",
			Handler: handlerFor("admin"),
			Guards:  []RouteGuard{func(from, to string) bool { return true }},
		},
	}

	m := MatchRoute(routes, "/admin")
	if m == nil {
		t.Fatal("expected match")
	}
	if !CheckGuards(m, "/", "/admin") {
		t.Fatal("expected guards to pass")
	}
}

func TestCheckGuardsFail(t *testing.T) {
	routes := []RouteConfig{
		{
			Path:    "/admin",
			Handler: handlerFor("admin"),
			Guards:  []RouteGuard{func(from, to string) bool { return false }},
		},
	}

	m := MatchRoute(routes, "/admin")
	if m == nil {
		t.Fatal("expected match")
	}
	if CheckGuards(m, "/", "/admin") {
		t.Fatal("expected guards to fail")
	}
}

func TestCheckGuardsMultiple(t *testing.T) {
	callOrder := []string{}
	routes := []RouteConfig{
		{
			Path:    "/admin",
			Handler: handlerFor("admin"),
			Guards: []RouteGuard{
				func(from, to string) bool {
					callOrder = append(callOrder, "first")
					return true
				},
				func(from, to string) bool {
					callOrder = append(callOrder, "second")
					return false
				},
				func(from, to string) bool {
					callOrder = append(callOrder, "third")
					return true
				},
			},
		},
	}

	m := MatchRoute(routes, "/admin")
	CheckGuards(m, "/", "/admin")
	if len(callOrder) != 2 || callOrder[0] != "first" || callOrder[1] != "second" {
		t.Fatalf("expected [first, second], got %v", callOrder)
	}
}

func TestCheckGuardsInherited(t *testing.T) {
	parentGuardCalled := false
	routes := []RouteConfig{
		{
			Path:   "/admin",
			Guards: []RouteGuard{func(from, to string) bool { parentGuardCalled = true; return true }},
			Children: []RouteConfig{
				{Path: "/settings", Handler: handlerFor("settings")},
			},
		},
	}

	m := MatchRoute(routes, "/admin/settings")
	if m == nil {
		t.Fatal("expected match")
	}
	if len(m.Guards) != 1 {
		t.Fatalf("expected 1 inherited guard, got %d", len(m.Guards))
	}
	CheckGuards(m, "/", "/admin/settings")
	if !parentGuardCalled {
		t.Fatal("expected parent guard to be called")
	}
}

// --- RenderMatch ---

func TestRenderMatchNil(t *testing.T) {
	if RenderMatch(nil) != nil {
		t.Fatal("expected nil for nil match")
	}
}

func TestRenderMatchNoLayouts(t *testing.T) {
	m := &RouteMatch{
		Params:  Params{"id": "5"},
		Handler: func(p Params) Node { return Textf("user-%s", p["id"]) },
	}
	rendered := RenderMatch(m)
	tn, ok := rendered.(*TextNode)
	if !ok || tn.Content != "user-5" {
		t.Fatalf("expected 'user-5', got %v", rendered)
	}
}

// --- Edge cases ---

func TestMatchRouteNormalizesPath(t *testing.T) {
	routes := []RouteConfig{
		{Path: "/about", Handler: handlerFor("about")},
	}

	// Without leading slash.
	m := MatchRoute(routes, "about")
	if m == nil {
		t.Fatal("expected match for 'about' without leading slash")
	}
}

func TestMatchRouteEmptyPath(t *testing.T) {
	routes := []RouteConfig{
		{Path: "/", Handler: handlerFor("home")},
	}

	m := MatchRoute(routes, "")
	if m == nil {
		t.Fatal("expected match for empty path")
	}
}

func TestMatchRoutePriority(t *testing.T) {
	// First match wins.
	routes := []RouteConfig{
		{Path: "/user/new", Handler: handlerFor("new-user")},
		{Path: "/user/:id", Handler: handlerFor("user")},
	}

	m := MatchRoute(routes, "/user/new")
	if m == nil {
		t.Fatal("expected match")
	}
	node := m.Handler(m.Params)
	tn, ok := node.(*TextNode)
	if !ok || tn.Content != "new-user" {
		t.Fatalf("expected new-user (first match), got %v", node)
	}
}

func TestMatchRouteParamDoesNotMatchExtra(t *testing.T) {
	routes := []RouteConfig{
		{Path: "/user/:id", Handler: handlerFor("user")},
	}

	// Should not match — extra segment.
	m := MatchRoute(routes, "/user/42/posts")
	if m != nil {
		t.Fatal("expected no match for /user/42/posts")
	}
}

func TestMatchRouteLayoutWithHandler(t *testing.T) {
	// A route can have both a layout and a handler (leaf with layout).
	layout := func(outlet Node) Node {
		return El("main", nil, outlet)
	}
	routes := []RouteConfig{
		{
			Path:   "/",
			Layout: layout,
			Children: []RouteConfig{
				{Path: "", Handler: handlerFor("home")},
			},
		},
	}

	m := MatchRoute(routes, "/")
	if m == nil {
		t.Fatal("expected match")
	}
	rendered := RenderMatch(m)
	el, ok := rendered.(*Element)
	if !ok || el.Tag != "main" {
		t.Fatalf("expected <main>, got %v", rendered)
	}
}

func TestCheckGuardsNilMatch(t *testing.T) {
	if !CheckGuards(nil, "/", "/about") {
		t.Fatal("nil match should pass guards")
	}
}
