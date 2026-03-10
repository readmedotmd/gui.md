# Core Package (`gui`)

The `gui` package provides the foundational types for building backend-agnostic UI trees: nodes, components, a global store, a tree-diffing engine, and a renderer interface. It has zero external dependencies.

## Nodes

All renderable items implement the `Node` interface. The interface is sealed — external packages cannot satisfy it — ensuring that renderers only need to handle the known concrete types.

### Node Types

| Type | Description | Constructor |
|------|-------------|-------------|
| `*Element` | A named element with props and children (e.g. `div`, `box`) | `El()`, `Tag()` |
| `*TextNode` | Raw text content | `Text()`, `Textf()` |
| `*Fragment` | Groups children without a wrapper element | `Frag()` |
| `*ComponentNode` | Wraps a component invocation (functional or stateful) | `Comp()`, `Mount()`, `C()` |

### Element

The primary building block. Has a `Tag` string, a `Props` map, and a `Children` slice.

```go
type Element struct {
    Tag      string
    Props    Props
    Children []Node
}
```

### TextNode

Holds a single string of text content.

```go
type TextNode struct {
    Content string
}
```

### Fragment

Groups children without introducing a wrapper element. Useful when a component must return multiple siblings.

```go
type Fragment struct {
    Children []Node
}
```

## Constructors

### `Tag(name)` — Curried Element Builder

The primary way backends define their element vocabulary. Returns a three-stage curried function:

```
Tag(name) → func(attrs ...Attr) → func(children ...Node) → *Element
```

**Usage:**

```go
var Div = gui.Tag("div")

// Stage 1: apply attributes
// Stage 2: apply children
node := Div(gui.Attr_("class", "wrapper"))(
    gui.Text("hello"),
)
```

Each call to the final stage produces an independent `*Element`, so the same builder can be reused:

```go
builder := Div(gui.Attr_("id", "a"))
el1 := builder(gui.Text("one"))  // separate element
el2 := builder(gui.Text("two"))  // separate element
```

### `El(tag, props, children...)` — Direct Constructor

A lower-level escape hatch that creates an `*Element` directly. If `props` is nil, it is replaced with an empty `Props{}` so callers can always write to `el.Props` safely.

```go
el := gui.El("section", gui.Props{"id": "root"}, gui.Text("content"))
```

### `Text(s)` and `Textf(format, args...)`

Create `*TextNode` values. `Textf` works like `fmt.Sprintf`:

```go
gui.Text("Hello")
gui.Textf("Count: %d", 42)
```

### `Frag(children...)`

Creates a `*Fragment`. An empty fragment is valid.

```go
gui.Frag(
    gui.Text("line one"),
    gui.Text("line two"),
)
```

## Props and Attrs

`Props` is `map[string]any` — a string-keyed map of attributes and event handlers. Interpretation of values is left to the backend renderer.

`Attr` is `func(Props)` — a function that sets one or more props. Attrs compose naturally:

```go
type Attr func(Props)
```

### `Attr_(key, value)`

The generic escape hatch for setting an arbitrary prop:

```go
gui.Attr_("data-user-id", "42")
gui.Attr_("tabindex", 3)
gui.Attr_("hidden", true)
```

Backends define their own typed helpers on top of this pattern. See the [HTML](html.md) and [Terminal](term.md) docs.

## Events

`Event` carries data about a user interaction. The fields populated depend on the event type.

```go
type Event struct {
    Type  string // "click", "keypress", "input", "change", "mouseenter", "mouseleave"
    Key   string // keyboard events
    Value string // input/change events
    X, Y  int    // mouse position
}
```

`EventHandler` is a type alias for `func(Event)`. Using a type alias means plain `func(Event)` literals satisfy the type without explicit conversion:

```go
type EventHandler = func(Event)
```

Both `func()` (simple callbacks) and `func(Event)` (rich handlers) are supported in Props. Simple handlers are convenient for actions that don't need event data; rich handlers provide access to mouse coordinates, key names, and input values.

## Components

### Functional Components

`FuncComponent[T]` is a generic function: `func(props T, children []Node) Node`. The type parameter gives compile-time safety for props:

```go
type NavBarProps struct {
    Title string
}

func NavBar(props NavBarProps, children []gui.Node) gui.Node {
    return h.Header()(
        h.H1()(gui.Text(props.Title)),
        h.Nav()(gui.Frag(children...)),
    )
}
```

Wrap with `Comp()` to place in the tree — `T` is inferred from the function:

```go
gui.Comp(NavBar, NavBarProps{Title: "My App"},
    h.A(h.Href("/"))(gui.Text("Home")),
    h.A(h.Href("/about"))(gui.Text("About")),
)
```

### Renderable and Component[P]

`Renderable` is the non-generic interface used to store stateful components in the tree:

```go
type Renderable interface {
    Render() Node
}
```

`Component[P]` is the generic interface — `P` is the props type. It extends `Renderable` with an unexported `initBase` method that is automatically satisfied by embedding `BaseComponent[P, S]`:

```go
type Component[P any] interface {
    Renderable
    initBase(P, []Node)
}
```

### Stateful Components

Embed `BaseComponent[P, S]` where `P` is the props type and `S` is the state type, and implement `Render() Node`:

```go
type CounterState struct{ Count int }

type Counter struct {
    gui.BaseComponent[gui.Props, CounterState]
}

func (c *Counter) Render() gui.Node {
    s := c.State()  // returns CounterState — fully typed, no assertion needed
    return gui.El("div", nil, gui.Textf("Count: %d", s.Count))
}
```

For components that don't need typed props, use `gui.Props` as the first type parameter. `nil` is valid for map types, so `gui.Mount(c, nil)` works.

For typed props, define a struct:

```go
type UserCardProps struct { Name string }
type CardState struct { Role string }

type UserCard struct {
    gui.BaseComponent[UserCardProps, CardState]
}

func (u *UserCard) Render() gui.Node {
    name := u.Props().Name  // typed! no assertion needed
    return gui.El("div", nil, gui.Text(name))
}
```

Wrap with `Mount[P]()` to place in the tree. `P` is inferred from the component, giving compile-time type safety:

```go
// P inferred as gui.Props — nil is valid for map types:
gui.Mount(&Counter{}, nil)

// P inferred as UserCardProps — compile-time checked:
gui.Mount(&UserCard{}, UserCardProps{Name: "Alice"})

// Compile error — wrong props type:
gui.Mount(&UserCard{}, gui.Props{"name": "Alice"})  // ✗ won't compile
```

### Managed Components (`C`)

`C[P]()` wraps a stateful component for the tree, but unlike `Mount`, the framework manages instance creation, caching, and reuse automatically. Pass a zero-value exemplar (via `new(T)`) — it is only used for type inference:

```go
// Framework creates and manages the instance. State is preserved across renders.
gui.C(new(ContactForm), nil)
gui.C(new(UserCard), UserCardProps{Name: "Alice"})

// Compile error — wrong props type (same safety as Mount):
gui.C(new(UserCard), gui.Props{})  // ✗ won't compile
```

With a `Reconciler` (e.g. inside `dom.App`), instances are matched by **type + encounter order** — the Nth `ContactForm` in the tree reuses the Nth cached instance. Without a reconciler (standalone `Resolve`), a fresh instance is created each time (fine for static rendering).

**Lifecycle with reconciler:**

| Scenario | Lifecycle |
|----------|-----------|
| First render | `NewFunc()` → `InitFunc()` → `WillMount` → `Render` → `DidMount` |
| Re-render (same slot) | `InitFunc()` (updates props) → `WillUpdate` → `Render` → `DidUpdate` |
| Removed from tree | `dom.App` calls `DidUnmount`; reconciler deletes cached instance |
| Re-added after removal | Fresh instance — `WillMount` again (unmount destroys state) |

**When to use `C` vs `Mount`:**

| | `C` (managed) | `Mount` (manual) |
|---|---|---|
| Instance management | Framework creates + caches | You create + thread through |
| State preservation | Automatic (via reconciler) | You hold the pointer |
| Use case | Most components | Escape hatch, testing, pre-configured instances |

### BaseComponent[P, S] API

| Method | Description |
|--------|-------------|
| `Props() P` | Returns the typed props injected by `Mount()` or `C()` |
| `Children() []Node` | Returns children injected by `Mount()` or `C()` |
| `State() S` | Returns the current typed state |
| `SetState(s S)` | Replaces the entire state |
| `UpdateState(fn func(S) S)` | Applies a function to the current state |

### Lifecycle Hooks

Override these methods on your struct to hook into the component lifecycle:

| Hook | When called |
|------|-------------|
| `WillMount()` | Before the first `Render()` call |
| `DidMount()` | After the first `Render()` call |
| `WillUpdate()` | Before a re-render |
| `DidUpdate()` | After a re-render |
| `DidUnmount()` | When the component is removed |

The call order during `Resolve()` is: `WillMount` → `Render` → `DidMount`.

## Resolve

`Resolve(node Node) Node` recursively walks the tree and expands all `*ComponentNode` values into concrete `*Element`, `*TextNode`, or `*Fragment` values. Renderers should call this before traversing.

- Functional components: calls the captured closure with children and resolves the result
- Stateful components: calls lifecycle hooks, then `Render()`, and resolves the result
- Elements and Fragments: recursively resolves children
- TextNodes: returned as-is
- nil: returns nil

Nested components (a component whose `Render()` returns another `*ComponentNode`) are resolved recursively through all levels.

### ResolveTracked

`ResolveTracked(node Node, onComponent func(Renderable)) Node` works like `Resolve` but calls `onComponent` for every stateful component encountered during resolution. This lets callers (e.g. `dom.App`) discover which components are in the current tree so they can auto-wire `SetOnChange` and call `DidUnmount` on removal.

Passing a nil callback is valid and equivalent to `Resolve`.

### Reconciler

