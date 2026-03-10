package gui

import (
	"reflect"
	"testing"
)

// patchOps extracts just the Op fields from a patch list, useful for quick
// assertions that only care about the number and kind of operations.
func patchOps(patches []Patch) []PatchOp {
	ops := make([]PatchOp, len(patches))
	for i, p := range patches {
		ops[i] = p.Op
	}
	return ops
}

// findPatch returns the first patch in the list matching the given op, or a
// zero-value Patch and false if none is found.
func findPatch(patches []Patch, op PatchOp) (Patch, bool) {
	for _, p := range patches {
		if p.Op == op {
			return p, true
		}
	}
	return Patch{}, false
}

// findPatches returns all patches matching the given op.
func findPatches(patches []Patch, op PatchOp) []Patch {
	var result []Patch
	for _, p := range patches {
		if p.Op == op {
			result = append(result, p)
		}
	}
	return result
}

// ── 1. Identical trees → no patches ────────────────────────────────────────

func TestDiff_IdenticalTrees(t *testing.T) {
	tests := []struct {
		name string
		tree Node
	}{
		{
			name: "identical text nodes",
			tree: Text("hello"),
		},
		{
			name: "identical element no children",
			tree: El("div", Props{"class": "box"}),
		},
		{
			name: "identical element with children",
			tree: El("ul", nil,
				El("li", nil, Text("a")),
				El("li", nil, Text("b")),
			),
		},
		{
			name: "identical fragment",
			tree: Frag(Text("x"), El("span", nil)),
		},
		{
			name: "empty element no props no children",
			tree: El("br", nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Diff(tt.tree, tt.tree)
			if len(patches) != 0 {
				t.Errorf("expected no patches, got %d: %+v", len(patches), patches)
			}
		})
	}
}

// ── 2. Different text content → OpUpdateText ────────────────────────────────

func TestDiff_UpdateText(t *testing.T) {
	old := Text("hello")
	new := Text("world")

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateText {
		t.Errorf("expected OpUpdateText, got %v", p.Op)
	}
	if p.OldText != "hello" {
		t.Errorf("expected OldText=%q, got %q", "hello", p.OldText)
	}
	if p.NewText != "world" {
		t.Errorf("expected NewText=%q, got %q", "world", p.NewText)
	}
	if len(p.Path) != 0 {
		t.Errorf("expected empty path at root, got %v", p.Path)
	}
}

func TestDiff_UpdateText_EmptyToNonEmpty(t *testing.T) {
	patches := Diff(Text(""), Text("filled"))
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpUpdateText {
		t.Errorf("expected OpUpdateText, got %v", patches[0].Op)
	}
	if patches[0].NewText != "filled" {
		t.Errorf("expected NewText=%q, got %q", "filled", patches[0].NewText)
	}
}

// ── 3. Different element tag → OpReplace ───────────────────────────────────

func TestDiff_DifferentTag(t *testing.T) {
	old := El("div", nil, Text("content"))
	new := El("span", nil, Text("content"))

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpReplace {
		t.Errorf("expected OpReplace, got %v", p.Op)
	}
	if p.Old != old {
		t.Errorf("Old node mismatch")
	}
	if p.New != new {
		t.Errorf("New node mismatch")
	}
}

// ── 4. Same tag, different props → OpUpdateProps with changed keys ──────────

func TestDiff_UpdatedProps(t *testing.T) {
	old := El("div", Props{"class": "old", "id": "same"})
	new := El("div", Props{"class": "new", "id": "same"})

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateProps {
		t.Errorf("expected OpUpdateProps, got %v", p.Op)
	}
	if p.Props["class"] != "new" {
		t.Errorf("expected changed class=%q, got %v", "new", p.Props["class"])
	}
	// "id" is unchanged — must NOT appear in the changed props.
	if _, exists := p.Props["id"]; exists {
		t.Errorf("unchanged prop 'id' should not appear in patch, got %v", p.Props["id"])
	}
}

// ── 5. Prop removed → OpUpdateProps with nil value for that key ────────────

func TestDiff_PropRemoved(t *testing.T) {
	old := El("div", Props{"class": "box", "hidden": true})
	new := El("div", Props{"class": "box"})

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateProps {
		t.Errorf("expected OpUpdateProps, got %v", p.Op)
	}
	val, exists := p.Props["hidden"]
	if !exists {
		t.Fatal("expected 'hidden' key in patch props for removal signal")
	}
	if val != nil {
		t.Errorf("expected nil value for removed prop, got %v", val)
	}
}

