package tuix

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gdamore/tcell/v2"

	"github.com/Vinetwigs/tuix/ast"
	"github.com/Vinetwigs/tuix/backend"
	"github.com/Vinetwigs/tuix/component"
	"github.com/Vinetwigs/tuix/event"
	"github.com/Vinetwigs/tuix/layout"
	"github.com/Vinetwigs/tuix/parser"
	"github.com/Vinetwigs/tuix/runtime"
)

// App is a loaded TUIX application: component tree + reactive store +
// event dispatcher + focus state.
type App struct {
	root     *component.Component
	title    string
	store    *runtime.Store
	events   *event.Dispatcher
	focus    []*component.Component
	focusIdx int
	quit     bool
	screen   atomic.Pointer[backend.Screen]
}

// LoadString parses .tuix markup from a string and builds the App.
func LoadString(src string) (*App, error) {
	root, err := parser.ParseString(src)
	if err != nil {
		return nil, err
	}
	return fromAST(root)
}

// LoadFile parses a .tuix file (with <Include> support) and builds the App.
func LoadFile(path string) (*App, error) {
	root, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return fromAST(root)
}

// fromAST extracts <State> seeds, builds the component tree and wires
// store, dispatcher and focus.
func fromAST(root *ast.Node) (*App, error) {
	seeds := map[string]string{}
	var ui []*ast.Node
	for _, c := range root.Children {
		if component.NormalizedType(c.Name) == "State" {
			k := c.Attr("key", c.Attr("name", ""))
			if k == "" {
				return nil, fmt.Errorf("tuix: line %d: <State> without key attribute", c.Line)
			}
			v := c.Attr("value", "")
			if v == "" {
				v = c.Text
			}
			seeds[k] = v
			continue
		}
		ui = append(ui, c)
	}
	filtered := &ast.Node{Name: root.Name, Attrs: root.Attrs, Text: root.Text, Children: ui}

	tree, err := component.Build(filtered)
	if err != nil {
		return nil, err
	}
	app := &App{
		root:   tree,
		title:  root.Attr("title", ""),
		store:  runtime.New(seeds),
		events: event.NewDispatcher(),
	}
	// Seed Input bindings from value="..." when the key is still empty.
	component.Walk(tree, func(c *component.Component) {
		if c.Type == component.TypeInput {
			if k := c.BindKey(); k != "" {
				if app.store.Get(k) == "" {
					if v, ok := c.Attrs["value"]; ok && v != "" {
						app.store.Set(k, v)
					}
				}
			}
		}
	})
	app.focus = component.Focusables(tree)
	app.focusIdx = 0
	return app, nil
}

// Title returns the <App title="..."> value.
func (a *App) Title() string { return a.title }

// Tree returns the root component (introspection, tests, custom render).
func (a *App) Tree() *component.Component { return a.root }

// Store exposes the reactive state store.
func (a *App) Store() *runtime.Store { return a.store }

// Events exposes the event dispatcher.
func (a *App) Events() *event.Dispatcher { return a.events }

// Set assigns a state value and triggers UI refresh.
func (a *App) Set(key, val string) { a.store.Set(key, val) }

// Get reads a state value (computed keys included).
func (a *App) Get(key string) string { return a.store.Get(key) }

// Computed registers a derived value. See runtime.Store.Computed.
func (a *App) Computed(key string, fn func(get func(string) string) string) {
	a.store.Computed(key, fn)
}

// On registers an event handler by name.
func (a *App) On(name string, h event.Handler) { a.events.On(name, h) }

// Emit dispatches a custom event.
func (a *App) Emit(ev event.Event) { a.events.Emit(ev) }

// Quit requests the Run loop to stop.
func (a *App) Quit() { a.quit = true }

// Quitting reports whether Quit was requested.
func (a *App) Quitting() bool { return a.quit }

// Focused returns the currently focused component (may be nil).
func (a *App) Focused() *component.Component {
	if len(a.focus) == 0 || a.focusIdx < 0 || a.focusIdx >= len(a.focus) {
		return nil
	}
	return a.focus[a.focusIdx]
}

// SetFocus moves focus to index i (clamped/wrapped).
func (a *App) SetFocus(i int) {
	if len(a.focus) == 0 {
		return
	}
	n := len(a.focus)
	a.focusIdx = ((i % n) + n) % n
	a.events.Emit(event.Event{Type: event.FocusChange, TargetID: a.Focused().ID})
}

// FocusNext moves focus forward (wraps).
func (a *App) FocusNext() { a.SetFocus(a.focusIdx + 1) }

// FocusPrev moves focus backward (wraps).
func (a *App) FocusPrev() { a.SetFocus(a.focusIdx - 1) }

// cursorPos normalizes an Input cursor to [0, n]; negative means end.
func cursorPos(f *component.Component, n int) int {
	if f.Cursor < 0 || f.Cursor > n {
		return n
	}
	return f.Cursor
}

// cursorLineCol converts an absolute rune offset into (line, col).
func cursorLineCol(rs []rune, pos int) (line, col int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(rs) {
		pos = len(rs)
	}
	start := 0
	for i := 0; i < pos; i++ {
		if rs[i] == '\n' {
			line++
			start = i + 1
		}
	}
	return line, pos - start
}

