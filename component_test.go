package gui

import (
	"testing"
)

// --- Test state types ---

type counterState struct{ Count int }
type greetState struct {
	Name    string
	Greeted bool
}

// --- Test component types ---

// testCounter is a minimal stateful component with counterState.
type testCounter struct {
	BaseComponent[Props, counterState]
}

func (c *testCounter) Render() Node {
	s := c.State()
	return El("div", nil, Textf("Count: %d", s.Count))
}

// lifecycleTracker records every lifecycle hook and Render call in order,
// allowing tests to assert the exact call sequence.
type lifecycleTracker struct {
	BaseComponent[Props, greetState]
	Calls []string
}

func (l *lifecycleTracker) WillMount() { l.Calls = append(l.Calls, "WillMount") }
func (l *lifecycleTracker) DidMount()  { l.Calls = append(l.Calls, "DidMount") }
func (l *lifecycleTracker) Render() Node {
	l.Calls = append(l.Calls, "Render")
	return El("span", nil, Text(l.State().Name))
}

// wrapperComponent returns a ComponentNode as its output so that
// Resolve must recurse through two levels of component.
type wrapperComponent struct {
	BaseComponent[Props, counterState]
}

func (w *wrapperComponent) Render() Node {
	inner := &testCounter{}
	inner.SetState(counterState{Count: 7})
	return Mount(inner, nil)
}

// --- Compile-time interface checks ---

var (
	_ Node       = (*ComponentNode)(nil)
	_ Renderable = (*testCounter)(nil)
	_ Renderable = (*lifecycleTracker)(nil)
)

// --- Functional component helpers ---

// greetFunc is a FuncComponent used across multiple test cases.
func greetFunc(props Props, children []Node) Node {
	name, _ := props["name"].(string)
	return El("p", nil, Textf("Hello, %s!", name))
}

// passthroughFunc simply returns the first child, used to test children
// propagation through Comp.
func passthroughFunc(props Props, children []Node) Node {
	if len(children) == 0 {
		return El("div", nil)
	}
	return El("div", nil, children...)
}

// --- Tests ---

// TestFuncComponent_ReturnsCorrectNode verifies that a functional component
// returns a properly constructed node when called directly.
func TestFuncComponent_ReturnsCorrectNode(t *testing.T) {
	node := greetFunc(Props{"name": "Alice"}, nil)
	el, ok := node.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", node)
	}
	if el.Tag != "p" {
		t.Errorf("tag: got %q, want %q", el.Tag, "p")
	}
	if len(el.Children) != 1 {
		t.Fatalf("children: got %d, want 1", len(el.Children))
	}
	txt, ok := el.Children[0].(*TextNode)
	if !ok {
		t.Fatalf("child type: expected *TextNode, got %T", el.Children[0])
	}
	if txt.Content != "Hello, Alice!" {
		t.Errorf("content: got %q, want %q", txt.Content, "Hello, Alice!")
	}
}

// TestComp_WrapsWithPropsAndChildren verifies that Comp stores the closure
// and children on the returned ComponentNode. Props are captured inside
// the closure, not stored on the node.
func TestComp_WrapsWithPropsAndChildren(t *testing.T) {
	child := Text("child")
	cn := Comp(greetFunc, Props{"name": "Bob"}, child)

	if cn.Func == nil {
		t.Fatal("Func field must not be nil")
	}
	if cn.Stateful != nil {
		t.Error("Stateful must be nil for a functional component")
	}
	if len(cn.Children) != 1 {
		t.Fatalf("children count: got %d, want 1", len(cn.Children))
	}
	if cn.Children[0] != child {
		t.Error("children[0] must be the child node passed to Comp")
	}
}

// TestComp_NilPropsCreatesValidClosure verifies that passing nil props to Comp
// creates a working closure that can be resolved without panic.
func TestComp_NilPropsCreatesValidClosure(t *testing.T) {
	cn := Comp(greetFunc, nil)
	if cn.Func == nil {
		t.Fatal("Func must not be nil")
	}
	// Resolving should work — greetFunc handles nil props gracefully.
	result := Resolve(cn)
	if result == nil {
		t.Error("Resolve must return a non-nil node")
	}
}

