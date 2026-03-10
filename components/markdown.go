// Package components provides reusable, backend-agnostic UI components built
// on top of the gui framework. Components in this package target HTML output
// via gui/html and are designed to be shared across any gui-based application.
package components

import (
	gui "github.com/readmedotmd/gui.md"
	"strings"
)

// MaxMarkdownSize is the maximum input size (in bytes) that RenderMarkdown
// will process. Inputs exceeding this limit are truncated to prevent CPU and
// memory exhaustion from pathologically large documents.
const MaxMarkdownSize = 1 << 20 // 1 MiB

// RenderMarkdown converts a markdown string into a gui Node tree using
// standard HTML elements. Supported syntax:
//
//   - Headings: # H1 through #### H4
//   - Bold: **text** or ***bold+italic***
//   - Italic: *text*
//   - Strikethrough: ~~text~~
//   - Inline code: `code`
//   - Fenced code blocks: ```lang ... ```
//   - Unordered lists: lines starting with "- ", "* ", or "+ "
//   - Ordered lists: lines starting with "1. ", "2. ", etc.
//   - Blockquotes: lines starting with "> "
//   - Horizontal rules: "---", "***", or "___"
//   - Links: [text](url)   (opens in a new tab)
//   - Images: ![alt](url)  (data:, http:, and https: URLs accepted)
//   - Tables: GFM-style pipe tables with optional header separator row
func RenderMarkdown(text string) gui.Node {
	if len(text) > MaxMarkdownSize {
		text = text[:MaxMarkdownSize]
	}
	lines := strings.Split(text, "\n")
	var nodes []gui.Node

	inCodeBlock := false
	var codeLines []string

	inUL := false
	inOL := false
	var listItems []gui.Node

	inTable := false
	var tableRows [][]string
	tableHasHeader := false

	// Accumulate consecutive plain-text lines into a single paragraph.
	var paraLines []string
	flushPara := func() {
		if len(paraLines) > 0 {
			nodes = append(nodes, gui.P()(RenderInline(strings.Join(paraLines, " "))))
			paraLines = nil
		}
	}

	flushList := func() {
		if inUL {
			nodes = append(nodes, gui.Ul()(listItems...))
			listItems = nil
			inUL = false
		} else if inOL {
			nodes = append(nodes, gui.Ol()(listItems...))
			listItems = nil
			inOL = false
		}
	}

	flushTable := func() {
		if !inTable {
			return
		}
		inTable = false
		if len(tableRows) == 0 {
			tableRows = nil
			tableHasHeader = false
			return
		}
		var tableChildren []gui.Node
		if tableHasHeader {
			var cells []gui.Node
			for _, cell := range tableRows[0] {
				cells = append(cells, gui.Th()(RenderInline(cell)))
			}
			tableChildren = append(tableChildren, gui.Thead()(gui.Tr()(cells...)))
			tableRows = tableRows[1:]
		}
		if len(tableRows) > 0 {
			var bodyRows []gui.Node
			for _, row := range tableRows {
				var cells []gui.Node
				for _, cell := range row {
					cells = append(cells, gui.Td()(RenderInline(cell)))
				}
				bodyRows = append(bodyRows, gui.Tr()(cells...))
			}
			tableChildren = append(tableChildren, gui.Tbody()(bodyRows...))
		}
		nodes = append(nodes, gui.Table(gui.Class("md-table"))(tableChildren...))
		tableRows = nil
		tableHasHeader = false
	}

	for _, line := range lines {
		// Fenced code block
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				nodes = append(nodes, gui.Pre()(gui.Code()(gui.Text(strings.Join(codeLines, "\n")))))
				codeLines = nil
				inCodeBlock = false
			} else {
				flushPara()
				flushList()
				flushTable()
				inCodeBlock = true
			}
			continue
		}
		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Image
		if imgNode := tryParseImage(line); imgNode != nil {
			flushPara()
			flushList()
			flushTable()
			nodes = append(nodes, imgNode)
			continue
		}

		// Horizontal rule
		if trimmed == "---" || trimmed == "***" || trimmed == "___" ||
			trimmed == "- - -" || trimmed == "* * *" || trimmed == "_ _ _" {
			flushPara()
			flushList()
			flushTable()
			nodes = append(nodes, gui.Hr()())
			continue
		}

		// Headings (check longest prefix first)
		if strings.HasPrefix(line, "#### ") {
			flushPara()
			flushList()
			flushTable()
			nodes = append(nodes, gui.H4()(RenderInline(strings.TrimPrefix(line, "#### "))))
			continue
		}
		if strings.HasPrefix(line, "### ") {
			flushPara()
			flushList()
			flushTable()
			nodes = append(nodes, gui.H3()(RenderInline(strings.TrimPrefix(line, "### "))))
			continue
		}
		if strings.HasPrefix(line, "## ") {
			flushPara()
			flushList()
			flushTable()
			nodes = append(nodes, gui.H2()(RenderInline(strings.TrimPrefix(line, "## "))))
			continue
		}
		if strings.HasPrefix(line, "# ") {
			flushPara()
			flushList()
			flushTable()
			nodes = append(nodes, gui.H1()(RenderInline(strings.TrimPrefix(line, "# "))))
			continue
		}

		// Blockquote
		if strings.HasPrefix(line, "> ") || line == ">" {
			flushPara()
			flushList()
			flushTable()
			content := strings.TrimPrefix(strings.TrimPrefix(line, "> "), ">")
			nodes = append(nodes, gui.Blockquote()(RenderInline(content)))
			continue
		}

		// Unordered list: "- ", "+ ", "* "
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "+ ") || strings.HasPrefix(line, "* ") {
			flushPara()
			flushTable()
			if inOL {
				nodes = append(nodes, gui.Ol()(listItems...))
				listItems = nil
				inOL = false
			}
			inUL = true
			listItems = append(listItems, gui.Li()(RenderInline(line[2:])))
			continue
		}

		// Ordered list: "1. ", "2. ", etc.
		if olContent, ok := parseOLItem(line); ok {
			flushPara()
			flushTable()
			if inUL {
				nodes = append(nodes, gui.Ul()(listItems...))
				listItems = nil
				inUL = false
			}
			inOL = true
			listItems = append(listItems, gui.Li()(RenderInline(olContent)))
			continue
		}

		// Table row
		if strings.HasPrefix(trimmed, "|") && strings.Contains(line, "|") {
			flushPara()
			flushList()
			if isTableSeparator(trimmed) {
				tableHasHeader = true
				continue
			}
			inTable = true
			if cols := parseTableRow(trimmed); len(cols) > 0 {
				tableRows = append(tableRows, cols)
			}
			continue
		} else if inTable {
			flushPara()
			flushTable()
		}

		// Empty line — flush any open list/table/paragraph
		if line == "" {
			flushPara()
			flushList()
			flushTable()
			continue
		}

		// Accumulate plain text into the current paragraph
		flushList()
		flushTable()
		paraLines = append(paraLines, trimmed)
	}

	flushPara()
	flushList()
	if inCodeBlock && len(codeLines) > 0 {
		nodes = append(nodes, gui.Pre()(gui.Code()(gui.Text(strings.Join(codeLines, "\n")))))
	}
	flushTable()

	return gui.Frag(nodes...)
}