// lineStartOffset returns the offset of the first rune of pos's line.
func lineStartOffset(rs []rune, pos int) int {
	if pos > len(rs) {
		pos = len(rs)
	}
	for pos > 0 && rs[pos-1] != '\n' {
		pos--
	}
	return pos
}

// lineEndOffset returns the offset just past the last rune of pos's
// line (the '\n' itself, or end of buffer).
func lineEndOffset(rs []rune, pos int) int {
	if pos > len(rs) {
		pos = len(rs)
	}
	for pos < len(rs) && rs[pos] != '\n' {
		pos++
	}
	return pos
}

// offsetAtLineCol maps (line, col) back to an absolute offset, clamping
// col to the line length and line to the last line.
func offsetAtLineCol(rs []rune, line, col int) int {
	s := 0
	for l := 0; l < line && s < len(rs); {
		if rs[s] == '\n' {
			l++
		}
		s++
	}
	e := s
	for e < len(rs) && rs[e] != '\n' {
		e++
	}
	if col < 0 {
		col = 0
	}
	if s+col > e {
		return e
	}
	return s + col
}

// inputView returns the visible rune window of a single-line value and
// the cursor column inside it, scrolling so the cursor stays visible.
func inputView(val string, cursor, visW int) (win []rune, col int) {
	rs := []rune(val)
	if visW <= 0 {
		return nil, 0
	}
	start := 0
	if cursor >= visW {
		start = cursor - visW + 1
	}
	if start > len(rs) {
		start = len(rs)
	}
	end := start + visW
	if end > len(rs) {
		end = len(rs)
	}
	return rs[start:end], cursor - start
}

// maskValue hides an input value for password fields: echo="*" (or any
// single character; echo="true" also means "*"). Absent, "false" or "0"
// disables masking; the stored value is always the real one.
func maskValue(val, echo string) string {
	echo = strings.TrimSpace(echo)
	if echo == "" || echo == "false" || echo == "0" {
		return val
	}
	m := '*'
	if echo != "true" {
		m = []rune(echo)[0]
	}
	return strings.Repeat(string(m), len([]rune(val)))
}

// attrInt reads an integer attribute with a default fallback.
func attrInt(c *component.Component, key string, def int) int {
	v, err := strconv.Atoi(strings.TrimSpace(c.Attr(key, "")))
	if err != nil {
		return def
	}
	return v
}

// wrapLines word-wraps text to width cells, preserving explicit
// newlines. Overlong words are hard-split. Width <= 0 returns the
// text unchanged.
func wrapLines(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	var out []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		cur := ""
		flush := func() {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		}
		for _, wd := range words {
			wr := []rune(wd)
			if len(wr) > width {
				flush()
				for len(wr) > width {
					out = append(out, string(wr[:width]))
					wr = wr[width:]
				}
				cur = string(wr)
				continue
			}
			if cur == "" {
				cur = wd
			} else if len([]rune(cur))+1+len(wr) <= width {
				cur += " " + wd
			} else {
				flush()
				cur = wd
			}
		}
		flush()
	}
	return out
}

// listLines returns the rows of a List component: the newline-split
// value of the state key named by its source attribute.
func (a *App) listLines(c *component.Component) []string {
	src := strings.TrimSpace(c.Attr("source", ""))
	if src == "" {
		return nil
	}
	val := a.store.Get(src)
	if val == "" {
		return nil
	}
	return strings.Split(val, "\n")
}

// clampListSel keeps the selection inside [0, len) (or -1 when empty).
func clampListSel(c *component.Component, n int) {
	if n <= 0 {
		c.Sel = -1
		return
	}
	if c.Sel < 0 {
		c.Sel = 0
	}
	if c.Sel >= n {
		c.Sel = n - 1
	}
}

// listOffset computes the first visible row so the selection stays
// visible inside a viewport of h rows.
func listOffset(sel, n, h int) int {
	if h <= 0 || n <= 0 || sel < 0 {
		return 0
	}
	if sel < h {
		return 0
	}
	off := sel - h + 1
	if off > n-1 {
		off = n - 1
	}
	if off < 0 {
		off = 0
	}
	return off
}

// moveListSel shifts the selection by delta, clamps it and emits a
// Select event when it actually changed.
func (a *App) moveListSel(c *component.Component, delta int) {
	lines := a.listLines(c)
	if len(lines) == 0 {
		c.Sel = -1
		return
	}
	old := c.Sel
	if old < 0 {
		c.Sel = 0
	} else {
		c.Sel += delta
	}
	clampListSel(c, len(lines))
	if c.Sel != old && c.Sel >= 0 && c.Sel < len(lines) {
		a.events.Emit(event.Event{
			Type:     event.Select,
			Name:     c.Attr("onSelect", ""),
			TargetID: c.ID,
			Value:    lines[c.Sel],
		})
	}
}

// Layout computes rectangles for a width x height screen.
func (a *App) Layout(width, height int) layout.Map {
	return layout.Compute(a.root, width, height)
}

