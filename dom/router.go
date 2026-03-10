//go:build js && wasm

package dom

import (
	gui "github.com/readmedotmd/gui.md"
	"strings"
	"syscall/js"
)

// Router provides hash-based routing for single-page WASM applications.
// It listens for hashchange events on the window and exposes the current
// route (the fragment after "#") as a simple string. Route changes are
// synchronised through a [gui.Store] so subscribers are notified in a
// familiar way.
//
// Usage:
//
//	router := dom.NewRouter()
//	defer router.Release()
//
//	unsub := router.Subscribe(func(route, prev string) {
//	    fmt.Println("navigated from", prev, "to", route)
//	})
//	defer unsub()
//
//	router.Navigate("/about")
type Router struct {
	store    *gui.Store[string]
	callback js.Func
}

// NewRouter creates a Router, reads the initial hash from the URL, and
// begins listening for hashchange events on the window.
func NewRouter() *Router {
	initial := normalizeHash(js.Global().Get("location").Get("hash").String())

	r := &Router{
		store: gui.NewStore(initial),
	}

	r.callback = js.FuncOf(func(this js.Value, args []js.Value) any {
		hash := js.Global().Get("location").Get("hash").String()
		r.store.Set(normalizeHash(hash))
		return nil
	})

	js.Global().Call("addEventListener", "hashchange", r.callback)
	return r
}

// Route returns the current route path (e.g. "/about").
func (r *Router) Route() string {
	return r.store.Get()
}

// Navigate changes the route by setting location.hash. This triggers a
// hashchange event which updates the store and notifies subscribers —
// there is a single code path for all route updates.
func (r *Router) Navigate(path string) {
	js.Global().Get("location").Set("hash", "#"+strings.TrimPrefix(path, "/"))
}

// Subscribe registers fn to be called whenever the route changes.
// It returns an unsubscribe function.
func (r *Router) Subscribe(fn func(route, prevRoute string)) func() {
	return r.store.Subscribe(fn)
}

// Release removes the hashchange listener and frees the JS callback.
// Call this when the router is no longer needed.
func (r *Router) Release() {
	js.Global().Call("removeEventListener", "hashchange", r.callback)
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
