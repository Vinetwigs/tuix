// Package event defines the TUIX event system: keyboard input,
// focus/navigation actions and custom application events.
package event

import (
	"sync"

	"github.com/gdamore/tcell/v2"
)

// Type classifies an event.
type Type int

const (
	// KeyPress is a raw keyboard event from the backend.
	KeyPress Type = iota
	// Activate fires when a Button is pressed (Enter/Space).
	Activate
	// InputChange fires when an Input value changes.
	InputChange
	// FocusChange fires when keyboard focus moves.
	FocusChange
	// Custom is a user-defined event emitted from Go code.
	Custom
	// Quit requests application shutdown.
	Quit
	// Select fires when a List selection changes (keyboard or mouse).
	Select
)

// Event is a single occurrence in the system.
type Event struct {
	Type     Type
	Key      tcell.Key
	Rune     rune
	Mod      tcell.ModMask
	Name     string // handler name (e.g. onActivate="onOk") or custom name
	TargetID string // component id that originated the event
	Value    string // extra payload (e.g. input content)
}

// Handler handles one event.
type Handler func(ev Event)

// Dispatcher routes named handlers plus a global observer list.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	global   []Handler
}

// NewDispatcher creates an empty dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: map[string][]Handler{}}
}

// On registers a handler for a name. The name matches the markup
// attribute (e.g. <Button onActivate="save"> -> On("save", ...)).
// The special name "*" receives every event.
func (d *Dispatcher) On(name string, h Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if name == "*" {
		d.global = append(d.global, h)
		return
	}
	d.handlers[name] = append(d.handlers[name], h)
}

// Emit dispatches ev to matching handlers and to global observers.
func (d *Dispatcher) Emit(ev Event) {
	d.mu.RLock()
	list := append([]Handler{}, d.handlers[ev.Name]...)
	glob := append([]Handler{}, d.global...)
	d.mu.RUnlock()
	for _, h := range list {
		h(ev)
	}
	for _, h := range glob {
		h(ev)
	}
}

// Has reports whether at least one handler exists for name.
func (d *Dispatcher) Has(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.handlers[name]) > 0
}
