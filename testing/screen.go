// Package guitesting provides a React Testing Library-style API for testing
// gui.md component trees. It renders components to an in-memory tree and
// provides high-level queries (by text, role, test ID) and interaction
// helpers (click, type, etc.) that mirror how a user would interact with
// the rendered UI.
//
// Example:
//
//	screen := guitesting.Render(MyComponent())
//	btn := screen.GetByText("Submit")
//	screen.Click(btn)
//	screen.AssertText("Success")
package guitesting

import (
	"fmt"
	"strings"

	gui "github.com/readmedotmd/gui.md"
	"github.com/readmedotmd/gui.md/html"
)

// Screen is the main entry point for querying and interacting with a rendered
// component tree. It mirrors React Testing Library's "screen" object.
type Screen struct {
	root       gui.Node
	reconciler *gui.Reconciler
	renderFn   func() gui.Node
	htmlR      *html.Renderer
}

// Render renders a gui.Node tree and returns a Screen for querying it.
// For stateful components that need re-rendering, use RenderFunc instead.
func Render(node gui.Node) *Screen {
	resolved := gui.Resolve(node)
	return &Screen{
		root:  resolved,
		htmlR: html.New(),
	}
}

// RenderFunc renders a component tree produced by fn and returns a Screen.
// Calling Rerender() will re-invoke fn and update the tree. This is useful
// for testing stateful components that change over time.
func RenderFunc(fn func() gui.Node) *Screen {
	rec := gui.NewReconciler()
	resolved := rec.Resolve(fn(), nil)
	return &Screen{
		root:       resolved,
		reconciler: rec,
		renderFn:   fn,
		htmlR:      html.New(),
	}
}

// Rerender re-invokes the render function and updates the tree.
// Only works with screens created via RenderFunc.
func (s *Screen) Rerender() {
	if s.renderFn == nil {
		panic("guitesting: Rerender called on non-func screen; use RenderFunc")
	}
	s.root = s.reconciler.Resolve(s.renderFn(), nil)
}

// Root returns the resolved root node of the rendered tree.
func (s *Screen) Root() gui.Node { return s.root }

// HTML returns the rendered HTML string of the current tree.
func (s *Screen) HTML() string {
	return s.htmlR.RenderString(s.root)
}

// Debug prints the HTML to stdout for debugging. Returns the Screen for chaining.
func (s *Screen) Debug() *Screen {
	fmt.Println(s.HTML())
	return s
}

// ---------------------------------------------------------------------------
// Query helpers
// ---------------------------------------------------------------------------

// NodeRef is a reference to a node in the rendered tree, along with its path.
type NodeRef struct {
	Node gui.Node
	Path []int // index path from root
}

// Element returns the underlying *gui.Element or panics.
func (n *NodeRef) Element() *gui.Element {
	if el, ok := n.Node.(*gui.Element); ok {
		return el
	}
	panic(fmt.Sprintf("guitesting: NodeRef is %T, not *gui.Element", n.Node))
}

// Text returns the text content of this node (recursively collected).
func (n *NodeRef) Text() string {
	return collectText(n.Node)
}

// Prop returns the prop value for the given key, or nil.
func (n *NodeRef) Prop(key string) any {
	if el, ok := n.Node.(*gui.Element); ok {
		return el.Props[key]
	}
	return nil
}

