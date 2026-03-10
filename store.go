package gui

import "sync"

// Store is a generic, Zustand-like global state container.
// T is the state type — typically a struct defined by the user.
// It is safe for concurrent use.
//
// Example:
//
//	type AppState struct {
//	    Count int
//	    Name  string
//	}
//	store := gui.NewStore(AppState{Count: 0, Name: "World"})
type Store[T any] struct {
	mu        sync.RWMutex
	state     T
	prevState T
	listeners map[int]func(state, prevState T)
	nextID    int
}

// NewStore creates a new Store with the given initial state.
func NewStore[T any](initial T) *Store[T] {
	return &Store[T]{
		state:     initial,
		listeners: make(map[int]func(state, prevState T)),
	}
}

// Get returns the current state. It is safe to call concurrently.
func (s *Store[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Set replaces the entire state and notifies all subscribers with the new
// state and the previous state. Notifications are delivered outside of the
// store's lock, so subscribers may safely call Get, Set, or Update without
// deadlocking.
func (s *Store[T]) Set(newState T) {
	s.mu.Lock()
	s.prevState = s.state
	s.state = newState
	cur := s.state
	prev := s.prevState
	fns := make([]func(state, prevState T), 0, len(s.listeners))
	for _, fn := range s.listeners {
		fns = append(fns, fn)
	}
	s.mu.Unlock()

	for _, fn := range fns {
		fn(cur, prev)
	}
}

// Update applies fn to the current state, stores the result, and notifies
// all subscribers. It is the ergonomic way to make partial updates without
// replacing fields that should remain unchanged.
//
// Example:
//
//	store.Update(func(s AppState) AppState {
//	    s.Count++
//	    return s
//	})
func (s *Store[T]) Update(fn func(T) T) {
	s.mu.Lock()
	s.prevState = s.state
	s.state = fn(s.state)
	cur := s.state
	prev := s.prevState
	fns := make([]func(state, prevState T), 0, len(s.listeners))
	for _, lfn := range s.listeners {
		fns = append(fns, lfn)
	}
	s.mu.Unlock()

	for _, lfn := range fns {
		lfn(cur, prev)
	}
}

// Subscribe registers fn as a listener that is called after every state
// change, receiving both the new state and the previous state. It returns an
// unsubscribe function that removes the listener; calling the unsubscribe
// function more than once is safe and has no effect.
func (s *Store[T]) Subscribe(fn func(state, prevState T)) func() {
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	s.listeners[id] = fn
	s.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.listeners, id)
			s.mu.Unlock()
		})
	}
}
