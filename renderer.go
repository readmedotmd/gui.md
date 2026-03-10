package gui

import "io"

// Renderer is the interface that pluggable backends implement.
// A backend receives a fully-resolved Node tree and writes it to an
// io.Writer (for stream-based output) or returns it as a string.
//
// Backends should call [Resolve] on the incoming node before traversing,
// so that all [ComponentNode] values are expanded into concrete types.
type Renderer interface {
	// Render writes the node tree to w.
	Render(node Node, w io.Writer) error
	// RenderString returns the render as a string.
	RenderString(node Node) string
}
