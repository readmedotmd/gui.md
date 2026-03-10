// Package gui provides core node types and builder utilities for constructing
// UI element trees. It is backend-agnostic: concrete renderers (terminal, DOM,
// etc.) consume the node tree produced by these primitives.
package gui

import (
	"fmt"
	"regexp"
)

// validTag matches valid HTML/XML tag names: starts with a letter, followed by
// letters, digits, or hyphens. This prevents tag-name injection attacks where
// crafted strings like "div onload=alert(1)" could break out of the tag.
var validTag = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)

// ValidateTag checks whether tag is a safe, well-formed element name.
// It panics with a descriptive message when the tag is invalid, since an
// invalid tag name is always a programming error, not a runtime condition.
func ValidateTag(tag string) {
	if !validTag.MatchString(tag) {
		panic(fmt.Sprintf("gui: invalid tag name %q: must match [a-zA-Z][a-zA-Z0-9-]*", tag))
	}
}

// Node is the interface all renderable items implement.
// The unexported isNode method prevents external packages from accidentally
// satisfying the interface with unrelated types.
type Node interface {
	isNode()
}

// Element represents a named element with props and children (e.g. "div", "box").
// It is the primary building block of a UI tree.
type Element struct {
	// Tag identifies the element kind, e.g. "div", "button", "box".
	Tag string

	// Props holds the element's attributes and event handlers.
	Props Props

	// Children contains the ordered child nodes of this element.
	Children []Node
}

func (e *Element) isNode() {}

// TextNode represents raw text content inside a UI tree.
type TextNode struct {
	// Content is the literal text string to be rendered.
	Content string
}

func (t *TextNode) isNode() {}

// Fragment groups children without introducing a wrapper element in the tree.
// It is useful when a component must return multiple children but cannot or
// should not add an extra container node.
type Fragment struct {
	// Children contains the ordered child nodes of this fragment.
	Children []Node
}

func (f *Fragment) isNode() {}

// Props holds element attributes and event handlers as a string-keyed map.
// Values may be any type; interpretation is left to the backend renderer.
type Props map[string]any

// Attr is a function that sets a prop on an element.
// Used with the curried Tag element builders to apply attributes in a
// composable, functional style.
type Attr func(Props)

// Tag returns a curried element builder for the given tag name.
// This is the shared utility that backends use to define their elements.
//
// Example usage:
//
//	var Div = gui.Tag("div")
//	Div(Class("x"))(Text("hi"))
func Tag(name string) func(attrs ...Attr) func(children ...Node) *Element {
	ValidateTag(name)
	return func(attrs ...Attr) func(children ...Node) *Element {
		return func(children ...Node) *Element {
			props := Props{}
			for _, attr := range attrs {
				attr(props)
			}
			return &Element{Tag: name, Props: props, Children: children}
		}
	}
}

// El creates an Element directly with a tag, props, and children.
// This is a low-level escape hatch — prefer the curried Tag builders.
// If props is nil, it is replaced with an empty Props map so callers may
// always write to the returned element's Props without a nil-map panic.
func El(tag string, props Props, children ...Node) *Element {
	ValidateTag(tag)
	if props == nil {
		props = Props{}
	}
	return &Element{Tag: tag, Props: props, Children: children}
}

// Text creates a TextNode from a plain string.
func Text(s string) *TextNode {
	return &TextNode{Content: s}
}

// Textf creates a TextNode using fmt.Sprintf-style formatting.
// It accepts the same format string and variadic arguments as fmt.Sprintf.
func Textf(format string, args ...any) *TextNode {
	return &TextNode{Content: fmt.Sprintf(format, args...)}
}

// Frag creates a Fragment that groups children without a wrapper element.
func Frag(children ...Node) *Fragment {
	return &Fragment{Children: children}
}

// Attr_ sets an arbitrary key-value prop and returns an Attr that can be
// passed to any Tag builder. It is the escape hatch for backends that need
// props not covered by their typed helpers.
func Attr_(key string, value any) Attr {
	return func(p Props) { p[key] = value }
}