// HandleKey processes one keyboard input. It returns true when the
// application requested to quit.
func (a *App) HandleKey(key tcell.Key, r rune, mod tcell.ModMask) bool {
	switch key {
	case tcell.KeyCtrlC, tcell.KeyCtrlD:
		a.quit = true
		a.events.Emit(event.Event{Type: event.Quit})
		return true
	case tcell.KeyEscape:
		a.quit = true
		a.events.Emit(event.Event{Type: event.Quit})
		return true
	case tcell.KeyTab:
		a.FocusNext()
		return false
	case tcell.KeyBacktab:
		a.FocusPrev()
		return false
	}

	f := a.Focused()

	// A focused Slider uses Left/Right/Home/End to change its value;
	// Up/Down still move focus through the generic handling below.
	if f != nil && f.Type == component.TypeSlider {
		switch key {
		case tcell.KeyLeft:
			a.adjustSlider(f, -1)
			return false
		case tcell.KeyRight:
			a.adjustSlider(f, +1)
			return false
		case tcell.KeyHome:
			_, minV, maxV, step := a.sliderRange(f)
			a.setSliderValue(f, minV, minV, maxV, step)
			return false
		case tcell.KeyEnd:
			_, minV, maxV, step := a.sliderRange(f)
			a.setSliderValue(f, maxV, minV, maxV, step)
			return false
		}
	}

	// Navigation arrows move focus, except inside inputs and lists.
	// Single-line inputs: Up/Down hop between fields, Left/Right move
	// the cursor. Multiline inputs: all four arrows move inside the
	// text (Up/Down hop out only at the first/last line); Tab leaves.
	switch key {
	case tcell.KeyDown, tcell.KeyRight:
		if f == nil || (f.Type != component.TypeInput && f.Type != component.TypeList) {
			a.FocusNext()
			return false
		}
		if f.Type == component.TypeInput && key == tcell.KeyDown &&
			f.Attr("multiline", "false") != "true" {
			a.FocusNext()
			return false
		}
		if f.Type == component.TypeList && key == tcell.KeyRight {
			a.FocusNext()
			return false
		}
	case tcell.KeyUp, tcell.KeyLeft:
		if f == nil || (f.Type != component.TypeInput && f.Type != component.TypeList) {
			a.FocusPrev()
			return false
		}
		if f.Type == component.TypeInput && key == tcell.KeyUp &&
			f.Attr("multiline", "false") != "true" {
			a.FocusPrev()
			return false
		}
		if f.Type == component.TypeList && key == tcell.KeyLeft {
			a.FocusPrev()
			return false
		}
	}

	if f != nil {
		switch f.Type {
		case component.TypeButton:
			if key == tcell.KeyEnter || r == ' ' {
				name := f.Attr("onActivate", f.Attr("onClick", ""))
				a.events.Emit(event.Event{
					Type:     event.Activate,
					Key:      key,
					Rune:     r,
					Mod:      mod,
					Name:     name,
					TargetID: f.ID,
				})
				if name == "" {
					// No handler: default buttons with id quit the app
					// only when explicitly marked quit="true".
					if f.Attr("quit", "false") == "true" {
						a.quit = true
					}
				}
				return false
			}
		case component.TypeInput:
			k := f.BindKey()
			cur := ""
			if k != "" {
				cur = a.store.Get(k)
			}
			multi := f.Attr("multiline", "false") == "true"
			runes := []rune(cur)
			pos := cursorPos(f, len(runes))
			commit := func(s string, cursor int) {
				if k == "" {
					return
				}
				a.store.Set(k, s)
				f.Cursor = cursor
				a.events.Emit(event.Event{
					Type:     event.InputChange,
					Key:      key,
					Rune:     r,
					Name:     f.Attr("onChange", ""),
					TargetID: f.ID,
					Value:    s,
				})
			}
			switch key {
			case tcell.KeyLeft:
				if pos > 0 {
					f.Cursor = pos - 1
				}
				return false
			case tcell.KeyRight:
				if pos < len(runes) {
					f.Cursor = pos + 1
				}
				return false
			case tcell.KeyUp:
				// Single-line inputs never reach here (handled
				// above); multiline moves a line up, or hops out
				// past the first line.
				if multi {
					if line, col := cursorLineCol(runes, pos); line > 0 {
						f.Cursor = offsetAtLineCol(runes, line-1, col)
					} else {
						a.FocusPrev()
					}
					return false
				}
				a.FocusPrev()
				return false
			case tcell.KeyDown:
				if multi {
					nlines := strings.Count(cur, "\n") + 1
					if line, col := cursorLineCol(runes, pos); line+1 < nlines {
						f.Cursor = offsetAtLineCol(runes, line+1, col)
					} else {
						a.FocusNext()
					}
					return false
				}
				a.FocusNext()
				return false
			case tcell.KeyHome:
				if multi {
					f.Cursor = lineStartOffset(runes, pos)
				} else {
					f.Cursor = 0
				}
				return false
			case tcell.KeyEnd:
				if multi {
					f.Cursor = lineEndOffset(runes, pos)
				} else {
					f.Cursor = -1
				}
				return false
			case tcell.KeyEnter:
				if multi {
					commit(cur+"\n", -1)
					return false
				}
				name := f.Attr("onEnter", f.Attr("onActivate", ""))
				a.events.Emit(event.Event{
					Type:     event.Activate,
					Key:      key,
					Name:     name,
					TargetID: f.ID,
					Value:    cur,
				})
				// Enter in a single-line input moves to next field.
				a.FocusNext()
				return false
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if k != "" && pos > 0 {
					commit(string(runes[:pos-1])+string(runes[pos:]), pos-1)
				}
				return false
			case tcell.KeyDelete:
				if k != "" && pos < len(runes) {
					commit(string(runes[:pos])+string(runes[pos+1:]), pos)
				}
				return false
			case tcell.KeyRune:
				if k != "" && r != 0 && (multi || r != '\n') {
					commit(string(runes[:pos])+string(r)+string(runes[pos:]), pos+1)
				}
				return false
			}
		case component.TypeCheckbox:
			if key == tcell.KeyEnter || r == ' ' {
				a.toggleCheckbox(f)
				return false
			}
		case component.TypeList:
			lines := a.listLines(f)
			switch key {
			case tcell.KeyUp:
				a.moveListSel(f, -1)
				return false
			case tcell.KeyDown:
				a.moveListSel(f, +1)
				return false
			case tcell.KeyEnter:
				if f.Sel >= 0 && f.Sel < len(lines) {
					a.events.Emit(event.Event{
						Type:     event.Activate,
						Key:      key,
						Name:     f.Attr("onActivate", f.Attr("onClick", "")),
						TargetID: f.ID,
						Value:    lines[f.Sel],
					})
				}
				return false
			}
			if r == ' ' {
				if f.Sel >= 0 && f.Sel < len(lines) {
					a.events.Emit(event.Event{
						Type:     event.Activate,
						Rune:     r,
						Name:     f.Attr("onActivate", f.Attr("onClick", "")),
						TargetID: f.ID,
						Value:    lines[f.Sel],
					})
				}
				return false
			}
		}
	}

	// Global key event for custom shortcuts.
	a.events.Emit(event.Event{Type: event.KeyPress, Key: key, Rune: r, Mod: mod})
	return false
}