`Reconciler` manages the instance cache for components created via `C`. It is used automatically by `dom.App` — most users don't interact with it directly.

```go
rec := gui.NewReconciler()

// Each call reuses cached instances for C-nodes and cleans up removed ones.
resolved := rec.Resolve(tree, func(c gui.Renderable) {
    // called for every stateful component (both C-managed and Mount-managed)
})
```

Instance matching is by **type + encounter order** (same as React without keys). After each `Resolve` call, cache entries not seen in the current cycle are deleted.

## Store

`Store[T]` is a generic, thread-safe state container inspired by Zustand. `T` is typically a struct.

```go
type AppState struct {
    Count int
    Name  string
}

store := gui.NewStore(AppState{Count: 0, Name: "World"})
```

### Methods

| Method | Description |
|--------|-------------|
| `Get() T` | Returns the current state (read-locked) |
| `Set(newState T)` | Replaces the entire state and notifies subscribers |
| `Update(fn func(T) T)` | Applies `fn` to current state, stores result, notifies subscribers |
| `Subscribe(fn func(state, prevState T)) func()` | Registers a listener; returns an unsubscribe function |

### Thread Safety

All methods are safe for concurrent use. The store uses `sync.RWMutex` internally. Subscriber notifications are delivered **outside** the lock, so subscribers can safely call `Get()`, `Set()`, or `Update()` without deadlocking.

### Subscriptions

Subscribers receive both the new state and the previous state:

```go
unsub := store.Subscribe(func(cur, prev AppState) {
    fmt.Printf("Count changed from %d to %d\n", prev.Count, cur.Count)
})

// Later:
unsub()   // safe to call multiple times
```

### Update Pattern

`Update` is the idiomatic way to make partial changes without replacing unrelated fields:

```go
store.Update(func(s AppState) AppState {
    s.Count++
    return s
})
```

**Slice fields**: when appending to slices inside `Update`, copy first to avoid sharing the underlying array with the previous state:

```go
store.Update(func(s AppState) AppState {
    items := make([]string, len(s.Items), len(s.Items)+1)
    copy(items, s.Items)
    s.Items = append(items, "new item")
    return s
})
```

## Diff

`Diff(old, new Node) []Patch` compares two **resolved** node trees and returns a minimal list of patches. Both trees should already be resolved (no `*ComponentNode` values).

### Patch Operations

| Op | Description | Relevant Fields |
|----|-------------|-----------------|
| `OpReplace` | Entire node replaced | `Old`, `New` |
| `OpUpdateProps` | Props changed on an element (same tag) | `Props` (changed keys only; `nil` value = removal) |
| `OpUpdateText` | Text content changed | `OldText`, `NewText` |
| `OpInsertChild` | New child added | `New`, `Index` |
| `OpRemoveChild` | Child removed | `Old`, `Index` |

### Patch Structure

```go
type Patch struct {
    Op      PatchOp
    Path    []int   // index path from root to the target node
    Old     Node
    New     Node
    Props   Props
    OldText string
    NewText string
    Index   int     // child index for Insert/Remove
}
```

### Behavior Details

- **Path**: an `[]int` of child indices from the root to the target. Root patches have an empty (non-nil) path.
- **Unchanged props** are excluded from `OpUpdateProps` — only changed, added, or removed keys appear.
- **Function-valued props** (event handlers — both `func()` and `func(Event)`) are always considered changed, since Go functions cannot be meaningfully compared.
- **RemoveChild patches** are emitted in **reverse index order** (highest index first) so that a renderer can splice them without index shifting.
- **InsertChild patches** are emitted in ascending index order.
- **nil vs nil** produces no patches. **nil vs non-nil** (or vice versa) produces an `OpReplace`.
- Path slices are copied — mutating one patch's path does not corrupt another's.

## Renderer Interface

```go
type Renderer interface {
    Render(node Node, w io.Writer) error
    RenderString(node Node) string
}
```

Backends implement this interface. Both built-in renderers (`html.Renderer` and `term.Renderer`) call `Resolve()` internally, so callers don't need to resolve manually.

See the [HTML renderer](html.md) and [Terminal renderer](term.md) docs for backend-specific details.

## Source Files

| File | Contents |
|------|----------|
| `node.go` | `Node` interface, `Element`, `TextNode`, `Fragment`, `Props`, `Attr`, `Tag`, `El`, `Text`, `Textf`, `Frag`, `Attr_` |
| `component.go` | `FuncComponent[T]`, `ComponentNode`, `Comp[T]`, `Mount[P]`, `C[P]`, `Renderable`, `Component[P]`, `BaseComponent[P, S]`, lifecycle hooks, `Resolve`, `ResolveTracked` |
| `reconciler.go` | `Reconciler`, `NewReconciler` |
| `store.go` | `Store[T]`, `NewStore`, `Get`, `Set`, `Update`, `Subscribe` |
| `event.go` | `Event`, `EventHandler` |
| `diff.go` | `PatchOp`, `Patch`, `Diff`, `isHandlerFunc` |
| `renderer.go` | `Renderer` interface |
