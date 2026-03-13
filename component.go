package gui

import (
	"fmt"
	"reflect"
)

// --- Functional Components ---

// FuncComponent is a generic function that takes typed props and children,
// and returns a Node. The type parameter T specifies the props type.
//
//	type NavBarProps struct { Title string }
//	func NavBar(props NavBarProps, children []gui.Node) gui.Node { ... }
type FuncComponent[T any] func(props T, children []Node) Node

// ComponentNode wraps a component invocation in the node tree.
// It holds either a functional component closure, a Renderable (stateful),
// or a managed component descriptor (set by C, consumed by Reconciler).
type ComponentNode struct {
	Func     func([]Node) Node // closure — typed props already captured
	Stateful Renderable        // manual instance (Mount)
	Children []Node

	// Managed component fields (set by C, consumed by Reconciler).
	TypeKey  string            // e.g. "main.ContactForm" — cache key
	NewFunc  func() Renderable // creates a zero-value instance
	InitFunc func(Renderable)  // sets props/children via initBase
}

func (c *ComponentNode) isNode() {}

// Comp wraps a functional component for use in the tree. The type parameter
// T is inferred from the function signature, giving compile-time type safety
// for props.
//
//	type NavBarProps struct { Title string }
//	gui.Comp(NavBar, NavBarProps{Title: "App"}, child1, child2)
//
// Existing code using gui.Props still works — T is inferred as Props:
//
//	gui.Comp(Greeting, gui.Props{"name": "Alice"})
func Comp[T any](fn FuncComponent[T], props T, children ...Node) *ComponentNode {
	return &ComponentNode{
		Func: func(children []Node) Node {
			return fn(props, children)
		},
		Children: children,
	}
}

// Mount wraps a stateful component for use in the tree.
// Sets props and children on the component's base before returning.
// The type parameter P is inferred from the component's embedded
// BaseComponent[P, S], giving compile-time type safety for props.
//
//	gui.Mount(&Counter{}, gui.Props{"label": "clicks"})
//	gui.Mount(&UserCard{}, UserCardProps{Name: "Alice"})  // typed props
func Mount[P any](c Component[P], props P, children ...Node) *ComponentNode {
	c.initBase(props, children)
	return &ComponentNode{Stateful: c, Children: children}
}

// C wraps a managed stateful component for use in the tree. The framework
// creates, caches, and reuses instances automatically — the caller passes
// a zero-value exemplar (via new(T)) purely for type inference.
//
// When used with a [Reconciler] (e.g. inside dom.App), the Nth instance of
// a given type encountered in the tree is matched to the Nth cached instance,
// preserving state across renders. Without a reconciler (standalone [Resolve]),
// a fresh instance is created every time (fine for static HTML rendering).
//
//	gui.C(new(ContactForm), nil)
//	gui.C(new(UserCard), UserCardProps{Name: "Alice"})
func C[P any](exemplar Component[P], props P, children ...Node) *ComponentNode {
	typ := reflect.TypeOf(exemplar).Elem()
	return &ComponentNode{
		Children: children,
		TypeKey:  typ.PkgPath() + "." + typ.Name(),
		NewFunc: func() Renderable {
			v := reflect.New(typ).Interface()
			r, ok := v.(Renderable)
			if !ok {
				panic(fmt.Sprintf("gui.C: type %T does not implement Renderable", v))
			}
			return r
		},
		InitFunc: func(r Renderable) {
			c, ok := r.(Component[P])
			if !ok {
				panic(fmt.Sprintf("gui.C: type %T does not implement Component[%s]", r, typ.String()))
			}
			c.initBase(props, children)
		},
	}
}

// --- Renderable interface (non-generic) ---

// Renderable is the non-generic interface for component storage in the tree.
// Any type that has a Render method satisfies Renderable.
type Renderable interface {
	Render() Node
}

// --- Component interface (generic) ---

// Component is the generic interface for stateful components.
// The type parameter P specifies the props type.
// Implement Render() and embed BaseComponent[P, S] for state management.
type Component[P any] interface {
	Renderable
	initBase(P, []Node)
}

// --- BaseComponent (generic) ---

// BaseComponent provides typed props access, typed state management,
// and lifecycle hooks. Embed in your component structs.
// P is the props type, S is the state type.
//
//	type CounterState struct { Count int }
//
//	type Counter struct {
//	    gui.BaseComponent[gui.Props, CounterState]
//	}
//
//	func (c *Counter) Render() gui.Node {
//	    s := c.State()  // CounterState — typed!
//	    return gui.El("div", nil, gui.Textf("Count: %d", s.Count))
//	}
//
// For typed props:
//
//	type UserCardProps struct { Name string }
//	type CardState struct { Role string }
//
//	type UserCard struct {
//	    gui.BaseComponent[UserCardProps, CardState]
//	}
//
//	func (u *UserCard) Render() gui.Node {
//	    name := u.Props().Name  // typed! no assertion needed
//	    return gui.El("div", nil, gui.Text(name))
//	}
type BaseComponent[P, S any] struct {
	props     P
	children  []Node
	state     S
	onChange  func()
	notifying bool // re-entrancy guard for onChange
	mounted   bool // true after the first Resolve cycle completes
}

// initBase sets props and children. Called by Mount().
func (c *BaseComponent[P, S]) initBase(props P, children []Node) {
	c.props = props
	c.children = children
}

// Props returns the component's typed props.
func (c *BaseComponent[P, S]) Props() P { return c.props }

// Children returns the component's children nodes.
func (c *BaseComponent[P, S]) Children() []Node { return c.children }

// State returns the current typed state.
func (c *BaseComponent[P, S]) State() S { return c.state }