// RenderCells renders the UI into a headless string grid (one string
// per row) for tests and screenshots. No terminal is required.
func (a *App) RenderCells(width, height int) []string {
	grid := make([][]rune, height)
	for y := range grid {
		grid[y] = make([]rune, width)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}
	rects := a.Layout(width, height)
	setText := func(x, y int, s string) {
		if y < 0 || y >= height {
			return
		}
		for i, r := range []rune(s) {
			if x+i < 0 || x+i >= width {
				continue
			}
			grid[y][x+i] = r
		}
	}
	drawBox := func(r layout.Rect, title string) {
		if r.W < 2 || r.H < 2 {
			return
		}
		setText(r.X, r.Y, "+"+strings.Repeat("-", r.W-2)+"+")
		setText(r.X, r.Bottom()-1, "+"+strings.Repeat("-", r.W-2)+"+")
		for y := r.Y + 1; y < r.Bottom()-1; y++ {
			setText(r.X, y, "|")
			setText(r.Right()-1, y, "|")
		}
		if title != "" {
			t := " " + a.store.Render(title) + " "
			rs := []rune(t)
			if len(rs) > r.W-2 {
				rs = rs[:r.W-2]
			}
			setText(r.X+1, r.Y, string(rs))
		}
	}
	clip := func(s string, w int) string {
		rs := []rune(s)
		if len(rs) > w {
			rs = rs[:w]
		}
		return string(rs)
	}
	center := func(s string, w int) string {
		rs := []rune(s)
		if len(rs) >= w {
			return clip(s, w)
		}
		pad := (w - len(rs)) / 2
		return strings.Repeat(" ", pad) + s
	}

	focused := a.Focused()
	var draw func(c *component.Component)
	draw = func(c *component.Component) {
		r, ok := rects[c]
		if !ok {
			return
		}
		isFocused := focused != nil && focused == c
		switch c.Type {
		case component.TypeApp, component.TypeColumn, component.TypeRow,
			component.TypeGrid, component.TypeSpacer:
			// Transparent containers.
		case component.TypeBox:
			if c.Attr("border", "true") != "false" {
				drawBox(r, c.Title())
			}
		case component.TextType:
			txt := a.store.Render(c.Text)
			var tlines []string
			if c.Attr("wrap", "true") == "false" {
				if idx := strings.Index(txt, "\n"); idx >= 0 {
					txt = txt[:idx]
				}
				tlines = []string{txt}
			} else {
				tlines = wrapLines(txt, r.W)
			}
			if ml := attrInt(c, "maxLines", 0); ml > 0 && len(tlines) > ml {
				tlines = tlines[:ml]
			}
			for i, ln := range tlines {
				if i >= r.H {
					break
				}
				setText(r.X, r.Y+i, clip(ln, r.W))
			}
		case component.TypeButton:
			label := a.store.Render(c.Attrs["label"])
			if isFocused {
				label = "> " + label + " <"
			}
			if c.Attr("border", "true") != "false" {
				drawBox(r, "")
				if r.H >= 3 {
					setText(r.X+1, r.Y+r.H/2, clip(center(label, r.W-2), r.W-2))
				} else if r.H >= 1 {
					setText(r.X, r.Y, clip(center(label, r.W), r.W))
				}
			} else {
				s := "[ " + label + " ]"
				if isFocused {
					s = "[> " + label + " <]"
				}
				setText(r.X, r.Y, clip(s, r.W))
			}
		case component.TypeInput:
			val := ""
			if k := c.BindKey(); k != "" {
				val = a.store.Get(k)
			} else {
				val = c.Attr("value", "")
			}
			multi := c.Attr("multiline", "false") == "true"
			shown := maskValue(val, c.Attr("echo", ""))
			bordered := c.Attr("border", "true") != "false"
			if multi {
				ilines := strings.Split(shown, "\n")
				ix, iy, iw, ih := r.X, r.Y, r.W, r.H
				if bordered {
					drawBox(r, "")
					ix++
					iy++
					iw -= 2
					ih -= 2
				}
				if iw >= 1 && ih >= 1 {
					if shown == "" {
						ln := c.Attr("placeholder", "")
						if isFocused {
							ln += "_"
						}
						setText(ix, iy, clip(ln, iw))
					} else {
						start := len(ilines) - ih
						if start < 0 {
							start = 0
						}
						cline, ccol := -1, 0
						if isFocused {
							cline, ccol = cursorLineCol([]rune(val), cursorPos(c, len([]rune(val))))
						}
						for i := start; i < len(ilines) && i-start < ih; i++ {
							ln := ilines[i]
							if isFocused && i == cline {
								lr := []rune(ln)
								at := ccol
								if at > len(lr) {
									at = len(lr)
								}
								if at < 0 {
									at = 0
								}
								ln = string(lr[:at]) + "_" + string(lr[at:])
							}
							setText(ix, iy+i-start, clip(ln, iw))
						}
					}
				}
				break
			}
			if bordered {
				drawBox(r, "")
			}
			ix, iw := r.X, r.W
			iy := r.Y
			if bordered {
				ix++
				iw -= 2
				iy = r.Y + r.H/2
			}
			if iw < 1 {
				break
			}
			if val == "" {
				ln := c.Attr("placeholder", "")
				if isFocused {
					ln += "_"
				}
				setText(ix, iy, clip(ln, iw))
				break
			}
			pos := cursorPos(c, len([]rune(val)))
			win, col := inputView(shown, pos, iw)
			wrs := string(win)
			if isFocused {
				wrs = string(win[:col]) + "_" + string(win[col:])
			}
			setText(ix, iy, clip(wrs, iw))
		case component.TypeList:
			lines := a.listLines(c)
			clampListSel(c, len(lines))
			lx, ly, lw, lh := r.X, r.Y, r.W, r.H
			if c.Attr("border", "false") != "false" {
				drawBox(r, c.Title())
				lx++
				ly++
				lw -= 2
				lh -= 2
			}
			if lw < 1 || lh < 1 {
				break
			}
			if len(lines) == 0 {
				setText(lx, ly, clip(c.Attr("empty", "(empty)"), lw))
				break
			}
			off := listOffset(c.Sel, len(lines), lh)
			for i := 0; i < lh && off+i < len(lines); i++ {
				mark := "  "
				if off+i == c.Sel {
					mark = "> "
				}
				setText(lx, ly+i, clip(mark+lines[off+i], lw))
			}
		case component.TypeDivider:
			ch := firstRune(c.Attr("char", ""), '-')
			if c.Attr("vertical", "false") == "true" {
				if c.Attr("char", "") == "" {
					ch = '|'
				}
				for y := r.Y; y < r.Bottom(); y++ {
					setText(r.X, y, string(ch))
				}
			} else {
				setText(r.X, r.Y, clip(rowOf(ch, a.store.Render(c.Attr("label", "")), r.W), r.W))
			}
		case component.TypeProgress:
			fill := firstRune(c.Attr("fill", ""), '#')
			empty := firstRune(c.Attr("empty", ""), '-')
			bar := progressBar(
				atof(a.boundValue(c), 0), atof(c.Attr("max", "100"), 100), r.W,
				fill, empty,
				c.Attr("brackets", "true") != "false",
				c.Attr("showValue", "true") != "false",
			)
			setText(r.X, r.Y, clip(bar, r.W))
		case component.TypeCheckbox:
			mark := "[ ]"
			if boolValue(a.boundValue(c)) {
				mark = "[x]"
			}
			line := mark
			if lbl := a.store.Render(c.Attr("label", "")); lbl != "" {
				line += " " + lbl
			}
			if isFocused {
				line = "> " + line
			}
			setText(r.X, r.Y, clip(line, r.W))
		case component.TypeSparkline:
			setText(r.X, r.Y, clip(a.sparklineString(c, r.W), r.W))
		case component.TypeSlider:
			v, minV, maxV, _ := a.sliderRange(c)
			bar := sliderBar(
				v, minV, maxV, r.W,
				firstRune(c.Attr("fill", ""), '='),
				firstRune(c.Attr("empty", ""), '-'),
				firstRune(c.Attr("knob", ""), 'O'),
				c.Attr("brackets", "true") != "false",
			)
			setText(r.X, r.Y, clip(bar, r.W))
		case component.TypeBarChart:
			ramp := []rune(c.Attr("chars", defaultRamp))
			txt := barChart(a.sourceValue(c), r.W, r.H, ramp, atof(c.Attr("max", "0"), 0))
			for i, ln := range strings.Split(txt, "\n") {
				if i >= r.H {
					break
				}
				setText(r.X, r.Y+i, clip(ln, r.W))
			}
		case component.TypeTable:
			header, rows, widths := a.tableData(c)
			maxRows := attrInt(c, "rows", 5)
			bx, by, bw, bh := r.X, r.Y, r.W, r.H
			if c.Attr("border", "false") == "true" {
				drawBox(r, c.Title())
				bx++
				by++
				bw -= 2
				bh -= 2
			}
			if bw < 1 || bh < 1 {
				break
			}
			lines := tableLines(header, rows, widths, bw, maxRows)
			if len(lines) == 0 {
				setText(bx, by, clip(c.Attr("empty", "(no rows)"), bw))
				break
			}
			for i, ln := range lines {
				if i >= bh {
					break
				}
				setText(bx, by+i, clip(ln, bw))
			}
		default:
			// Custom components: render children only.
		}
		for _, ch := range c.Children {
			draw(ch)
		}
	}
	draw(a.root)

	out := make([]string, height)
	for y, row := range grid {
		out[y] = strings.TrimRight(string(row), " ")
	}
	return out
}

