package adapter

import (
	"errors"
	"testing"
	"time"
)

func TestStreamEventTypeValues(t *testing.T) {
	types := []struct {
		typ  StreamEventType
		want int
	}{
		{EventToken, 0},
		{EventDone, 1},
		{EventError, 2},
		{EventToolUse, 3},
		{EventToolResult, 4},
		{EventSystem, 5},
		{EventThinking, 6},
		{EventPermissionRequest, 7},
		{EventPermissionResult, 8},
		{EventProgress, 9},
		{EventFileChange, 10},
		{EventSubAgent, 11},
		{EventCostUpdate, 12},
	}
	for _, tc := range types {
		if int(tc.typ) != tc.want {
			t.Errorf("StreamEventType %d: expected %d", tc.typ, tc.want)
		}
	}
}

func TestFileChangeOpValues(t *testing.T) {
	tests := []struct {
		op   FileChangeOp
		want string
	}{
		{FileCreated, "created"},
		{FileEdited, "edited"},
		{FileDeleted, "deleted"},
		{FileRenamed, "renamed"},
	}
	for _, tc := range tests {
		if string(tc.op) != tc.want {
			t.Errorf("FileChangeOp %q: expected %q", tc.op, tc.want)
		}
	}
}

func TestStreamTokenEvents(t *testing.T) {
	m := NewMock()
	tokens := []string{"Hello", " ", "world", "!"}
	go func() {
		for _, tok := range tokens {
			m.Emit(StreamEvent{Type: EventToken, Token: tok})
		}
		m.Emit(StreamEvent{Type: EventDone})
	}()

	var collected []string
	for ev := range m.Receive() {
		if ev.Type == EventDone {
			break
		}
		if ev.Type == EventToken {
			collected = append(collected, ev.Token)
		}
	}

	if len(collected) != len(tokens) {
		t.Fatalf("expected %d tokens, got %d", len(tokens), len(collected))
	}
	for i, tok := range tokens {
		if collected[i] != tok {
			t.Errorf("token %d: got %q, want %q", i, collected[i], tok)
		}
	}
}

func TestStreamThinkingEvents(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{Type: EventThinking, Thinking: "Let me consider..."})
		m.Emit(StreamEvent{Type: EventThinking, Thinking: "I should check."})
		m.Emit(StreamEvent{Type: EventToken, Token: "Here's what I found."})
		m.Emit(StreamEvent{Type: EventDone})
	}()

	var thinking, tokens int
	for ev := range m.Receive() {
		switch ev.Type {
		case EventThinking:
			thinking++
		case EventToken:
			tokens++
		case EventDone:
			goto done
		}
	}
done:
	if thinking != 2 {
		t.Errorf("expected 2 thinking, got %d", thinking)
	}
	if tokens != 1 {
		t.Errorf("expected 1 token, got %d", tokens)
	}
}

func TestStreamToolUseAndResult(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{
			Type: EventToolUse, ToolCallID: "tc-1", ToolName: "Read",
			ToolInput: map[string]any{"file_path": "/tmp/foo"}, ToolStatus: "running",
		})
		m.Emit(StreamEvent{
			Type: EventToolResult, ToolCallID: "tc-1",
			ToolOutput: "file contents", ToolStatus: "complete",
		})
		m.Emit(StreamEvent{Type: EventDone})
	}()

	var toolUse, toolResult *StreamEvent
	for ev := range m.Receive() {
		switch ev.Type {
		case EventToolUse:
			cp := ev
			toolUse = &cp
		case EventToolResult:
			cp := ev
			toolResult = &cp
		case EventDone:
			goto done
		}
	}
done:
	if toolUse == nil || toolResult == nil {
		t.Fatal("missing tool events")
	}
	if toolUse.ToolCallID != toolResult.ToolCallID {
		t.Error("ToolCallID mismatch")
	}
	if toolUse.ToolName != "Read" {
		t.Errorf("ToolName: got %q", toolUse.ToolName)
	}
}

func TestStreamParallelToolCalls(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{Type: EventToolUse, ToolCallID: "tc-1", ToolName: "Grep", ToolStatus: "running"})
		m.Emit(StreamEvent{Type: EventToolUse, ToolCallID: "tc-2", ToolName: "Read", ToolStatus: "running"})
		m.Emit(StreamEvent{Type: EventToolResult, ToolCallID: "tc-2", ToolStatus: "complete"})
		m.Emit(StreamEvent{Type: EventToolResult, ToolCallID: "tc-1", ToolStatus: "complete"})
		m.Emit(StreamEvent{Type: EventDone})
	}()

	inflight := map[string]bool{}
	for ev := range m.Receive() {
		switch ev.Type {
		case EventToolUse:
			inflight[ev.ToolCallID] = true
		case EventToolResult:
			if !inflight[ev.ToolCallID] {
				t.Errorf("result for unknown tool call %q", ev.ToolCallID)
			}
			delete(inflight, ev.ToolCallID)
		case EventDone:
			goto done
		}
	}
done:
	if len(inflight) != 0 {
		t.Errorf("unclosed tool calls: %v", inflight)
	}
}

