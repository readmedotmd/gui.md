package gui

import "testing"

// --- Test component types for reconciler ---

// rTracker records all lifecycle hooks in order.
type rTracker struct {
	BaseComponent[Props, counterState]
	Calls []string
}

func (r *rTracker) WillMount()  { r.Calls = append(r.Calls, "WillMount") }
func (r *rTracker) DidMount()   { r.Calls = append(r.Calls, "DidMount") }
func (r *rTracker) WillUpdate() { r.Calls = append(r.Calls, "WillUpdate") }
func (r *rTracker) DidUpdate()  { r.Calls = append(r.Calls, "DidUpdate") }
func (r *rTracker) DidUnmount() { r.Calls = append(r.Calls, "DidUnmount") }
func (r *rTracker) Render() Node {
	r.Calls = append(r.Calls, "Render")
	return El("div", nil, Textf("Count: %d", r.State().Count))
}

// rTrackerInit sets state in WillMount to verify it survives across renders.
type rTrackerInit struct {
	BaseComponent[Props, counterState]
}

func (r *rTrackerInit) WillMount() {
	r.SetState(counterState{Count: 42})
}

func (r *rTrackerInit) Render() Node {
	return El("div", nil, Textf("Count: %d", r.State().Count))
}

// rPropsComponent has typed props for testing prop updates.
type rPropsData struct{ Name string }

type rPropsComponent struct {
	BaseComponent[rPropsData, struct{}]
}

func (r *rPropsComponent) Render() Node {
	return El("span", nil, Text(r.Props().Name))
}

// --- Reconciler tests ---

func TestReconciler_FirstRender_Lifecycle(t *testing.T) {
	rec := NewReconciler()
	tree := C(new(rTracker), nil)

	var got []Renderable
	rec.Resolve(tree, func(c Renderable) { got = append(got, c) })

	if len(got) != 1 {
		t.Fatalf("collected %d components, want 1", len(got))
	}
	tracker := got[0].(*rTracker)
	want := []string{"WillMount", "Render", "DidMount"}
	if len(tracker.Calls) != len(want) {
		t.Fatalf("calls: got %v, want %v", tracker.Calls, want)
	}
	for i, c := range tracker.Calls {
		if c != want[i] {
			t.Errorf("call[%d]: got %q, want %q", i, c, want[i])
		}
	}
}

func TestReconciler_Rerender_ReusesInstance(t *testing.T) {
	rec := NewReconciler()
	tree := func() Node { return C(new(rTracker), nil) }

	var first, second Renderable
	rec.Resolve(tree(), func(c Renderable) { first = c })
	rec.Resolve(tree(), func(c Renderable) { second = c })

	if first != second {
		t.Fatal("expected same instance across renders")
	}

	tracker := first.(*rTracker)
	want := []string{
		"WillMount", "Render", "DidMount",
		"WillUpdate", "Render", "DidUpdate",
	}
	if len(tracker.Calls) != len(want) {
		t.Fatalf("calls: got %v, want %v", tracker.Calls, want)
	}
	for i, c := range tracker.Calls {
		if c != want[i] {
			t.Errorf("call[%d]: got %q, want %q", i, c, want[i])
		}
	}
}

func TestReconciler_StatePreserved(t *testing.T) {
	rec := NewReconciler()
	tree := func() Node { return C(new(rTrackerInit), nil) }

	// First render: WillMount sets Count=42.
	result := rec.Resolve(tree(), nil)

	el := result.(*Element)
	txt := el.Children[0].(*TextNode)
	if txt.Content != "Count: 42" {
		t.Errorf("first render content: got %q, want %q", txt.Content, "Count: 42")
	}

	// Second render: state should be preserved (WillMount not called again).
	result = rec.Resolve(tree(), nil)
	el = result.(*Element)
	txt = el.Children[0].(*TextNode)
	if txt.Content != "Count: 42" {
		t.Errorf("second render content: got %q, want %q", txt.Content, "Count: 42")
	}
}

func TestReconciler_Removal_DeletesFromCache(t *testing.T) {
	rec := NewReconciler()
	show := true
	tree := func() Node {
		if show {
			return C(new(rTracker), nil)
		}
		return El("div", nil, Text("empty"))
	}

	var first Renderable
	rec.Resolve(tree(), func(c Renderable) { first = c })

	// Remove the component.
	show = false
	var collected []Renderable
	rec.Resolve(tree(), func(c Renderable) { collected = append(collected, c) })

	if len(collected) != 0 {
		t.Fatalf("expected 0 components after removal, got %d", len(collected))
	}

	// Re-add: should get a fresh instance (not the old one).
	show = true
	var reAdded Renderable
	rec.Resolve(tree(), func(c Renderable) { reAdded = c })

	if reAdded == first {
		t.Error("re-added component must be a new instance, not the old one")
	}

	// Fresh instance should have WillMount lifecycle.
	tracker := reAdded.(*rTracker)
	if len(tracker.Calls) < 1 || tracker.Calls[0] != "WillMount" {
		t.Errorf("re-added calls: got %v, want WillMount first", tracker.Calls)
	}
}