// ClickAt processes a left mouse click at cell (x, y) on a width x
// height screen. Hit-testing follows document order (topmost last
// drawn wins); transparent containers are skipped so clicks pass
// through to interactive components behind them.
//
//   - Clicking a Button activates it (same as Enter).
//   - Clicking an Input moves keyboard focus to it.
//   - Clicking any component with onActivate/onClick fires that handler.
//
// It reports whether some component handled the click.
func (a *App) ClickAt(x, y, w, h int) bool {
	rects := a.Layout(w, h)
	var order []*component.Component
	component.Walk(a.root, func(c *component.Component) { order = append(order, c) })
	for i := len(order) - 1; i >= 0; i-- {
		c := order[i]
		r, ok := rects[c]
		if !ok || r.W <= 0 || r.H <= 0 {
			continue
		}
		if x < r.X || x >= r.Right() || y < r.Y || y >= r.Bottom() {
			continue
		}
		if c.Type == component.TypeInput {
			a.focusComponent(c)
			val := ""
			if k := c.BindKey(); k != "" {
				val = a.store.Get(k)
			} else {
				val = c.Attr("value", "")
			}
			inner, visW := r.X, r.W
			bordered := c.Attr("border", "true") != "false"
			if bordered {
				inner++
				visW -= 2
			}
			_ = visW
			if c.Attr("multiline", "false") == "true" {
				lines := strings.Split(val, "\n")
				top, visH := r.Y, r.H
				if bordered {
					top++
					visH -= 2
				}
				start := len(lines) - visH
				if start < 0 {
					start = 0
				}
				ln := (y - top) + start
				if ln < 0 {
					ln = 0
				}
				if ln >= len(lines) {
					ln = len(lines) - 1
				}
				if ln >= 0 {
					c.Cursor = offsetAtLineCol([]rune(val), ln, x-inner)
				}
				return true
			}
			// Same scroll window as rendering, so the click
			// lands on the visible rune.
			n := len([]rune(val))
			start := 0
			if pos := cursorPos(c, n); visW > 0 && pos >= visW {
				start = pos - visW + 1
			}
			at := start + (x - inner)
			if at < 0 {
				at = 0
			}
			if at > n {
				at = n
			}
			c.Cursor = at
			return true
		}
		if c.Type == component.TypeList {
			a.focusComponent(c)
			lines := a.listLines(c)
			clampListSel(c, len(lines))
			// Map the clicked row to a line index (skip border).
			top := r.Y
			visH := r.H
			if c.Attr("border", "false") != "false" {
				top++
				visH -= 2
			}
			idx := listOffset(c.Sel, len(lines), visH) + (y - top)
			if idx >= 0 && idx < len(lines) && idx != c.Sel {
				c.Sel = idx
				a.events.Emit(event.Event{
					Type:     event.Select,
					Name:     c.Attr("onSelect", ""),
					TargetID: c.ID,
					Value:    lines[c.Sel],
				})
			}
			return true
		}
		if c.Type == component.TypeSlider {
			a.focusComponent(c)
			v, minV, maxV, step := a.sliderRange(c)
			start, trackW := sliderTrack(c, r.W, v)
			pos := x - (r.X + start)
			frac := 0.0
			if trackW > 1 {
				frac = float64(pos) / float64(trackW-1)
			}
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			a.setSliderValue(c, minV+frac*(maxV-minV), minV, maxV, step)
			return true
		}
		if c.Type == component.TypeCheckbox {
			a.focusComponent(c)
			a.toggleCheckbox(c)
			return true
		}
		name := c.Attr("onActivate", c.Attr("onClick", ""))
		if c.Type == component.TypeButton || name != "" {
			a.focusComponent(c)
			a.events.Emit(event.Event{
				Type:     event.Activate,
				Name:     name,
				TargetID: c.ID,
			})
			if name == "" && c.Attr("quit", "false") == "true" {
				a.quit = true
			}
			return true
		}
		// Non-interactive: keep looking underneath.
	}
	return false
}

