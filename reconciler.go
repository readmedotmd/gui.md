package gui

import "strconv"

// Reconciler manages the lifecycle of components created via [C]. It caches
// instances by type and encounter order (like React without keys), so the Nth
// instance of a given component type in the tree always maps to the same
// cached struct, preserving state across renders.
//
// Use [NewReconciler] to create one. dom.App creates a reconciler
// automatically; standalone callers can use one for stateful HTML rendering.
type Reconciler struct {
	cache map[string]Renderable
}

// NewReconciler creates a Reconciler with an empty instance cache.
func NewReconciler() *Reconciler {
	return &Reconciler{
		cache: make(map[string]Renderable),
	}
}

// Resolve recursively resolves ComponentNodes into concrete nodes, managing
// the instance cache for C-nodes. onComponent is called for every stateful
// component encountered (both C-managed and Mount-managed), allowing callers
// to wire SetOnChange and track removals. Passing nil is valid.
func (r *Reconciler) Resolve(node Node, onComponent func(Renderable)) Node {
	counters := make(map[string]int)
	seen := make(map[string]struct{})
	result := r.resolve(node, onComponent, counters, seen)

	// Remove cache entries not seen this cycle.
	for key := range r.cache {
		if _, ok := seen[key]; !ok {
			delete(r.cache, key)
		}
	}

	return result
}

func (r *Reconciler) resolve(node Node, onComponent func(Renderable), counters map[string]int, seen map[string]struct{}) Node {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *ComponentNode:
		if n.Func != nil {
			return r.resolve(n.Func(n.Children), onComponent, counters, seen)
		}
		if n.NewFunc != nil {
			// Managed component: look up or create instance.
			idx := counters[n.TypeKey]
			counters[n.TypeKey] = idx + 1
			cacheKey := n.TypeKey + ":" + strconv.Itoa(idx)
			seen[cacheKey] = struct{}{}

			inst, exists := r.cache[cacheKey]
			if !exists {
				inst = n.NewFunc()
				r.cache[cacheKey] = inst
			}
			n.InitFunc(inst)

			return r.resolveStateful(inst, onComponent, counters, seen)
		}
		if n.Stateful != nil {
			return r.resolveStateful(n.Stateful, onComponent, counters, seen)
		}
		return nil
	case *Element:
		return &Element{Tag: n.Tag, Props: n.Props, Children: r.resolveAndFlatten(n.Children, onComponent, counters, seen)}
	case *Fragment:
		return &Fragment{Children: r.resolveAndFlatten(n.Children, onComponent, counters, seen)}
	case *TextNode:
		return n
	default:
		return n
	}
}

func (r *Reconciler) resolveStateful(inst Renderable, onComponent func(Renderable), counters map[string]int, seen map[string]struct{}) Node {
	mt, tracked := inst.(mountTracker)
	if tracked && mt.isMounted() {
		if wu, ok := inst.(willUpdater); ok {
			wu.WillUpdate()
		}
		result := r.resolve(inst.Render(), onComponent, counters, seen)
		if du, ok := inst.(didUpdater); ok {
			du.DidUpdate()
		}
		if onComponent != nil {
			onComponent(inst)
		}
		return result
	}
	// First mount.
	if wm, ok := inst.(willMounter); ok {
		wm.WillMount()
	}
	result := r.resolve(inst.Render(), onComponent, counters, seen)
	if dm, ok := inst.(didMounter); ok {
		dm.DidMount()
	}
	if tracked {
		mt.setMounted()
	}
	if onComponent != nil {
		onComponent(inst)
	}
	return result
}

func (r *Reconciler) resolveAndFlatten(children []Node, onComponent func(Renderable), counters map[string]int, seen map[string]struct{}) []Node {
	var result []Node
	for _, child := range children {
		resolved := r.resolve(child, onComponent, counters, seen)
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
