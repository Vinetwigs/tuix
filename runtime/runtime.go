// Package runtime implements the reactive state of TUIX apps.
//
// The Store holds string key/value pairs. UI text can embed bindings
// with the {key} syntax; every Set notifies subscribers so the UI can
// re-render automatically. Derived values are supported through
// Computed: a computed key is re-evaluated from other keys on each
// mutation and behaves like a normal key on Get/Render.
package runtime

import (
	"sort"
	"strings"
	"sync"
)

// Store is a thread-safe reactive key/value state container.
type Store struct {
	mu        sync.RWMutex
	vals      map[string]string
	computeds map[string]func(get func(string) string) string
	watchers  []func(key, val string)
	version   uint64
	changed   chan struct{}
}

// New creates a store pre-filled with initial values (may be nil).
func New(initial map[string]string) *Store {
	s := &Store{
		vals:      map[string]string{},
		computeds: map[string]func(func(string) string) string{},
		changed:   make(chan struct{}, 1),
	}
	for k, v := range initial {
		s.vals[k] = v
	}
	return s
}

// Get returns the value for key, evaluating computed keys on the fly.
// Unknown keys return "".
func (s *Store) Get(key string) string {
	s.mu.RLock()
	fn, isComputed := s.computeds[key]
	vals := make(map[string]string, len(s.vals))
	for k, v := range s.vals {
		vals[k] = v
	}
	// Snapshot computed funcs to evaluate without holding the lock.
	compCopy := make(map[string]func(func(string) string) string, len(s.computeds))
	for k, f := range s.computeds {
		compCopy[k] = f
	}
	s.mu.RUnlock()

	if isComputed {
		get := func(k string) string {
			if v, ok := vals[k]; ok {
				return v
			}
			if cf, ok := compCopy[k]; ok && k != key {
				return cf(func(inner string) string { return vals[inner] })
			}
			return ""
		}
		return fn(get)
	}
	return vals[key]
}

// Set assigns val to key, refreshes nothing eagerly (computed keys are
// lazy), bumps the version and notifies watchers/subscribers.
func (s *Store) Set(key, val string) {
	s.mu.Lock()
	if old, ok := s.vals[key]; ok && old == val {
		s.mu.Unlock()
		return
	}
	// A manual Set on a computed key converts it back to a plain key.
	delete(s.computeds, key)
	s.vals[key] = val
	s.version++
	watchers := append([]func(string, string){}, s.watchers...)
	s.mu.Unlock()

	for _, w := range watchers {
		w(key, val)
	}
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

// Computed registers a derived value: fn receives a get function and
// returns the derived string. It is re-evaluated on every Get/Render,
// so it always reflects current state.
func (s *Store) Computed(key string, fn func(get func(string) string) string) {
	s.mu.Lock()
	s.computeds[key] = fn
	s.version++
	s.mu.Unlock()
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

// OnChange registers a callback invoked synchronously on every Set.
func (s *Store) OnChange(fn func(key, val string)) {
	s.mu.Lock()
	s.watchers = append(s.watchers, fn)
	s.mu.Unlock()
}

// Changed returns a channel signaled (non-blocking, coalesced) after
// each mutation. Useful for render loops.
func (s *Store) Changed() <-chan struct{} {
	return s.changed
}

// Version returns a monotonically increasing counter bumped on mutation.
func (s *Store) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// Snapshot returns a copy of plain (non-computed) values.
func (s *Store) Snapshot() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.vals))
	for k, v := range s.vals {
		out[k] = v
	}
	return out
}

// Keys returns sorted known keys (plain + computed).
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	set := map[string]bool{}
	for k := range s.vals {
		set[k] = true
	}
	for k := range s.computeds {
		set[k] = true
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Render interpolates {key} placeholders in text using current values.
// Unknown keys render as "". Use "{{" and "}}" to escape braces.
func (s *Store) Render(text string) string {
	var b strings.Builder
	i := 0
	for i < len(text) {
		c := text[i]
		if c == '{' {
			if i+1 < len(text) && text[i+1] == '{' {
				b.WriteByte('{')
				i += 2
				continue
			}
			end := strings.IndexByte(text[i:], '}')
			if end < 0 {
				b.WriteString(text[i:])
				break
			}
			key := strings.TrimSpace(text[i+1 : i+end])
			b.WriteString(s.Get(key))
			i += end + 1
			continue
		}
		if c == '}' && i+1 < len(text) && text[i+1] == '}' {
			b.WriteByte('}')
			i += 2
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}
