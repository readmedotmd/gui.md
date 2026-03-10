//go:build js && wasm

package dom

import (
	gui "github.com/readmedotmd/gui.md"
	"syscall/js"
)

// onChangeWirer is satisfied by gui.BaseComponent — used to auto-wire
// SetOnChange so that stateful components trigger re-renders.
type onChangeWirer interface{ SetOnChange(func()) }

// didUnmounter is satisfied by gui.BaseComponent — used to fire
// DidUnmount when a component disappears from the tree.
type didUnmounter interface{ DidUnmount() }

// App manages the lifecycle of a DOM application. It resolves the component
// tree, auto-wires SetOnChange on stateful components so that state changes
// trigger re-renders, and calls DidUnmount when components are removed from
// the tree.
//
// Usage:
//
//	app := dom.NewApp(container, func() gui.Node {
//	    return buildApp(store.Get())
//	})
//	defer app.Release()
//	app.Run()
type App struct {
	renderer      *Renderer
	root          func() gui.Node
	mounted       map[gui.Renderable]struct{}
	reconciler    *gui.Reconciler
	renderPending bool    // true while a rAF callback is scheduled
	rafCb         js.Func // persistent requestAnimationFrame callback
}

// NewApp creates an App that renders into the given container element.
// The root function is called on every render to produce the node tree.
func NewApp(container js.Value, root func() gui.Node) *App {
	a := &App{
		renderer:   New(container),
		root:       root,
		mounted:    make(map[gui.Renderable]struct{}),
		reconciler: gui.NewReconciler(),
	}
	a.rafCb = js.FuncOf(func(this js.Value, args []js.Value) any {
		a.renderPending = false
		a.doRender()
		return nil
	})
	return a
}

// Render schedules a render on the next animation frame. Multiple calls
// between frames are coalesced into a single render pass, preventing
// high-frequency state updates (e.g. streaming WebSocket tokens) from
// blocking the JS event loop and dropping keyboard input.
func (a *App) Render() {
	if a.renderPending {
		return
	}
	a.renderPending = true
	js.Global().Call("requestAnimationFrame", a.rafCb)
}

// doRender resolves the root tree, auto-wires new stateful components,
// cleans up removed components, and patches the DOM.
func (a *App) doRender() {
	seen := make(map[gui.Renderable]struct{})

	resolved := a.reconciler.Resolve(a.root(), func(c gui.Renderable) {
		seen[c] = struct{}{}
	})

	// Wire new components.
	for c := range seen {
		if _, ok := a.mounted[c]; !ok {
			if w, ok := c.(onChangeWirer); ok {
				w.SetOnChange(a.Render)
			}
		}
	}

	// Clean up removed components.
	for c := range a.mounted {
		if _, ok := seen[c]; !ok {
			if w, ok := c.(onChangeWirer); ok {
				w.SetOnChange(nil)
			}
			if u, ok := c.(didUnmounter); ok {
				u.DidUnmount()
			}
		}
	}

	a.mounted = seen
	a.renderer.UpdateTree(resolved)
}

// Run performs the initial synchronous render and then blocks the goroutine
// forever. This is the typical entry point for a WASM main function.
func (a *App) Run() {
	a.doRender()
	select {}
}

// Release unwires all mounted components (SetOnChange(nil) + DidUnmount)
// and releases the underlying renderer and the rAF callback.
func (a *App) Release() {
	for c := range a.mounted {
		if w, ok := c.(onChangeWirer); ok {
			w.SetOnChange(nil)
		}
		if u, ok := c.(didUnmounter); ok {
			u.DidUnmount()
		}
	}
	a.mounted = nil
	a.rafCb.Release()
	a.renderer.Release()
}
