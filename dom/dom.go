//go:build js && wasm

// Package dom provides a live DOM renderer for the gui library that runs in
// WebAssembly. It mounts a gui.Node tree into a browser DOM element, wires up
// event listeners for handler props (onclick, oninput, etc.), and supports
// incremental updates via the gui.Diff engine.
package dom

import (
	gui "github.com/readmedotmd/gui.md"
	stdhtml "html"
	"io"
	"strconv"
	"strings"
	"syscall/js"
)

// focusState captures the identity, cursor position, and DOM value of the
// currently focused input/textarea so that focus and content can be restored
// after DOM patches destroy and recreate elements.
type focusState struct {
	active         js.Value
	tagName        string
	id             string
	className      string
	selectionStart int
	selectionEnd   int
	value          string // live DOM value at save time
}

// handlerSlot holds a stable js.Func whose closure dispatches to the latest
// Go handler. This avoids js.Func alloc/Release/DOM-set churn on re-renders —
// only the current pointer is swapped.
type handlerSlot struct {
	jsfn    js.Func // stable — created once per (element, event) pair
	current any     // latest Go handler: func() or func(gui.Event)
}

// maxDOMDepth is the maximum nesting depth for recursive DOM node creation.
// This prevents stack overflow from pathologically deep trees.
const maxDOMDepth = 512

// Renderer renders gui.Node trees into live browser DOM elements.
// It resolves components, builds DOM nodes, and wires event listeners for
// handler props. Use [New] to create a renderer mounted on a container element.
type Renderer struct {
	doc          js.Value
	container    js.Value
	nextElemID   int
	handlers     map[string]*handlerSlot // key: "<elemID>:<event>"
	prevTree     gui.Node
	objectURLs   []string // tracked object URLs for cleanup
}

// New creates a DOM Renderer that mounts into the given container element.
// Typically called with document.getElementById("app").
// Panics if the container is null or undefined — this usually means the
// element does not exist yet (e.g. WASM started before DOMContentLoaded).
func New(container js.Value) *Renderer {
	if container.IsNull() || container.IsUndefined() {
		panic("dom.New: container is null or undefined; ensure the element exists (e.g. wait for DOMContentLoaded)")
	}
	return &Renderer{
		doc:       js.Global().Get("document"),
		container: container,
		handlers:  make(map[string]*handlerSlot),
	}
}

// Render resolves components in node and writes an HTML string representation
// to w. This satisfies gui.Renderer but for DOM usage prefer [Update] which
// creates live DOM nodes with event handlers.
func (r *Renderer) Render(node gui.Node, w io.Writer) error {
	resolved := gui.Resolve(node)
	s := r.renderToString(resolved)
	_, err := io.WriteString(w, s)
	return err
}

// RenderString returns the node tree as an HTML string.
// For live DOM rendering with events, use [Update] instead.
func (r *Renderer) RenderString(node gui.Node) string {
	var buf strings.Builder
	r.Render(node, &buf) //nolint:errcheck
	return buf.String()
}

// Update re-resolves the root function, diffs against the previous tree,
// and applies patches to the live DOM. On the first call it replaces the
// container's content entirely.
func (r *Renderer) Update(root func() gui.Node) {
	r.UpdateTree(gui.Resolve(root()))
}

// UpdateTree diffs newTree (which must already be resolved) against the
// previous tree and applies patches to the live DOM. On the first call it
// replaces the container's content entirely.
//
// Use this when you need to resolve the tree yourself (e.g. via
// [gui.ResolveTracked]) and want to avoid double-resolving.
func (r *Renderer) UpdateTree(newTree gui.Node) {
	if r.prevTree == nil {
		// First render: clear container and mount fresh DOM.
		r.releaseCallbacks()
		r.container.Set("innerHTML", "")
		domNode := r.createDOMNode(newTree)
		if !domNode.IsNull() && !domNode.IsUndefined() {
			r.container.Call("appendChild", domNode)
		}
		r.prevTree = newTree
		return
	}

	patches := gui.Diff(r.prevTree, newTree)
	if len(patches) == 0 {
		return
	}

	fs := r.saveFocus()
	r.applyPatches(patches, r.container)
	r.restoreFocus(fs)
	r.prevTree = newTree
}

