package adapter

import (
	"testing"
	"time"
)

func TestRoleValues(t *testing.T) {
	tests := []struct {
		role Role
		want string
	}{
		{RoleUser, "user"},
		{RoleAssistant, "assistant"},
		{RoleSystem, "system"},
		{RoleTool, "tool"},
	}
	for _, tc := range tests {
		if string(tc.role) != tc.want {
			t.Errorf("Role %q: expected %q", tc.role, tc.want)
		}
	}
}

func TestContentTypeValues(t *testing.T) {
	tests := []struct {
		ct   ContentType
		want string
	}{
		{ContentText, "text"},
		{ContentCode, "code"},
		{ContentImage, "image"},
		{ContentFile, "file"},
		{ContentToolUse, "tool_use"},
		{ContentToolResult, "tool_result"},
	}
	for _, tc := range tests {
		if string(tc.ct) != tc.want {
			t.Errorf("ContentType %q: expected %q", tc.ct, tc.want)
		}
	}
}

func TestTextContent(t *testing.T) {
	blocks := TextContent("hello world")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Type != ContentText {
		t.Errorf("Type: got %q", blocks[0].Type)
	}
	if blocks[0].Text != "hello world" {
		t.Errorf("Text: got %q", blocks[0].Text)
	}
}

func TestTextContentEmpty(t *testing.T) {
	blocks := TextContent("")
	if len(blocks) != 1 || blocks[0].Text != "" {
		t.Errorf("unexpected: %+v", blocks)
	}
}

func TestContentBlockVariants(t *testing.T) {
	// Code
	b := ContentBlock{Type: ContentCode, Text: "fmt.Println()", Language: "go"}
	if b.Language != "go" {
		t.Errorf("Language: got %q", b.Language)
	}

	// Image
	b = ContentBlock{Type: ContentImage, Data: []byte{0x89, 0x50}, MimeType: "image/png"}
	if b.MimeType != "image/png" || len(b.Data) != 2 {
		t.Errorf("Image: %+v", b)
	}

	// File
	b = ContentBlock{Type: ContentFile, Data: []byte("key: value"), MimeType: "application/yaml"}
	if b.Type != ContentFile {
		t.Errorf("Type: got %q", b.Type)
	}

	// ToolUse
	tc := &ToolCall{ID: "tc-1", Name: "Read", Input: map[string]any{"file": "/tmp"}, Status: "running"}
	b = ContentBlock{Type: ContentToolUse, ToolCall: tc}
	if b.ToolCall.Name != "Read" {
		t.Errorf("ToolCall.Name: got %q", b.ToolCall.Name)
	}

	// ToolResult
	tc = &ToolCall{ID: "tc-1", Name: "Read", Output: "contents", Status: "complete"}
	b = ContentBlock{Type: ContentToolResult, ToolCall: tc}
	if b.ToolCall.Status != "complete" {
		t.Errorf("ToolCall.Status: got %q", b.ToolCall.Status)
	}
}

func TestMessageSimple(t *testing.T) {
	now := time.Now()
	msg := Message{
		ID:        "msg-1",
		Role:      RoleUser,
		Content:   TextContent("hello"),
		Timestamp: now,
		Metadata:  map[string]string{"source": "web"},
	}

	if msg.ID != "msg-1" || msg.Role != RoleUser {
		t.Errorf("unexpected: %+v", msg)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(msg.Content))
	}
	if msg.Metadata["source"] != "web" {
		t.Errorf("Metadata: got %v", msg.Metadata)
	}
}

func TestMessageMultiModal(t *testing.T) {
	msg := Message{
		ID:   "msg-2",
		Role: RoleUser,
		Content: []ContentBlock{
			{Type: ContentText, Text: "What is this?"},
			{Type: ContentImage, Data: []byte{0xFF}, MimeType: "image/jpeg"},
			{Type: ContentFile, Data: []byte("data"), MimeType: "text/csv"},
		},
	}
	if len(msg.Content) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(msg.Content))
	}
}

func TestMessageWithToolCalls(t *testing.T) {
	msg := Message{
		ID:   "msg-3",
		Role: RoleAssistant,
		Content: []ContentBlock{
			{Type: ContentText, Text: "Let me read that file."},
			{Type: ContentToolUse, ToolCall: &ToolCall{ID: "tc-1", Name: "Read", Status: "running"}},
			{Type: ContentToolUse, ToolCall: &ToolCall{ID: "tc-2", Name: "Grep", Status: "running"}},
		},
	}

	toolCalls := 0
	for _, b := range msg.Content {
		if b.Type == ContentToolUse {
			toolCalls++
		}
	}
	if toolCalls != 2 {
		t.Errorf("expected 2, got %d", toolCalls)
	}
}

func TestConversation(t *testing.T) {
	now := time.Now()
	conv := Conversation{
		ID:      "conv-1",
		Adapter: "claude-code",
		Title:   "Fix login bug",
		Messages: []Message{
			{ID: "m1", Role: RoleUser, Content: TextContent("fix it"), Timestamp: now},
			{ID: "m2", Role: RoleAssistant, Content: TextContent("done"), Timestamp: now.Add(time.Second)},
		},
		CreatedAt: now,
		UpdatedAt: now.Add(time.Second),
		Metadata:  map[string]string{"branch": "fix/login"},
	}

	if conv.ID != "conv-1" || conv.Title != "Fix login bug" {
		t.Errorf("unexpected: %+v", conv)
	}
	if len(conv.Messages) != 2 {
		t.Errorf("Messages: got %d", len(conv.Messages))
	}
	if conv.Metadata["branch"] != "fix/login" {
		t.Errorf("Metadata: got %v", conv.Metadata)
	}
}

func TestToolCallFields(t *testing.T) {
	tc := ToolCall{
		ID:     "tc-99",
		Name:   "Bash",
		Input:  map[string]any{"command": "ls"},
		Output: "file1\nfile2\n",
		Status: "complete",
	}

	if tc.ID != "tc-99" || tc.Name != "Bash" || tc.Status != "complete" {
		t.Errorf("unexpected: %+v", tc)
	}
	input, ok := tc.Input.(map[string]any)
	if !ok {
		t.Fatalf("Input type: got %T", tc.Input)
	}
	if input["command"] != "ls" {
		t.Errorf("Input[command]: got %v", input["command"])
	}
}
