//go:build js && wasm

package dom

import (
	gui "github.com/readmedotmd/gui.md"
	"strings"
	"syscall/js"
)

// RouterMode controls whether the router uses hash fragments or the History API.
type RouterMode int

const (
	// HashMode uses location.hash and hashchange events (default).
	// URLs look like: http://example.com/#/about
	HashMode RouterMode = iota

	// HistoryMode uses the History API (pushState/popstate).
	// URLs look like: http://example.com/about
	// Requires server-side configuration to serve index.html for all routes.
	HistoryMode
)

// RouterOption configures a Router.
type RouterOption func(*routerConfig)

type routerConfig struct {
	mode   RouterMode
	routes []gui.RouteConfig
	guards []gui.RouteGuard // global guards (BeforeEach)
}

// WithHistoryMode sets the router to use the History API instead of hash fragments.
func WithHistoryMode() RouterOption {
	return func(c *routerConfig) { c.mode = HistoryMode }
}

// WithRoutes sets the declarative route configuration for the router.
// When routes are configured, use router.Render() to get the matched page Node.
func WithRoutes(routes ...gui.RouteConfig) RouterOption {
	return func(c *routerConfig) { c.routes = routes }
}

// BeforeEach registers a global navigation guard. Guards are checked in order
// before every navigation. If any guard returns false, navigation is cancelled.
func BeforeEach(guard gui.RouteGuard) RouterOption {
	return func(c *routerConfig) { c.guards = append(c.guards, guard) }
}

// Route creates a RouteConfig for use with WithRoutes.
func Route(path string, handler func(gui.Params) gui.Node, children ...gui.RouteConfig) gui.RouteConfig {
	return gui.RouteConfig{
		Path:     path,
		Handler:  handler,
		Children: children,
	}
}

// RouteWithLayout creates a RouteConfig with a layout wrapper.
func RouteWithLayout(path string, layout func(outlet gui.Node) gui.Node, children ...gui.RouteConfig) gui.RouteConfig {
	return gui.RouteConfig{
		Path:     path,
		Layout:   layout,
		Children: children,
	}
}

// RouteWithGuards creates a RouteConfig with guards.
func RouteWithGuards(path string, handler func(gui.Params) gui.Node, guards []gui.RouteGuard, children ...gui.RouteConfig) gui.RouteConfig {
	return gui.RouteConfig{
		Path:     path,
		Handler:  handler,
		Guards:   guards,
		Children: children,
	}
}

// Router provides hash-based or history-based routing for single-page WASM
// applications. It listens for URL changes and exposes the current route
// through a [gui.Store]. It supports declarative route configuration with
// pattern matching, named parameters, nested routes, layouts, and guards.
//
// Basic usage (manual matching):
//
//	router := dom.NewRouter()
//	defer router.Release()
//	router.Navigate("/about")
//
// Declarative usage:
//
//	router := dom.NewRouter(
//	    dom.WithRoutes(
//	        dom.Route("/", homePage),
//	        dom.Route("/user/:id", userPage),
//	    ),
//	)
//	defer router.Release()
//	page := router.Render() // returns matched page Node
type Router struct {
	store    *gui.Store[string]
	callback js.Func
	config   routerConfig
}

// NewRouter creates a Router, reads the initial URL, and begins listening
// for navigation events. Options configure the routing mode, route table,
// and global guards.
func NewRouter(opts ...RouterOption) *Router {
	cfg := routerConfig{mode: HashMode}
	for _, opt := range opts {
		opt(&cfg)
	}

	r := &Router{
		config: cfg,
	}

	var initial string
	switch cfg.mode {
	case HistoryMode:
		initial = normalizeHash(js.Global().Get("location").Get("pathname").String())
	default:
		initial = normalizeHash(js.Global().Get("location").Get("hash").String())
	}

	r.store = gui.NewStore(initial)

	switch cfg.mode {
	case HistoryMode:
		r.callback = js.FuncOf(func(this js.Value, args []js.Value) any {
			path := js.Global().Get("location").Get("pathname").String()
			r.handleNavigation(normalizeHash(path))
			return nil
		})
		js.Global().Call("addEventListener", "popstate", r.callback)
	default:
		r.callback = js.FuncOf(func(this js.Value, args []js.Value) any {
			hash := js.Global().Get("location").Get("hash").String()
			r.handleNavigation(normalizeHash(hash))
			return nil
		})
		js.Global().Call("addEventListener", "hashchange", r.callback)
	}

	return r
}

