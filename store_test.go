package gui

import (
	"sync"
	"sync/atomic"
	"testing"
)

// testState is the shared state type used across all Store tests.
type testState struct {
	Count int
	Name  string
	Items []string
}

// TestNewStore_InitialState verifies that Get returns the exact state passed
// to NewStore.
func TestNewStore_InitialState(t *testing.T) {
	initial := testState{Count: 42, Name: "hello", Items: []string{"a", "b"}}
	s := NewStore(initial)

	got := s.Get()
	if got.Count != initial.Count {
		t.Errorf("Count: got %d, want %d", got.Count, initial.Count)
	}
	if got.Name != initial.Name {
		t.Errorf("Name: got %q, want %q", got.Name, initial.Name)
	}
	if len(got.Items) != len(initial.Items) {
		t.Errorf("Items length: got %d, want %d", len(got.Items), len(initial.Items))
	}
}

// TestSet_ReplacesState verifies that Set fully replaces the stored state.
func TestSet_ReplacesState(t *testing.T) {
	s := NewStore(testState{Count: 1, Name: "old"})

	next := testState{Count: 99, Name: "new", Items: []string{"x"}}
	s.Set(next)

	got := s.Get()
	if got.Count != 99 {
		t.Errorf("Count: got %d, want 99", got.Count)
	}
	if got.Name != "new" {
		t.Errorf("Name: got %q, want %q", got.Name, "new")
	}
	if len(got.Items) != 1 || got.Items[0] != "x" {
		t.Errorf("Items: got %v, want [x]", got.Items)
	}
}

// TestUpdate_AppliesFunction verifies that Update increments a field without
// touching others.
func TestUpdate_AppliesFunction(t *testing.T) {
	s := NewStore(testState{Count: 10, Name: "keep"})

	s.Update(func(st testState) testState {
		st.Count++
		return st
	})

	got := s.Get()
	if got.Count != 11 {
		t.Errorf("Count: got %d, want 11", got.Count)
	}
	if got.Name != "keep" {
		t.Errorf("Name should be unchanged; got %q", got.Name)
	}
}

// TestUpdate_ReceivesCurrentState ensures the function passed to Update sees
// the state that was current at the time of the call.
func TestUpdate_ReceivesCurrentState(t *testing.T) {
	s := NewStore(testState{Count: 5})

	var seen int
	s.Update(func(st testState) testState {
		seen = st.Count
		return st
	})

	if seen != 5 {
		t.Errorf("Update fn received Count=%d, want 5", seen)
	}
}

// TestSet_NotifiesSubscribers verifies that a subscriber is called with the
// new state and the previous state after Set.
func TestSet_NotifiesSubscribers(t *testing.T) {
	s := NewStore(testState{Count: 1})

	var (
		gotNew  testState
		gotPrev testState
		called  int
	)
	s.Subscribe(func(state, prevState testState) {
		called++
		gotNew = state
		gotPrev = prevState
	})

	s.Set(testState{Count: 2, Name: "after"})

	if called != 1 {
		t.Fatalf("subscriber called %d times, want 1", called)
	}
	if gotNew.Count != 2 {
		t.Errorf("new state Count: got %d, want 2", gotNew.Count)
	}
	if gotPrev.Count != 1 {
		t.Errorf("prev state Count: got %d, want 1", gotPrev.Count)
	}
}

// TestUpdate_NotifiesSubscribers verifies that subscribers receive the correct
// new and previous states after Update.
func TestUpdate_NotifiesSubscribers(t *testing.T) {
	s := NewStore(testState{Count: 3})

	var gotNew, gotPrev testState
	s.Subscribe(func(state, prevState testState) {
		gotNew = state
		gotPrev = prevState
	})

	s.Update(func(st testState) testState {
		st.Count += 7
		return st
	})

	if gotNew.Count != 10 {
		t.Errorf("new Count: got %d, want 10", gotNew.Count)
	}
	if gotPrev.Count != 3 {
		t.Errorf("prev Count: got %d, want 3", gotPrev.Count)
	}
}

// TestSubscribe_Unsubscribe verifies that calling the returned unsubscribe
// function prevents future notifications.
func TestSubscribe_Unsubscribe(t *testing.T) {
	s := NewStore(testState{})

	var calls int
	unsub := s.Subscribe(func(_, _ testState) { calls++ })

	s.Set(testState{Count: 1}) // should notify
	unsub()
	s.Set(testState{Count: 2}) // should NOT notify
	s.Set(testState{Count: 3}) // should NOT notify

	if calls != 1 {
		t.Errorf("subscriber called %d times after unsubscribe, want 1", calls)
	}
}