// TestBaseComponent_StateZeroValueInitially checks that State() returns the
// zero value of the type parameter before any SetState call.
func TestBaseComponent_StateZeroValueInitially(t *testing.T) {
	c := &testCounter{}
	s := c.State()
	if s.Count != 0 {
		t.Errorf("initial Count: got %d, want 0", s.Count)
	}
}

// TestBaseComponent_SetState_ReplacesState verifies that SetState stores the
// new value and that State() returns it immediately after.
func TestBaseComponent_SetState_ReplacesState(t *testing.T) {
	c := &testCounter{}
	c.SetState(counterState{Count: 42})
	if c.State().Count != 42 {
		t.Errorf("Count after SetState: got %d, want 42", c.State().Count)
	}
}

// TestBaseComponent_UpdateState_AppliesFunction verifies that UpdateState
// applies the given function to the current state and stores the result.
func TestBaseComponent_UpdateState_AppliesFunction(t *testing.T) {
	c := &testCounter{}
	c.SetState(counterState{Count: 10})
	c.UpdateState(func(s counterState) counterState {
		s.Count += 5
		return s
	})
	if c.State().Count != 15 {
		t.Errorf("Count after UpdateState: got %d, want 15", c.State().Count)
	}
}

// TestBaseComponent_UpdateState_SeesCurrentState verifies that the function
// passed to UpdateState receives the state that existed before the call.
func TestBaseComponent_UpdateState_SeesCurrentState(t *testing.T) {
	c := &testCounter{}
	c.SetState(counterState{Count: 3})
	var seen int
	c.UpdateState(func(s counterState) counterState {
		seen = s.Count
		s.Count++
		return s
	})
	if seen != 3 {
		t.Errorf("UpdateState fn received Count=%d, want 3", seen)
	}
	if c.State().Count != 4 {
		t.Errorf("Count after UpdateState: got %d, want 4", c.State().Count)
	}
}

// TestBaseComponent_Props_ReturnsMountedProps verifies that Props() returns
// the props injected by Mount().
func TestBaseComponent_Props_ReturnsMountedProps(t *testing.T) {
	c := &testCounter{}
	Mount(c, Props{"label": "clicks"})
	if c.Props()["label"] != "clicks" {
		t.Errorf("Props label: got %v, want %q", c.Props()["label"], "clicks")
	}
}

// TestBaseComponent_Children_ReturnsMountedChildren verifies that Children()
// returns the child nodes injected by Mount().
func TestBaseComponent_Children_ReturnsMountedChildren(t *testing.T) {
	child1 := Text("a")
	child2 := Text("b")
	c := &testCounter{}
	Mount(c, nil, child1, child2)
	got := c.Children()
	if len(got) != 2 {
		t.Fatalf("Children len: got %d, want 2", len(got))
	}
	if got[0] != child1 || got[1] != child2 {
		t.Error("Children values do not match those passed to Mount")
	}
}

// TestLifecycle_HooksCalledInOrder verifies the lifecycle sequence:
// WillMount -> Render -> DidMount, exactly once each, in that order.
func TestLifecycle_HooksCalledInOrder(t *testing.T) {
	tracker := &lifecycleTracker{}
	tracker.SetState(greetState{Name: "Eve"})

	cn := Mount(tracker, nil)
	Resolve(cn)

	want := []string{"WillMount", "Render", "DidMount"}
	if len(tracker.Calls) != len(want) {
		t.Fatalf("call count: got %d (%v), want %d (%v)",
			len(tracker.Calls), tracker.Calls, len(want), want)
	}
	for i, call := range tracker.Calls {
		if call != want[i] {
			t.Errorf("call[%d]: got %q, want %q", i, call, want[i])
		}
	}
}

