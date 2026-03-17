package gui

import "strings"

// RouteConfig defines a declarative route with optional pattern parameters,
// nested children, a layout wrapper, and a handler that produces a Node.
//
// Patterns support named parameters prefixed with ":" (e.g. "/user/:id") and
// a trailing "*" wildcard that matches the rest of the path (e.g. "/files/*path").
//
// Example:
//
//	gui.RouteConfig{
//	    Path:    "/user/:id",
//	    Handler: func(p gui.Params) gui.Node { return userPage(p["id"]) },
//	}
type RouteConfig struct {
	// Path is the URL pattern to match (e.g. "/", "/user/:id", "/files/*path").
	Path string

	// Handler produces the page Node for this route. It receives any
	// extracted path parameters. Nil when the route is layout-only.
	Handler func(Params) Node

	// Layout wraps the matched child's output. It receives the child Node
	// (the "outlet") and returns the layout Node. When nil, the child is
	// rendered without a wrapper.
	Layout func(outlet Node) Node

	// Children defines nested routes. A child's Path is relative to the
	// parent's Path. An empty child Path ("") matches the parent exactly
	// (index route).
	Children []RouteConfig

	// Guards are checked in order before this route is entered.
	// If any guard returns false, navigation is cancelled.
	Guards []RouteGuard

	// segments caches the parsed pattern. Populated lazily on first match.
	segments []routeSegment
	parsed   bool
}

// Params holds named route parameters extracted from the URL.
type Params map[string]string

// RouteGuard is called before entering a route. Return false to cancel navigation.
// The from and to arguments are the previous and next paths.
type RouteGuard func(from, to string) bool

// RouteMatch is the result of matching a path against a route configuration tree.
type RouteMatch struct {
	// Params contains named parameters extracted from the path.
	Params Params

	// Handler is the matched route's handler function.
	Handler func(Params) Node

	// Layouts is the chain of layout functions from outermost to innermost.
	Layouts []func(outlet Node) Node

	// Guards is the chain of guard functions from outermost to innermost.
	Guards []RouteGuard
}

// routeSegment is a parsed segment of a route pattern.
type routeSegment struct {
	literal  string // non-empty for literal segments
	param    string // non-empty for ":param" segments
	wildcard string // non-empty for "*wildcard" segments (always last)
}

// parsePattern splits a route pattern into segments.
func parsePattern(pattern string) []routeSegment {
	pattern = strings.TrimPrefix(pattern, "/")
	if pattern == "" {
		return nil
	}
	parts := strings.Split(pattern, "/")
	segs := make([]routeSegment, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "*") {
			segs = append(segs, routeSegment{wildcard: p[1:]})
			break // wildcard consumes the rest
		}
		if strings.HasPrefix(p, ":") {
			segs = append(segs, routeSegment{param: p[1:]})
		} else {
			segs = append(segs, routeSegment{literal: p})
		}
	}
	return segs
}

// getSegments returns the parsed pattern segments, caching them on first access.
func (r *RouteConfig) getSegments() []routeSegment {
	if !r.parsed {
		r.segments = parsePattern(r.Path)
		r.parsed = true
	}
	return r.segments
}

// MatchRoute finds the first matching route in the configuration tree for the
// given path. It returns nil if no route matches.
func MatchRoute(routes []RouteConfig, path string) *RouteMatch {
	return matchRoutes(routes, normalizePath(path), nil, nil)
}

func matchRoutes(routes []RouteConfig, path string, layouts []func(Node) Node, guards []RouteGuard) *RouteMatch {
	for i := range routes {
		r := &routes[i]
		if m := matchSingle(r, path, layouts, guards); m != nil {
			return m
		}
	}
	return nil
}

func matchSingle(r *RouteConfig, path string, layouts []func(Node) Node, guards []RouteGuard) *RouteMatch {
	segs := r.getSegments()

	// Collect guards for this level.
	allGuards := guards
	if len(r.Guards) > 0 {
		allGuards = make([]RouteGuard, len(guards)+len(r.Guards))
		copy(allGuards, guards)
		copy(allGuards[len(guards):], r.Guards)
	}

	// Collect layouts for this level.
	allLayouts := layouts
	if r.Layout != nil {
		allLayouts = make([]func(Node) Node, len(layouts)+1)
		copy(allLayouts, layouts)
		allLayouts[len(layouts)] = r.Layout
	}

	pathSegs := splitPath(path)
	params := Params{}

	// Try to match this route's pattern against the beginning of the path.
	consumed, ok := matchSegments(segs, pathSegs, params)
	if !ok {
		return nil
	}

	remaining := joinPath(pathSegs[consumed:])

	// If there are children, try to match them first against the remaining path.
	if len(r.Children) > 0 {
		if m := matchRoutes(r.Children, remaining, allLayouts, allGuards); m != nil {
			// Merge params from this level.
			for k, v := range params {
				if _, exists := m.Params[k]; !exists {
					m.Params[k] = v
				}
			}
			return m
		}
	}

	// Leaf match: the entire path must be consumed (unless wildcard ate it).
	if remaining != "/" && remaining != "" {
		return nil
	}

	if r.Handler == nil {
		return nil
	}

	return &RouteMatch{
		Params:  params,
		Handler: r.Handler,
		Layouts: allLayouts,
		Guards:  allGuards,
	}
}

func matchSegments(segs []routeSegment, pathSegs []string, params Params) (consumed int, ok bool) {
	pi := 0
	for _, seg := range segs {
		if seg.wildcard != "" {
			// Wildcard consumes everything remaining.
			rest := strings.Join(pathSegs[pi:], "/")
			params[seg.wildcard] = rest
			return len(pathSegs), true
		}
		if pi >= len(pathSegs) {
			return 0, false
		}
		if seg.param != "" {
			params[seg.param] = pathSegs[pi]
			pi++
		} else if seg.literal != "" {
			if pathSegs[pi] != seg.literal {
				return 0, false
			}
			pi++
		}
	}
	return pi, true
}

func splitPath(path string) []string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func joinPath(segs []string) string {
	if len(segs) == 0 {
		return "/"
	}
	return "/" + strings.Join(segs, "/")
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// RenderMatch produces the final Node for a RouteMatch by calling the handler
// and wrapping the result in the layout chain (innermost first, outermost last).
func RenderMatch(m *RouteMatch) Node {
	if m == nil {
		return nil
	}
	node := m.Handler(m.Params)
	// Apply layouts from innermost to outermost.
	for i := len(m.Layouts) - 1; i >= 0; i-- {
		node = m.Layouts[i](node)
	}
	return node
}

// CheckGuards runs the guard chain for a RouteMatch. Returns true if all
// guards pass (or if there are no guards).
func CheckGuards(m *RouteMatch, from, to string) bool {
	if m == nil {
		return true
	}
	for _, g := range m.Guards {
		if !g(from, to) {
			return false
		}
	}
	return true
}
