package gui

import "testing"

func TestEventConstruction(t *testing.T) {
	e := Event{
		Type: "click",
		X:    10,
		Y:    20,
	}
	if e.Type != "click" {
		t.Errorf("expected Type=%q, got %q", "click", e.Type)
	}
	if e.X != 10 || e.Y != 20 {
		t.Errorf("expected X=10 Y=20, got X=%d Y=%d", e.X, e.Y)
	}
}

func TestEventKeyboard(t *testing.T) {
	e := Event{
		Type: "keypress",
		Key:  "Enter",
	}
	if e.Key != "Enter" {
		t.Errorf("expected Key=%q, got %q", "Enter", e.Key)
	}
}

func TestEventHandlerTypeAlias(t *testing.T) {
	// Verify that a func(Event) literal satisfies EventHandler without conversion.
	var handler EventHandler = func(e Event) {
		_ = e.Type
	}
	handler(Event{Type: "test"})
}

func TestEventInput(t *testing.T) {
	e := Event{
		Type:  "input",
		Value: "hello",
	}
	if e.Value != "hello" {
		t.Errorf("expected Value=%q, got %q", "hello", e.Value)
	}
}