func TestStreamPermissionRequest(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{
			Type: EventPermissionRequest,
			Permission: &PermissionRequest{
				ToolCallID: "tc-5", ToolName: "Bash",
				ToolInput:   map[string]any{"command": "rm -rf /tmp/stuff"},
				Description: "Delete /tmp/stuff recursively",
			},
		})
		m.Emit(StreamEvent{Type: EventDone})
	}()

	var perm *PermissionRequest
	for ev := range m.Receive() {
		if ev.Type == EventPermissionRequest {
			perm = ev.Permission
		}
		if ev.Type == EventDone {
			break
		}
	}

	if perm == nil {
		t.Fatal("missing permission request")
	}
	if perm.ToolCallID != "tc-5" || perm.ToolName != "Bash" {
		t.Errorf("unexpected: %+v", perm)
	}
	if perm.Description != "Delete /tmp/stuff recursively" {
		t.Errorf("Description: got %q", perm.Description)
	}
}

func TestStreamFileChangeEvents(t *testing.T) {
	tests := []struct {
		op      FileChangeOp
		path    string
		oldPath string
	}{
		{FileCreated, "/tmp/new.go", ""},
		{FileEdited, "/tmp/main.go", ""},
		{FileDeleted, "/tmp/old.go", ""},
		{FileRenamed, "/tmp/new_name.go", "/tmp/old_name.go"},
	}

	for _, tc := range tests {
		m := NewMock()
		go func() {
			m.Emit(StreamEvent{
				Type:       EventFileChange,
				FileChange: &FileChange{Op: tc.op, Path: tc.path, OldPath: tc.oldPath},
			})
			m.Emit(StreamEvent{Type: EventDone})
		}()

		var fc *FileChange
		for ev := range m.Receive() {
			if ev.Type == EventFileChange {
				fc = ev.FileChange
			}
			if ev.Type == EventDone {
				break
			}
		}

		if fc == nil {
			t.Fatalf("missing file change for %s", tc.op)
		}
		if fc.Op != tc.op {
			t.Errorf("Op: got %q, want %q", fc.Op, tc.op)
		}
		if fc.Path != tc.path {
			t.Errorf("Path: got %q", fc.Path)
		}
		if tc.oldPath != "" && fc.OldPath != tc.oldPath {
			t.Errorf("OldPath: got %q", fc.OldPath)
		}
	}
}

func TestStreamSubAgentLifecycle(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{
			Type: EventSubAgent,
			SubAgent: &SubAgentEvent{
				AgentID: "sa-1", AgentName: "researcher",
				Status: "started", Prompt: "Find all TODOs",
			},
		})
		m.Emit(StreamEvent{
			Type: EventSubAgent,
			SubAgent: &SubAgentEvent{
				AgentID: "sa-1", AgentName: "researcher",
				Status: "completed", Result: "Found 5 TODOs",
			},
		})
		m.Emit(StreamEvent{Type: EventDone})
	}()

	var events []*SubAgentEvent
	for ev := range m.Receive() {
		if ev.Type == EventSubAgent {
			cp := *ev.SubAgent
			events = append(events, &cp)
		}
		if ev.Type == EventDone {
			break
		}
	}

	if len(events) != 2 {
		t.Fatalf("expected 2, got %d", len(events))
	}
	if events[0].Status != "started" || events[1].Status != "completed" {
		t.Errorf("statuses: %q, %q", events[0].Status, events[1].Status)
	}
	if events[0].AgentID != events[1].AgentID {
		t.Error("AgentID should match")
	}
}

func TestStreamCostUpdate(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{
			Type: EventCostUpdate,
			Usage: &TokenUsage{
				InputTokens: 1500, OutputTokens: 300,
				CacheRead: 200, CacheWrite: 50, TotalCost: 0.0087,
			},
		})
		m.Emit(StreamEvent{Type: EventDone})
	}()

	var usage *TokenUsage
	for ev := range m.Receive() {
		if ev.Type == EventCostUpdate {
			usage = ev.Usage
		}
		if ev.Type == EventDone {
			break
		}
	}

	if usage == nil {
		t.Fatal("missing cost update")
	}
	if usage.InputTokens != 1500 || usage.OutputTokens != 300 {
		t.Errorf("tokens: %d/%d", usage.InputTokens, usage.OutputTokens)
	}
	if usage.TotalCost != 0.0087 {
		t.Errorf("cost: %f", usage.TotalCost)
	}
}

func TestStreamProgressEvents(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{Type: EventProgress, ProgressPct: 0.25, ProgressMsg: "Scanning..."})
		m.Emit(StreamEvent{Type: EventProgress, ProgressPct: 0.75, ProgressMsg: "Analyzing..."})
		m.Emit(StreamEvent{Type: EventProgress, ProgressPct: 1.0, ProgressMsg: "Done"})
		m.Emit(StreamEvent{Type: EventDone})
	}()

	var pcts []float64
	for ev := range m.Receive() {
		if ev.Type == EventProgress {
			pcts = append(pcts, ev.ProgressPct)
		}
		if ev.Type == EventDone {
			break
		}
	}

	if len(pcts) != 3 || pcts[0] != 0.25 || pcts[2] != 1.0 {
		t.Errorf("pcts: %v", pcts)
	}
}

