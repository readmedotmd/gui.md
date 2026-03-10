//go:build js && wasm

// dom_app is a single-page WASM application demonstrating all NanoGUI features:
//
//   - gui.Store[AppState]         — global state for routing (blue badge)
//   - gui.BaseComponent[P, S]    — stateful component for the contact form (pink badge)
//   - gui.FuncComponent[T]       — NavBar with typed props (green badge)
//   - gui/dom.App                — auto-wires SetOnChange and DidUnmount for stateful components
//   - gui/dom.Router             — hash-based SPA navigation
//   - Event handling             — onclick, oninput, onchange
//   - Full CSS styling           — embedded via gui.StyleEl
//
// Run:
//
//	./run.sh
//	# Open http://localhost:9090
package main

import (
	gui "github.com/readmedotmd/gui.md"
	"github.com/readmedotmd/gui.md/dom"
	"syscall/js"
)

// ---------------------------------------------------------------------------
// State types
// ---------------------------------------------------------------------------

// AppState holds global state in a gui.Store. The Route field is synced
// with the hash router — changing it re-renders the whole app.
type AppState struct {
	Route string
}

// FormState holds local state for the ContactForm stateful component.
type FormState struct {
	Name      string
	Email     string
	Message   string
	Submitted bool
}

// ---------------------------------------------------------------------------
// CSS
// ---------------------------------------------------------------------------

const appCSS = `
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    background: #f0f2f5;
    color: #1a1a2e;
    line-height: 1.6;
}

.app { max-width: 800px; margin: 0 auto; padding: 20px; }

/* Nav */
.navbar {
    display: flex; align-items: center; justify-content: space-between;
    background: #1a1a2e; color: #fff; padding: 14px 24px; border-radius: 10px;
    margin-bottom: 24px;
}
.navbar .brand { font-size: 1.25rem; font-weight: 700; }
.nav-links { display: flex; gap: 8px; }
.nav-link {
    padding: 6px 14px; border-radius: 6px; cursor: pointer;
    background: transparent; border: 1px solid rgba(255,255,255,0.2);
    color: #ccc; font-size: 0.9rem; transition: all 0.15s;
}
.nav-link:hover { background: rgba(255,255,255,0.1); color: #fff; }
.nav-link.active { background: #e94560; border-color: #e94560; color: #fff; }

/* Cards */
.card {
    background: #fff; border-radius: 10px; padding: 24px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.08); margin-bottom: 20px;
}
.card h2 { margin-bottom: 12px; color: #1a1a2e; }
.card p { color: #555; }

/* Badge */
.badge {
    display: inline-block; padding: 3px 10px; border-radius: 12px;
    font-size: 0.75rem; font-weight: 600; vertical-align: middle; margin-left: 8px;
}
.badge-store { background: #dbeafe; color: #1e40af; }
.badge-component { background: #fce7f3; color: #9d174d; }
.badge-functional { background: #d1fae5; color: #065f46; }

/* Feature list */
.features { list-style: none; padding: 0; }
.features li {
    padding: 10px 0; border-bottom: 1px solid #eee; color: #444;
}
.features li:last-child { border-bottom: none; }

/* Form */
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-weight: 600; margin-bottom: 4px; font-size: 0.9rem; }
.form-group input, .form-group textarea, .form-group select {
    width: 100%; padding: 10px 12px; border: 1px solid #ddd; border-radius: 6px;
    font-size: 0.95rem; font-family: inherit; transition: border-color 0.15s;
}
.form-group input:focus, .form-group textarea:focus, .form-group select:focus {
    outline: none; border-color: #e94560;
}
.form-group textarea { resize: vertical; min-height: 100px; }

.btn {
    display: inline-block; padding: 10px 24px; border: none; border-radius: 6px;
    background: #e94560; color: #fff; font-size: 1rem; cursor: pointer;
    transition: background 0.15s;
}
.btn:hover { background: #c73650; }

/* Preview */
.preview { background: #f8f9fa; border-left: 3px solid #e94560; padding: 16px; border-radius: 0 6px 6px 0; }
.preview h3 { margin-bottom: 8px; color: #1a1a2e; }
.preview p { color: #555; font-size: 0.9rem; }

/* Success */
.success { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; padding: 16px; border-radius: 6px; }

/* Footer */
.footer { text-align: center; padding: 20px 0; color: #999; font-size: 0.85rem; }

/* About grid */
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
@media (max-width: 600px) { .grid { grid-template-columns: 1fr; } }
`

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func badgeStore() gui.Node {
	return gui.Span(gui.Class("badge badge-store"))(gui.Text("gui.Store"))
}

func badgeComponent() gui.Node {
	return gui.Span(gui.Class("badge badge-component"))(gui.Text("dom.App + BaseComponent[P,S]"))
}

func badgeFunctional() gui.Node {
	return gui.Span(gui.Class("badge badge-functional"))(gui.Text("FuncComponent[T]"))
}

func displayOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// ---------------------------------------------------------------------------
// NavBar — functional component with typed props (gui.FuncComponent[T])
//
// A plain function: receives typed props, returns nodes. No local state,
// no runtime type assertions needed.
// ---------------------------------------------------------------------------

// NavBarProps defines the typed props for the NavBar component.
type NavBarProps struct {
	Route      string
	OnNavigate func(gui.Event)
}

func NavBar(props NavBarProps, _ []gui.Node) gui.Node {
	route := props.Route
	onNav := props.OnNavigate

	link := func(path, label string) gui.Node {
		cls := "nav-link"
		if route == path {
			cls = "nav-link active"
		}
		return gui.Button(
			gui.Class(cls),
			gui.On("click", func(e gui.Event) {
				e.Value = path
				onNav(e)
			}),
		)(gui.Text(label))
	}

	return gui.Nav(gui.Class("navbar"))(
		gui.Span(gui.Class("brand"))(
			gui.Text("NanoGUI Demo"),
			badgeFunctional(),
		),
		gui.Div(gui.Class("nav-links"))(
			link("/", "Home"),
			link("/about", "About"),
			link("/contact", "Contact"),
		),
	)
}

// ---------------------------------------------------------------------------
// ContactForm — stateful component (gui.BaseComponent[gui.Props, FormState])
//
// Local form state lives inside the component. SetState/UpdateState
// automatically trigger re-renders via dom.App's auto-wired SetOnChange.
// ---------------------------------------------------------------------------

type ContactForm struct {
	gui.BaseComponent[gui.Props, FormState]
}

func (c *ContactForm) WillMount() {
	c.SetState(FormState{})
}

func (c *ContactForm) Render() gui.Node {
	s := c.State()

	if s.Submitted {
		return gui.Div(gui.Class("card"))(
			gui.Div(gui.Class("success"))(
				gui.H2()(gui.Text("Message Sent!")),
				gui.P()(gui.Textf("Thanks %s, we'll get back to you at %s.", s.Name, s.Email)),
			),
		)
	}

	return gui.Frag(
		gui.Div(gui.Class("card"))(
			gui.H2()(
				gui.Text("Get in Touch"),
				badgeComponent(),
			),
			gui.P(gui.Style("color:#888; font-size:0.85rem; margin-bottom:16px"))(
				gui.Text(
					"Form state lives in BaseComponent[Props, FormState]. dom.App auto-wires SetOnChange so SetState triggers re-render.",
				),
			),
			gui.Div(gui.Class("form-group"))(
				gui.Label()(gui.Text("Name")),
				gui.Input(
					gui.Type("text"),
					gui.Placeholder("Your name"),
					gui.Value(s.Name),
					gui.On("input", func(e gui.Event) {
						c.UpdateState(func(s FormState) FormState {
							s.Name = e.Value
							return s
						})
					}),
				)(),
			),
			gui.Div(gui.Class("form-group"))(
				gui.Label()(gui.Text("Email")),
				gui.Input(
					gui.Type("email"),
					gui.Placeholder("you@example.com"),
					gui.Value(s.Email),
					gui.On("input", func(e gui.Event) {
						c.UpdateState(func(s FormState) FormState {
							s.Email = e.Value
							return s
						})
					}),
				)(),
			),
			gui.Div(gui.Class("form-group"))(
				gui.Label()(gui.Text("Message")),
				gui.Textarea(
					gui.Placeholder("What's on your mind?"),
					gui.On("input", func(e gui.Event) {
						c.UpdateState(func(s FormState) FormState {
							s.Message = e.Value
							return s
						})
					}),
				)(gui.Text(s.Message)),
			),
			gui.Div(gui.Class("form-group"))(
				gui.Label()(gui.Text("Priority")),
				gui.Select(
					gui.On("change", func(e gui.Event) {}),
				)(
					gui.Option(gui.Value("normal"))(gui.Text("Normal")),
					gui.Option(gui.Value("high"))(gui.Text("High")),
					gui.Option(gui.Value("urgent"))(gui.Text("Urgent")),
				),
			),
			gui.Button(
				gui.Class("btn"),
				gui.On("click", func(e gui.Event) {
					c.UpdateState(func(s FormState) FormState {
						s.Submitted = true
						return s
					})
				}),
			)(gui.Text("Send Message")),
		),

		// Live preview card
		gui.Div(gui.Class("card preview"))(
			gui.H3()(gui.Text("Live Preview")),
			gui.P()(gui.Textf("Name: %s", displayOr(s.Name, "(empty)"))),
			gui.P()(gui.Textf("Email: %s", displayOr(s.Email, "(empty)"))),
			gui.P()(gui.Textf("Message: %s", displayOr(s.Message, "(empty)"))),
		),
	)
}

// ---------------------------------------------------------------------------
// Pages
// ---------------------------------------------------------------------------