// focusComponent moves keyboard focus to c when it is focusable.
func (a *App) focusComponent(c *component.Component) {
	for idx, f := range a.focus {
		if f == c {
			a.SetFocus(idx)
			return
		}
	}
}

// Notify wakes the Run loop so it redraws with the latest state.
// It is goroutine-safe and a no-op while Run is not active: call it
// after mutating state from background goroutines (timers, file
// watchers, network) to refresh the screen immediately.
func (a *App) Notify() {
	if s := a.screen.Load(); s != nil {
		s.Wakeup()
	}
}

// Run starts the interactive terminal loop. It blocks until Quit is
// requested (Esc, Ctrl+C) or a quit button is activated.
// Mouse support is enabled: left click activates buttons and focuses
// inputs; everything else also works from the keyboard.
func (a *App) Run() error {
	s, err := backend.New()
	if err != nil {
		return err
	}
	defer s.Fini()
	defer a.screen.Store(nil)
	a.screen.Store(s)
	s.EnableMouse()

	for !a.quit {
		w, h := s.Size()
		if w <= 0 || h <= 0 {
			w, h = 80, 24
		}
		s.Clear()
		a.renderToScreen(s, w, h)
		s.Show()

		ev := s.Poll()
		switch t := ev.(type) {
		case *tcell.EventKey:
			a.HandleKey(t.Key(), t.Rune(), t.Modifiers())
		case *tcell.EventMouse:
			x, y := t.Position()
			if t.Buttons()&tcell.Button1 != 0 {
				a.ClickAt(x, y, w, h)
			}
		case *tcell.EventResize:
			// Redrawn next iteration.
			_ = t
		}
	}
	return nil
}