func TestStreamProgressIndeterminate(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{Type: EventProgress, ProgressPct: -1, ProgressMsg: "Working..."})
		m.Emit(StreamEvent{Type: EventDone})
	}()

	ev := <-m.Receive()
	if ev.ProgressPct != -1 {
		t.Errorf("expected -1, got %f", ev.ProgressPct)
	}
}

func TestStreamErrorEvent(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{
			Type:  EventError,
			Error: &AdapterError{Code: ErrRateLimited, Message: "429 Too Many Requests"},
		})
	}()

	ev := <-m.Receive()
	if ev.Type != EventError {
		t.Fatalf("expected EventError, got %d", ev.Type)
	}
	var ae *AdapterError
	if !errors.As(ev.Error, &ae) || ae.Code != ErrRateLimited {
		t.Fatalf("expected ErrRateLimited, got %v", ev.Error)
	}
}

func TestStreamEventsHaveTimestamps(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{Type: EventToken, Token: "hi"})
		m.Emit(StreamEvent{Type: EventDone})
	}()

	ev := <-m.Receive()
	if ev.Timestamp.IsZero() {
		t.Error("Emit should set a timestamp")
	}
}

func TestStreamEventPreservesExplicitTimestamp(t *testing.T) {
	m := NewMock()
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	go func() {
		m.Emit(StreamEvent{Type: EventToken, Token: "hi", Timestamp: ts})
	}()

	ev := <-m.Receive()
	if !ev.Timestamp.Equal(ts) {
		t.Errorf("expected %v, got %v", ts, ev.Timestamp)
	}
}

func TestStreamFullTurnSequence(t *testing.T) {
	m := NewMock()
	go func() {
		m.Emit(StreamEvent{Type: EventThinking, Thinking: "I need to read the file"})
		m.Emit(StreamEvent{Type: EventPermissionRequest, Permission: &PermissionRequest{ToolCallID: "tc-1", ToolName: "Read", Description: "Read /tmp/main.go"}})
		m.Emit(StreamEvent{Type: EventToolUse, ToolCallID: "tc-1", ToolName: "Read", ToolStatus: "running"})
		m.Emit(StreamEvent{Type: EventProgress, ProgressPct: 0.5, ProgressMsg: "Reading..."})
		m.Emit(StreamEvent{Type: EventToolResult, ToolCallID: "tc-1", ToolOutput: "package main", ToolStatus: "complete"})
		m.Emit(StreamEvent{Type: EventFileChange, FileChange: &FileChange{Op: FileEdited, Path: "/tmp/main.go"}})
		m.Emit(StreamEvent{Type: EventToken, Token: "I've updated "})
		m.Emit(StreamEvent{Type: EventToken, Token: "the file."})
		m.Emit(StreamEvent{Type: EventCostUpdate, Usage: &TokenUsage{InputTokens: 100, OutputTokens: 20, TotalCost: 0.001}})
		m.Emit(StreamEvent{Type: EventDone})
	}()

	var eventTypes []StreamEventType
	for ev := range m.Receive() {
		eventTypes = append(eventTypes, ev.Type)
		if ev.Type == EventDone {
			break
		}
	}

	expected := []StreamEventType{
		EventThinking, EventPermissionRequest, EventToolUse, EventProgress,
		EventToolResult, EventFileChange, EventToken, EventToken,
		EventCostUpdate, EventDone,
	}

	if len(eventTypes) != len(expected) {
		t.Fatalf("expected %d events, got %d", len(expected), len(eventTypes))
	}
	for i, et := range expected {
		if eventTypes[i] != et {
			t.Errorf("event %d: expected %d, got %d", i, et, eventTypes[i])
		}
	}
}

func TestTokenUsageZeroValues(t *testing.T) {
	u := TokenUsage{}
	if u.InputTokens != 0 || u.OutputTokens != 0 || u.TotalCost != 0 {
		t.Errorf("zero value should be all zeros: %+v", u)
	}
}

func TestPermissionRequestFields(t *testing.T) {
	pr := PermissionRequest{
		ToolCallID: "tc-1", ToolName: "Write",
		ToolInput:   map[string]any{"file_path": "/x"},
		Description: "Write to /x",
	}
	if pr.ToolCallID != "tc-1" || pr.ToolName != "Write" || pr.Description != "Write to /x" {
		t.Errorf("unexpected: %+v", pr)
	}
}

func TestSubAgentEventFields(t *testing.T) {
	sa := SubAgentEvent{AgentID: "sa-1", AgentName: "researcher", Status: "started", Prompt: "find bugs"}
	if sa.AgentID != "sa-1" || sa.Status != "started" || sa.Prompt != "find bugs" {
		t.Errorf("unexpected: %+v", sa)
	}
}

func TestFileChangeFields(t *testing.T) {
	fc := FileChange{Op: FileRenamed, Path: "/new", OldPath: "/old"}
	if fc.Op != FileRenamed || fc.Path != "/new" || fc.OldPath != "/old" {
		t.Errorf("unexpected: %+v", fc)
	}
}