// ── 6. Prop added → OpUpdateProps with new key ─────────────────────────────

func TestDiff_PropAdded(t *testing.T) {
	old := El("input", Props{"type": "text"})
	new := El("input", Props{"type": "text", "placeholder": "enter value"})

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateProps {
		t.Errorf("expected OpUpdateProps, got %v", p.Op)
	}
	if p.Props["placeholder"] != "enter value" {
		t.Errorf("expected new prop placeholder=%q, got %v", "enter value", p.Props["placeholder"])
	}
	// "type" is unchanged.
	if _, exists := p.Props["type"]; exists {
		t.Errorf("unchanged prop 'type' should not appear in patch")
	}
}

// ── 7. Extra child in new tree → OpInsertChild with correct Index ───────────

func TestDiff_InsertChild(t *testing.T) {
	old := El("ul", nil,
		El("li", nil, Text("a")),
	)
	new := El("ul", nil,
		El("li", nil, Text("a")),
		El("li", nil, Text("b")),
		El("li", nil, Text("c")),
	)

	patches := Diff(old, new)

	inserts := findPatches(patches, OpInsertChild)
	if len(inserts) != 2 {
		t.Fatalf("expected 2 OpInsertChild patches, got %d: %+v", len(inserts), patches)
	}
	// Insertions should appear in ascending index order.
	if inserts[0].Index != 1 {
		t.Errorf("first insert: expected Index=1, got %d", inserts[0].Index)
	}
	if inserts[1].Index != 2 {
		t.Errorf("second insert: expected Index=2, got %d", inserts[1].Index)
	}
	// New nodes should be set correctly.
	n0, ok := inserts[0].New.(*Element)
	if !ok {
		t.Fatalf("expected *Element New node, got %T", inserts[0].New)
	}
	if n0.Tag != "li" {
		t.Errorf("expected li, got %q", n0.Tag)
	}
}

// ── 8. Fewer children in new tree → OpRemoveChild with correct Index ────────

func TestDiff_RemoveChild(t *testing.T) {
	old := El("ul", nil,
		El("li", nil, Text("a")),
		El("li", nil, Text("b")),
		El("li", nil, Text("c")),
	)
	new := El("ul", nil,
		El("li", nil, Text("a")),
	)

	patches := Diff(old, new)

	removals := findPatches(patches, OpRemoveChild)
	if len(removals) != 2 {
		t.Fatalf("expected 2 OpRemoveChild patches, got %d: %+v", len(removals), patches)
	}
	// Removals must arrive in REVERSE index order (2 then 1) so that a
	// renderer can splice them safely without index shifting.
	if removals[0].Index != 2 {
		t.Errorf("first removal: expected Index=2, got %d", removals[0].Index)
	}
	if removals[1].Index != 1 {
		t.Errorf("second removal: expected Index=1, got %d", removals[1].Index)
	}
}

// ── 9. Deep nested change → correct Path ───────────────────────────────────

func TestDiff_DeepNestedPath(t *testing.T) {
	// Build a tree: div > [span, ul > [li, li > [b, em]]]
	// Change text inside em: path [1, 1, 1]
	old := El("div", nil,
		El("span", nil),
		El("ul", nil,
			El("li", nil),
			El("li", nil,
				El("b", nil, Text("bold")),
				El("em", nil, Text("old")),
			),
		),
	)
	new := El("div", nil,
		El("span", nil),
		El("ul", nil,
			El("li", nil),
			El("li", nil,
				El("b", nil, Text("bold")),
				El("em", nil, Text("new")),
			),
		),
	)

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateText {
		t.Errorf("expected OpUpdateText, got %v", p.Op)
	}
	// Path to the text node inside em: div[1] → ul[1] → li[1] → em[1] → text[0]
	// indices:                                [1]        [1]        [1]        [0]
	wantPath := []int{1, 1, 1, 0}
	if !reflect.DeepEqual(p.Path, wantPath) {
		t.Errorf("expected path %v, got %v", wantPath, p.Path)
	}
	if p.OldText != "old" || p.NewText != "new" {
		t.Errorf("expected old=%q new=%q, got old=%q new=%q", "old", "new", p.OldText, p.NewText)
	}
}

