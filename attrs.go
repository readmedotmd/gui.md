package gui

// Class sets the "class" attribute.
func Class(v string) Attr { return func(p Props) { p["class"] = v } }

// Id sets the "id" attribute.
func Id(v string) Attr { return func(p Props) { p["id"] = v } }

// Style sets the "style" attribute.
func Style(v string) Attr { return func(p Props) { p["style"] = v } }

// Href sets the "href" attribute.
func Href(v string) Attr { return func(p Props) { p["href"] = v } }

// Src sets the "src" attribute.
func Src(v string) Attr { return func(p Props) { p["src"] = v } }

// Alt sets the "alt" attribute.
func Alt(v string) Attr { return func(p Props) { p["alt"] = v } }

// Type sets the "type" attribute.
func Type(v string) Attr { return func(p Props) { p["type"] = v } }

// Accept sets the "accept" attribute (used on file inputs).
func Accept(v string) Attr { return func(p Props) { p["accept"] = v } }

// Name sets the "name" attribute.
func Name(v string) Attr { return func(p Props) { p["name"] = v } }

// Value sets the "value" attribute.
func Value(v string) Attr { return func(p Props) { p["value"] = v } }

// Placeholder sets the "placeholder" attribute.
func Placeholder(v string) Attr { return func(p Props) { p["placeholder"] = v } }

// Action sets the "action" attribute (forms).
func Action(v string) Attr { return func(p Props) { p["action"] = v } }

// Method sets the "method" attribute (forms).
func Method(v string) Attr { return func(p Props) { p["method"] = v } }

// Disabled sets the "disabled" boolean attribute.
// When v is true the attribute is rendered as a bare flag (e.g. disabled).
// When v is false the attribute is omitted entirely.
func Disabled(v bool) Attr { return func(p Props) { p["disabled"] = v } }

// Checked sets the "checked" boolean attribute.
// When v is true the attribute is rendered as a bare flag (e.g. checked).
// When v is false the attribute is omitted entirely.
func Checked(v bool) Attr { return func(p Props) { p["checked"] = v } }

// Data sets a "data-*" attribute.
// For example, Data("user-id", "42") sets data-user-id="42".
func Data(key, value string) Attr { return func(p Props) { p["data-"+key] = value } }

// On registers a rich event handler prop under the key "on<event>".
// The handler receives an [Event] with type-specific data (mouse
// coordinates, key name, input value, etc.).
// Event handlers are omitted from HTML string output but are available to
// interactive backends (e.g. the DOM renderer) that inspect Props.
func On(event string, handler func(Event)) Attr {
	return func(p Props) { p["on"+event] = handler }
}

// OnClick registers a simple click handler under the key "onclick".
// For handlers that need event data (coordinates, etc.), use [On] instead.
func OnClick(handler func()) Attr {
	return func(p Props) { p["onclick"] = handler }
}