// TestMount_SetsPropsAndChildren verifies that Mount injects props and children
// into the component's base before returning the ComponentNode.
func TestMount_SetsPropsAndChildren(t *testing.T) {
	child := Text("kid")
	c := &testCounter{}
	cn := Mount(c, Props{"x": 1}, child)

	// ComponentNode must reference the component.
	if cn.Stateful != c {
		t.Error("ComponentNode.Stateful must point to the mounted component")
	}
	// props visible via the component's own accessors.
	if c.Props()["x"] != 1 {
		t.Errorf("component Props x: got %v, want 1", c.Props()["x"])
	}
	if len(cn.Children) != 1 || cn.Children[0] != child {
		t.Error("ComponentNode.Children does not match what was passed to Mount")
	}
	if len(c.Children()) != 1 || c.Children()[0] != child {
		t.Error("component Children() does not match what was passed to Mount")
	}
}

// TestMount_NilPropsDefaultsToNil verifies that passing nil props to Mount
// results in a nil Props on the component (nil is the zero value for a map).
func TestMount_NilPropsDefaultsToNil(t *testing.T) {
	c := &testCounter{}
	Mount(c, nil)
	if c.Props() != nil {
		t.Error("component Props() must be nil after Mount with nil props")
	}
}

// TestResolve_FunctionalComponent verifies that Resolve calls the FuncComponent
// and returns the resolved element tree.
func TestResolve_FunctionalComponent(t *testing.T) {
	cn := Comp(greetFunc, Props{"name": "Carol"})
	result := Resolve(cn)

	el, ok := result.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", result)
	}
	if el.Tag != "p" {
		t.Errorf("tag: got %q, want %q", el.Tag, "p")
	}
	if len(el.Children) != 1 {
		t.Fatalf("children: got %d, want 1", len(el.Children))
	}
	txt, ok := el.Children[0].(*TextNode)
	if !ok {
		t.Fatalf("child type: expected *TextNode, got %T", el.Children[0])
	}
	if txt.Content != "Hello, Carol!" {
		t.Errorf("content: got %q, want %q", txt.Content, "Hello, Carol!")
	}
}

// TestResolve_StatefulComponent verifies that Resolve calls Render on a
// stateful component and returns the resolved concrete node.
func TestResolve_StatefulComponent(t *testing.T) {
	c := &testCounter{}
	c.SetState(counterState{Count: 5})
	cn := Mount(c, nil)
	result := Resolve(cn)

	el, ok := result.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", result)
	}
	if el.Tag != "div" {
		t.Errorf("tag: got %q, want %q", el.Tag, "div")
	}
	if len(el.Children) != 1 {
		t.Fatalf("children: got %d, want 1", len(el.Children))
	}
	txt, ok := el.Children[0].(*TextNode)
	if !ok {
		t.Fatalf("child: expected *TextNode, got %T", el.Children[0])
	}
	if txt.Content != "Count: 5" {
		t.Errorf("content: got %q, want %q", txt.Content, "Count: 5")
	}
}

// TestResolve_NestedComponents verifies that Resolve recurses through a
// component whose Render returns another ComponentNode (two levels deep).
func TestResolve_NestedComponents(t *testing.T) {
	outer := &wrapperComponent{}
	cn := Mount(outer, nil)
	result := Resolve(cn)

	// The outer component returns Mount(&testCounter{...}), which Resolve
	// must further resolve to the testCounter's concrete Element output.
	el, ok := result.(*Element)
	if !ok {
		t.Fatalf("expected *Element after double resolution, got %T", result)
	}
	if el.Tag != "div" {
		t.Errorf("tag: got %q, want %q", el.Tag, "div")
	}
	if len(el.Children) != 1 {
		t.Fatalf("children: got %d, want 1", len(el.Children))
	}
	txt, ok := el.Children[0].(*TextNode)
	if !ok {
		t.Fatalf("child: expected *TextNode, got %T", el.Children[0])
	}
	if txt.Content != "Count: 7" {
		t.Errorf("content: got %q, want %q", txt.Content, "Count: 7")
	}
}

