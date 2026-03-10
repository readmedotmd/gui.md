package guitesting

import (
	"testing"

	gui "github.com/readmedotmd/gui.md"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func div(attrs ...gui.Attr) func(children ...gui.Node) *gui.Element {
	return gui.Div(attrs...)
}

func btn(attrs ...gui.Attr) func(children ...gui.Node) *gui.Element {
	return gui.Button(attrs...)
}

func input(attrs ...gui.Attr) func(children ...gui.Node) *gui.Element {
	return gui.Input(attrs...)
}

// ---------------------------------------------------------------------------
// Render
// ---------------------------------------------------------------------------

func TestRenderReturnsScreen(t *testing.T) {
	s := Render(div()(gui.Text("hello")))
	if s == nil {
		t.Fatal("expected non-nil screen")
	}
	if s.Root() == nil {
		t.Fatal("expected non-nil root")
	}
}

func TestRenderHTML(t *testing.T) {
	s := Render(div(gui.Class("x"))(gui.Text("hello")))
	html := s.HTML()
	if html != `<div class="x">hello</div>` {
		t.Errorf("got %q", html)
	}
}

// ---------------------------------------------------------------------------
// GetByText / QueryByText
// ---------------------------------------------------------------------------

func TestGetByTextFindsText(t *testing.T) {
	s := Render(div()(
		gui.Span()(gui.Text("hello")),
		gui.Span()(gui.Text("world")),
	))

	ref := s.GetByText("world")
	if ref == nil {
		t.Fatal("expected to find 'world'")
	}
	if ref.Text() != "world" {
		t.Errorf("text: got %q", ref.Text())
	}
}

func TestGetByTextPanicsWhenNotFound(t *testing.T) {
	s := Render(div()(gui.Text("hello")))
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	s.GetByText("nonexistent")
}

func TestQueryByTextReturnsNil(t *testing.T) {
	s := Render(div()(gui.Text("hello")))
	if s.QueryByText("nonexistent") != nil {
		t.Error("expected nil")
	}
}

func TestQueryAllByTextMultipleMatches(t *testing.T) {
	s := Render(div()(
		gui.Span()(gui.Text("item")),
		gui.Span()(gui.Text("item")),
		gui.Span()(gui.Text("other")),
	))

	refs := s.QueryAllByText("item")
	if len(refs) < 2 {
		t.Errorf("expected at least 2 matches, got %d", len(refs))
	}
}

// ---------------------------------------------------------------------------
// GetByTestId / QueryByTestId
// ---------------------------------------------------------------------------

func TestGetByTestId(t *testing.T) {
	s := Render(div()(
		gui.Span(gui.Data("testid", "name"))(gui.Text("Alice")),
		gui.Span(gui.Data("testid", "age"))(gui.Text("30")),
	))

	ref := s.GetByTestId("name")
	if ref.Text() != "Alice" {
		t.Errorf("text: got %q", ref.Text())
	}
}

func TestGetByTestIdPanicsWhenNotFound(t *testing.T) {
	s := Render(div()())
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	s.GetByTestId("missing")
}

func TestQueryByTestIdReturnsNil(t *testing.T) {
	s := Render(div()())
	if s.QueryByTestId("missing") != nil {
		t.Error("expected nil")
	}
}

// ---------------------------------------------------------------------------
// GetByRole / QueryByRole
// ---------------------------------------------------------------------------

func TestGetByRoleButton(t *testing.T) {
	s := Render(div()(
		btn()(gui.Text("Click me")),
	))

	ref := s.GetByRole("button")
	if ref.Text() != "Click me" {
		t.Errorf("text: got %q", ref.Text())
	}
}

func TestGetByRoleLink(t *testing.T) {
	s := Render(div()(
		gui.A(gui.Href("/"))(gui.Text("Home")),
	))

	ref := s.GetByRole("link")
	if ref.Text() != "Home" {
		t.Errorf("text: got %q", ref.Text())
	}
}

func TestGetByRoleHeading(t *testing.T) {
	s := Render(div()(
		gui.H1()(gui.Text("Title")),
	))

	ref := s.GetByRole("heading")
	if ref.Text() != "Title" {
		t.Errorf("text: got %q", ref.Text())
	}
}

func TestGetByRoleTextbox(t *testing.T) {
	s := Render(div()(
		input(gui.Type("text"), gui.Placeholder("Enter name"))(),
	))

	ref := s.GetByRole("textbox")
	if ref == nil {
		t.Fatal("expected to find textbox")
	}
}

func TestGetByRoleTextboxTextarea(t *testing.T) {
	s := Render(div()(
		gui.Textarea(gui.Placeholder("Write here"))(),
	))

	ref := s.GetByRole("textbox")
	if ref == nil {
		t.Fatal("expected to find textbox (textarea)")
	}
}

func TestGetByRoleCheckbox(t *testing.T) {
	s := Render(div()(
		input(gui.Type("checkbox"))(),
	))

	ref := s.GetByRole("checkbox")
	if ref == nil {
		t.Fatal("expected to find checkbox")
	}
}

func TestGetByRoleList(t *testing.T) {
	s := Render(gui.Ul()(
		gui.Li()(gui.Text("one")),
		gui.Li()(gui.Text("two")),
	))

	ref := s.GetByRole("list")
	if ref == nil {
		t.Fatal("expected to find list")
	}
}

func TestGetByRoleListitem(t *testing.T) {
	s := Render(gui.Ul()(
		gui.Li()(gui.Text("one")),
	))

	ref := s.GetByRole("listitem")
	if ref.Text() != "one" {
		t.Errorf("text: got %q", ref.Text())
	}
}

func TestGetByRoleExplicit(t *testing.T) {
	s := Render(div(gui.Attr_("role", "alert"))(gui.Text("Error!")))

	ref := s.GetByRole("alert")
	if ref.Text() != "Error!" {
		t.Errorf("text: got %q", ref.Text())
	}
}

func TestGetByRolePanicsWhenNotFound(t *testing.T) {
	s := Render(div()())
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	s.GetByRole("button")
}

func TestQueryAllByRole(t *testing.T) {
	s := Render(div()(
		btn()(gui.Text("A")),
		btn()(gui.Text("B")),
		gui.Span()(gui.Text("C")),
	))

	refs := s.QueryAllByRole("button")
	if len(refs) != 2 {
		t.Errorf("expected 2, got %d", len(refs))
	}
}

// ---------------------------------------------------------------------------
// GetByPlaceholder / QueryByPlaceholder
// ---------------------------------------------------------------------------

func TestGetByPlaceholder(t *testing.T) {
	s := Render(div()(
		input(gui.Placeholder("Search..."))(),
	))

	ref := s.GetByPlaceholder("Search")
	if ref == nil {
		t.Fatal("expected to find input")
	}
}

func TestGetByPlaceholderPanicsWhenNotFound(t *testing.T) {
	s := Render(div()())
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	s.GetByPlaceholder("missing")
}

// ---------------------------------------------------------------------------
// QueryAllByTag
// ---------------------------------------------------------------------------

func TestQueryAllByTag(t *testing.T) {
	s := Render(div()(
		gui.Span()(gui.Text("a")),
		gui.Span()(gui.Text("b")),
		gui.P()(gui.Text("c")),
	))

	refs := s.QueryAllByTag("span")
	if len(refs) != 2 {
		t.Errorf("expected 2, got %d", len(refs))
	}
}

// ---------------------------------------------------------------------------
// NodeRef methods
// ---------------------------------------------------------------------------

func TestNodeRefElement(t *testing.T) {
	s := Render(btn(gui.Class("primary"))(gui.Text("Go")))
	ref := s.GetByRole("button")

	el := ref.Element()
	if el.Tag != "button" {
		t.Errorf("tag: got %q", el.Tag)
	}
}

func TestNodeRefElementPanicsOnTextNode(t *testing.T) {
	ref := &NodeRef{Node: gui.Text("hi")}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	ref.Element()
}

func TestNodeRefHasClass(t *testing.T) {
	s := Render(div(gui.Class("foo bar baz"))())
	ref := s.QueryAllByTag("div")[0]

	if !ref.HasClass("bar") {
		t.Error("expected HasClass(bar) = true")
	}
	if ref.HasClass("qux") {
		t.Error("expected HasClass(qux) = false")
	}
}

func TestNodeRefProp(t *testing.T) {
	s := Render(input(gui.Type("email"), gui.Placeholder("Email"))())
	ref := s.QueryAllByTag("input")[0]

	if ref.Prop("type") != "email" {
		t.Errorf("type: got %v", ref.Prop("type"))
	}
	if ref.Prop("placeholder") != "Email" {
		t.Errorf("placeholder: got %v", ref.Prop("placeholder"))
	}
	if ref.Prop("nonexistent") != nil {
		t.Error("expected nil for missing prop")
	}
}

func TestNodeRefPropOnTextNode(t *testing.T) {
	ref := &NodeRef{Node: gui.Text("hi")}
	if ref.Prop("anything") != nil {
		t.Error("expected nil")
	}
}

func TestNodeRefText(t *testing.T) {
	s := Render(div()(
		gui.Span()(gui.Text("hello ")),
		gui.Strong()(gui.Text("world")),
	))
	// The div should have combined text "hello world"
	refs := s.QueryAllByTag("div")
	if refs[0].Text() != "hello world" {
		t.Errorf("got %q", refs[0].Text())
	}
}

// ---------------------------------------------------------------------------
// Click
// ---------------------------------------------------------------------------

func TestClick(t *testing.T) {
	clicked := false
	s := Render(btn(gui.OnClick(func() { clicked = true }))(gui.Text("Go")))

	ref := s.GetByRole("button")
	s.Click(ref)

	if !clicked {
		t.Error("expected click handler to fire")
	}
}

func TestClickWithEventHandler(t *testing.T) {
	var received gui.Event
	handler := func(e gui.Event) { received = e }
	tree := gui.El("button", gui.Props{"onclick": handler}, gui.Text("Go"))

	s := Render(tree)
	ref := s.GetByRole("button")
	s.Click(ref)

	if received.Type != "click" {
		t.Errorf("event type: got %q", received.Type)
	}
}

func TestClickPanicsOnTextNode(t *testing.T) {
	ref := &NodeRef{Node: gui.Text("hi")}
	s := Render(gui.Text("hi"))
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	s.Click(ref)
}

func TestClickPanicsWithNoHandler(t *testing.T) {
	s := Render(btn()(gui.Text("Go")))
	ref := s.GetByRole("button")
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	s.Click(ref)
}

// ---------------------------------------------------------------------------
// FireEvent
// ---------------------------------------------------------------------------

func TestFireEvent(t *testing.T) {
	var received gui.Event
	handler := func(e gui.Event) { received = e }
	tree := gui.El("div", gui.Props{"onmouseenter": handler})

	s := Render(tree)
	refs := s.QueryAllByTag("div")
	s.FireEvent(refs[0], "mouseenter", gui.Event{X: 10, Y: 20})

	if received.Type != "mouseenter" {
		t.Errorf("type: got %q", received.Type)
	}
	if received.X != 10 {
		t.Errorf("X: got %d", received.X)
	}
}

func TestFireEventSimpleHandler(t *testing.T) {
	called := false
	tree := gui.El("div", gui.Props{"onfocus": func() { called = true }})

	s := Render(tree)
	refs := s.QueryAllByTag("div")
	s.FireEvent(refs[0], "focus", gui.Event{})

	if !called {
		t.Error("expected handler to fire")
	}
}

func TestFireEventPanicsNoHandler(t *testing.T) {
	s := Render(div()())
	refs := s.QueryAllByTag("div")
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	s.FireEvent(refs[0], "click", gui.Event{})
}

// ---------------------------------------------------------------------------
// Type
// ---------------------------------------------------------------------------

func TestType(t *testing.T) {
	var values []string
	handler := func(e gui.Event) { values = append(values, e.Value) }
	tree := gui.El("input", gui.Props{"oninput": handler})

	s := Render(tree)
	ref := s.QueryAllByTag("input")[0]
	s.Type(ref, "abc")

	if len(values) != 3 {
		t.Fatalf("expected 3 events, got %d", len(values))
	}
	if values[0] != "a" || values[1] != "ab" || values[2] != "abc" {
		t.Errorf("values: %v", values)
	}
}

func TestTypeWithOnChange(t *testing.T) {
	var value string
	handler := func(e gui.Event) { value = e.Value }
	tree := gui.El("input", gui.Props{"onchange": handler})

	s := Render(tree)
	ref := s.QueryAllByTag("input")[0]
	s.Type(ref, "hello")

	if value != "hello" {
		t.Errorf("value: got %q", value)
	}
}

func TestTypePanicsNoHandler(t *testing.T) {
	s := Render(input()())
	ref := s.QueryAllByTag("input")[0]
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	s.Type(ref, "text")
}

// ---------------------------------------------------------------------------
// Clear
// ---------------------------------------------------------------------------

func TestClear(t *testing.T) {
	var value string
	handler := func(e gui.Event) { value = e.Value }
	tree := gui.El("input", gui.Props{"oninput": handler})

	s := Render(tree)
	ref := s.QueryAllByTag("input")[0]
	s.Clear(ref)

	if value != "" {
		t.Errorf("value: got %q", value)
	}
}

func TestClearPanicsNoHandler(t *testing.T) {
	s := Render(input()())
	ref := s.QueryAllByTag("input")[0]
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	s.Clear(ref)
}

// ---------------------------------------------------------------------------
// KeyPress
// ---------------------------------------------------------------------------

func TestKeyPress(t *testing.T) {
	var key string
	handler := func(e gui.Event) { key = e.Key }
	tree := gui.El("input", gui.Props{"onkeypress": handler})

	s := Render(tree)
	ref := s.QueryAllByTag("input")[0]
	s.KeyPress(ref, "Enter")

	if key != "Enter" {
		t.Errorf("key: got %q", key)
	}
}

// ---------------------------------------------------------------------------
// ContainsText / TextContent
// ---------------------------------------------------------------------------

func TestContainsText(t *testing.T) {
	s := Render(div()(
		gui.P()(gui.Text("Hello World")),
		gui.P()(gui.Text("Goodbye")),
	))

	if !s.ContainsText("Hello World") {
		t.Error("expected to contain 'Hello World'")
	}
	if s.ContainsText("Missing") {
		t.Error("expected NOT to contain 'Missing'")
	}
}

func TestTextContent(t *testing.T) {
	s := Render(div()(
		gui.Text("A"),
		gui.Span()(gui.Text("B")),
	))

	if s.TextContent() != "AB" {
		t.Errorf("got %q", s.TextContent())
	}
}

// ---------------------------------------------------------------------------
// Assertions (Asserter)
// ---------------------------------------------------------------------------

func TestAssertTextVisible(t *testing.T) {
	s := Render(div()(gui.Text("hello")))
	s.Assert(t).TextVisible("hello")
}

func TestAssertTextNotVisible(t *testing.T) {
	s := Render(div()(gui.Text("hello")))
	s.Assert(t).TextNotVisible("world")
}

func TestAssertHasElement(t *testing.T) {
	s := Render(div()(gui.Span()()))
	s.Assert(t).HasElement("span")
}

func TestAssertHasNoElement(t *testing.T) {
	s := Render(div()())
	s.Assert(t).HasNoElement("span")
}

func TestAssertHasTestId(t *testing.T) {
	s := Render(div(gui.Data("testid", "foo"))())
	s.Assert(t).HasTestId("foo")
}

func TestAssertHasNoTestId(t *testing.T) {
	s := Render(div()())
	s.Assert(t).HasNoTestId("foo")
}

func TestAssertHasRole(t *testing.T) {
	s := Render(div()(btn()(gui.Text("Go"))))
	s.Assert(t).HasRole("button")
}

func TestAssertElementCount(t *testing.T) {
	s := Render(div()(
		gui.Span()(),
		gui.Span()(),
		gui.Span()(),
	))
	s.Assert(t).ElementCount("span", 3)
}

func TestAssertHTMLContains(t *testing.T) {
	s := Render(div(gui.Class("active"))(gui.Text("on")))
	s.Assert(t).HTMLContains(`class="active"`)
}

func TestAssertHTMLNotContains(t *testing.T) {
	s := Render(div()())
	s.Assert(t).HTMLNotContains("active")
}

func TestAssertNodeHasText(t *testing.T) {
	s := Render(btn()(gui.Text("Submit")))
	ref := s.GetByRole("button")
	s.Assert(t).NodeHasText(ref, "Submit")
}

func TestAssertNodeHasClass(t *testing.T) {
	s := Render(div(gui.Class("primary active"))())
	ref := s.QueryAllByTag("div")[0]
	s.Assert(t).NodeHasClass(ref, "active")
}

func TestAssertNodeHasProp(t *testing.T) {
	s := Render(input(gui.Type("email"))())
	ref := s.QueryAllByTag("input")[0]
	s.Assert(t).NodeHasProp(ref, "type", "email")
}

func TestAssertNodeEnabled(t *testing.T) {
	s := Render(btn()(gui.Text("Go")))
	ref := s.GetByRole("button")
	s.Assert(t).NodeEnabled(ref)
}

func TestAssertNodeDisabled(t *testing.T) {
	s := Render(btn(gui.Disabled(true))(gui.Text("Go")))
	ref := s.GetByRole("button")
	s.Assert(t).NodeDisabled(ref)
}

func TestAssertSnapshot(t *testing.T) {
	s := Render(div()(gui.Text("hello")))
	s.Assert(t).Snapshot("<div>hello</div>")
}

// ---------------------------------------------------------------------------
// AssertNodeRef
// ---------------------------------------------------------------------------

func TestAssertNodeRefHasText(t *testing.T) {
	s := Render(btn()(gui.Text("Click")))
	ref := s.GetByRole("button")
	AssertNode(t, ref).HasText("Click")
}

func TestAssertNodeRefContainsText(t *testing.T) {
	s := Render(div()(gui.Text("hello world")))
	ref := s.QueryAllByTag("div")[0]
	AssertNode(t, ref).ContainsText("hello")
}

func TestAssertNodeRefHasClass(t *testing.T) {
	s := Render(div(gui.Class("foo bar"))())
	ref := s.QueryAllByTag("div")[0]
	AssertNode(t, ref).HasClass("foo").HasClass("bar")
}

func TestAssertNodeRefHasTag(t *testing.T) {
	s := Render(btn()(gui.Text("Go")))
	ref := s.GetByRole("button")
	AssertNode(t, ref).HasTag("button")
}

func TestAssertNodeRefHasProp(t *testing.T) {
	s := Render(gui.A(gui.Href("/home"))(gui.Text("Home")))
	ref := s.GetByRole("link")
	AssertNode(t, ref).HasProp("href", "/home")
}

func TestAssertNodeRefHasChildren(t *testing.T) {
	s := Render(div()(gui.Text("a"), gui.Text("b")))
	ref := s.QueryAllByTag("div")[0]
	AssertNode(t, ref).HasChildren(2)
}

func TestAssertNodeRefIsEnabled(t *testing.T) {
	s := Render(btn()(gui.Text("Go")))
	ref := s.GetByRole("button")
	AssertNode(t, ref).IsEnabled()
}

func TestAssertNodeRefIsDisabled(t *testing.T) {
	s := Render(btn(gui.Disabled(true))(gui.Text("Go")))
	ref := s.GetByRole("button")
	AssertNode(t, ref).IsDisabled()
}

// ---------------------------------------------------------------------------
// Within (scoped queries)
// ---------------------------------------------------------------------------

func TestWithin(t *testing.T) {
	tree := div()(
		div(gui.Data("testid", "sidebar"))(
			gui.Span()(gui.Text("Sidebar")),
			btn()(gui.Text("Close")),
		),
		div(gui.Data("testid", "main"))(
			gui.Span()(gui.Text("Main")),
			btn()(gui.Text("Submit")),
		),
	)

	s := Render(tree)
	sidebar := s.GetByTestId("sidebar")
	scoped := Within(sidebar)

	// Should find "Close" button but not "Submit"
	ref := scoped.GetByRole("button")
	if ref.Text() != "Close" {
		t.Errorf("expected 'Close', got %q", ref.Text())
	}

	if scoped.QueryByText("Main") != nil {
		t.Error("Within(sidebar) should not see Main content")
	}
}

// ---------------------------------------------------------------------------
// QueryByProp
// ---------------------------------------------------------------------------

func TestQueryByProp(t *testing.T) {
	s := Render(div()(
		input(gui.Name("email"))(),
		input(gui.Name("password"))(),
	))

	ref := s.QueryByProp("name", "password")
	if ref == nil {
		t.Fatal("expected to find input")
	}
	if ref.Prop("name") != "password" {
		t.Errorf("name: got %v", ref.Prop("name"))
	}
}

func TestQueryByPropReturnsNil(t *testing.T) {
	s := Render(div()())
	if s.QueryByProp("name", "missing") != nil {
		t.Error("expected nil")
	}
}

// ---------------------------------------------------------------------------
// RenderFunc + Rerender
// ---------------------------------------------------------------------------

func TestRenderFuncAndRerender(t *testing.T) {
	count := 0
	renderFn := func() gui.Node {
		return div()(gui.Textf("Count: %d", count))
	}

	s := RenderFunc(renderFn)
	if !s.ContainsText("Count: 0") {
		t.Error("expected Count: 0")
	}

	count = 5
	s.Rerender()
	if !s.ContainsText("Count: 5") {
		t.Error("expected Count: 5")
	}
}

func TestRerenderPanicsOnNonFuncScreen(t *testing.T) {
	s := Render(div()())
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	s.Rerender()
}

// ---------------------------------------------------------------------------
// WaitFor
// ---------------------------------------------------------------------------

func TestWaitFor(t *testing.T) {
	count := 0
	renderFn := func() gui.Node {
		return div()(gui.Textf("v%d", count))
	}

	s := RenderFunc(renderFn)

	// Simulate delayed state change
	ok := WaitFor(s, func() bool {
		count++
		return s.ContainsText("v3")
	}, 10)

	if !ok {
		t.Error("expected WaitFor to succeed")
	}
}

func TestWaitForFailsIfNeverTrue(t *testing.T) {
	s := Render(div()(gui.Text("static")))
	ok := WaitFor(s, func() bool {
		return s.ContainsText("never")
	}, 5)

	if ok {
		t.Error("expected WaitFor to fail")
	}
}

// ---------------------------------------------------------------------------
// Complex integration: stateful counter component
// ---------------------------------------------------------------------------

func TestStatefulCounterIntegration(t *testing.T) {
	count := 0
	var increment func()

	renderFn := func() gui.Node {
		increment = func() { count++ }
		return div()(
			gui.Textf("Count: %d", count),
			btn(gui.OnClick(func() { increment() }))(gui.Text("+")),
		)
	}

	s := RenderFunc(renderFn)
	s.Assert(t).TextVisible("Count: 0")

	// Click the + button
	plusBtn := s.GetByText("+")
	s.Click(plusBtn)
	s.Rerender()
	s.Assert(t).TextVisible("Count: 1")

	// Click again
	plusBtn = s.GetByText("+")
	s.Click(plusBtn)
	s.Rerender()
	s.Assert(t).TextVisible("Count: 2")
}

// ---------------------------------------------------------------------------
// Complex integration: form with inputs
// ---------------------------------------------------------------------------

func TestFormIntegration(t *testing.T) {
	name := ""
	submitted := false

	renderFn := func() gui.Node {
		return gui.Form()(
			gui.El("input", gui.Props{
				"type":        "text",
				"placeholder": "Your name",
				"oninput": func(e gui.Event) {
					name = e.Value
				},
			}),
			btn(gui.OnClick(func() {
				submitted = true
			}))(gui.Text("Submit")),
			gui.P(gui.Data("testid", "preview"))(gui.Textf("Hello, %s", name)),
		)
	}

	s := RenderFunc(renderFn)

	// Type into the input
	nameInput := s.GetByPlaceholder("Your name")
	s.Type(nameInput, "Alice")
	s.Rerender()

	// Check preview
	preview := s.GetByTestId("preview")
	if preview.Text() != "Hello, Alice" {
		t.Errorf("preview: got %q", preview.Text())
	}

	// Submit
	submitBtn := s.GetByText("Submit")
	s.Click(submitBtn)

	if !submitted {
		t.Error("expected form to be submitted")
	}
}

// ---------------------------------------------------------------------------
// Complex integration: todo list
// ---------------------------------------------------------------------------

func TestTodoListIntegration(t *testing.T) {
	todos := []string{"Buy milk", "Write tests"}

	renderFn := func() gui.Node {
		items := make([]gui.Node, len(todos))
		for i, todo := range todos {
			items[i] = gui.Li()(gui.Text(todo))
		}
		return div()(
			gui.H1()(gui.Text("Todo List")),
			gui.Ul()(items...),
		)
	}

	s := RenderFunc(renderFn)

	// Verify heading
	s.Assert(t).HasRole("heading")
	heading := s.GetByRole("heading")
	AssertNode(t, heading).HasText("Todo List")

	// Verify list items
	items := s.QueryAllByRole("listitem")
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	AssertNode(t, items[0]).HasText("Buy milk")
	AssertNode(t, items[1]).HasText("Write tests")

	// Add a todo and rerender
	todos = append(todos, "Ship feature")
	s.Rerender()

	items = s.QueryAllByRole("listitem")
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	AssertNode(t, items[2]).HasText("Ship feature")
}

// ---------------------------------------------------------------------------
// Nested component rendering with Resolve
// ---------------------------------------------------------------------------

type badgeProps struct {
	Label string
}

func Badge(props badgeProps, children []gui.Node) gui.Node {
	return gui.Span(gui.Class("badge"))(gui.Text(props.Label))
}

func TestFuncComponentRendering(t *testing.T) {
	tree := div()(
		gui.Comp(Badge, badgeProps{Label: "New"}),
		gui.Text(" item"),
	)

	s := Render(tree)
	s.Assert(t).
		TextVisible("New").
		TextVisible("item").
		HasElement("span")

	badge := s.GetByText("New")
	AssertNode(t, badge).HasClass("badge")
}

// ---------------------------------------------------------------------------
// Fragments
// ---------------------------------------------------------------------------

func TestFragmentHandling(t *testing.T) {
	tree := div()(
		gui.Frag(
			gui.Text("A"),
			gui.Text("B"),
		),
		gui.Text("C"),
	)

	s := Render(tree)
	if s.TextContent() != "ABC" {
		t.Errorf("expected 'ABC', got %q", s.TextContent())
	}
}

// ---------------------------------------------------------------------------
// Debug (just ensure it doesn't panic)
// ---------------------------------------------------------------------------

func TestDebugDoesNotPanic(t *testing.T) {
	s := Render(div()(gui.Text("debug me")))
	s.Debug()
}

// ---------------------------------------------------------------------------
// HasClass on non-element
// ---------------------------------------------------------------------------

func TestHasClassOnNonElement(t *testing.T) {
	ref := &NodeRef{Node: gui.Text("hi")}
	if ref.HasClass("foo") {
		t.Error("expected false for text node")
	}
}

// ---------------------------------------------------------------------------
// Implicit role: navigation, form, table
// ---------------------------------------------------------------------------

func TestGetByRoleNavigation(t *testing.T) {
	s := Render(gui.Nav()(gui.Text("nav")))
	ref := s.GetByRole("navigation")
	if ref.Text() != "nav" {
		t.Errorf("got %q", ref.Text())
	}
}

func TestGetByRoleForm(t *testing.T) {
	s := Render(gui.Form()(gui.Text("form")))
	ref := s.GetByRole("form")
	if ref.Text() != "form" {
		t.Errorf("got %q", ref.Text())
	}
}

func TestGetByRoleTable(t *testing.T) {
	s := Render(gui.Table()(gui.Tr()(gui.Td()(gui.Text("cell")))))
	ref := s.GetByRole("table")
	if ref == nil {
		t.Fatal("expected table")
	}
}

func TestGetByRoleRow(t *testing.T) {
	s := Render(gui.Table()(gui.Tr()(gui.Td()(gui.Text("cell")))))
	ref := s.GetByRole("row")
	if ref == nil {
		t.Fatal("expected row")
	}
}

func TestGetByRoleCell(t *testing.T) {
	s := Render(gui.Table()(gui.Tr()(gui.Td()(gui.Text("cell")))))
	ref := s.GetByRole("cell")
	if ref.Text() != "cell" {
		t.Errorf("got %q", ref.Text())
	}
}
