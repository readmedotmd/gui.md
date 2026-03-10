package components_test

import (
	"strings"
	"testing"

	gui "github.com/readmedotmd/gui.md"
	"github.com/readmedotmd/gui.md/components"
	"github.com/readmedotmd/gui.md/html"
)

func render(node gui.Node) string {
	return html.New().RenderString(node)
}

// ---- RenderMarkdown ---------------------------------------------------------

func TestHeadings(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"# H1", "<h1>H1</h1>"},
		{"## H2", "<h2>H2</h2>"},
		{"### H3", "<h3>H3</h3>"},
		{"#### H4", "<h4>H4</h4>"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := render(components.RenderMarkdown(tt.input))
			if got != tt.want {
				t.Errorf("\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestParagraph(t *testing.T) {
	got := render(components.RenderMarkdown("Hello world"))
	if !strings.Contains(got, "<p>Hello world</p>") {
		t.Errorf("expected paragraph, got: %s", got)
	}
}

func TestMultiLineParagraph(t *testing.T) {
	got := render(components.RenderMarkdown("line one\nline two"))
	if !strings.Contains(got, "<p>line one line two</p>") {
		t.Errorf("expected merged paragraph, got: %s", got)
	}
}

func TestBold(t *testing.T) {
	got := render(components.RenderInline("**bold**"))
	if !strings.Contains(got, "<strong>bold</strong>") {
		t.Errorf("expected bold, got: %s", got)
	}
}

func TestItalic(t *testing.T) {
	got := render(components.RenderInline("*italic*"))
	if !strings.Contains(got, "<em>italic</em>") {
		t.Errorf("expected italic, got: %s", got)
	}
}

func TestBoldItalic(t *testing.T) {
	got := render(components.RenderInline("***both***"))
	if !strings.Contains(got, "<strong><em>both</em></strong>") {
		t.Errorf("expected bold+italic, got: %s", got)
	}
}

func TestStrikethrough(t *testing.T) {
	got := render(components.RenderInline("~~strike~~"))
	if !strings.Contains(got, "line-through") {
		t.Errorf("expected strikethrough, got: %s", got)
	}
}

func TestInlineCode(t *testing.T) {
	got := render(components.RenderInline("`code`"))
	if !strings.Contains(got, "<code>code</code>") {
		t.Errorf("expected inline code, got: %s", got)
	}
}

func TestLink(t *testing.T) {
	got := render(components.RenderInline("[text](https://example.com)"))
	if !strings.Contains(got, `href="https://example.com"`) {
		t.Errorf("expected link, got: %s", got)
	}
	if !strings.Contains(got, "text") {
		t.Errorf("expected link text, got: %s", got)
	}
	if !strings.Contains(got, `target="_blank"`) {
		t.Errorf("expected target=_blank, got: %s", got)
	}
}

func TestFencedCodeBlock(t *testing.T) {
	md := "```go\nfmt.Println(\"hi\")\n```"
	got := render(components.RenderMarkdown(md))
	if !strings.Contains(got, "<pre>") || !strings.Contains(got, "<code>") {
		t.Errorf("expected code block, got: %s", got)
	}
}

func TestUnorderedList(t *testing.T) {
	md := "- item1\n- item2\n- item3"
	got := render(components.RenderMarkdown(md))
	if !strings.Contains(got, "<ul>") || !strings.Contains(got, "<li>") {
		t.Errorf("expected unordered list, got: %s", got)
	}
	if strings.Count(got, "<li>") != 3 {
		t.Errorf("expected 3 list items, got: %s", got)
	}
}

func TestOrderedList(t *testing.T) {
	md := "1. first\n2. second"
	got := render(components.RenderMarkdown(md))
	if !strings.Contains(got, "<ol>") || !strings.Contains(got, "<li>") {
		t.Errorf("expected ordered list, got: %s", got)
	}
}

func TestBlockquote(t *testing.T) {
	got := render(components.RenderMarkdown("> quoted text"))
	if !strings.Contains(got, "<blockquote>") {
		t.Errorf("expected blockquote, got: %s", got)
	}
}

func TestHorizontalRule(t *testing.T) {
	for _, sep := range []string{"---", "***", "___"} {
		t.Run(sep, func(t *testing.T) {
			got := render(components.RenderMarkdown(sep))
			if !strings.Contains(got, "<hr />") {
				t.Errorf("expected hr for %q, got: %s", sep, got)
			}
		})
	}
}

func TestImage(t *testing.T) {
	md := "![alt text](https://example.com/img.png)"
	got := render(components.RenderMarkdown(md))
	if !strings.Contains(got, `src="https://example.com/img.png"`) {
		t.Errorf("expected img src, got: %s", got)
	}
	if !strings.Contains(got, `alt="alt text"`) {
		t.Errorf("expected alt text, got: %s", got)
	}
}

func TestImageRejectsUnsafeScheme(t *testing.T) {
	md := "![x](javascript:alert(1))"
	got := render(components.RenderMarkdown(md))
	// The unsafe URL should not appear as an href or src attribute.
	if strings.Contains(got, `href="javascript:`) || strings.Contains(got, `src="javascript:`) {
		t.Errorf("should not render javascript: URLs as href/src, got: %s", got)
	}
}

func TestLinkRejectsJavascriptScheme(t *testing.T) {
	got := render(components.RenderInline("[click](javascript:alert(1))"))
	if strings.Contains(got, `href="javascript:`) {
		t.Errorf("should not render javascript: URLs as href, got: %s", got)
	}
}

func TestTable(t *testing.T) {
	md := "| Name | Age |\n|------|-----|\n| Alice | 30 |"
	got := render(components.RenderMarkdown(md))
	if !strings.Contains(got, "<table") {
		t.Errorf("expected table, got: %s", got)
	}
	if !strings.Contains(got, "<thead>") {
		t.Errorf("expected thead, got: %s", got)
	}
	if !strings.Contains(got, "<th>") {
		t.Errorf("expected th, got: %s", got)
	}
	if !strings.Contains(got, "<td>") {
		t.Errorf("expected td, got: %s", got)
	}
}

func TestEmptyInput(t *testing.T) {
	got := render(components.RenderMarkdown(""))
	if got != "" {
		t.Errorf("expected empty output for empty input, got: %s", got)
	}
}

func TestUnclosedCodeBlock(t *testing.T) {
	md := "```\nunclosed code"
	got := render(components.RenderMarkdown(md))
	if !strings.Contains(got, "<pre>") {
		t.Errorf("expected unclosed code block to flush, got: %s", got)
	}
}

func TestMixedContent(t *testing.T) {
	md := "# Title\n\nParagraph text.\n\n- list item\n\n> quote"
	got := render(components.RenderMarkdown(md))
	if !strings.Contains(got, "<h1>") {
		t.Errorf("missing heading in mixed content")
	}
	if !strings.Contains(got, "<p>") {
		t.Errorf("missing paragraph in mixed content")
	}
	if !strings.Contains(got, "<ul>") {
		t.Errorf("missing list in mixed content")
	}
	if !strings.Contains(got, "<blockquote>") {
		t.Errorf("missing blockquote in mixed content")
	}
}

func TestMaxMarkdownSize(t *testing.T) {
	// Input larger than MaxMarkdownSize should be truncated, not crash.
	large := strings.Repeat("x", components.MaxMarkdownSize+100)
	got := render(components.RenderMarkdown(large))
	// Should produce output without panicking.
	if len(got) == 0 {
		t.Error("expected non-empty output for large input")
	}
}

// ---- RenderInline edge cases ------------------------------------------------

func TestUnmatchedMarkers(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unmatched bold", "**open"},
		{"unmatched italic", "*open"},
		{"unmatched code", "`open"},
		{"unmatched strike", "~~open"},
		{"unmatched link bracket", "[text"},
		{"unmatched link paren", "[text](url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic.
			got := render(components.RenderInline(tt.input))
			if len(got) == 0 {
				t.Error("expected non-empty output for unmatched marker")
			}
		})
	}
}