// Release frees all JavaScript callback functions and revokes any tracked
// object URLs. Call this when the renderer is no longer needed to prevent
// memory leaks.
func (r *Renderer) Release() {
	r.releaseCallbacks()
	r.revokeObjectURLs()
}

func (r *Renderer) releaseCallbacks() {
	for k, slot := range r.handlers {
		slot.jsfn.Release()
		delete(r.handlers, k)
	}
}

// elemID returns a stable integer identifier for a DOM element, assigning one
// via an expando property (_rid) on first access.
func (r *Renderer) elemID(el js.Value) int {
	rid := el.Get("_rid")
	if rid.IsUndefined() {
		r.nextElemID++
		el.Set("_rid", r.nextElemID)
		return r.nextElemID
	}
	return rid.Int()
}

// createDOMNode recursively creates browser DOM elements from a gui.Node tree.
func (r *Renderer) createDOMNode(node gui.Node) js.Value {
	return r.createDOMNodeAt(node, 0)
}

// createDOMNodeAt is the depth-tracked implementation of createDOMNode.
func (r *Renderer) createDOMNodeAt(node gui.Node, depth int) js.Value {
	if node == nil || depth > maxDOMDepth {
		return js.Null()
	}

	switch n := node.(type) {
	case *gui.Element:
		el := r.doc.Call("createElement", n.Tag)
		r.applyProps(el, n.Props)
		for _, child := range n.Children {
			childDOM := r.createDOMNodeAt(child, depth+1)
			if !childDOM.IsNull() && !childDOM.IsUndefined() {
				el.Call("appendChild", childDOM)
			}
		}
		// For <select>, re-apply the value after options are appended.
		// Setting .value before options exist is a no-op in the browser.
		if n.Tag == "select" {
			if v, ok := n.Props["value"]; ok {
				el.Set("value", js.ValueOf(v).String())
			}
		}
		return el

	case *gui.TextNode:
		return r.doc.Call("createTextNode", n.Content)

	case *gui.Fragment:
		frag := r.doc.Call("createDocumentFragment")
		for _, child := range n.Children {
			childDOM := r.createDOMNodeAt(child, depth+1)
			if !childDOM.IsNull() && !childDOM.IsUndefined() {
				frag.Call("appendChild", childDOM)
			}
		}
		return frag

	default:
		return js.Null()
	}
}

// applyProps sets HTML attributes and wires event listeners on a DOM element.
func (r *Renderer) applyProps(el js.Value, props gui.Props) {
	for k, v := range props {
		if strings.HasPrefix(k, "on") && len(k) > 2 {
			event := k[2:] // "onclick" -> "click"
			r.setEventHandler(el, event, v)
			continue
		}

		switch val := v.(type) {
		case bool:
			if val {
				el.Call("setAttribute", k, "")
			}
		case nil:
			el.Call("removeAttribute", k)
		default:
			// "value" must be set as a DOM property so it updates the live
			// content of inputs/textareas (setAttribute only sets the default).
			if k == "value" {
				el.Set("value", js.ValueOf(v).String())
			} else {
				el.Call("setAttribute", k, js.ValueOf(v).String())
			}
		}
	}
}