// TestResolve_PassesThroughElement verifies that a plain *Element is returned
// unchanged (but with its children recursively resolved).
func TestResolve_PassesThroughElement(t *testing.T) {
	original := El("section", Props{"id": "root"}, Text("hello"))
	result := Resolve(original)

	el, ok := result.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", result)
	}
	if el.Tag != "section" {
		t.Errorf("tag: got %q, want %q", el.Tag, "section")
	}
	if el.Props["id"] != "root" {
		t.Errorf("props id: got %v, want %q", el.Props["id"], "root")
	}
	if len(el.Children) != 1 {
		t.Fatalf("children: got %d, want 1", len(el.Children))
	}
	txt, ok := el.Children[0].(*TextNode)
	if !ok {
		t.Fatalf("child: expected *TextNode, got %T", el.Children[0])
	}
	if txt.Content != "hello" {
		t.Errorf("text content: got %q, want %q", txt.Content, "hello")
	}
}

// TestResolve_PassesThroughTextNode verifies that Resolve returns a *TextNode
// as-is without wrapping or modification.
func TestResolve_PassesThroughTextNode(t *testing.T) {
	tn := Text("raw text")
	result := Resolve(tn)

	got, ok := result.(*TextNode)
	if !ok {
		t.Fatalf("expected *TextNode, got %T", result)
	}
	if got != tn {
		t.Error("Resolve must return the same *TextNode pointer for leaf nodes")
	}
	if got.Content != "raw text" {
		t.Errorf("content: got %q, want %q", got.Content, "raw text")
	}
}

// TestResolve_PassesThroughFragment verifies that Resolve resolves a Fragment's
// children and returns a new Fragment with those resolved children.
func TestResolve_PassesThroughFragment(t *testing.T) {
	frag := Frag(Text("a"), Text("b"))
	result := Resolve(frag)

	f, ok := result.(*Fragment)
	if !ok {
		t.Fatalf("expected *Fragment, got %T", result)
	}
	if len(f.Children) != 2 {
		t.Fatalf("children: got %d, want 2", len(f.Children))
	}
	for i, want := range []string{"a", "b"} {
		txt, ok := f.Children[i].(*TextNode)
		if !ok {
			t.Fatalf("children[%d]: expected *TextNode, got %T", i, f.Children[i])
		}
		if txt.Content != want {
			t.Errorf("children[%d] content: got %q, want %q", i, txt.Content, want)
		}
	}
}

// TestResolve_HandlesNil verifies that Resolve returns nil when passed nil,
// without panicking.
func TestResolve_HandlesNil(t *testing.T) {
	result := Resolve(nil)
	if result != nil {
		t.Errorf("Resolve(nil): expected nil, got %v", result)
	}
}

// TestResolve_ComponentWithChildrenPassedThrough verifies that a FuncComponent
// receives and correctly forwards children from Comp.
func TestResolve_ComponentWithChildrenPassedThrough(t *testing.T) {
	child := Text("inner")
	cn := Comp(passthroughFunc, nil, child)
	result := Resolve(cn)

	el, ok := result.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", result)
	}
	if len(el.Children) != 1 {
		t.Fatalf("children: got %d, want 1", len(el.Children))
	}
	txt, ok := el.Children[0].(*TextNode)
	if !ok {
		t.Fatalf("child: expected *TextNode, got %T", el.Children[0])
	}
	if txt.Content != "inner" {
		t.Errorf("content: got %q, want %q", txt.Content, "inner")
	}
}

// TestNonZeroInitialState verifies that state pre-set via SetState before
// Mount is available inside Render after Resolve.
func TestNonZeroInitialState(t *testing.T) {
	c := &testCounter{}
	c.SetState(counterState{Count: 99})
	cn := Mount(c, nil)
	result := Resolve(cn)

	el, ok := result.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", result)
	}
	txt, ok := el.Children[0].(*TextNode)
	if !ok {
		t.Fatalf("child: expected *TextNode, got %T", el.Children[0])
	}
	if txt.Content != "Count: 99" {
		t.Errorf("content: got %q, want %q", txt.Content, "Count: 99")
	}
}

