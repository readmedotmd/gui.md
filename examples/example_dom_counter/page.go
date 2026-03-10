package main

import (
	gui "github.com/readmedotmd/gui.md"
)

// buildUI constructs the counter UI tree. The count value is read from the
// global store so that both CLI and WASM entry points can share this function.
func buildUI(count int) gui.Node {
	return gui.Html()(
		gui.Head()(
			gui.Title()(gui.Text("DOM Counter")),
			gui.StyleEl()(gui.Text(`
				body { font-family: sans-serif; text-align: center; margin-top: 50px; }
				button { font-size: 24px; padding: 8px 20px; margin: 0 10px; cursor: pointer; }
				.count { font-size: 48px; margin: 20px 0; }
			`)),
		),
		gui.Body()(
			gui.H1()(gui.Text("DOM Counter")),
			gui.Div(gui.Class("count"))(gui.Textf("%d", count)),
			gui.Div()(
				gui.Button(gui.Id("decrement"))(gui.Text("-")),
				gui.Button(gui.Id("increment"))(gui.Text("+")),
			),
			gui.P()(gui.Text("Click the buttons to change the count.")),
		),
	)
}