// setEventHandler wires a Go callback as the element's on<event> property.
// Using the property (el.onclick = fn) instead of addEventListener ensures
// that each element has at most one handler per event — subsequent calls
// replace the previous handler rather than accumulating listeners.
// Supports both func() (simple handlers) and func(gui.Event) (rich handlers).
//
// Handler slots are stable: the js.Func is created once per (element, event)
// pair and its closure dispatches to a swappable pointer. Re-renders only
// update the pointer — no js.Func alloc/Release/DOM write.
func (r *Renderer) setEventHandler(el js.Value, event string, handler any) {
	switch handler.(type) {
	case func(), func(gui.Event):
		// supported
	default:
		return // unsupported handler type
	}

	key := strconv.Itoa(r.elemID(el)) + ":" + event

	// Fast path: slot exists — just swap the handler pointer.
	if slot, ok := r.handlers[key]; ok {
		slot.current = handler
		return
	}

	// Slow path: first time seeing this (element, event) pair.
	slot := &handlerSlot{current: handler}
	slot.jsfn = js.FuncOf(func(this js.Value, args []js.Value) any {
		switch fn := slot.current.(type) {
		case func():
			fn()
		case func(gui.Event):
			evt := gui.Event{Type: event}
			if len(args) > 0 {
				jsEvt := args[0]
				evt = extractEventData(event, jsEvt)
				evt.JSRaw = jsEvt
				evt.PreventDefault = func() { jsEvt.Call("preventDefault") }
			}
			// Revoke previous object URLs and track new ones.
			if event == "paste" && len(evt.ImageURLs) > 0 {
				r.revokeObjectURLs()
				r.objectURLs = append(r.objectURLs[:0], evt.ImageURLs...)
			}
			fn(evt)
		}
		return nil
	})
	r.handlers[key] = slot
	el.Set("on"+event, slot.jsfn)
}

// extractEventData reads relevant fields from a browser event into a gui.Event.
func extractEventData(eventType string, jsEvt js.Value) gui.Event {
	evt := gui.Event{Type: eventType}

	switch eventType {
	case "click", "mousedown", "mouseup", "mousemove", "mouseenter", "mouseleave":
		evt.X = jsEvt.Get("clientX").Int()
		evt.Y = jsEvt.Get("clientY").Int()

	case "keypress", "keydown", "keyup":
		evt.Key = jsEvt.Get("key").String()
		evt.ShiftKey = jsEvt.Get("shiftKey").Bool()

	case "input", "change":
		target := jsEvt.Get("target")
		if !target.IsNull() && !target.IsUndefined() {
			evt.Value = target.Get("value").String()
		}

	case "paste":
		cd := jsEvt.Get("clipboardData")
		if !cd.IsNull() && !cd.IsUndefined() {
			items := cd.Get("items")
			n := items.Length()
			const maxPasteImages = 10
			for i := 0; i < n; i++ {
				if len(evt.ImageURLs) >= maxPasteImages {
					break
				}
				item := items.Index(i)
				if item.Get("kind").String() == "file" &&
					strings.HasPrefix(item.Get("type").String(), "image/") {
					file := item.Call("getAsFile")
					if !file.IsNull() && !file.IsUndefined() {
						u := js.Global().Get("URL").Call("createObjectURL", file)
						evt.ImageURLs = append(evt.ImageURLs, u.String())
					}
				}
			}
		}

	case "touchstart", "touchend", "touchmove", "touchcancel":
		touches := jsEvt.Get("touches")
		if !touches.IsUndefined() && touches.Length() > 0 {
			touch := touches.Index(0)
			evt.X = touch.Get("clientX").Int()
			evt.Y = touch.Get("clientY").Int()
		}

	case "dragstart", "dragend", "dragover", "dragenter", "dragleave", "drop":
		evt.X = jsEvt.Get("clientX").Int()
		evt.Y = jsEvt.Get("clientY").Int()

	case "wheel":
		evt.X = jsEvt.Get("clientX").Int()
		evt.Y = jsEvt.Get("clientY").Int()

	case "focus", "blur", "focusin", "focusout":
		target := jsEvt.Get("target")
		if !target.IsNull() && !target.IsUndefined() {
			evt.Value = target.Get("value").String()
		}
	}

	return evt
}

