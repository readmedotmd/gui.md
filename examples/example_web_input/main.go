// HTTP server demonstrating server-side rendering with NanoGUI.
// Serves a form input. On submit, re-renders with the input shown in a preview div.
//
// Run: cd example_web_input && go run .
// Open: http://localhost:9090
package main

import (
	"fmt"
	gui "github.com/readmedotmd/gui.md"
	"github.com/readmedotmd/gui.md/html"
	"net/http"
	"os"
	"strings"
)

// FormState holds the current form submission value.
type FormState struct {
	Input string
}

func renderPage(state FormState) string {
	r := html.New()

	preview := "(nothing typed yet)"
	if state.Input != "" {
		preview = state.Input
	}

	page := gui.Html()(
		gui.Head()(
			gui.Title()(gui.Text("NanoGUI Web Input")),
			gui.Meta(gui.Attr_("charset", "utf-8"))(),
			gui.StyleEl()(gui.Text(`
				body { font-family: sans-serif; max-width: 600px; margin: 40px auto; padding: 0 20px; }
				.preview { padding: 16px; background: #f0f0f0; border-radius: 8px; margin-top: 16px; }
				input[type=text] { padding: 8px; width: 100%; font-size: 16px; box-sizing: border-box; }
				.meta { color: #666; font-size: 14px; margin-top: 8px; }
			`)),
		),
		gui.Body()(
			gui.H1()(gui.Text("NanoGUI Web Input")),
			gui.Form(gui.Action("/"), gui.Method("POST"))(
				gui.Label()(gui.Text("Type something:")),
				gui.Br()(),
				gui.Input(gui.Type("text"), gui.Name("input"), gui.Value(state.Input),
					gui.Placeholder("Type here and submit..."),
					gui.Attr_("autofocus", true))(),
				gui.Br()(), gui.Br()(),
				gui.Button(gui.Type("submit"))(gui.Text("Update Preview")),
			),
			gui.Div(gui.Class("preview"))(
				gui.H2()(gui.Text("Preview")),
				gui.P()(gui.Text(preview)),
				gui.P(gui.Class("meta"))(gui.Textf("Length: %d | Words: %d",
					len(state.Input),
					len(strings.Fields(state.Input)),
				)),
			),
		),
	)
	return r.RenderString(page)
}

func main() {
	store := gui.NewStore(FormState{Input: ""})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm() //nolint:errcheck // best-effort form parsing
			store.Set(FormState{Input: r.FormValue("input")})
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, renderPage(store.Get()))
	})

	addr := ":9090"
	fmt.Fprintf(os.Stderr, "Listening on http://localhost%s\n", addr)
	http.ListenAndServe(addr, nil) //nolint:errcheck // exits on signal
}