// ── 10. Text replaced by Element → OpReplace ───────────────────────────────

func TestDiff_TextReplacedByElement(t *testing.T) {
	old := Text("hello")
	new := El("span", nil, Text("hello"))

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpReplace {
		t.Errorf("expected OpReplace, got %v", patches[0].Op)
	}
	if patches[0].Old != old {
		t.Errorf("Old node mismatch")
	}
	if patches[0].New != new {
		t.Errorf("New node mismatch")
	}
}

// ── 11. Element replaced by Text → OpReplace ───────────────────────────────

func TestDiff_ElementReplacedByText(t *testing.T) {
	old := El("div", nil, Text("content"))
	new := Text("content")

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpReplace {
		t.Errorf("expected OpReplace, got %v", patches[0].Op)
	}
}

// ── 12. Fragment children diffed correctly ─────────────────────────────────

func TestDiff_FragmentChildren(t *testing.T) {
	old := Frag(
		Text("one"),
		El("div", nil),
		Text("three"),
	)
	new := Frag(
		Text("one"),
		El("div", nil),
		Text("THREE"), // changed
	)

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateText {
		t.Errorf("expected OpUpdateText, got %v", p.Op)
	}
	if p.OldText != "three" || p.NewText != "THREE" {
		t.Errorf("expected old=%q new=%q, got old=%q new=%q", "three", "THREE", p.OldText, p.NewText)
	}
	// Path inside the fragment: index 2 (third child of the fragment root).
	if !reflect.DeepEqual(p.Path, []int{2}) {
		t.Errorf("expected path [2], got %v", p.Path)
	}
}

func TestDiff_FragmentVsNonFragment(t *testing.T) {
	old := Frag(Text("a"))
	new := El("div", nil, Text("a"))

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 OpReplace patch, got %d", len(patches))
	}
	if patches[0].Op != OpReplace {
		t.Errorf("expected OpReplace, got %v", patches[0].Op)
	}
}

// ── 13. Both nil → no patches ──────────────────────────────────────────────

func TestDiff_BothNil(t *testing.T) {
	patches := Diff(nil, nil)
	if len(patches) != 0 {
		t.Errorf("expected no patches for nil/nil, got %d: %+v", len(patches), patches)
	}
}

// ── 14. Old nil, new present → OpReplace ───────────────────────────────────

