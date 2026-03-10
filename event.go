package gui

// Event carries data about a user interaction. The fields populated depend
// on the event Type — for example, keyboard events fill Key while mouse
// events fill X and Y.
type Event struct {
	Type           string   // "click", "keypress", "input", "change", "mouseenter", "mouseleave"
	Key            string   // keyboard events
	ShiftKey       bool     // keyboard events: whether the Shift key was held
	Value          string   // input/change events
	X, Y           int      // mouse position (terminal coordinates or client coordinates)
	ImageURLs      []string // paste events: object URLs for any image files in the clipboard
	PreventDefault func()   // call to prevent the browser's default action for this event
	// JSRaw holds the raw platform event object in DOM/WASM contexts
	// (concrete type: syscall/js.Value). Nil in all other backends.
	// Use a type assertion to access clipboard data, file lists, etc.
	JSRaw any
}

// EventHandler is the function signature for event callbacks that receive
// event data. Using a type alias (=) so that func(Event) literals satisfy
// EventHandler without an explicit conversion.
type EventHandler = func(Event)