// RenderInline processes inline markdown spans within a single line.
// Handles bold+italic (***), bold (**), italic (*), strikethrough (~~),
// inline code (`), and links ([text](url)).
// Inline spans may be nested (e.g. bold text inside a list item).
func RenderInline(text string) gui.Node {
	type marker struct{ s, typ string }
	// Priority: longer/higher-precedence markers come first.
	markers := []marker{
		{"***", "bolditalic"},
		{"**", "bold"},
		{"~~", "strike"},
		{"`", "code"},
		{"[", "link"},
		{"*", "italic"},
	}

	var nodes []gui.Node
	remaining := text

	for len(remaining) > 0 {
		earliest := -1
		var chosen marker

		for _, m := range markers {
			if idx := strings.Index(remaining, m.s); idx >= 0 && (earliest < 0 || idx < earliest) {
				earliest = idx
				chosen = m
			}
		}

		if earliest < 0 {
			nodes = append(nodes, gui.Text(remaining))
			break
		}
		if earliest > 0 {
			nodes = append(nodes, gui.Text(remaining[:earliest]))
		}
		remaining = remaining[earliest+len(chosen.s):]

		switch chosen.typ {
		case "code":
			if end := strings.Index(remaining, "`"); end < 0 {
				nodes = append(nodes, gui.Text("`"))
			} else {
				nodes = append(nodes, gui.Code()(gui.Text(remaining[:end])))
				remaining = remaining[end+1:]
			}
		case "bolditalic":
			if end := strings.Index(remaining, "***"); end < 0 {
				nodes = append(nodes, gui.Text("***"))
			} else {
				nodes = append(nodes, gui.Strong()(gui.Em()(RenderInline(remaining[:end]))))
				remaining = remaining[end+3:]
			}
		case "bold":
			if end := strings.Index(remaining, "**"); end < 0 {
				nodes = append(nodes, gui.Text("**"))
			} else {
				nodes = append(nodes, gui.Strong()(RenderInline(remaining[:end])))
				remaining = remaining[end+2:]
			}
		case "italic":
			if end := strings.Index(remaining, "*"); end < 0 {
				nodes = append(nodes, gui.Text("*"))
			} else {
				nodes = append(nodes, gui.Em()(RenderInline(remaining[:end])))
				remaining = remaining[end+1:]
			}
		case "strike":
			if end := strings.Index(remaining, "~~"); end < 0 {
				nodes = append(nodes, gui.Text("~~"))
			} else {
				nodes = append(nodes, gui.Span(gui.Style("text-decoration:line-through"))(RenderInline(remaining[:end])))
				remaining = remaining[end+2:]
			}
		case "link":
			closeBracket := strings.Index(remaining, "]")
			if closeBracket < 0 || !strings.HasPrefix(remaining[closeBracket:], "](") {
				nodes = append(nodes, gui.Text("["))
			} else {
				linkText := remaining[:closeBracket]
				remaining = remaining[closeBracket+2:]
				if closeParen := strings.Index(remaining, ")"); closeParen < 0 {
					nodes = append(nodes, gui.Text("["+linkText+"]("))
				} else {
					url := remaining[:closeParen]
					remaining = remaining[closeParen+1:]
					if isSafeURL(url) {
						nodes = append(nodes, gui.A(
							gui.Href(url),
							gui.Attr_("target", "_blank"),
							gui.Attr_("rel", "noopener noreferrer"),
						)(RenderInline(linkText)))
					} else {
						// Unsafe scheme — render as plain text.
						nodes = append(nodes, gui.Text("["+linkText+"]("+url+")"))
					}
				}
			}
		}
	}

	return gui.Frag(nodes...)
}