func homePage() gui.Node {
	return gui.Div()(
		gui.Div(gui.Class("card"))(
			gui.H2()(gui.Text("Welcome to NanoGUI")),
			gui.P()(
				gui.Text(
					"A tiny, dependency-free Go UI library that renders to HTML, the terminal, and the browser DOM via WebAssembly.",
				),
			),
		),
		gui.Div(gui.Class("card"))(
			gui.H2()(gui.Text("Features")),
			gui.Ul(gui.Class("features"))(
				gui.Li()(gui.Text("Type-safe functional components with gui.Comp()")),
				gui.Li()(gui.Text("Stateful components with BaseComponent[P, S]")),
				gui.Li()(gui.Text("Generic store with subscriptions")),
				gui.Li()(gui.Text("Virtual DOM diffing and patching")),
				gui.Li()(gui.Text("Hash-based router for SPA navigation")),
				gui.Li()(gui.Text("Event handling: click, input, change, keyboard")),
				gui.Li()(gui.Text("Three backends: HTML, DOM (WASM), Terminal")),
			),
		),
		gui.Div(gui.Class("card"))(
			gui.H2()(gui.Text("Patterns in This Demo")),
			gui.Ul(gui.Class("features"))(
				gui.Li()(
					gui.Strong()(gui.Text("Navigation")),
					badgeStore(),
					gui.Text(" — route stored in gui.Store[AppState], hash router syncs it"),
				),
				gui.Li()(
					gui.Strong()(gui.Text("NavBar")),
					badgeFunctional(),
					gui.Text(" — typed props via FuncComponent[NavBarProps], no local state"),
				),
				gui.Li()(
					gui.Strong()(gui.Text("Contact Form")),
					badgeComponent(),
					gui.Text(
						" — stateful component with typed local state, SetState triggers re-render",
					),
				),
			),
		),
	)
}

func aboutPage() gui.Node {
	return gui.Div()(
		gui.Div(gui.Class("card"))(
			gui.H2()(gui.Text("About This Demo")),
			gui.P()(
				gui.Text(
					"This single-page application is built entirely with NanoGUI. It demonstrates routing, components, state management, and styling — all in pure Go compiled to WebAssembly.",
				),
			),
		),
		gui.Div(gui.Class("grid"))(
			gui.Div(gui.Class("card"))(
				gui.H2()(gui.Text("Store"), badgeStore()),
				gui.P()(
					gui.Text(
						"The route lives in a gui.Store[AppState]. The hash router writes to it, and a subscription triggers re-renders. Global state that multiple parts of the app read.",
					),
				),
			),
			gui.Div(gui.Class("card"))(
				gui.H2()(gui.Text("Stateful Component"), badgeComponent()),
				gui.P()(
					gui.Text(
						"The contact form embeds BaseComponent[Props, FormState]. State is local to the component. dom.App auto-wires SetOnChange so SetState/UpdateState trigger re-renders.",
					),
				),
			),
			gui.Div(gui.Class("card"))(
				gui.H2()(gui.Text("Functional Component"), badgeFunctional()),
				gui.P()(
					gui.Text(
						"NavBar is a FuncComponent[NavBarProps] — it receives typed props (Route, OnNavigate) with compile-time safety. No type assertions, no local state, no lifecycle.",
					),
				),
			),
			gui.Div(gui.Class("card"))(
				gui.H2()(gui.Text("Rendering")),
				gui.P()(
					gui.Text(
						"The same node tree API renders to static HTML, live DOM (WASM), or styled terminal output.",
					),
				),
			),
		),
	)
}

// ---------------------------------------------------------------------------
// Layout
// ---------------------------------------------------------------------------

func buildApp(state AppState, navigate func(string)) gui.Node {
	var page gui.Node
	switch state.Route {
	case "/about":
		page = aboutPage()
	case "/contact":
		page = gui.C(new(ContactForm), nil)
	default:
		page = homePage()
	}

	return gui.Div(gui.Class("app"))(
		gui.StyleEl()(gui.Text(appCSS)),
		gui.Comp(NavBar, NavBarProps{
			Route: state.Route,
			OnNavigate: func(e gui.Event) {
				navigate(e.Value)
			},
		}),
		gui.Div(gui.Style("text-align:right; margin-bottom:8px"))(
			gui.Span(gui.Style("font-size:0.8rem; color:#888"))(
				gui.Textf("Current route: %s", state.Route),
			),
			badgeStore(),
		),
		page,
		gui.Footer(gui.Class("footer"))(
			gui.Text("Built with NanoGUI — Go + WebAssembly"),
		),
	)
}

// ---------------------------------------------------------------------------
// main — WASM entry point
// ---------------------------------------------------------------------------

func main() {
	appStore := gui.NewStore(AppState{Route: "/"})

	container := js.Global().Get("document").Call("getElementById", "app")
	router := dom.NewRouter()
	defer router.Release()

	// Sync router → store.
	appStore.Set(AppState{Route: router.Route()})
	router.Subscribe(func(route, _ string) {
		appStore.Set(AppState{Route: route})
	})

	app := dom.NewApp(container, func() gui.Node {
		return buildApp(appStore.Get(), router.Navigate)
	})
	defer app.Release()

	// Re-render on store changes (route navigation).
	appStore.Subscribe(func(_, _ AppState) { app.Render() })

	app.Run()
}