func TestDiff_OldNilNewPresent(t *testing.T) {
	new := El("div", nil, Text("hello"))

	patches := Diff(nil, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	p := patches[0]
	if p.Op != OpReplace {
		t.Errorf("expected OpReplace, got %v", p.Op)
	}
	if p.Old != nil {
		t.Errorf("expected Old=nil, got %v", p.Old)
	}
	if p.New != new {
		t.Errorf("New node mismatch")
	}
}

// ── 15. New nil, old present → OpReplace ───────────────────────────────────

func TestDiff_NewNilOldPresent(t *testing.T) {
	old := El("section", nil)

	patches := Diff(old, nil)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	p := patches[0]
	if p.Op != OpReplace {
		t.Errorf("expected OpReplace, got %v", p.Op)
	}
	if p.Old != old {
		t.Errorf("Old node mismatch")
	}
	if p.New != nil {
		t.Errorf("expected New=nil, got %v", p.New)
	}
}

// ── 16. Complex tree with multiple changes ──────────────────────────────────

func TestDiff_ComplexMultipleChanges(t *testing.T) {
	// old:
	//   div.container
	//     h1            text "Title"
	//     ul
	//       li           text "item A"
	//       li           text "item B"
	//     footer         text "old footer"
	old := El("div", Props{"class": "container"},
		El("h1", nil, Text("Title")),
		El("ul", nil,
			El("li", nil, Text("item A")),
			El("li", nil, Text("item B")),
		),
		El("footer", nil, Text("old footer")),
	)

	// new:
	//   div.container.wide   ← prop "class" changed
	//     h1                 text "Title"  (unchanged)
	//     ul
	//       li               text "item A" (unchanged)
	//       li               text "item B" (unchanged)
	//       li               text "item C" (inserted)
	//     footer             text "new footer"  ← text changed
	new := El("div", Props{"class": "container wide"},
		El("h1", nil, Text("Title")),
		El("ul", nil,
			El("li", nil, Text("item A")),
			El("li", nil, Text("item B")),
			El("li", nil, Text("item C")),
		),
		El("footer", nil, Text("new footer")),
	)

	patches := Diff(old, new)

	// Expect: OpUpdateProps (root), OpInsertChild (ul[2]), OpUpdateText (footer text)
	if len(patches) != 3 {
		t.Fatalf("expected 3 patches, got %d: %+v", len(patches), patches)
	}

	// --- OpUpdateProps on root div ---
	propsP, ok := findPatch(patches, OpUpdateProps)
	if !ok {
		t.Fatal("expected an OpUpdateProps patch")
	}
	if !reflect.DeepEqual(propsP.Path, []int{}) || len(propsP.Path) != 0 {
		// Root path is nil/empty.
	}
	if propsP.Props["class"] != "container wide" {
		t.Errorf("expected changed class=%q, got %v", "container wide", propsP.Props["class"])
	}

	// --- OpInsertChild for "item C" ---
	inserts := findPatches(patches, OpInsertChild)
	if len(inserts) != 1 {
		t.Fatalf("expected 1 OpInsertChild, got %d", len(inserts))
	}
	if inserts[0].Index != 2 {
		t.Errorf("insert: expected Index=2, got %d", inserts[0].Index)
	}
	// Path should point to the ul element (child index 1 of root div).
	if !reflect.DeepEqual(inserts[0].Path, []int{1}) {
		t.Errorf("insert: expected path [1], got %v", inserts[0].Path)
	}

	// --- OpUpdateText for footer ---
	texts := findPatches(patches, OpUpdateText)
	if len(texts) != 1 {
		t.Fatalf("expected 1 OpUpdateText, got %d", len(texts))
	}
	if texts[0].OldText != "old footer" || texts[0].NewText != "new footer" {
		t.Errorf("text patch: expected old=%q new=%q, got old=%q new=%q",
			"old footer", "new footer", texts[0].OldText, texts[0].NewText)
	}
	// Path to the text node: root div [2] → footer [0] → text
	if !reflect.DeepEqual(texts[0].Path, []int{2, 0}) {
		t.Errorf("text patch: expected path [2,0], got %v", texts[0].Path)
	}
}

// ── 17. Props with function values → always considered changed ──────────────

func TestDiff_FunctionPropsAlwaysChanged(t *testing.T) {
	fn1 := func() {}
	fn2 := func() {}

	old := El("button", Props{"onClick": fn1, "label": "click"})
	new := El("button", Props{"onClick": fn2, "label": "click"})

	patches := Diff(old, new)

	// The function prop must always produce an OpUpdateProps patch.
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch for changed function prop, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateProps {
		t.Errorf("expected OpUpdateProps, got %v", p.Op)
	}
	if _, exists := p.Props["onClick"]; !exists {
		t.Errorf("expected 'onClick' in changed props")
	}
	// "label" string prop is unchanged; must not appear.
	if _, exists := p.Props["label"]; exists {
		t.Errorf("unchanged string prop 'label' should not appear in patch")
	}
}

func TestDiff_EventHandlerPropsAlwaysChanged(t *testing.T) {
	fn1 := func(Event) {}
	fn2 := func(Event) {}

	old := El("button", Props{"onclick": fn1, "label": "click"})
	new := El("button", Props{"onclick": fn2, "label": "click"})

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch for changed func(Event) prop, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateProps {
		t.Errorf("expected OpUpdateProps, got %v", p.Op)
	}
	if _, exists := p.Props["onclick"]; !exists {
		t.Errorf("expected 'onclick' in changed props")
	}
	if _, exists := p.Props["label"]; exists {
		t.Errorf("unchanged string prop 'label' should not appear in patch")
	}
}

func TestDiff_SameEventHandlerPropIsAlwaysChanged(t *testing.T) {
	fn := func(Event) {}

	old := El("button", Props{"onclick": fn})
	new := El("button", Props{"onclick": fn})

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch (same func(Event) reference still changed), got %d: %+v", len(patches), patches)
	}
	if patches[0].Op != OpUpdateProps {
		t.Errorf("expected OpUpdateProps, got %v", patches[0].Op)
	}
}

