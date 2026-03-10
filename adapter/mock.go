package adapter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Mock is a full-featured mock that implements Adapter and all optional interfaces.
// Use it in tests to validate adapter-consuming code without a real backend.
type Mock struct {
	mu              sync.Mutex
	status          AdapterStatus
	config          AdapterConfig
	events          chan StreamEvent
	messages        []Message
	conversations   []Conversation
	sessionID       string
	statusCallbacks []func(AdapterStatus)
	permissionCh    chan permissionDecision
	started         bool
	cancelled       bool
	healthy         bool
	StartErr        error
	SendErr         error
	HealthErr       error
}

type permissionDecision struct {
	toolCallID string
	approved   bool
}

// NewMock creates a new Mock adapter in the idle state.
func NewMock() *Mock {
	return &Mock{
		status:       StatusIdle,
		events:       make(chan StreamEvent, 256),
		permissionCh: make(chan permissionDecision, 16),
		healthy:      true,
		sessionID:    "session-001",
	}
}

// --- Core Adapter interface ---

func (m *Mock) Start(ctx context.Context, cfg AdapterConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.StartErr != nil {
		return m.StartErr
	}

	select {
	case <-ctx.Done():
		return &AdapterError{Code: ErrTimeout, Message: "start cancelled", Err: ctx.Err()}
	default:
	}

	m.config = cfg
	m.started = true
	m.setStatusLocked(StatusRunning)
	return nil
}

func (m *Mock) Send(ctx context.Context, msg Message, opts ...SendOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SendErr != nil {
		return m.SendErr
	}
	if m.status != StatusRunning {
		return &AdapterError{Code: ErrUnknown, Message: "adapter not running"}
	}

	select {
	case <-ctx.Done():
		return &AdapterError{Code: ErrTimeout, Message: "send cancelled", Err: ctx.Err()}
	default:
	}

	var sendOpts SendOptions
	for _, opt := range opts {
		opt(&sendOpts)
	}

	m.messages = append(m.messages, msg)
	return nil
}

func (m *Mock) Cancel() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelled = true
	return nil
}

func (m *Mock) Receive() <-chan StreamEvent {
	return m.events
}

func (m *Mock) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == StatusStopped {
		return nil
	}

	m.setStatusLocked(StatusStopped)
	close(m.events)
	return nil
}

func (m *Mock) Status() AdapterStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *Mock) Capabilities() AdapterCapabilities {
	return AdapterCapabilities{
		SupportsStreaming:    true,
		SupportsImages:       true,
		SupportsFiles:        true,
		SupportsToolUse:      true,
		SupportsMCP:          true,
		SupportsThinking:     true,
		SupportsCancellation: true,
		SupportsHistory:      true,
		SupportsSubAgents:    true,
		MaxContextWindow:     200000,
		SupportedModels:      []string{"model-a", "model-b"},
	}
}

func (m *Mock) Health(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.HealthErr != nil {
		return m.HealthErr
	}
	if !m.healthy {
		return &AdapterError{Code: ErrCrashed, Message: "adapter process died"}
	}
	return nil
}

// --- SessionProvider ---

func (m *Mock) SessionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessionID
}

// --- HistoryClearer ---

func (m *Mock) ClearHistory(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
	return nil
}

// --- HistoryProvider ---

func (m *Mock) GetHistory(ctx context.Context) ([]Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]Message, len(m.messages))
	copy(cp, m.messages)
	return cp, nil
}

// --- ConversationManager ---

func (m *Mock) ListConversations(ctx context.Context) ([]Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]Conversation, len(m.conversations))
	copy(cp, m.conversations)
	return cp, nil
}

func (m *Mock) ResumeConversation(ctx context.Context, conversationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.conversations {
		if c.ID == conversationID {
			m.messages = make([]Message, len(c.Messages))
			copy(m.messages, c.Messages)
			return nil
		}
	}
	return fmt.Errorf("conversation %q not found", conversationID)
}

// --- PermissionResponder ---

func (m *Mock) RespondPermission(ctx context.Context, toolCallID string, approved bool) error {
	select {
	case m.permissionCh <- permissionDecision{toolCallID: toolCallID, approved: approved}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// --- StatusListener ---

func (m *Mock) OnStatusChange(fn func(AdapterStatus)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCallbacks = append(m.statusCallbacks, fn)
}

// --- Test helpers ---

// Emit sends an event to the Receive channel. If Timestamp is zero, it is
// set to time.Now(). This is exported so tests can simulate adapter output.
func (m *Mock) Emit(ev StreamEvent) {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}
	m.events <- ev
}

// Config returns the config passed to the last Start call.
func (m *Mock) Config() AdapterConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.config
}

// Cancelled reports whether Cancel was called.
func (m *Mock) Cancelled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cancelled
}

// SetHealthy sets the health state of the mock.
func (m *Mock) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

// SetSessionID sets the session ID returned by SessionID().
func (m *Mock) SetSessionID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionID = id
}

// SetConversations sets the conversations returned by ListConversations.
func (m *Mock) SetConversations(convos []Conversation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conversations = convos
}

// PermissionCh returns the internal permission channel for test assertions.
func (m *Mock) PermissionCh() <-chan permissionDecision {
	return m.permissionCh
}

// --- internal ---

func (m *Mock) setStatusLocked(s AdapterStatus) {
	m.status = s
	for _, fn := range m.statusCallbacks {
		fn(s)
	}
}