// applyPatches applies a set of diff patches to the live DOM.
func (r *Renderer) applyPatches(patches []gui.Patch, container js.Value) {
	// When the root tree is a Fragment its children were flattened directly
	// into the container (DocumentFragment is transparent on appendChild).
	// In that case diff paths are relative to the Fragment, whose children
	// map 1-to-1 to the container's childNodes — so we use the container
	// itself as the navigation root. For a single-element root we use the
	// container's sole child as before.
	var root js.Value
	if _, isFrag := r.prevTree.(*gui.Fragment); isFrag {
		root = container
	} else {
		root = container.Get("firstChild")
		if root.IsNull() || root.IsUndefined() {
			return
		}
	}

	for _, p := range patches {
		switch p.Op {
		case gui.OpReplace:
			target := r.navigate(root, p.Path)
			if target.IsNull() || target.IsUndefined() {
				continue
			}
			newNode := r.createDOMNode(p.New)
			parent := target.Get("parentNode")
			if !parent.IsNull() && !parent.IsUndefined() {
				parent.Call("replaceChild", newNode, target)
			}

		case gui.OpUpdateProps:
			target := r.navigate(root, p.Path)
			if target.IsNull() || target.IsUndefined() {
				continue
			}
			for k, v := range p.Props {
				if strings.HasPrefix(k, "on") && len(k) > 2 {
					event := k[2:]
					r.setEventHandler(target, event, v)
					continue
				}
				if v == nil {
					target.Call("removeAttribute", k)
				} else if bv, ok := v.(bool); ok {
					if bv {
						target.Call("setAttribute", k, "")
					} else {
						target.Call("removeAttribute", k)
					}
				} else if k == "value" {
					newVal := js.ValueOf(v).String()
					currentVal := target.Get("value").String()
					// Skip value writes on the focused element — the browser's
					// value is authoritative while the user is typing. Writing
					// a stale value from the virtual tree drops keystrokes.
					if currentVal != newVal && !target.Equal(r.doc.Get("activeElement")) {
						target.Set("value", newVal)
					}
				} else {
					target.Call("setAttribute", k, js.ValueOf(v).String())
				}
			}

		case gui.OpUpdateText:
			target := r.navigate(root, p.Path)
			if target.IsNull() || target.IsUndefined() {
				continue
			}
			target.Set("textContent", p.NewText)

		case gui.OpInsertChild:
			parent := r.navigate(root, p.Path)
			if parent.IsNull() || parent.IsUndefined() {
				continue
			}
			newChild := r.createDOMNode(p.New)
			children := parent.Get("childNodes")
			if p.Index >= children.Length() {
				parent.Call("appendChild", newChild)
			} else {
				ref := children.Index(p.Index)
				parent.Call("insertBefore", newChild, ref)
			}

		case gui.OpRemoveChild:
			parent := r.navigate(root, p.Path)
			if parent.IsNull() || parent.IsUndefined() {
				continue
			}
			children := parent.Get("childNodes")
			if p.Index < children.Length() {
				child := children.Index(p.Index)
				parent.Call("removeChild", child)
			}
		}
	}
}

// navigate walks the DOM tree by the index path to find a specific node.
func (r *Renderer) navigate(root js.Value, path []int) js.Value {
	current := root
	for _, idx := range path {
		children := current.Get("childNodes")
		if idx >= children.Length() {
			return js.Null()
		}
		current = children.Index(idx)
		if current.IsNull() || current.IsUndefined() {
			return js.Null()
		}
	}
	return current
}

// saveFocus captures the focused element's identity and cursor position if
// it is an input or textarea inside the renderer's container.
func (r *Renderer) saveFocus() *focusState {
	active := r.doc.Get("activeElement")
	if active.IsNull() || active.IsUndefined() {
		return nil
	}
	tag := strings.ToLower(active.Get("tagName").String())
	if tag != "textarea" && tag != "input" {
		return nil
	}
	// Only save if the active element is inside our container.
	if !r.container.Call("contains", active).Bool() {
		return nil
	}
	var selStart, selEnd int
	if ss := active.Get("selectionStart"); !ss.IsNull() && !ss.IsUndefined() {
		selStart = ss.Int()
	}
	if se := active.Get("selectionEnd"); !se.IsNull() && !se.IsUndefined() {
		selEnd = se.Int()
	}
	return &focusState{
		active:         active,
		tagName:        tag,
		id:             active.Get("id").String(),
		className:      active.Get("className").String(),
		selectionStart: selStart,
		selectionEnd:   selEnd,
		value:          active.Get("value").String(),
	}
}