func TestPlainText(t *testing.T) {
	got := render(components.RenderInline("just plain text"))
	if got != "just plain text" {
		t.Errorf("expected plain text passthrough, got: %s", got)
	}
}

func TestNestedInline(t *testing.T) {
	got := render(components.RenderInline("**bold with *italic* inside**"))
	if !strings.Contains(got, "<strong>") && !strings.Contains(got, "<em>") {
		t.Errorf("expected nested inline formatting, got: %s", got)
	}
}

// ---- parseOLItem ------------------------------------------------------------

func TestOrderedListVariousNumbers(t *testing.T) {
	md := "1. first\n10. tenth\n99. ninety-nine"
	got := render(components.RenderMarkdown(md))
	if strings.Count(got, "<li>") != 3 {
		t.Errorf("expected 3 OL items, got: %s", got)
	}
}

// ---- parseTableRow / isTableSeparator ---------------------------------------

func TestTableSeparatorVariants(t *testing.T) {
	md := "| A |\n|:---:|\n| B |"
	got := render(components.RenderMarkdown(md))
	if !strings.Contains(got, "<thead>") {
		t.Errorf("expected header with alignment separator, got: %s", got)
	}
}

func TestListMarkerVariants(t *testing.T) {
	for _, prefix := range []string{"- ", "+ ", "* "} {
		t.Run(prefix, func(t *testing.T) {
			md := prefix + "item"
			got := render(components.RenderMarkdown(md))
			if !strings.Contains(got, "<ul>") {
				t.Errorf("expected UL for prefix %q, got: %s", prefix, got)
			}
		})
	}
}

func TestBlockquoteEmptyLine(t *testing.T) {
	got := render(components.RenderMarkdown(">"))
	if !strings.Contains(got, "<blockquote>") {
		t.Errorf("expected blockquote for bare >, got: %s", got)
	}
}

func TestDataImageURL(t *testing.T) {
	md := "![pic](data:image/png;base64,abc123)"
	got := render(components.RenderMarkdown(md))
	if !strings.Contains(got, "data:image/png") {
		t.Errorf("expected data: image URL, got: %s", got)
	}
}