// TestSubscribe_MultipleSubscribers verifies that all registered subscribers
// receive notifications.
func TestSubscribe_MultipleSubscribers(t *testing.T) {
	s := NewStore(testState{})

	const n = 5
	counts := make([]int, n)
	for i := range counts {
		i := i
		s.Subscribe(func(_, _ testState) { counts[i]++ })
	}

	s.Set(testState{Count: 1})

	for i, c := range counts {
		if c != 1 {
			t.Errorf("subscriber %d called %d times, want 1", i, c)
		}
	}
}

// TestSubscriber_ReceivesCorrectPrevState verifies the prev state accumulates
// correctly across multiple Set calls.
func TestSubscriber_ReceivesCorrectPrevState(t *testing.T) {
	s := NewStore(testState{Count: 0})

	type pair struct{ new, prev int }
	var pairs []pair

	s.Subscribe(func(state, prevState testState) {
		pairs = append(pairs, pair{state.Count, prevState.Count})
	})

	s.Set(testState{Count: 1})
	s.Set(testState{Count: 2})
	s.Set(testState{Count: 3})

	want := []pair{{1, 0}, {2, 1}, {3, 2}}
	if len(pairs) != len(want) {
		t.Fatalf("got %d notifications, want %d", len(pairs), len(want))
	}
	for i, p := range pairs {
		if p != want[i] {
			t.Errorf("notification %d: got %+v, want %+v", i, p, want[i])
		}
	}
}

// TestConcurrentAccess exercises concurrent Get, Set, and Update from many
// goroutines. The race detector will surface any data races.
func TestConcurrentAccess(t *testing.T) {
	s := NewStore(testState{Count: 0})

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Readers.
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = s.Get()
		}()
	}

	// Writers via Set.
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			s.Set(testState{Count: 1})
		}()
	}

	// Writers via Update.
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			s.Update(func(st testState) testState {
				st.Count++
				return st
			})
		}()
	}

	wg.Wait()
}

// TestSubscriberCallsUpdateNoDeadlock verifies that a subscriber can safely
// call Update (or Set) without deadlocking, because notifications are
// delivered outside the store's lock.
func TestSubscriberCallsUpdateNoDeadlock(t *testing.T) {
	s := NewStore(testState{Count: 0})

	var fired atomic.Bool
	s.Subscribe(func(state, _ testState) {
		if state.Count == 1 && !fired.Swap(true) {
			// Calling Update from inside a subscriber must not deadlock.
			s.Update(func(st testState) testState {
				st.Count = 100
				return st
			})
		}
	})

	s.Set(testState{Count: 1})

	got := s.Get()
	if got.Count != 100 {
		t.Errorf("Count after nested Update: got %d, want 100", got.Count)
	}
}

// TestUpdate_SliceField verifies that Update can safely append to a slice
// field without corrupting state.
func TestUpdate_SliceField(t *testing.T) {
	s := NewStore(testState{Items: []string{"a"}})

	s.Update(func(st testState) testState {
		// Append to a copy to avoid sharing the underlying array.
		items := make([]string, len(st.Items), len(st.Items)+1)
		copy(items, st.Items)
		st.Items = append(items, "b")
		return st
	})

	got := s.Get()
	if len(got.Items) != 2 || got.Items[0] != "a" || got.Items[1] != "b" {
		t.Errorf("Items: got %v, want [a b]", got.Items)
	}
}

// TestZeroValueState verifies that NewStore accepts a zero-value state and
// that Get, Set, and Update all work correctly from that baseline.
func TestZeroValueState(t *testing.T) {
	s := NewStore(testState{})

	got := s.Get()
	if got.Count != 0 || got.Name != "" || got.Items != nil {
		t.Errorf("unexpected zero state: %+v", got)
	}

	s.Set(testState{Count: 7})
	if s.Get().Count != 7 {
		t.Errorf("Count after Set: got %d, want 7", s.Get().Count)
	}

	s.Update(func(st testState) testState {
		st.Name = "zero-start"
		return st
	})
	if s.Get().Name != "zero-start" {
		t.Errorf("Name after Update: got %q", s.Get().Name)
	}
}

// TestUnsubscribe_Idempotent verifies that calling the unsubscribe function
// multiple times does not panic or produce unexpected side effects.
func TestUnsubscribe_Idempotent(t *testing.T) {
	s := NewStore(testState{})

	var calls int
	unsub := s.Subscribe(func(_, _ testState) { calls++ })

	unsub()
	unsub() // second call must not panic
	unsub() // third call must not panic

	s.Set(testState{Count: 1})

	if calls != 0 {
		t.Errorf("subscriber called %d times after unsubscribe, want 0", calls)
	}
}
