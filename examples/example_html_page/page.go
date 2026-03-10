package main

import (
	gui "github.com/readmedotmd/gui.md"
	"github.com/readmedotmd/gui.md/html"
)

type PageState struct {
	Theme string
	Lang  string
}

// NavBar is a functional component that renders a header with navigation links.
func NavBar(props gui.Props, children []gui.Node) gui.Node {
	title, _ := props["title"].(string)
	return gui.Header()(
		gui.H1()(gui.Text(title)),
		gui.Nav()(gui.Frag(children...)),
	)
}

// UserCardProps defines the typed props for UserCard.
type UserCardProps struct {
	Name string
}

// CardState holds the state for UserCard.
type CardState struct {
	Role string
}

// UserCard is a stateful component that renders a user card with name and role.
type UserCard struct {
	gui.BaseComponent[UserCardProps, CardState]
}

// Render implements gui.Renderable for UserCard.
func (u *UserCard) Render() gui.Node {
	name := u.Props().Name // typed! no assertion needed
	s := u.State()
	role := s.Role
	if role == "" {
		role = "member"
	}
	return gui.Div(gui.Class("card"))(
		gui.H2()(gui.Text(name)),
		gui.P()(gui.Textf("Role: %s", role)),
	)
}

func renderPage() string {
	store := gui.NewStore(PageState{Theme: "dark", Lang: "en"})
	state := store.Get()

	page := gui.Html()(
		gui.Body(gui.Class(state.Theme))(
			gui.Comp(NavBar, gui.Props{"title": "My App"},
				gui.A(gui.Href("/"))(gui.Text("Home")),
				gui.A(gui.Href("/about"))(gui.Text("About")),
			),
			gui.Mount(&UserCard{}, UserCardProps{Name: "Alice"}),
			gui.Footer()(
				gui.Textf("Language: %s", state.Lang),
			),
		),
	)

	r := html.New()
	return r.RenderString(page)
}