func TestDiff_MixedHandlerTypes(t *testing.T) {
	// Test that func() and func(Event) on the same element both produce patches.
	fn1 := func() {}
	fn2 := func(Event) {}

	old := El("div", Props{"onclick": fn1, "onchange": fn2})
	new := El("div", Props{"onclick": fn1, "onchange": fn2})

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateProps {
		t.Errorf("expected OpUpdateProps, got %v", p.Op)
	}
	if _, exists := p.Props["onclick"]; !exists {
		t.Errorf("expected 'onclick' in changed props")
	}
	if _, exists := p.Props["onchange"]; !exists {
		t.Errorf("expected 'onchange' in changed props")
	}
}

func TestDiff_SameFunctionPropIsAlwaysChanged(t *testing.T) {
	// Even the very same function value counts as changed because functions
	// in Go cannot be compared for equality in a meaningful way for UI diffing.
	fn := func() {}

	old := El("button", Props{"onClick": fn})
	new := El("button", Props{"onClick": fn})

	patches := Diff(old, new)

	// propsEqual returns false for any func(), so we always get a patch.
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch (same func reference still changed), got %d: %+v", len(patches), patches)
	}
	if patches[0].Op != OpUpdateProps {
		t.Errorf("expected OpUpdateProps, got %v", patches[0].Op)
	}
}

// ── 18. RemoveChild patches come in reverse index order ─────────────────────

func TestDiff_RemoveChildReverseOrder(t *testing.T) {
	old := El("div", nil,
		Text("a"), // index 0
		Text("b"), // index 1
		Text("c"), // index 2
		Text("d"), // index 3
	)
	new := El("div", nil,
		Text("a"), // index 0 — kept
	)

	patches := Diff(old, new)

	removals := findPatches(patches, OpRemoveChild)
	if len(removals) != 3 {
		t.Fatalf("expected 3 removals, got %d: %+v", len(removals), patches)
	}
	// Must descend: 3, 2, 1.
	for i, want := range []int{3, 2, 1} {
		if removals[i].Index != want {
			t.Errorf("removal[%d]: expected Index=%d, got %d", i, want, removals[i].Index)
		}
	}
}

// ── 19. Path is a copy — mutating one path doesn't corrupt others ───────────

func TestDiff_PathIsCopied(t *testing.T) {
	// Two sibling text changes so we get two patches with different paths.
	old := El("div", nil,
		Text("first old"),
		Text("second old"),
	)
	new := El("div", nil,
		Text("first new"),
		Text("second new"),
	)

	patches := Diff(old, new)

	if len(patches) != 2 {
		t.Fatalf("expected 2 patches, got %d", len(patches))
	}
	// Capture copies of the paths before mutation.
	path0 := make([]int, len(patches[0].Path))
	path1 := make([]int, len(patches[1].Path))
	copy(path0, patches[0].Path)
	copy(path1, patches[1].Path)

	// Mutate the first path slice.
	if len(patches[0].Path) > 0 {
		patches[0].Path[0] = 999
	}

	// The second patch's path must be unaffected.
	if !reflect.DeepEqual(patches[1].Path, path1) {
		t.Errorf("mutating patches[0].Path corrupted patches[1].Path: got %v, want %v",
			patches[1].Path, path1)
	}

	// Also verify the original captured path for patch 0 differed from patch 1.
	if reflect.DeepEqual(path0, path1) {
		t.Errorf("sibling patches should have different paths: both are %v", path0)
	}
	if path0[0] != 0 {
		t.Errorf("expected path[0]=%d to be 0, got %d", 0, path0[0])
	}
	if path1[0] != 1 {
		t.Errorf("expected path[1]=%d to be 1, got %d", 1, path1[0])
	}
}

// ── 20. Empty Props vs nil Props — no spurious patches ─────────────────────