// SetState replaces the component state and triggers a re-render
// if an onChange callback has been registered via [SetOnChange].
// Calls to SetState during an active onChange (e.g. from WillMount
// during a render cycle) are applied but do not re-trigger onChange.
func (c *BaseComponent[P, S]) SetState(s S) {
	c.state = s
	c.fireOnChange()
}

// UpdateState applies a function to the current state and triggers a
// re-render if an onChange callback has been registered via [SetOnChange].
// Like [SetState], re-entrant calls during onChange are applied silently.
//
//	c.UpdateState(func(s CounterState) CounterState {
//	    s.Count++
//	    return s
//	})
func (c *BaseComponent[P, S]) UpdateState(fn func(S) S) {
	c.state = fn(c.state)
	c.fireOnChange()
}

func (c *BaseComponent[P, S]) fireOnChange() {
	if c.onChange != nil && !c.notifying {
		c.notifying = true
		defer func() { c.notifying = false }()
		c.onChange()
	}
}

// SetOnChange registers a function that is called after every [SetState]
// or [UpdateState]. In interactive backends this is typically wired to a
// render function so that state changes automatically update the UI.
func (c *BaseComponent[P, S]) SetOnChange(fn func()) {
	c.onChange = fn
}

// --- Lifecycle hooks (override in your struct) ---

// WillMount is called before the first render.
func (c *BaseComponent[P, S]) WillMount() {}

// DidMount is called after the first render.
func (c *BaseComponent[P, S]) DidMount() {}

// WillUpdate is called before re-rendering.
func (c *BaseComponent[P, S]) WillUpdate() {}

// DidUpdate is called after re-rendering.
func (c *BaseComponent[P, S]) DidUpdate() {}

// DidUnmount is called when the component is removed.
func (c *BaseComponent[P, S]) DidUnmount() {}

// isMounted reports whether the component has completed its first render.
func (c *BaseComponent[P, S]) isMounted() bool { return c.mounted }

// setMounted marks the component as mounted after the first Resolve cycle.
func (c *BaseComponent[P, S]) setMounted() { c.mounted = true }

// --- Lifecycle interfaces for type-assertion dispatch ---

type willMounter interface{ WillMount() }
type didMounter interface{ DidMount() }
type willUpdater interface{ WillUpdate() }
type didUpdater interface{ DidUpdate() }

// mountTracker is satisfied by BaseComponent. Resolve uses it to
// distinguish first-mount (WillMount/DidMount) from subsequent
// renders (WillUpdate/DidUpdate).
type mountTracker interface {
	isMounted() bool
	setMounted()
}

// --- Resolve ---

// Resolve recursively resolves ComponentNodes into concrete
// Element/Text/Fragment nodes. Renderers call this before rendering.
func Resolve(node Node) Node {
	return resolve(node, nil)
}

// ResolveTracked works like [Resolve] but calls onComponent for every
// stateful component encountered during resolution. This lets callers
// (e.g. dom.App) discover which components are in the current tree so
// they can auto-wire SetOnChange and call DidUnmount on removal.
// Passing a nil callback is valid and equivalent to [Resolve].
func ResolveTracked(node Node, onComponent func(Renderable)) Node {
	return resolve(node, onComponent)
}

func resolve(node Node, onComponent func(Renderable)) Node {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *ComponentNode:
		if n.Func != nil {
			return resolve(n.Func(n.Children), onComponent)
		}
		if n.NewFunc != nil {
			// C-node without reconciler: create a fresh instance each time.
			inst := n.NewFunc()
			n.InitFunc(inst)
			if wm, ok := inst.(willMounter); ok {
				wm.WillMount()
			}
			result := resolve(inst.Render(), onComponent)
			if dm, ok := inst.(didMounter); ok {
				dm.DidMount()
			}
			if mt, ok := inst.(mountTracker); ok {
				mt.setMounted()
			}
			if onComponent != nil {
				onComponent(inst)
			}
			return result
		}
		if n.Stateful != nil {
			mt, tracked := n.Stateful.(mountTracker)
			if tracked && mt.isMounted() {
				// Re-render: call WillUpdate/DidUpdate.
				if wu, ok := n.Stateful.(willUpdater); ok {
					wu.WillUpdate()
				}
				result := resolve(n.Stateful.Render(), onComponent)
				if du, ok := n.Stateful.(didUpdater); ok {
					du.DidUpdate()
				}
				if onComponent != nil {
					onComponent(n.Stateful)
				}
				return result
			}
			// First mount: call WillMount/DidMount.
			if wm, ok := n.Stateful.(willMounter); ok {
				wm.WillMount()
			}
			result := resolve(n.Stateful.Render(), onComponent)
			if dm, ok := n.Stateful.(didMounter); ok {
				dm.DidMount()
			}
			if tracked {
				mt.setMounted()
			}
			if onComponent != nil {
				onComponent(n.Stateful)
			}
			return result
		}
		return nil
	case *Element:
		return &Element{Tag: n.Tag, Props: n.Props, Children: resolveAndFlatten(n.Children, onComponent)}
	case *Fragment:
		return &Fragment{Children: resolveAndFlatten(n.Children, onComponent)}
	case *TextNode:
		return n
	default:
		return n
	}
}

// resolveAndFlatten resolves each child and inlines any Fragment results
// into the parent's child list. This ensures the virtual tree matches the
// browser DOM structure where DocumentFragment children are transparent.
func resolveAndFlatten(children []Node, onComponent func(Renderable)) []Node {
	var result []Node
	for _, child := range children {
		resolved := resolve(child, onComponent)
		if resolved == nil {
			continue
		}
		if frag, ok := resolved.(*Fragment); ok {
			result = append(result, frag.Children...)
		} else {
			result = append(result, resolved)
		}
	}
	return result
}