func (a *App) renderToScreen(s *backend.Screen, w, h int) {
	rects := a.Layout(w, h)
	focused := a.Focused()
	var draw func(c *component.Component)
	draw = func(c *component.Component) {
		r, ok := rects[c]
		if !ok {
			return
		}
		isFocused := focused != nil && focused == c
		st := c.Style.ToTcell()
		focusSt := st.Reverse(true)
		if isFocused && (c.Type == component.TypeButton || c.Type == component.TypeInput) {
			st = focusSt
		}
		switch c.Type {
		case component.TypeApp, component.TypeColumn, component.TypeRow,
			component.TypeGrid, component.TypeSpacer:
			// Transparent.
		case component.TypeBox:
			if c.Attr("border", "true") != "false" {
				s.DrawBox(r, a.store.Render(c.Title()), c.Style.ToTcell())
			}
		case component.TextType:
			txt := a.store.Render(c.Text)
			var tlines []string
			if c.Attr("wrap", "true") == "false" {
				if idx := strings.Index(txt, "\n"); idx >= 0 {
					txt = txt[:idx]
				}
				tlines = []string{txt}
			} else {
				tlines = wrapLines(txt, r.W)
			}
			if ml := attrInt(c, "maxLines", 0); ml > 0 && len(tlines) > ml {
				tlines = tlines[:ml]
			}
			for i, ln := range tlines {
				if i >= r.H {
					break
				}
				s.DrawText(r.X, r.Y+i, r.W, ln, st, false)
			}
		case component.TypeButton:
			label := a.store.Render(c.Attr("label", ""))
			if c.Attr("border", "true") != "false" {
				s.DrawBox(r, "", st)
				if r.H >= 1 && r.W >= 2 {
					inner := r.W - 2
					rs := []rune(label)
					if isFocused {
						label = "> " + label + " <"
						rs = []rune(label)
					}
					if len(rs) > inner {
						label = string(rs[:inner])
					} else {
						pad := (inner - len(rs)) / 2
						if pad < 0 {
							pad = 0
						}
						label = strings.Repeat(" ", pad) + label
					}
					s.DrawText(r.X+1, r.Y+r.H/2, inner, label, st, false)
				}
			} else {
				mark := "[ " + label + " ]"
				s.DrawText(r.X, r.Y, r.W, mark, st, false)
			}
		case component.TypeInput:
			val := ""
			if k := c.BindKey(); k != "" {
				val = a.store.Get(k)
			} else {
				val = c.Attr("value", "")
			}
			shown := maskValue(val, c.Attr("echo", ""))
			multi := c.Attr("multiline", "false") == "true"
			bordered := c.Attr("border", "true") != "false"
			if multi {
				ilines := strings.Split(shown, "\n")
				ix, iy, iw, ih := r.X, r.Y, r.W, r.H
				if bordered {
					s.DrawBox(r, "", st)
					ix++
					iy++
					iw -= 2
					ih -= 2
				}
				if iw >= 1 && ih >= 1 {
					if shown == "" {
						s.DrawText(ix, iy, iw, c.Attr("placeholder", ""), st, false)
					} else {
						start := len(ilines) - ih
						if start < 0 {
							start = 0
						}
						for i := start; i < len(ilines) && i-start < ih; i++ {
							s.DrawText(ix, iy+i-start, iw, ilines[i], st, false)
						}
						if isFocused {
							cline, ccol := cursorLineCol([]rune(val), cursorPos(c, len([]rune(val))))
							if cline >= start && cline-start < ih {
								mline := []rune(ilines[cline])
								if ccol > len(mline) {
									ccol = len(mline)
								}
								if ccol < 0 {
									ccol = 0
								}
								if cx := ix + ccol; cx < ix+iw {
									s.SetCell(cx, iy+(cline-start), '_', focusSt)
								}
							}
						}
					}
				}
				break
			}
			if bordered {
				s.DrawBox(r, "", st)
			}
			ix, iw := r.X, r.W
			iy := r.Y
			if bordered {
				ix++
				iw -= 2
				iy = r.Y + r.H/2
			}
			if iw < 1 {
				break
			}
			if val == "" {
				ln := c.Attr("placeholder", "")
				if isFocused {
					ln += "_"
				}
				s.DrawText(ix, iy, iw, ln, st, false)
				break
			}
			pos := cursorPos(c, len([]rune(val)))
			win, col := inputView(shown, pos, iw)
			s.DrawText(ix, iy, iw, string(win), st, false)
			if isFocused && col >= 0 && col < iw {
				s.SetCell(ix+col, iy, '_', focusSt)
			}
		case component.TypeList:
			lines := a.listLines(c)
			clampListSel(c, len(lines))
			lx, ly, lw, lh := r.X, r.Y, r.W, r.H
			if c.Attr("border", "false") != "false" {
				s.DrawBox(r, a.store.Render(c.Title()), c.Style.ToTcell())
				lx++
				ly++
				lw -= 2
				lh -= 2
			}
			if lw >= 1 && lh >= 1 {
				if len(lines) == 0 {
					s.DrawText(lx, ly, lw, c.Attr("empty", "(empty)"), st, false)
				} else {
					off := listOffset(c.Sel, len(lines), lh)
					for i := 0; i < lh && off+i < len(lines); i++ {
						row := "  " + lines[off+i]
						rst := st
						if off+i == c.Sel {
							row = "> " + lines[off+i]
							if isFocused {
								rst = focusSt
							}
						}
						s.DrawText(lx, ly+i, lw, row, rst, true)
					}
				}
			}
		case component.TypeDivider:
			ch := firstRune(c.Attr("char", ""), '-')
			if c.Attr("vertical", "false") == "true" {
				if c.Attr("char", "") == "" {
					ch = '|'
				}
				for y := r.Y; y < r.Bottom(); y++ {
					s.SetCell(r.X, y, ch, st)
				}
			} else if r.W > 0 {
				s.DrawText(r.X, r.Y, r.W, rowOf(ch, a.store.Render(c.Attr("label", "")), r.W), st, false)
			}
		case component.TypeProgress:
			fill := firstRune(c.Attr("fill", ""), '#')
			empty := firstRune(c.Attr("empty", ""), '-')
			bar := progressBar(
				atof(a.boundValue(c), 0), atof(c.Attr("max", "100"), 100), r.W,
				fill, empty,
				c.Attr("brackets", "true") != "false",
				c.Attr("showValue", "true") != "false",
			)
			s.DrawText(r.X, r.Y, r.W, bar, st, false)
		case component.TypeCheckbox:
			mark := "[ ]"
			if boolValue(a.boundValue(c)) {
				mark = "[x]"
			}
			line := mark
			if lbl := a.store.Render(c.Attr("label", "")); lbl != "" {
				line += " " + lbl
			}
			bst := st
			if isFocused {
				bst = focusSt
			}
			s.DrawText(r.X, r.Y, r.W, line, bst, false)
		case component.TypeSparkline:
			s.DrawText(r.X, r.Y, r.W, a.sparklineString(c, r.W), st, false)
		case component.TypeSlider:
			v, minV, maxV, _ := a.sliderRange(c)
			bar := sliderBar(
				v, minV, maxV, r.W,
				firstRune(c.Attr("fill", ""), '='),
				firstRune(c.Attr("empty", ""), '-'),
				firstRune(c.Attr("knob", ""), 'O'),
				c.Attr("brackets", "true") != "false",
			)
			s.DrawText(r.X, r.Y, r.W, bar, st, false)
		case component.TypeBarChart:
			ramp := []rune(c.Attr("chars", defaultRamp))
			txt := barChart(a.sourceValue(c), r.W, r.H, ramp, atof(c.Attr("max", "0"), 0))
			for i, ln := range strings.Split(txt, "\n") {
				if i >= r.H {
					break
				}
				s.DrawText(r.X, r.Y+i, r.W, ln, st, false)
			}
		case component.TypeTable:
			header, rows, widths := a.tableData(c)
			maxRows := attrInt(c, "rows", 5)
			bx, by, bw, bh := r.X, r.Y, r.W, r.H
			if c.Attr("border", "false") == "true" {
				s.DrawBox(r, a.store.Render(c.Title()), st)
				bx++
				by++
				bw -= 2
				bh -= 2
			}
			if bw >= 1 && bh >= 1 {
				lines := tableLines(header, rows, widths, bw, maxRows)
				if len(lines) == 0 {
					s.DrawText(bx, by, bw, c.Attr("empty", "(no rows)"), st, false)
				} else {
					for i, ln := range lines {
						if i >= bh {
							break
						}
						s.DrawText(bx, by+i, bw, ln, st, false)
					}
				}
			}
		default:
			// Custom: children only.
		}
		for _, ch := range c.Children {
			draw(ch)
		}
	}
	draw(a.root)
}
