package guitesting

import (
	"fmt"
	"strings"
	"testing"

	gui "github.com/readmedotmd/gui.md"
	"github.com/readmedotmd/gui.md/html"
)

// Asserter wraps a Screen with a testing.T for fluent assertions.
// Use screen.Assert(t) to get one.
type Asserter struct {
	t      *testing.T
	screen *Screen
}

// Assert returns an Asserter bound to the given testing.T.
func (s *Screen) Assert(t *testing.T) *Asserter {
	t.Helper()
	return &Asserter{t: t, screen: s}
}

// TextVisible asserts that the rendered tree contains the given text.
func (a *Asserter) TextVisible(text string) *Asserter {
	a.t.Helper()
	if !a.screen.ContainsText(text) {
		a.t.Errorf("expected text %q to be visible in:\n%s", text, a.screen.HTML())
	}
	return a
}

// TextNotVisible asserts that the rendered tree does not contain the given text.
func (a *Asserter) TextNotVisible(text string) *Asserter {
	a.t.Helper()
	if a.screen.ContainsText(text) {
		a.t.Errorf("expected text %q to NOT be visible in:\n%s", text, a.screen.HTML())
	}
	return a
}

// HasElement asserts that at least one element with the given tag exists.
func (a *Asserter) HasElement(tag string) *Asserter {
	a.t.Helper()
	refs := a.screen.QueryAllByTag(tag)
	if len(refs) == 0 {
		a.t.Errorf("expected at least one <%s> element in:\n%s", tag, a.screen.HTML())
	}
	return a
}

// HasNoElement asserts that no element with the given tag exists.
func (a *Asserter) HasNoElement(tag string) *Asserter {
	a.t.Helper()
	refs := a.screen.QueryAllByTag(tag)
	if len(refs) > 0 {
		a.t.Errorf("expected no <%s> elements, found %d", tag, len(refs))
	}
	return a
}

// HasTestId asserts that an element with data-testid=id exists.
func (a *Asserter) HasTestId(id string) *Asserter {
	a.t.Helper()
	if a.screen.QueryByTestId(id) == nil {
		a.t.Errorf("expected element with data-testid=%q in:\n%s", id, a.screen.HTML())
	}
	return a
}

// HasNoTestId asserts that no element with data-testid=id exists.
func (a *Asserter) HasNoTestId(id string) *Asserter {
	a.t.Helper()
	if a.screen.QueryByTestId(id) != nil {
		a.t.Errorf("expected no element with data-testid=%q", id)
	}
	return a
}

// HasRole asserts that at least one element matching the role exists.
func (a *Asserter) HasRole(role string) *Asserter {
	a.t.Helper()
	refs := a.screen.QueryAllByRole(role)
	if len(refs) == 0 {
		a.t.Errorf("expected at least one element with role %q in:\n%s", role, a.screen.HTML())
	}
	return a
}

// ElementCount asserts that exactly n elements with the given tag exist.
func (a *Asserter) ElementCount(tag string, n int) *Asserter {
	a.t.Helper()
	refs := a.screen.QueryAllByTag(tag)
	if len(refs) != n {
		a.t.Errorf("expected %d <%s> elements, got %d", n, tag, len(refs))
	}
	return a
}

// HTMLContains asserts the rendered HTML contains the substring.
func (a *Asserter) HTMLContains(substr string) *Asserter {
	a.t.Helper()
	if !strings.Contains(a.screen.HTML(), substr) {
		a.t.Errorf("expected HTML to contain %q in:\n%s", substr, a.screen.HTML())
	}
	return a
}

// HTMLNotContains asserts the rendered HTML does NOT contain the substring.
func (a *Asserter) HTMLNotContains(substr string) *Asserter {
	a.t.Helper()
	if strings.Contains(a.screen.HTML(), substr) {
		a.t.Errorf("expected HTML to NOT contain %q", substr)
	}
	return a
}

// NodeHasText asserts that the given node ref has the expected text content.
func (a *Asserter) NodeHasText(ref *NodeRef, text string) *Asserter {
	a.t.Helper()
	got := ref.Text()
	if got != text {
		a.t.Errorf("expected node text %q, got %q", text, got)
	}
	return a
}

// NodeHasClass asserts the node has the given CSS class.
func (a *Asserter) NodeHasClass(ref *NodeRef, class string) *Asserter {
	a.t.Helper()
	if !ref.HasClass(class) {
		a.t.Errorf("expected node to have class %q", class)
	}
	return a
}

// NodeHasProp asserts the node has the given prop with the expected value.
func (a *Asserter) NodeHasProp(ref *NodeRef, key string, value any) *Asserter {
	a.t.Helper()
	got := ref.Prop(key)
	if fmt.Sprint(got) != fmt.Sprint(value) {
		a.t.Errorf("expected prop %q=%v, got %v", key, value, got)
	}
	return a
}