// HasClass reports whether the node has the given CSS class.
func (n *NodeRef) HasClass(class string) bool {
	if el, ok := n.Node.(*gui.Element); ok {
		if c, ok := el.Props["class"].(string); ok {
			for _, part := range strings.Fields(c) {
				if part == class {
					return true
				}
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// GetBy* queries — panic if not found (like RTL's getBy*)
// ---------------------------------------------------------------------------

// GetByText finds the first node whose recursive text content contains text.
func (s *Screen) GetByText(text string) *NodeRef {
	ref := s.QueryByText(text)
	if ref == nil {
		panic(fmt.Sprintf("guitesting: GetByText(%q): not found in:\n%s", text, s.HTML()))
	}
	return ref
}

// GetByTestId finds the first element with data-testid matching id.
func (s *Screen) GetByTestId(id string) *NodeRef {
	ref := s.QueryByTestId(id)
	if ref == nil {
		panic(fmt.Sprintf("guitesting: GetByTestId(%q): not found in:\n%s", id, s.HTML()))
	}
	return ref
}

// GetByRole finds the first element whose tag or "role" prop matches role.
// Implicit role mapping: "button" matches <button>, "link" matches <a>,
// "heading" matches <h1>-<h6>, "textbox" matches <input>/<textarea>,
// "list" matches <ul>/<ol>, "listitem" matches <li>.
func (s *Screen) GetByRole(role string) *NodeRef {
	ref := s.QueryByRole(role)
	if ref == nil {
		panic(fmt.Sprintf("guitesting: GetByRole(%q): not found in:\n%s", role, s.HTML()))
	}
	return ref
}

// GetByPlaceholder finds the first element with a matching placeholder prop.
func (s *Screen) GetByPlaceholder(text string) *NodeRef {
	ref := s.QueryByPlaceholder(text)
	if ref == nil {
		panic(fmt.Sprintf("guitesting: GetByPlaceholder(%q): not found", text))
	}
	return ref
}

// ---------------------------------------------------------------------------
// QueryBy* queries — return nil if not found (like RTL's queryBy*)
// ---------------------------------------------------------------------------

// QueryByText returns the first node containing text, or nil.
func (s *Screen) QueryByText(text string) *NodeRef {
	refs := s.QueryAllByText(text)
	if len(refs) == 0 {
		return nil
	}
	return refs[0]
}

// QueryByTestId returns the first element with data-testid=id, or nil.
func (s *Screen) QueryByTestId(id string) *NodeRef {
	refs := s.QueryAllByTestId(id)
	if len(refs) == 0 {
		return nil
	}
	return refs[0]
}

// QueryByRole returns the first element matching role, or nil.
func (s *Screen) QueryByRole(role string) *NodeRef {
	refs := s.QueryAllByRole(role)
	if len(refs) == 0 {
		return nil
	}
	return refs[0]
}

// QueryByPlaceholder returns the first element with matching placeholder, or nil.
func (s *Screen) QueryByPlaceholder(text string) *NodeRef {
	refs := s.QueryAllByPlaceholder(text)
	if len(refs) == 0 {
		return nil
	}
	return refs[0]
}

// ---------------------------------------------------------------------------
// QueryAllBy* queries — return all matches (like RTL's queryAllBy*)
// ---------------------------------------------------------------------------

// QueryAllByText returns all Element or Fragment nodes whose recursive text
// content contains text. Raw TextNodes are excluded — the returned refs are
// always the nearest parent element, which is more useful for interactions
// (clicking, reading props, etc.), mirroring React Testing Library behavior.
// Results are ordered deepest-first (most specific element first).
func (s *Screen) QueryAllByText(text string) []*NodeRef {
	var results []*NodeRef
	walk(s.root, nil, func(node gui.Node, path []int) {
		// Skip raw text nodes — return containing elements instead.
		if _, isText := node.(*gui.TextNode); isText {
			return
		}
		if strings.Contains(collectText(node), text) {
			cp := make([]int, len(path))
			copy(cp, path)
			results = append(results, &NodeRef{Node: node, Path: cp})
		}
	})
	// Deepest matches first (most specific element).
	if len(results) > 1 {
		for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
			results[i], results[j] = results[j], results[i]
		}
	}
	return results
}

// QueryAllByTestId returns all elements with data-testid=id.
func (s *Screen) QueryAllByTestId(id string) []*NodeRef {
	var results []*NodeRef
	walk(s.root, nil, func(node gui.Node, path []int) {
		if el, ok := node.(*gui.Element); ok {
			if tid, ok := el.Props["data-testid"].(string); ok && tid == id {
				cp := make([]int, len(path))
				copy(cp, path)
				results = append(results, &NodeRef{Node: node, Path: cp})
			}
		}
	})
	return results
}

// QueryAllByRole returns all elements matching role.
func (s *Screen) QueryAllByRole(role string) []*NodeRef {
	var results []*NodeRef
	walk(s.root, nil, func(node gui.Node, path []int) {
		if el, ok := node.(*gui.Element); ok {
			if matchesRole(el, role) {
				cp := make([]int, len(path))
				copy(cp, path)
				results = append(results, &NodeRef{Node: node, Path: cp})
			}
		}
	})
	return results
}

// QueryAllByPlaceholder returns all elements with matching placeholder prop.
func (s *Screen) QueryAllByPlaceholder(text string) []*NodeRef {
	var results []*NodeRef
	walk(s.root, nil, func(node gui.Node, path []int) {
		if el, ok := node.(*gui.Element); ok {
			if ph, ok := el.Props["placeholder"].(string); ok && strings.Contains(ph, text) {
				cp := make([]int, len(path))
				copy(cp, path)
				results = append(results, &NodeRef{Node: node, Path: cp})
			}
		}
	})
	return results
}

// QueryAllByTag returns all elements with the given tag name.
func (s *Screen) QueryAllByTag(tag string) []*NodeRef {
	var results []*NodeRef
	walk(s.root, nil, func(node gui.Node, path []int) {
		if el, ok := node.(*gui.Element); ok && el.Tag == tag {
			cp := make([]int, len(path))
			copy(cp, path)
			results = append(results, &NodeRef{Node: node, Path: cp})
		}
	})
	return results
}

// ---------------------------------------------------------------------------
// Interaction helpers
// ---------------------------------------------------------------------------

// Click simulates a click on the given node by invoking its "onclick" or
// "onclick" event handler. Supports both func() and func(gui.Event).
func (s *Screen) Click(ref *NodeRef) {
	el, ok := ref.Node.(*gui.Element)
	if !ok {
		panic("guitesting: Click requires an Element node")
	}
	if handler, ok := el.Props["onclick"].(func()); ok {
		handler()
		return
	}
	if handler, ok := el.Props["onclick"].(func(gui.Event)); ok {
		handler(gui.Event{Type: "click"})
		return
	}
	panic(fmt.Sprintf("guitesting: Click: no onclick handler on <%s>", el.Tag))
}

// FireEvent fires a named event on the node. It looks for "on<event>" in props.
func (s *Screen) FireEvent(ref *NodeRef, event string, ev gui.Event) {
	el, ok := ref.Node.(*gui.Element)
	if !ok {
		panic("guitesting: FireEvent requires an Element node")
	}
	key := "on" + event
	if handler, ok := el.Props[key].(func()); ok {
		handler()
		return
	}
	if handler, ok := el.Props[key].(func(gui.Event)); ok {
		ev.Type = event
		handler(ev)
		return
	}
	panic(fmt.Sprintf("guitesting: FireEvent(%q): no handler on <%s>", event, el.Tag))
}

// Type simulates typing text into an input element by firing an "oninput"
// event with Value set for each character, then a final event with the full text.
func (s *Screen) Type(ref *NodeRef, text string) {
	el, ok := ref.Node.(*gui.Element)
	if !ok {
		panic("guitesting: Type requires an Element node")
	}
	key := "oninput"
	if handler, ok := el.Props[key].(func(gui.Event)); ok {
		// Fire per-character events, then the full string
		for i := 1; i <= len(text); i++ {
			handler(gui.Event{Type: "input", Value: text[:i]})
		}
		return
	}
	if handler, ok := el.Props["onchange"].(func(gui.Event)); ok {
		handler(gui.Event{Type: "change", Value: text})
		return
	}
	panic(fmt.Sprintf("guitesting: Type: no oninput/onchange handler on <%s>", el.Tag))
}

// Clear simulates clearing an input by firing oninput with empty value.
func (s *Screen) Clear(ref *NodeRef) {
	el, ok := ref.Node.(*gui.Element)
	if !ok {
		panic("guitesting: Clear requires an Element node")
	}
	if handler, ok := el.Props["oninput"].(func(gui.Event)); ok {
		handler(gui.Event{Type: "input", Value: ""})
		return
	}
	if handler, ok := el.Props["onchange"].(func(gui.Event)); ok {
		handler(gui.Event{Type: "change", Value: ""})
		return
	}
	panic(fmt.Sprintf("guitesting: Clear: no oninput/onchange handler on <%s>", el.Tag))
}

// KeyPress simulates a keypress event on the node.
func (s *Screen) KeyPress(ref *NodeRef, key string) {
	s.FireEvent(ref, "keypress", gui.Event{Key: key})
}

// ---------------------------------------------------------------------------
// Assertion helpers (for use with testing.T)
// ---------------------------------------------------------------------------

// ContainsText reports whether the rendered tree contains the given text.
func (s *Screen) ContainsText(text string) bool {
	return strings.Contains(collectText(s.root), text)
}

// TextContent returns the full text content of the rendered tree.
func (s *Screen) TextContent() string {
	return collectText(s.root)
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

// walk traverses the node tree depth-first and calls fn for each node.
func walk(node gui.Node, path []int, fn func(gui.Node, []int)) {
	if node == nil {
		return
	}
	fn(node, path)
	switch n := node.(type) {
	case *gui.Element:
		for i, child := range n.Children {
			walk(child, append(path, i), fn)
		}
	case *gui.Fragment:
		for i, child := range n.Children {
			walk(child, append(path, i), fn)
		}
	}
}

// collectText recursively collects all text content from a node tree.
func collectText(node gui.Node) string {
	if node == nil {
		return ""
	}
	switch n := node.(type) {
	case *gui.TextNode:
		return n.Content
	case *gui.Element:
		var sb strings.Builder
		for _, child := range n.Children {
			sb.WriteString(collectText(child))
		}
		return sb.String()
	case *gui.Fragment:
		var sb strings.Builder
		for _, child := range n.Children {
			sb.WriteString(collectText(child))
		}
		return sb.String()
	}
	return ""
}

// matchesRole checks if an element matches a WAI-ARIA role, either explicitly
// via a "role" prop or implicitly from the tag name.
func matchesRole(el *gui.Element, role string) bool {
	// Explicit role prop
	if r, ok := el.Props["role"].(string); ok && r == role {
		return true
	}
	// Implicit role from tag
	switch role {
	case "button":
		return el.Tag == "button"
	case "link":
		return el.Tag == "a"
	case "heading":
		switch el.Tag {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			return true
		}
	case "textbox":
		if el.Tag == "textarea" {
			return true
		}
		if el.Tag == "input" {
			tp, _ := el.Props["type"].(string)
			return tp == "" || tp == "text" || tp == "email" || tp == "password" || tp == "search" || tp == "tel" || tp == "url"
		}
	case "list":
		return el.Tag == "ul" || el.Tag == "ol"
	case "listitem":
		return el.Tag == "li"
	case "checkbox":
		if el.Tag == "input" {
			tp, _ := el.Props["type"].(string)
			return tp == "checkbox"
		}
	case "radio":
		if el.Tag == "input" {
			tp, _ := el.Props["type"].(string)
			return tp == "radio"
		}
	case "img", "image":
		return el.Tag == "img"
	case "navigation":
		return el.Tag == "nav"
	case "form":
		return el.Tag == "form"
	case "table":
		return el.Tag == "table"
	case "row":
		return el.Tag == "tr"
	case "cell":
		return el.Tag == "td"
	case "columnheader":
		return el.Tag == "th"
	}
	return false
}