// TestStateIsTyped is a compile-time check that State() returns the exact type
// parameter with no type assertions required. If this file compiles, the test
// passes.
func TestStateIsTyped(t *testing.T) {
	c := &testCounter{}
	c.SetState(counterState{Count: 3})

	// No type assertion — State() is counterState directly.
	var s counterState = c.State()
	if s.Count != 3 {
		t.Errorf("Count: got %d, want 3", s.Count)
	}
}

// TestComponentNodeSatisfiesNode verifies that *ComponentNode implements Node
// by assigning it to a Node variable at runtime.
func TestComponentNodeSatisfiesNode(t *testing.T) {
	cn := Comp(greetFunc, nil)
	var n Node = cn
	if n == nil {
		t.Error("*ComponentNode assigned to Node must not be nil")
	}
}

// TestResolve_RecursivelyResolvesElementChildren verifies that children that
// are ComponentNodes inside a plain Element are also resolved.
func TestResolve_RecursivelyResolvesElementChildren(t *testing.T) {
	inner := &testCounter{}
	inner.SetState(counterState{Count: 2})

	// A plain element wrapping a ComponentNode child.
	tree := El("div", nil, Mount(inner, nil))
	result := Resolve(tree)

	el, ok := result.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", result)
	}
	if el.Tag != "div" {
		t.Errorf("tag: got %q, want %q", el.Tag, "div")
	}
	if len(el.Children) != 1 {
		t.Fatalf("children: got %d, want 1", len(el.Children))
	}
	// After resolution the child must be a concrete *Element, not a *ComponentNode.
	child, ok := el.Children[0].(*Element)
	if !ok {
		t.Fatalf("resolved child: expected *Element, got %T", el.Children[0])
	}
	if child.Tag != "div" {
		t.Errorf("child tag: got %q, want %q", child.Tag, "div")
	}
	txt, ok := child.Children[0].(*TextNode)
	if !ok {
		t.Fatalf("grandchild: expected *TextNode, got %T", child.Children[0])
	}
	if txt.Content != "Count: 2" {
		t.Errorf("grandchild content: got %q, want %q", txt.Content, "Count: 2")
	}
}

// TestResolve_FragmentWithComponentChildren verifies that ComponentNode
// children inside a Fragment are recursively resolved.
func TestResolve_FragmentWithComponentChildren(t *testing.T) {
	c1 := &testCounter{}
	c1.SetState(counterState{Count: 1})
	c2 := &testCounter{}
	c2.SetState(counterState{Count: 2})

	frag := Frag(Mount(c1, nil), Mount(c2, nil))
	result := Resolve(frag)

	f, ok := result.(*Fragment)
	if !ok {
		t.Fatalf("expected *Fragment, got %T", result)
	}
	if len(f.Children) != 2 {
		t.Fatalf("children: got %d, want 2", len(f.Children))
	}
	for i, want := range []string{"Count: 1", "Count: 2"} {
		el, ok := f.Children[i].(*Element)
		if !ok {
			t.Fatalf("children[%d]: expected *Element, got %T", i, f.Children[i])
		}
		txt, ok := el.Children[0].(*TextNode)
		if !ok {
			t.Fatalf("children[%d] text: expected *TextNode, got %T", i, el.Children[0])
		}
		if txt.Content != want {
			t.Errorf("children[%d] content: got %q, want %q", i, txt.Content, want)
		}
	}
}

// --- ResolveTracked tests ---

// TestResolveTracked_CollectsStatefulComponents verifies that the callback
// receives a stateful component exactly once.
func TestResolveTracked_CollectsStatefulComponents(t *testing.T) {
	c := &testCounter{}
	c.SetState(counterState{Count: 1})
	cn := Mount(c, nil)

	var collected []Renderable
	result := ResolveTracked(cn, func(comp Renderable) {
		collected = append(collected, comp)
	})

	if len(collected) != 1 {
		t.Fatalf("collected count: got %d, want 1", len(collected))
	}
	if collected[0] != c {
		t.Error("collected component must be the same pointer as the mounted component")
	}
	// Result should still be a valid resolved tree.
	el, ok := result.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", result)
	}
	if el.Tag != "div" {
		t.Errorf("tag: got %q, want %q", el.Tag, "div")
	}
}

