// Package backend is the tcell rendering backend of TUIX.
//
// It owns the terminal screen and exposes minimal drawing primitives
// used by the App render loop: text runs and bordered boxes. Layout
// rectangles come from the layout package so markup, runtime and
// rendering keep evolving independently.
package backend

import (
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/Vinetwigs/tuix/layout"
)

// Screen wraps a tcell screen with TUIX drawing helpers.
type Screen struct {
	t tcell.Screen
}

// New creates and initializes the real terminal screen.
func New() (*Screen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	s.Clear()
	return &Screen{t: s}, nil
}

// Tcell returns the underlying tcell screen for advanced use.
func (s *Screen) Tcell() tcell.Screen { return s.t }

// Size returns current screen dimensions in cells.
func (s *Screen) Size() (int, int) { return s.t.Size() }

// Clear wipes the screen buffer.
func (s *Screen) Clear() { s.t.Clear() }

// Show flushes the buffer to the terminal.
func (s *Screen) Show() { s.t.Show() }

// Fini restores the terminal.
func (s *Screen) Fini() { s.t.Fini() }

// Poll blocks until the next terminal event.
func (s *Screen) Poll() tcell.Event { return s.t.PollEvent() }

// EnableMouse turns on mouse event reporting (clicks, drags, wheel).
// Call it after New; pair with DisableMouse or Fini on exit.
func (s *Screen) EnableMouse() { s.t.EnableMouse() }

// DisableMouse turns off mouse event reporting.
func (s *Screen) DisableMouse() { s.t.DisableMouse() }

// wakeupEvent unblocks a pending Poll so the loop redraws. Unknown
// events are ignored by the App run loop by design.
type wakeupEvent struct{ at time.Time }

func (w wakeupEvent) When() time.Time { return w.at }

// Wakeup unblocks a pending Poll so the application can redraw.
// It is goroutine-safe: use it together with App.Notify to refresh
// the UI from background goroutines (timers, file watchers, ...).
func (s *Screen) Wakeup() { _ = s.t.PostEvent(wakeupEvent{at: time.Now()}) }

// SetCell writes one rune cell.
func (s *Screen) SetCell(x, y int, r rune, st tcell.Style) {
	s.t.SetContent(x, y, r, nil, st)
}

// DrawText writes text clipped to width w starting at (x, y).
// Short text is padded with spaces up to w when fill is true.
func (s *Screen) DrawText(x, y, w int, text string, st tcell.Style, fill bool) {
	runes := []rune(text)
	for i := 0; i < w; i++ {
		r := ' '
		if i < len(runes) {
			r = runes[i]
		} else if !fill {
			break
		}
		s.t.SetContent(x+i, y, r, nil, st)
	}
}

// DrawBox draws a bordered box with an optional title.
func (s *Screen) DrawBox(r layout.Rect, title string, st tcell.Style) {
	if r.W < 2 || r.H < 2 {
		return
	}
	// Corners and edges (single-line box drawing).
	s.t.SetContent(r.X, r.Y, '+', nil, st)
	s.t.SetContent(r.Right()-1, r.Y, '+', nil, st)
	s.t.SetContent(r.X, r.Bottom()-1, '+', nil, st)
	s.t.SetContent(r.Right()-1, r.Bottom()-1, '+', nil, st)
	for x := r.X + 1; x < r.Right()-1; x++ {
		s.t.SetContent(x, r.Y, '-', nil, st)
		s.t.SetContent(x, r.Bottom()-1, '-', nil, st)
	}
	for y := r.Y + 1; y < r.Bottom()-1; y++ {
		s.t.SetContent(r.X, y, '|', nil, st)
		s.t.SetContent(r.Right()-1, y, '|', nil, st)
	}
	if title != "" {
		label := " " + title + " "
		runes := []rune(label)
		for i, ch := range runes {
			x := r.X + 1 + i
			if x >= r.Right()-1 {
				break
			}
			s.t.SetContent(x, r.Y, ch, nil, st)
		}
	}
}

// DrawHLine draws a horizontal filler line of spaces (clears a row span).
func (s *Screen) DrawHLine(x, y, w int, st tcell.Style) {
	for i := 0; i < w; i++ {
		s.t.SetContent(x+i, y, ' ', nil, st)
	}
}
