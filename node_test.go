package gui

import (
	"testing"
)

// Compile-time interface satisfaction checks.
// If any of these types stop implementing Node, the build will fail with a
// clear error rather than a confusing runtime panic.
var (
	_ Node = (*Element)(nil)
	_ Node = (*TextNode)(nil)
	_ Node = (*Fragment)(nil)
)

func TestEl(t *testing.T) {
	t.Run("creates element with tag props and children", func(t *testing.T) {
		child := Text("hello")
		el := El("div", Props{"class": "container"}, child)

		if el.Tag != "div" {
			t.Errorf("expected tag %q, got %q", "div", el.Tag)
		}
		if el.Props["class"] != "container" {
			t.Errorf("expected prop class=%q, got %v", "container", el.Props["class"])
		}
		if len(el.Children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(el.Children))
		}
		got, ok := el.Children[0].(*TextNode)
		if !ok {
			t.Fatalf("expected *TextNode child, got %T", el.Children[0])
		}
		if got.Content != "hello" {
			t.Errorf("expected child content %q, got %q", "hello", got.Content)
		}
	})

	t.Run("nil props defaults to empty map", func(t *testing.T) {
		el := El("span", nil)
		if el.Props == nil {
			t.Error("expected non-nil Props, got nil")
		}
		// Writing to the map must not panic.
		el.Props["id"] = "test"
		if el.Props["id"] != "test" {
			t.Errorf("expected id=%q after write, got %v", "test", el.Props["id"])
		}
	})

	t.Run("empty element with no attrs and no children is valid", func(t *testing.T) {
		el := El("br", nil)
		if el.Tag != "br" {
			t.Errorf("expected tag %q, got %q", "br", el.Tag)
		}
		if len(el.Children) != 0 {
			t.Errorf("expected 0 children, got %d", len(el.Children))
		}
	})

	t.Run("multiple children are preserved in order", func(t *testing.T) {
		a, b, c := Text("a"), Text("b"), Text("c")
		el := El("ul", nil, a, b, c)
		if len(el.Children) != 3 {
			t.Fatalf("expected 3 children, got %d", len(el.Children))
		}
		for i, want := range []string{"a", "b", "c"} {
			got, ok := el.Children[i].(*TextNode)
			if !ok {
				t.Fatalf("child %d: expected *TextNode, got %T", i, el.Children[i])
			}
			if got.Content != want {
				t.Errorf("child %d: expected %q, got %q", i, want, got.Content)
			}
		}
	})
}

func TestText(t *testing.T) {
	t.Run("creates text node with correct content", func(t *testing.T) {
		tn := Text("hello world")
		if tn.Content != "hello world" {
			t.Errorf("expected %q, got %q", "hello world", tn.Content)
		}
	})

	t.Run("empty string is valid", func(t *testing.T) {
		tn := Text("")
		if tn.Content != "" {
			t.Errorf("expected empty string, got %q", tn.Content)
		}
	})
}

func TestTextf(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []any
		want   string
	}{
		{
			name:   "no format args",
			format: "plain text",
			args:   nil,
			want:   "plain text",
		},
		{
			name:   "single string arg",
			format: "Hello, %s!",
			args:   []any{"world"},
			want:   "Hello, world!",
		},
		{
			name:   "integer arg",
			format: "count: %d",
			args:   []any{42},
			want:   "count: 42",
		},
		{
			name:   "multiple args",
			format: "%s=%v",
			args:   []any{"key", true},
			want:   "key=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tn := Textf(tt.format, tt.args...)
			if tn.Content != tt.want {
				t.Errorf("expected %q, got %q", tt.want, tn.Content)
			}
		})
	}
}

func TestFrag(t *testing.T) {
	t.Run("groups children without wrapper", func(t *testing.T) {
		a, b := Text("a"), Text("b")
		f := Frag(a, b)
		if len(f.Children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(f.Children))
		}
		first, ok := f.Children[0].(*TextNode)
		if !ok {
			t.Fatalf("expected *TextNode, got %T", f.Children[0])
		}
		if first.Content != "a" {
			t.Errorf("expected %q, got %q", "a", first.Content)
		}
	})

	t.Run("empty frag is valid", func(t *testing.T) {
		f := Frag()
		if f == nil {
			t.Fatal("expected non-nil Fragment")
		}
		if len(f.Children) != 0 {
			t.Errorf("expected 0 children, got %d", len(f.Children))
		}
	})

	t.Run("frag satisfies Node", func(t *testing.T) {
		// Store in a Node variable to exercise the interface.
		var n Node = Frag(Text("x"))
		if n == nil {
			t.Error("expected non-nil Node")
		}
	})
}