// TestResolveTracked_IgnoresFuncComponents verifies that the callback is
// not called for FuncComponent nodes — only stateful components are tracked.
func TestResolveTracked_IgnoresFuncComponents(t *testing.T) {
	cn := Comp(greetFunc, Props{"name": "Alice"})

	var collected []Renderable
	ResolveTracked(cn, func(comp Renderable) {
		collected = append(collected, comp)
	})

	if len(collected) != 0 {
		t.Errorf("collected count: got %d, want 0 (FuncComponents should not be collected)", len(collected))
	}
}

// TestResolveTracked_CollectsNestedComponents verifies that stateful
// components nested inside Elements and Fragments are collected.
func TestResolveTracked_CollectsNestedComponents(t *testing.T) {
	c := &testCounter{}
	c.SetState(counterState{Count: 3})

	tree := El("div", nil, Frag(Mount(c, nil)))

	var collected []Renderable
	ResolveTracked(tree, func(comp Renderable) {
		collected = append(collected, comp)
	})

	if len(collected) != 1 {
		t.Fatalf("collected count: got %d, want 1", len(collected))
	}
	if collected[0] != c {
		t.Error("collected component must be the nested counter")
	}
}

// TestResolveTracked_MultipleComponents verifies that multiple stateful
// components in the same tree are all collected.
func TestResolveTracked_MultipleComponents(t *testing.T) {
	c1 := &testCounter{}
	c1.SetState(counterState{Count: 1})
	c2 := &testCounter{}
	c2.SetState(counterState{Count: 2})

	tree := El("div", nil, Mount(c1, nil), Mount(c2, nil))

	var collected []Renderable
	ResolveTracked(tree, func(comp Renderable) {
		collected = append(collected, comp)
	})

	if len(collected) != 2 {
		t.Fatalf("collected count: got %d, want 2", len(collected))
	}
	if collected[0] != c1 || collected[1] != c2 {
		t.Error("collected components must match c1 and c2 in order")
	}
}

// TestResolveTracked_ComponentInsideComponentRender verifies that when a
// component's Render returns Mount(inner, nil), both components are collected.
func TestResolveTracked_ComponentInsideComponentRender(t *testing.T) {
	outer := &wrapperComponent{}
	cn := Mount(outer, nil)

	var collected []Renderable
	ResolveTracked(cn, func(comp Renderable) {
		collected = append(collected, comp)
	})

	// wrapperComponent.Render() creates a new testCounter and returns
	// Mount(inner, nil). Both outer and inner should be collected.
	if len(collected) != 2 {
		t.Fatalf("collected count: got %d, want 2", len(collected))
	}
	// Inner is collected first (resolved during outer's Render), then outer.
	if collected[1] != outer {
		t.Error("second collected component must be the outer wrapper")
	}
}

// TestResolveTracked_NilCallback verifies that passing nil doesn't panic
// and produces the same result as Resolve.
func TestResolveTracked_NilCallback(t *testing.T) {
	c := &testCounter{}
	c.SetState(counterState{Count: 42})
	cn := Mount(c, nil)

	result := ResolveTracked(cn, nil)

	el, ok := result.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", result)
	}
	txt, ok := el.Children[0].(*TextNode)
	if !ok {
		t.Fatalf("child: expected *TextNode, got %T", el.Children[0])
	}
	if txt.Content != "Count: 42" {
		t.Errorf("content: got %q, want %q", txt.Content, "Count: 42")
	}
}

// TestMultipleUpdateState_Accumulates verifies that UpdateState called
// multiple times accumulates correctly.
func TestMultipleUpdateState_Accumulates(t *testing.T) {
	c := &testCounter{}
	for range 5 {
		c.UpdateState(func(s counterState) counterState {
			s.Count++
			return s
		})
	}
	if c.State().Count != 5 {
		t.Errorf("Count after 5 increments: got %d, want 5", c.State().Count)
	}
}
