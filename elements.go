package gui

// Block elements — structural containers for page layout.
var (
	Div     = Tag("div")
	Section = Tag("section")
	Article = Tag("article")
	Header  = Tag("header")
	Footer  = Tag("footer")
	Main    = Tag("main")
	Nav     = Tag("nav")
	Aside   = Tag("aside")
)

// Headings — six levels of section headings.
var (
	H1 = Tag("h1")
	H2 = Tag("h2")
	H3 = Tag("h3")
	H4 = Tag("h4")
	H5 = Tag("h5")
	H6 = Tag("h6")
)

// Inline elements — text-level semantics and links.
var (
	Span   = Tag("span")
	A      = Tag("a")
	Strong = Tag("strong")
	Em     = Tag("em")
	I      = Tag("i")
	Code   = Tag("code")
	Pre    = Tag("pre")
)

// Text content — paragraphs, quotations, and lists.
var (
	P          = Tag("p")
	Blockquote = Tag("blockquote")
	Ul         = Tag("ul")
	Ol         = Tag("ol")
	Li         = Tag("li")
)

// Form elements — interactive controls and containers.
var (
	Form     = Tag("form")
	Label    = Tag("label")
	Button   = Tag("button")
	Select   = Tag("select")
	Option   = Tag("option")
	Textarea = Tag("textarea")
	Input    = Tag("input")
)

// Void / self-closing elements — rendered without a closing tag.
// These elements may not have child nodes; children are silently ignored by
// the HTML renderer when the element tag is in the void set.
var (
	Br   = Tag("br")
	Hr   = Tag("hr")
	Img  = Tag("img")
	Meta = Tag("meta")
	Link = Tag("link")
)

// Interactive elements — disclosure widgets.
var (
	Details = Tag("details")
	Summary = Tag("summary")
)

// Table elements — tabular data presentation.
var (
	Table = Tag("table")
	Thead = Tag("thead")
	Tbody = Tag("tbody")
	Tr    = Tag("tr")
	Th    = Tag("th")
	Td    = Tag("td")
)

// Document-level elements — full HTML document structure.
var (
	Html    = Tag("html")
	Head    = Tag("head")
	Body    = Tag("body")
	Title   = Tag("title")
	Script  = Tag("script")
	StyleEl = Tag("style")
)