func TestTag(t *testing.T) {
	t.Run("empty tag call produces correct element", func(t *testing.T) {
		Div := Tag("div")
		el := Div()()
		if el.Tag != "div" {
			t.Errorf("expected tag %q, got %q", "div", el.Tag)
		}
		if len(el.Props) != 0 {
			t.Errorf("expected 0 props, got %d", len(el.Props))
		}
		if len(el.Children) != 0 {
			t.Errorf("expected 0 children, got %d", len(el.Children))
		}
	})

	t.Run("tag with attr and text child", func(t *testing.T) {
		P := Tag("p")
		el := P(Attr_("class", "x"))(Text("hi"))
		if el.Tag != "p" {
			t.Errorf("expected tag %q, got %q", "p", el.Tag)
		}
		if el.Props["class"] != "x" {
			t.Errorf("expected class=%q, got %v", "x", el.Props["class"])
		}
		if len(el.Children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(el.Children))
		}
		got, ok := el.Children[0].(*TextNode)
		if !ok {
			t.Fatalf("expected *TextNode, got %T", el.Children[0])
		}
		if got.Content != "hi" {
			t.Errorf("expected %q, got %q", "hi", got.Content)
		}
	})

	t.Run("multiple attrs merged into props", func(t *testing.T) {
		Btn := Tag("button")
		el := Btn(Attr_("id", "submit"), Attr_("disabled", true))()
		if el.Props["id"] != "submit" {
			t.Errorf("expected id=%q, got %v", "submit", el.Props["id"])
		}
		if el.Props["disabled"] != true {
			t.Errorf("expected disabled=true, got %v", el.Props["disabled"])
		}
	})

	t.Run("nested tag builders produce correct tree", func(t *testing.T) {
		Div := Tag("div")
		Span := Tag("span")
		el := Div()(Span()(Text("nested")))
		if el.Tag != "div" {
			t.Errorf("outer tag: expected %q, got %q", "div", el.Tag)
		}
		if len(el.Children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(el.Children))
		}
		inner, ok := el.Children[0].(*Element)
		if !ok {
			t.Fatalf("expected *Element child, got %T", el.Children[0])
		}
		if inner.Tag != "span" {
			t.Errorf("inner tag: expected %q, got %q", "span", inner.Tag)
		}
		if len(inner.Children) != 1 {
			t.Fatalf("expected 1 inner child, got %d", len(inner.Children))
		}
		txt, ok := inner.Children[0].(*TextNode)
		if !ok {
			t.Fatalf("expected *TextNode inner child, got %T", inner.Children[0])
		}
		if txt.Content != "nested" {
			t.Errorf("expected %q, got %q", "nested", txt.Content)
		}
	})

	t.Run("each call to the builder produces an independent element", func(t *testing.T) {
		Div := Tag("div")
		builder := Div(Attr_("id", "a"))
		el1 := builder(Text("one"))
		el2 := builder(Text("two"))

		if el1 == el2 {
			t.Error("expected distinct Element pointers")
		}
		if len(el1.Children) != 1 || len(el2.Children) != 1 {
			t.Errorf("each element should have exactly 1 child")
		}
	})
}

func TestAttr_(t *testing.T) {
	t.Run("sets arbitrary string key", func(t *testing.T) {
		p := Props{}
		Attr_("data-value", "hello")(p)
		if p["data-value"] != "hello" {
			t.Errorf("expected %q, got %v", "hello", p["data-value"])
		}
	})

	t.Run("sets integer value", func(t *testing.T) {
		p := Props{}
		Attr_("tabindex", 3)(p)
		if p["tabindex"] != 3 {
			t.Errorf("expected 3, got %v", p["tabindex"])
		}
	})

	t.Run("sets bool value", func(t *testing.T) {
		p := Props{}
		Attr_("hidden", false)(p)
		if p["hidden"] != false {
			t.Errorf("expected false, got %v", p["hidden"])
		}
	})

	t.Run("overwrites existing key", func(t *testing.T) {
		p := Props{"class": "old"}
		Attr_("class", "new")(p)
		if p["class"] != "new" {
			t.Errorf("expected %q, got %v", "new", p["class"])
		}
	})
}

func TestValidateTag(t *testing.T) {
	t.Run("valid tags accepted", func(t *testing.T) {
		valid := []string{"div", "h1", "my-component", "x", "A", "custom-elem"}
		for _, tag := range valid {
			ValidateTag(tag) // should not panic
		}
	})

	t.Run("rejects injection attempt", func(t *testing.T) {
		bad := []string{
			"div onload='alert(1)'",
			"div\"><script>",
			"",
			"123",
			"div.class",
			"a b",
			"onclick=",
		}
		for _, tag := range bad {
			t.Run(tag, func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic for tag %q", tag)
					}
				}()
				ValidateTag(tag)
			})
		}
	})
}

func TestElRejectsInvalidTag(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid tag in El()")
		}
	}()
	El("div onclick=alert(1)", nil)
}

func TestTagRejectsInvalidTag(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid tag in Tag()")
		}
	}()
	Tag("div onclick=alert(1)")
}

func TestNestedEl(t *testing.T) {
	t.Run("deeply nested El calls produce correct tree", func(t *testing.T) {
		tree := El("section", Props{"id": "root"},
			El("article", nil,
				El("p", nil, Text("paragraph")),
			),
		)

		if tree.Tag != "section" {
			t.Errorf("expected section, got %q", tree.Tag)
		}
		if tree.Props["id"] != "root" {
			t.Errorf("expected id=root, got %v", tree.Props["id"])
		}
		if len(tree.Children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(tree.Children))
		}

		article, ok := tree.Children[0].(*Element)
		if !ok {
			t.Fatalf("expected *Element, got %T", tree.Children[0])
		}
		if article.Tag != "article" {
			t.Errorf("expected article, got %q", article.Tag)
		}
		if len(article.Children) != 1 {
			t.Fatalf("expected 1 grandchild, got %d", len(article.Children))
		}

		para, ok := article.Children[0].(*Element)
		if !ok {
			t.Fatalf("expected *Element grandchild, got %T", article.Children[0])
		}
		if para.Tag != "p" {
			t.Errorf("expected p, got %q", para.Tag)
		}
		if len(para.Children) != 1 {
			t.Fatalf("expected 1 great-grandchild, got %d", len(para.Children))
		}

		txt, ok := para.Children[0].(*TextNode)
		if !ok {
			t.Fatalf("expected *TextNode, got %T", para.Children[0])
		}
		if txt.Content != "paragraph" {
			t.Errorf("expected %q, got %q", "paragraph", txt.Content)
		}
	})
}
