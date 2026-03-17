// Package html provides an HTML string renderer backend for the gui library.
package html

import (
	"errors"
	"fmt"
	gui "github.com/readmedotmd/gui.md"
	stdhtml "html"
	"io"
	"sort"
	"strings"
)

// maxRenderDepth is the maximum nesting depth for recursive rendering.
// This prevents stack overflow from pathologically deep trees.
const maxRenderDepth = 512

// ErrMaxDepth is returned when the node tree exceeds maxRenderDepth levels.
var ErrMaxDepth = errors.New("html: maximum render depth exceeded")

// Renderer renders a gui.Node tree to HTML strings.
// It resolves functional and stateful components via [gui.Resolve] before
// traversing, so callers do not need to resolve manually.
//
// Renderer implements the [gui.Renderer] interface; the compile-time check
// is in html_test.go.
type Renderer struct{}

// New creates a new HTML Renderer.
func New() *Renderer { return &Renderer{} }

// Render resolves components in node and writes the resulting HTML to w.
// It returns the first write error encountered, if any.
// Returns [ErrMaxDepth] if the tree exceeds 512 levels of nesting.
func (r *Renderer) Render(node gui.Node, w io.Writer) error {
	resolved := gui.Resolve(node)
	return r.renderNode(resolved, w, 0)
}

// RenderString returns the node tree rendered as an HTML string.
// Any write errors are silently discarded because strings.Builder never
// returns an error.
func (r *Renderer) RenderString(node gui.Node) string {
	var buf strings.Builder
	r.Render(node, &buf) //nolint:errcheck // strings.Builder never errors
	return buf.String()
}

// renderNode dispatches to the appropriate renderer based on the concrete
// node type. A nil node produces no output.
func (r *Renderer) renderNode(node gui.Node, w io.Writer, depth int) error {
	if node == nil {
		return nil
	}
	if depth > maxRenderDepth {
		return ErrMaxDepth
	}
	switch n := node.(type) {
	case *gui.Element:
		return r.renderElement(n, w, depth)
	case *gui.TextNode:
		_, err := io.WriteString(w, stdhtml.EscapeString(n.Content))
		return err
	case *gui.Fragment:
		for _, child := range n.Children {
			if err := r.renderNode(child, w, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		// Unknown node types produce no output; this keeps the renderer
		// forward-compatible with future node kinds.
		return nil
	}
}

// renderElement writes an opening tag, its children, and a closing tag.
// Void elements (e.g. <br>, <img>) that have no children are written as
// self-closing tags and their children slice is ignored.
func (r *Renderer) renderElement(el *gui.Element, w io.Writer, depth int) error {
	io.WriteString(w, "<")  //nolint:errcheck
	io.WriteString(w, el.Tag) //nolint:errcheck
	r.renderProps(el.Props, w)

	if isVoidElement(el.Tag) && len(el.Children) == 0 {
		io.WriteString(w, " />") //nolint:errcheck
		return nil
	}

	io.WriteString(w, ">") //nolint:errcheck
	for _, child := range el.Children {
		if err := r.renderNode(child, w, depth+1); err != nil {
			return err
		}
	}
	io.WriteString(w, "</")  //nolint:errcheck
	io.WriteString(w, el.Tag) //nolint:errcheck
	io.WriteString(w, ">")    //nolint:errcheck
	return nil
}

// renderProps writes sorted HTML attributes from props.
// The keys are sorted to produce deterministic output, which simplifies
// testing and diffing of rendered HTML.
//
// Rendering rules:
//   - func() values (event handlers from [On]) are silently skipped.
//   - bool values: true renders as a bare attribute flag; false is omitted.
//   - All other values are rendered as key="value" with HTML escaping applied.
func (r *Renderer) renderProps(props gui.Props, w io.Writer) {
	if len(props) == 0 {
		return
	}
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := props[k]
		if !isSafePropKey(k) {
			continue
		}
		// Skip function-valued props (event handlers registered with On/OnClick).
		switch v.(type) {
		case func():
			continue
		case func(gui.Event):
			continue
		}
		if b, ok := v.(bool); ok {
			if b {
				io.WriteString(w, " ")  //nolint:errcheck
				io.WriteString(w, k)    //nolint:errcheck
			}
			// false boolean props are omitted entirely.
			continue
		}
		io.WriteString(w, ` `)                                       //nolint:errcheck
		io.WriteString(w, k)                                         //nolint:errcheck
		io.WriteString(w, `="`)                                      //nolint:errcheck
		io.WriteString(w, stdhtml.EscapeString(fmt.Sprint(v)))       //nolint:errcheck
		io.WriteString(w, `"`)                                       //nolint:errcheck
	}
}

// isSafePropKey reports whether k is safe to use as an HTML attribute name.
// It rejects keys containing whitespace, quotes, '=', '<', '>', or '/'.
func isSafePropKey(k string) bool {
	for _, r := range k {
		switch r {
		case ' ', '\t', '\n', '\r', '"', '\'', '=', '<', '>', '/', '`':
			return false
		}
	}
	return k != ""
}

// isVoidElement reports whether tag is an HTML void element.
// Void elements cannot have child content and are self-closed.
// See https://html.spec.whatwg.org/multipage/syntax.html#void-elements
func isVoidElement(tag string) bool {
	switch tag {
	case "area", "base", "br", "col", "embed", "hr", "img",
		"input", "link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}
