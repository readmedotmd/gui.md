//go:build js && wasm

package main

import (
	gui "github.com/readmedotmd/gui.md"
	"github.com/readmedotmd/gui.md/dom"
	"syscall/js"
)

func main() {
	store := gui.NewStore(0)

	doc := js.Global().Get("document")
	container := doc.Call("getElementById", "app")

	renderer := dom.New(container)
	defer renderer.Release()

	// Build the UI using the DOM renderer.
	renderer.Update(func() gui.Node {
		return buildUI(store.Get())
	})

	// Wire up button click handlers via JS event listeners.
	incBtn := doc.Call("getElementById", "increment")
	decBtn := doc.Call("getElementById", "decrement")

	incCb := js.FuncOf(func(this js.Value, args []js.Value) any {
		store.Update(func(n int) int { return n + 1 })
		return nil
	})
	defer incCb.Release()

	decCb := js.FuncOf(func(this js.Value, args []js.Value) any {
		store.Update(func(n int) int { return n - 1 })
		return nil
	})
	defer decCb.Release()

	incBtn.Call("addEventListener", "click", incCb)
	decBtn.Call("addEventListener", "click", decCb)

	// Re-render on state change.
	store.Subscribe(func(_, _ int) {
		renderer.Update(func() gui.Node {
			return buildUI(store.Get())
		})
	})

	// Block forever so callbacks remain alive.
	select {}
}
