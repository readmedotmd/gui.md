package main

import (
	gui "github.com/readmedotmd/gui.md"
	"github.com/readmedotmd/gui.md/html"
)

func renderPage() string {
	page := gui.Html()(
		gui.Head()(
			gui.Title()(gui.Text("Hello NanoGUI")),
		),
		gui.Body()(
			gui.H1()(gui.Text("Hello from NanoGUI!")),
			gui.P()(gui.Text("A Go port of NanoJSX.")),
		),
	)

	r := html.New()
	return r.RenderString(page)
}