func TestReconciler_MultipleSameType(t *testing.T) {
	rec := NewReconciler()
	tree := func() Node {
		return El("div", nil,
			C(new(rTracker), nil),
			C(new(rTracker), nil),
		)
	}

	var collected []Renderable
	rec.Resolve(tree(), func(c Renderable) { collected = append(collected, c) })

	if len(collected) != 2 {
		t.Fatalf("collected %d, want 2", len(collected))
	}
	if collected[0] == collected[1] {
		t.Error("two C-nodes of the same type must produce different instances")
	}

	// Second render: both instances reused.
	var collected2 []Renderable
	rec.Resolve(tree(), func(c Renderable) { collected2 = append(collected2, c) })

	if collected2[0] != collected[0] {
		t.Error("first instance not reused")
	}
	if collected2[1] != collected[1] {
		t.Error("second instance not reused")
	}
}

func TestReconciler_MixedCAndMount(t *testing.T) {
	rec := NewReconciler()
	manual := &testCounter{}
	manual.SetState(counterState{Count: 99})

	tree := func() Node {
		return El("div", nil,
			C(new(rTracker), nil),
			Mount(manual, nil),
		)
	}

	var collected []Renderable
	result := rec.Resolve(tree(), func(c Renderable) { collected = append(collected, c) })

	if len(collected) != 2 {
		t.Fatalf("collected %d, want 2", len(collected))
	}

	// Verify the Mount component is the manual instance.
	if collected[1] != manual {
		t.Error("second collected must be the manual Mount instance")
	}

	// Verify output: first child from C-node, second from Mount.
	el := result.(*Element)
	if len(el.Children) != 2 {
		t.Fatalf("children: got %d, want 2", len(el.Children))
	}
	child2 := el.Children[1].(*Element)
	txt := child2.Children[0].(*TextNode)
	if txt.Content != "Count: 99" {
		t.Errorf("mount child content: got %q, want %q", txt.Content, "Count: 99")
	}
}

func TestReconciler_PropsUpdated(t *testing.T) {
	rec := NewReconciler()
	name := "Alice"
	tree := func() Node {
		return C(new(rPropsComponent), rPropsData{Name: name})
	}

	result := rec.Resolve(tree(), nil)
	el := result.(*Element)
	txt := el.Children[0].(*TextNode)
	if txt.Content != "Alice" {
		t.Errorf("first render: got %q, want %q", txt.Content, "Alice")
	}

	// Update props.
	name = "Bob"
	result = rec.Resolve(tree(), nil)
	el = result.(*Element)
	txt = el.Children[0].(*TextNode)
	if txt.Content != "Bob" {
		t.Errorf("second render: got %q, want %q", txt.Content, "Bob")
	}
}

func TestReconciler_OnComponentCallback(t *testing.T) {
	rec := NewReconciler()
	manual := &testCounter{}

	tree := func() Node {
		return El("div", nil,
			C(new(rTracker), nil),
			Mount(manual, nil),
		)
	}

	var collected []Renderable
	rec.Resolve(tree(), func(c Renderable) { collected = append(collected, c) })

	if len(collected) != 2 {
		t.Fatalf("collected %d, want 2", len(collected))
	}

	// C-node instance is first (depth-first), manual Mount is second.
	if _, ok := collected[0].(*rTracker); !ok {
		t.Errorf("first: expected *rTracker, got %T", collected[0])
	}
	if collected[1] != manual {
		t.Error("second must be the manual Mount instance")
	}
}

func TestReconciler_NilCallback(t *testing.T) {
	rec := NewReconciler()
	tree := C(new(rTracker), nil)

	// Should not panic with nil callback.
	result := rec.Resolve(tree, nil)
	if result == nil {
		t.Error("expected non-nil result")
	}
}

// --- Standalone Resolve with C-nodes ---

func TestC_StandaloneResolve_CreatesInstance(t *testing.T) {
	tree := C(new(rTracker), nil)
	result := Resolve(tree)

	el, ok := result.(*Element)
	if !ok {
		t.Fatalf("expected *Element, got %T", result)
	}
	txt := el.Children[0].(*TextNode)
	if txt.Content != "Count: 0" {
		t.Errorf("content: got %q, want %q", txt.Content, "Count: 0")
	}
}

func TestC_StandaloneResolve_FreshEachTime(t *testing.T) {
	var instances []Renderable
	tree := func() Node { return C(new(rTrackerInit), nil) }

	ResolveTracked(tree(), func(c Renderable) { instances = append(instances, c) })
	ResolveTracked(tree(), func(c Renderable) { instances = append(instances, c) })

	if len(instances) != 2 {
		t.Fatalf("collected %d, want 2", len(instances))
	}
	if instances[0] == instances[1] {
		t.Error("standalone Resolve must create fresh instances each time")
	}
}

func TestC_NestedInElement(t *testing.T) {
	rec := NewReconciler()
	tree := El("div", nil, C(new(rTracker), nil))

	var collected []Renderable
	result := rec.Resolve(tree, func(c Renderable) { collected = append(collected, c) })

	if len(collected) != 1 {
		t.Fatalf("collected %d, want 1", len(collected))
	}
	el := result.(*Element)
	if el.Tag != "div" {
		t.Errorf("tag: got %q, want %q", el.Tag, "div")
	}
	child := el.Children[0].(*Element)
	txt := child.Children[0].(*TextNode)
	if txt.Content != "Count: 0" {
		t.Errorf("content: got %q, want %q", txt.Content, "Count: 0")
	}
}

func TestC_WithTypedProps(t *testing.T) {
	tree := C(new(rPropsComponent), rPropsData{Name: "typed"})
	result := Resolve(tree)

	el := result.(*Element)
	txt := el.Children[0].(*TextNode)
	if txt.Content != "typed" {
		t.Errorf("content: got %q, want %q", txt.Content, "typed")
	}
}