// handleNavigation processes a route change, running guards before updating state.
func (r *Router) handleNavigation(newPath string) {
	oldPath := r.store.Get()
	if newPath == oldPath {
		return
	}

	// Run global guards.
	for _, g := range r.config.guards {
		if !g(oldPath, newPath) {
			// Guard rejected — revert the URL.
			r.revertURL(oldPath)
			return
		}
	}

	// Run route-level guards if we have route config.
	if len(r.config.routes) > 0 {
		m := gui.MatchRoute(r.config.routes, newPath)
		if m != nil && !gui.CheckGuards(m, oldPath, newPath) {
			r.revertURL(oldPath)
			return
		}
	}

	r.store.Set(newPath)
}

// revertURL restores the browser URL to path without triggering navigation.
func (r *Router) revertURL(path string) {
	switch r.config.mode {
	case HistoryMode:
		js.Global().Get("history").Call("replaceState", nil, "", path)
	default:
		// For hash mode, we set location.hash which will fire hashchange,
		// but since the path matches store.Get(), handleNavigation is a no-op.
		js.Global().Get("location").Set("hash", "#"+strings.TrimPrefix(path, "/"))
	}
}

// Route returns the current route path (e.g. "/about", "/user/42").
func (r *Router) Route() string {
	return r.store.Get()
}

// Navigate changes the route. This triggers navigation events which update
// the store and notify subscribers. Guards are checked before the navigation
// is committed.
func (r *Router) Navigate(path string) {
	path = normalizeHash(path)

	switch r.config.mode {
	case HistoryMode:
		// Check guards before pushing state.
		oldPath := r.store.Get()
		for _, g := range r.config.guards {
			if !g(oldPath, path) {
				return
			}
		}
		if len(r.config.routes) > 0 {
			m := gui.MatchRoute(r.config.routes, path)
			if m != nil && !gui.CheckGuards(m, oldPath, path) {
				return
			}
		}
		js.Global().Get("history").Call("pushState", nil, "", path)
		r.store.Set(path)
	default:
		js.Global().Get("location").Set("hash", "#"+strings.TrimPrefix(path, "/"))
	}
}

// Params returns the named parameters extracted from the current route.
// Returns nil if no routes are configured or the current path doesn't match.
func (r *Router) Params() gui.Params {
	if len(r.config.routes) == 0 {
		return nil
	}
	m := gui.MatchRoute(r.config.routes, r.store.Get())
	if m == nil {
		return nil
	}
	return m.Params
}

// Match returns the full RouteMatch for the current path, or nil if no
// routes are configured or nothing matches.
func (r *Router) Match() *gui.RouteMatch {
	if len(r.config.routes) == 0 {
		return nil
	}
	return gui.MatchRoute(r.config.routes, r.store.Get())
}

// Render matches the current path against the route configuration and returns
// the resulting Node (with layouts applied). Returns nil if no routes are
// configured or no route matches.
func (r *Router) Render() gui.Node {
	return gui.RenderMatch(r.Match())
}

// Subscribe registers fn to be called whenever the route changes.
// It returns an unsubscribe function.
func (r *Router) Subscribe(fn func(route, prevRoute string)) func() {
	return r.store.Subscribe(fn)
}

// Release removes the navigation event listener and frees the JS callback.
// Call this when the router is no longer needed.
func (r *Router) Release() {
	switch r.config.mode {
	case HistoryMode:
		js.Global().Call("removeEventListener", "popstate", r.callback)
	default:
		js.Global().Call("removeEventListener", "hashchange", r.callback)
	}
	r.callback.Release()
}

// normalizeHash strips the leading "#", ensures a leading "/", and
// defaults to "/" for an empty hash.
func normalizeHash(hash string) string {
	hash = strings.TrimPrefix(hash, "#")
	if hash == "" {
		return "/"
	}
	if !strings.HasPrefix(hash, "/") {
		hash = "/" + hash
	}
	return hash
}