// parseOLItem checks if line is an ordered-list item ("1. text", "2. text", …).
// Returns the item content and true on match.
func parseOLItem(line string) (string, bool) {
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch >= '0' && ch <= '9' {
			continue
		}
		if i > 0 && ch == '.' && i+1 < len(line) && line[i+1] == ' ' {
			return line[i+2:], true
		}
		break
	}
	return "", false
}

// parseTableRow splits a GFM table row on | delimiters.
func parseTableRow(line string) []string {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return result
}

// isTableSeparator returns true when line is a GFM table header separator
// (e.g. "|---|:---:|---|").
func isTableSeparator(line string) bool {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	if len(parts) == 0 {
		return false
	}
	for _, p := range parts {
		for _, ch := range strings.TrimSpace(p) {
			if ch != '-' && ch != ':' {
				return false
			}
		}
	}
	return true
}

// isSafeURL reports whether url uses an allowed scheme for links.
// Only http:, https:, mailto:, and relative URLs (no colon before first slash)
// are considered safe. This prevents javascript: and other dangerous schemes.
func isSafeURL(url string) bool {
	lower := strings.ToLower(url)
	if strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") {
		return true
	}
	// Relative URLs: no colon before first slash (or no colon at all).
	colon := strings.Index(lower, ":")
	if colon < 0 {
		return true // no scheme at all — relative
	}
	slash := strings.Index(lower, "/")
	return slash >= 0 && slash < colon
}

// tryParseImage parses a markdown image line (![alt](url)).
// Accepts data:, http:, and https: URL schemes.
func tryParseImage(line string) gui.Node {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "![") {
		return nil
	}
	closeBracket := strings.Index(trimmed, "](")
	if closeBracket < 0 || !strings.HasSuffix(trimmed, ")") {
		return nil
	}
	url := trimmed[closeBracket+2 : len(trimmed)-1]
	if !strings.HasPrefix(url, "data:image/") &&
		!strings.HasPrefix(url, "https://") &&
		!strings.HasPrefix(url, "http://") {
		return nil
	}
	alt := trimmed[2:closeBracket]
	return gui.Img(gui.Src(url), gui.Alt(alt), gui.Class("md-image"))()
}