// NodeEnabled asserts the node does not have disabled=true.
func (a *Asserter) NodeEnabled(ref *NodeRef) *Asserter {
	a.t.Helper()
	if v, ok := ref.Prop("disabled").(bool); ok && v {
		a.t.Errorf("expected node to be enabled")
	}
	return a
}

// NodeDisabled asserts the node has disabled=true.
func (a *Asserter) NodeDisabled(ref *NodeRef) *Asserter {
	a.t.Helper()
	v, ok := ref.Prop("disabled").(bool)
	if !ok || !v {
		a.t.Errorf("expected node to be disabled")
	}
	return a
}

// Snapshot compares the rendered HTML to expected. On mismatch, it shows a diff-style error.
func (a *Asserter) Snapshot(expected string) *Asserter {
	a.t.Helper()
	got := a.screen.HTML()
	if got != expected {
		a.t.Errorf("snapshot mismatch:\nwant: %s\ngot:  %s", expected, got)
	}
	return a
}

// ---------------------------------------------------------------------------
// NodeRef assertions for chaining
// ---------------------------------------------------------------------------

// AssertNodeRef provides fluent assertions on a single NodeRef.
type AssertNodeRef struct {
	t   *testing.T
	ref *NodeRef
}

// AssertNode returns a fluent assertion object for a NodeRef.
func AssertNode(t *testing.T, ref *NodeRef) *AssertNodeRef {
	return &AssertNodeRef{t: t, ref: ref}
}

// HasText asserts the node's text content equals expected.
func (a *AssertNodeRef) HasText(text string) *AssertNodeRef {
	a.t.Helper()
	got := a.ref.Text()
	if got != text {
		a.t.Errorf("expected text %q, got %q", text, got)
	}
	return a
}

// ContainsText asserts the node's text content contains substr.
func (a *AssertNodeRef) ContainsText(substr string) *AssertNodeRef {
	a.t.Helper()
	got := a.ref.Text()
	if !strings.Contains(got, substr) {
		a.t.Errorf("expected text to contain %q, got %q", substr, got)
	}
	return a
}

// HasClass asserts the node has the given CSS class.
func (a *AssertNodeRef) HasClass(class string) *AssertNodeRef {
	a.t.Helper()
	if !a.ref.HasClass(class) {
		a.t.Errorf("expected class %q", class)
	}
	return a
}

// HasTag asserts the element has the given tag name.
func (a *AssertNodeRef) HasTag(tag string) *AssertNodeRef {
	a.t.Helper()
	el := a.ref.Element()
	if el.Tag != tag {
		a.t.Errorf("expected tag %q, got %q", tag, el.Tag)
	}
	return a
}

// HasProp asserts the node has the given prop value.
func (a *AssertNodeRef) HasProp(key string, value any) *AssertNodeRef {
	a.t.Helper()
	got := a.ref.Prop(key)
	if fmt.Sprint(got) != fmt.Sprint(value) {
		a.t.Errorf("expected prop %q=%v, got %v", key, value, got)
	}
	return a
}

// HasChildren asserts the element has exactly n children.
func (a *AssertNodeRef) HasChildren(n int) *AssertNodeRef {
	a.t.Helper()
	el := a.ref.Element()
	if len(el.Children) != n {
		a.t.Errorf("expected %d children, got %d", n, len(el.Children))
	}
	return a
}

// IsDisabled asserts the node has disabled=true.
func (a *AssertNodeRef) IsDisabled() *AssertNodeRef {
	a.t.Helper()
	v, ok := a.ref.Prop("disabled").(bool)
	if !ok || !v {
		a.t.Errorf("expected node to be disabled")
	}
	return a
}

// IsEnabled asserts the node does NOT have disabled=true.
func (a *AssertNodeRef) IsEnabled() *AssertNodeRef {
	a.t.Helper()
	if v, ok := a.ref.Prop("disabled").(bool); ok && v {
		a.t.Errorf("expected node to be enabled")
	}
	return a
}

// WaitFor is a helper for checking conditions after async state updates.
// It re-renders the screen up to maxRetries times and checks if the condition
// holds. This is useful for testing components with delayed state changes.
func WaitFor(s *Screen, check func() bool, maxRetries int) bool {
	for i := 0; i < maxRetries; i++ {
		if check() {
			return true
		}
		if s.renderFn != nil {
			s.Rerender()
		}
	}
	return false
}

// Within scopes queries to a subtree rooted at the given node.
func Within(ref *NodeRef) *Screen {
	return &Screen{
		root:  ref.Node,
		htmlR: html.New(),
	}
}

// QueryByProp finds the first element where props[key] matches value (as strings).
func (s *Screen) QueryByProp(key string, value any) *NodeRef {
	valStr := fmt.Sprint(value)
	var result *NodeRef
	walk(s.root, nil, func(node gui.Node, path []int) {
		if result != nil {
			return
		}
		if el, ok := node.(*gui.Element); ok {
			if fmt.Sprint(el.Props[key]) == valStr {
				cp := make([]int, len(path))
				copy(cp, path)
				result = &NodeRef{Node: node, Path: cp}
			}
		}
	})
	return result
}