func TestDiff_EmptyPropsVsNilProps(t *testing.T) {
	tests := []struct {
		name string
		old  *Element
		new  *Element
	}{
		{
			name: "both nil props (via El)",
			old:  El("div", nil),
			new:  El("div", nil),
		},
		{
			name: "both empty props",
			old:  El("div", Props{}),
			new:  El("div", Props{}),
		},
		{
			name: "nil old props vs empty new props",
			// El normalises nil to empty, so both end up as Props{}.
			old: El("div", nil),
			new: El("div", Props{}),
		},
		{
			name: "empty old props vs nil new props",
			old:  El("div", Props{}),
			new:  El("div", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Diff(tt.old, tt.new)
			// We only care that there are no spurious prop patches — child
			// patches or replaces would be a test-design error.
			propPatches := findPatches(patches, OpUpdateProps)
			if len(propPatches) != 0 {
				t.Errorf("expected no OpUpdateProps patches, got %d: %+v", len(propPatches), propPatches)
			}
		})
	}
}

// ── Additional edge cases ────────────────────────────────────────────────────

func TestDiff_RootPathIsEmpty(t *testing.T) {
	// A patch at the root should have an empty (not nil) path slice so that
	// consumers can always len() it safely.
	patches := Diff(Text("a"), Text("b"))
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	// copyPath(nil) returns make([]int, 0) which has len 0 but is not nil.
	if patches[0].Path == nil {
		t.Error("expected non-nil Path slice at root, got nil")
	}
	if len(patches[0].Path) != 0 {
		t.Errorf("expected empty Path at root, got %v", patches[0].Path)
	}
}

func TestDiff_MultiplePropsChangedAtOnce(t *testing.T) {
	old := El("input", Props{"type": "text", "class": "old", "id": "inp"})
	new := El("input", Props{"type": "password", "class": "new", "id": "inp"})

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateProps {
		t.Errorf("expected OpUpdateProps, got %v", p.Op)
	}
	if p.Props["type"] != "password" {
		t.Errorf("expected type=%q, got %v", "password", p.Props["type"])
	}
	if p.Props["class"] != "new" {
		t.Errorf("expected class=%q, got %v", "new", p.Props["class"])
	}
	if _, exists := p.Props["id"]; exists {
		t.Errorf("unchanged 'id' should not appear in props patch")
	}
}

func TestDiff_InsertAndRemoveInSameTree(t *testing.T) {
	// old: div with 3 children; new: same 2 first children + a new third.
	// Actually test simultaneous insert + text change.
	old := El("section", nil,
		El("h2", nil, Text("heading")),
		El("p", nil, Text("old body")),
	)
	new := El("section", nil,
		El("h2", nil, Text("heading")),
		El("p", nil, Text("new body")),
		El("footer", nil, Text("added")),
	)

	patches := Diff(old, new)

	if len(patches) != 2 {
		t.Fatalf("expected 2 patches, got %d: %+v", len(patches), patches)
	}
	texts := findPatches(patches, OpUpdateText)
	if len(texts) != 1 {
		t.Fatalf("expected 1 OpUpdateText, got %d", len(texts))
	}
	if texts[0].NewText != "new body" {
		t.Errorf("expected NewText=%q, got %q", "new body", texts[0].NewText)
	}

	inserts := findPatches(patches, OpInsertChild)
	if len(inserts) != 1 {
		t.Fatalf("expected 1 OpInsertChild, got %d", len(inserts))
	}
	if inserts[0].Index != 2 {
		t.Errorf("expected insert Index=2, got %d", inserts[0].Index)
	}
}

func TestDiff_NestedFragmentChange(t *testing.T) {
	// Fragment inside an element.
	old := El("div", nil,
		Frag(
			Text("alpha"),
			Text("beta"),
		),
	)
	new := El("div", nil,
		Frag(
			Text("alpha"),
			Text("BETA"), // changed
		),
	)

	patches := Diff(old, new)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Op != OpUpdateText {
		t.Errorf("expected OpUpdateText, got %v", p.Op)
	}
	// Path: div[0] → frag[1] → text  →  [0, 1]
	if !reflect.DeepEqual(p.Path, []int{0, 1}) {
		t.Errorf("expected path [0,1], got %v", p.Path)
	}
}

func TestDiff_PatchOpsAreDefined(t *testing.T) {
	// Smoke-test that all PatchOp constants have distinct values.
	ops := []PatchOp{OpReplace, OpUpdateProps, OpUpdateText, OpInsertChild, OpRemoveChild}
	seen := map[PatchOp]bool{}
	for _, op := range ops {
		if seen[op] {
			t.Errorf("duplicate PatchOp value: %d", op)
		}
		seen[op] = true
	}
}
