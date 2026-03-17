package gui

import "reflect"

// PatchOp represents the type of change in a patch.
type PatchOp int

const (
	// OpReplace replaces an entire node.
	OpReplace PatchOp = iota
	// OpUpdateProps updates changed props on an element.
	OpUpdateProps
	// OpUpdateText updates the text content of a TextNode.
	OpUpdateText
	// OpInsertChild inserts a new child at the given index.
	OpInsertChild
	// OpRemoveChild removes a child at the given index.
	OpRemoveChild
)

// Patch represents a single change between two node trees.
type Patch struct {
	Op      PatchOp
	Path    []int  // index path from root to the target node
	Old     Node   // the old node (for Replace, Remove)
	New     Node   // the new node (for Replace, Insert)
	Props   Props  // changed props (for UpdateProps); nil value = removal
	OldText string // old text (for UpdateText)
	NewText string // new text (for UpdateText)
	Index   int    // child index (for InsertChild, RemoveChild)
}

// Diff compares two resolved node trees and returns a list of patches.
// Both trees should already be resolved (no ComponentNodes).
func Diff(old, new Node) []Patch {
	var patches []Patch
	// Use a stack-based path buffer to avoid allocations during traversal.
	// Only snapshot (copy) the path when creating an actual patch.
	pathBuf := make([]int, 0, 16)
	diffNodes(old, new, pathBuf, &patches)
	return patches
}

func diffNodes(old, new Node, path []int, patches *[]Patch) {
	if old == nil && new == nil {
		return
	}
	if old == nil || new == nil {
		*patches = append(*patches, Patch{
			Op: OpReplace, Path: copyPath(path), Old: old, New: new,
		})
		return
	}

	switch o := old.(type) {
	case *TextNode:
		n, ok := new.(*TextNode)
		if !ok {
			*patches = append(*patches, Patch{Op: OpReplace, Path: copyPath(path), Old: old, New: new})
			return
		}
		if o.Content != n.Content {
			*patches = append(*patches, Patch{
				Op: OpUpdateText, Path: copyPath(path),
				OldText: o.Content, NewText: n.Content,
			})
		}

	case *Element:
		n, ok := new.(*Element)
		if !ok || o.Tag != n.Tag {
			*patches = append(*patches, Patch{Op: OpReplace, Path: copyPath(path), Old: old, New: new})
			return
		}
		if changed := diffProps(o.Props, n.Props); changed != nil {
			*patches = append(*patches, Patch{
				Op: OpUpdateProps, Path: copyPath(path), Props: changed,
			})
		}
		diffChildren(o.Children, n.Children, path, patches)

	case *Fragment:
		n, ok := new.(*Fragment)
		if !ok {
			*patches = append(*patches, Patch{Op: OpReplace, Path: copyPath(path), Old: old, New: new})
			return
		}
		diffChildren(o.Children, n.Children, path, patches)

	default:
		*patches = append(*patches, Patch{Op: OpReplace, Path: copyPath(path), Old: old, New: new})
	}
}

func diffChildren(oldKids, newKids []Node, parentPath []int, patches *[]Patch) {
	minLen := len(oldKids)
	if len(newKids) < minLen {
		minLen = len(newKids)
	}
	for i := 0; i < minLen; i++ {
		// Extend the path buffer in-place; diffNodes will extend further or
		// snapshot via copyPath only when emitting a patch. Slice back after
		// the recursive call so the buffer is reusable.
		childPath := append(parentPath, i) //nolint:gocritic // intentional reuse
		diffNodes(oldKids[i], newKids[i], childPath, patches)
	}
	// Extra new children → InsertChild
	for i := minLen; i < len(newKids); i++ {
		*patches = append(*patches, Patch{
			Op: OpInsertChild, Path: copyPath(parentPath),
			New: newKids[i], Index: i,
		})
	}
	// Extra old children → RemoveChild (reverse order for safe removal)
	for i := len(oldKids) - 1; i >= minLen; i-- {
		*patches = append(*patches, Patch{
			Op: OpRemoveChild, Path: copyPath(parentPath),
			Old: oldKids[i], Index: i,
		})
	}
}

// diffProps returns changed props between old and new, or nil if identical.
// Returning nil (not empty map) when there are no changes avoids a map
// allocation on the hot path where most elements are unchanged.
func diffProps(old, new Props) Props {
	// Fast path: check if anything changed before allocating.
	anyChange := false
	for k, nv := range new {
		ov, exists := old[k]
		if !exists || !propsEqual(ov, nv) {
			anyChange = true
			break
		}
	}
	if !anyChange {
		// Check for removals.
		for k := range old {
			if _, exists := new[k]; !exists {
				anyChange = true
				break
			}
		}
	}
	if !anyChange {
		return nil
	}

	changed := Props{}
	for k, nv := range new {
		ov, exists := old[k]
		if !exists || !propsEqual(ov, nv) {
			changed[k] = nv
		}
	}
	for k := range old {
		if _, exists := new[k]; !exists {
			changed[k] = nil // nil signals removal
		}
	}
	return changed
}

// isHandlerFunc reports whether v is a function type used for event handlers.
// Both func() (simple callbacks) and func(Event) (rich event handlers) are
// supported.
func isHandlerFunc(v any) bool {
	switch v.(type) {
	case func():
		return true
	case func(Event):
		return true
	}
	return false
}

func propsEqual(a, b any) bool {
	// Functions are never considered equal (always re-apply).
	if isHandlerFunc(a) || isHandlerFunc(b) {
		return false
	}
	// Fast path for common types — avoids reflect.DeepEqual overhead.
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case int:
		bv, ok := b.(int)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	}
	return reflect.DeepEqual(a, b)
}

func copyPath(p []int) []int {
	cp := make([]int, len(p))
	copy(cp, p)
	return cp
}