// restoreFocus re-focuses the element captured by saveFocus and restores the
// cursor/selection position. It is a no-op when fs is nil or focus was not lost.
func (r *Renderer) restoreFocus(fs *focusState) {
	if fs == nil {
		return
	}
	// If the original element is still focused, nothing to do.
	current := r.doc.Get("activeElement")
	if !current.IsNull() && !current.IsUndefined() && current.Equal(fs.active) {
		return
	}

	// Try to find the equivalent replacement element.
	var replacement js.Value
	if fs.id != "" {
		replacement = r.doc.Call("getElementById", fs.id)
	}
	if (replacement.IsNull() || replacement.IsUndefined()) && fs.className != "" {
		classes := strings.Fields(fs.className)
		escapedClasses := make([]string, len(classes))
		for i, c := range classes {
			escapedClasses[i] = escapeCSS(c)
		}
		selector := escapeCSS(fs.tagName) + "." + strings.Join(escapedClasses, ".")
		replacement = r.container.Call("querySelector", selector)
	}
	if replacement.IsNull() || replacement.IsUndefined() {
		// Last resort: find the first matching tag in the container.
		replacement = r.container.Call("querySelector", escapeCSS(fs.tagName))
	}
	if replacement.IsNull() || replacement.IsUndefined() {
		return
	}

	// Restore the DOM value captured before patching — the replacement
	// element was created by createDOMNode which may have written a stale
	// value from the virtual tree. The browser's value at save time is
	// the source of truth.
	replacement.Set("value", fs.value)
	replacement.Call("focus")
	replacement.Call("setSelectionRange", fs.selectionStart, fs.selectionEnd)
}

// revokeObjectURLs frees any previously tracked blob URLs created via
// createObjectURL. This prevents memory leaks from accumulated paste events.
func (r *Renderer) revokeObjectURLs() {
	urlAPI := js.Global().Get("URL")
	for _, u := range r.objectURLs {
		urlAPI.Call("revokeObjectURL", u)
	}
	r.objectURLs = r.objectURLs[:0]
}

// escapeCSS escapes CSS metacharacters in s so it can be safely used in a
// querySelector selector. Characters that have special meaning in CSS selectors
// are prefixed with a backslash.
func escapeCSS(s string) string {
	var buf strings.Builder
	for _, r := range s {
		switch r {
		case '!', '"', '#', '$', '%', '&', '\'', '(', ')', '*', '+', ',',
			'.', '/', ':', ';', '<', '=', '>', '?', '@', '[', '\\', ']',
			'^', '`', '{', '|', '}', '~', ' ':
			buf.WriteRune('\\')
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

// renderToString produces a simple HTML string from a resolved node tree.
// Used by Render/RenderString for gui.Renderer interface compliance.
func (r *Renderer) renderToString(node gui.Node) string {
	if node == nil {
		return ""
	}
	switch n := node.(type) {
	case *gui.Element:
		var buf strings.Builder
		buf.WriteString("<" + n.Tag + ">")
		for _, child := range n.Children {
			buf.WriteString(r.renderToString(child))
		}
		buf.WriteString("</" + n.Tag + ">")
		return buf.String()
	case *gui.TextNode:
		return stdhtml.EscapeString(n.Content)
	case *gui.Fragment:
		var buf strings.Builder
		for _, child := range n.Children {
			buf.WriteString(r.renderToString(child))
		}
		return buf.String()
	default:
		return ""
	}
}
