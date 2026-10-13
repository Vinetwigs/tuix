package tuix

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Vinetwigs/tuix/component"
	"github.com/Vinetwigs/tuix/event"
)

// This file holds the pure helpers behind the visual widgets (Divider,
// ProgressBar, Checkbox, Sparkline, Slider, BarChart, Table). Keeping
// them free of tcell and of the render loops makes them trivial to
// unit test.

// boundValue returns the store value behind a component's bind key,
// or its literal value attribute when it has no binding.
func (a *App) boundValue(c *component.Component) string {
	if k := c.BindKey(); k != "" {
		return a.store.Get(k)
	}
	return c.Attr("value", "")
}

// sourceValue prefers the store key named by source, then the bound or
// literal value. It is the shared lookup for data-driven widgets.
func (a *App) sourceValue(c *component.Component) string {
	if src := strings.TrimSpace(c.Attr("source", "")); src != "" {
		return a.store.Get(src)
	}
	return a.boundValue(c)
}

// firstRune returns the first rune of s, or def when s is empty.
func firstRune(s string, def rune) rune {
	for _, r := range s {
		return r
	}
	return def
}

// boolValue parses a state string as a boolean flag.
func boolValue(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

// atof parses a float with a default fallback.
func atof(s string, def float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return v
}

// formatVal prints a float without a trailing ".0" for whole numbers.
func formatVal(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// parseSeries splits a string of numbers separated by spaces, commas,
// semicolons or newlines into a slice of floats, skipping bad tokens.
func parseSeries(s string) []float64 {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == ';' || r == '\t' || r == '\n'
	})
	var out []float64
	for _, f := range fields {
		if v, err := strconv.ParseFloat(strings.TrimSpace(f), 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

// toggleCheckbox flips the bound flag of a Checkbox, stores it and
// emits an Activate event carrying the new value ("true"/"false").
func (a *App) toggleCheckbox(c *component.Component) {
	k := c.BindKey()
	next := !boolValue(a.store.Get(k))
	if k != "" {
		a.store.Set(k, strconv.FormatBool(next))
	}
	a.events.Emit(event.Event{
		Type:     event.Activate,
		Name:     c.Attr("onChange", c.Attr("onActivate", "")),
		TargetID: c.ID,
		Value:    strconv.FormatBool(next),
	})
}

// progressBar renders value/max as a filled bar of the given width.
// value is clamped to [0, max]; the percentage label and the square
// brackets around the bar are optional.
func progressBar(value, maxV float64, width int, fill, empty rune, brackets, showValue bool) string {
	if maxV <= 0 {
		maxV = 1
	}
	frac := value / maxV
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	suffix := ""
	if showValue {
		suffix = fmt.Sprintf(" %d%%", int(frac*100+0.5))
	}
	barW := width - len([]rune(suffix))
	if brackets {
		barW -= 2
	}
	if barW < 0 {
		barW = 0
	}
	filled := int(float64(barW)*frac + 0.5)
	var b strings.Builder
	if brackets {
		b.WriteByte('[')
	}
	for i := 0; i < barW; i++ {
		if i < filled {
			b.WriteRune(fill)
		} else {
			b.WriteRune(empty)
		}
	}
	if brackets {
		b.WriteByte(']')
	}
	b.WriteString(suffix)
	return b.String()
}

// sparkline renders a numeric series as a one-line chart on the given
// ramp. Bounds can be pinned per side: hasLo/hasHi select whether lo/hi
// are taken from the caller or derived from the series, so a single
// min or max is honoured while the other side still auto-scales. A flat
// series sits on the middle level. When there are more values than
// columns, only the most recent ones are kept.
func sparkline(series string, width int, ramp []rune, lo, hi float64, hasLo, hasHi bool) string {
	if width <= 0 || len(ramp) < 2 {
		return ""
	}
	nums := parseSeries(series)
	if len(nums) == 0 {
		return ""
	}
	if !hasLo || !hasHi {
		mn, mx := nums[0], nums[0]
		for _, v := range nums {
			if v < mn {
				mn = v
			}
			if v > mx {
				mx = v
			}
		}
		if !hasLo {
			lo = mn
		}
		if !hasHi {
			hi = mx
		}
	}
	if len(nums) > width {
		nums = nums[len(nums)-width:]
	}
	span := hi - lo
	mid := (len(ramp) - 1) / 2
	var b strings.Builder
	for _, v := range nums {
		idx := mid
		if span > 0 {
			idx = int((v - lo) / span * float64(len(ramp)-1))
		}
		if idx < 0 {
			idx = 0
		}
		if idx > len(ramp)-1 {
			idx = len(ramp) - 1
		}
		b.WriteRune(ramp[idx])
	}
	return b.String()
}

// defaultRamp is the character ramp used by Sparkline and BarChart.
const defaultRamp = "\u2581\u2582\u2583\u2584\u2585\u2586\u2587\u2588"

// sparklineString renders a Sparkline component at the given width,
// honouring the optional chars/min/max attributes. A single min or max
// pins just that side; the other keeps auto-scaling.
func (a *App) sparklineString(c *component.Component, width int) string {
	ramp := []rune(c.Attr("chars", defaultRamp))
	loS := strings.TrimSpace(c.Attr("min", ""))
	hiS := strings.TrimSpace(c.Attr("max", ""))
	return sparkline(a.sourceValue(c), width, ramp,
		atof(loS, 0), atof(hiS, 0), loS != "", hiS != "")
}

// sliderRange reads a Slider's bounds and current value from the
// component and its bound state.
func (a *App) sliderRange(c *component.Component) (v, minV, maxV, step float64) {
	minV = atof(c.Attr("min", "0"), 0)
	maxV = atof(c.Attr("max", "100"), 100)
	if maxV <= minV {
		maxV = minV + 1
	}
	step = atof(c.Attr("step", "1"), 1)
	if step <= 0 {
		step = 1
	}
	v = atof(a.boundValue(c), minV)
	return
}

// sliderTrack returns the track start offset and width for a slider
// drawn in a component of the given width showing the given value.
func sliderTrack(c *component.Component, width int, value float64) (start, trackW int) {
	suffix := " " + formatVal(value)
	trackW = width - len([]rune(suffix))
	start = 0
	if c.Attr("brackets", "true") != "false" {
		trackW -= 2
		start = 1
	}
	if trackW < 1 {
		trackW = 1
	}
	return
}

// setSliderValue clamps target to the slider range, snaps it to the
// step, stores it and emits an InputChange event.
func (a *App) setSliderValue(c *component.Component, target, minV, maxV, step float64) {
	if target < minV {
		target = minV
	}
	if target > maxV {
		target = maxV
	}
	target = minV + float64(int((target-minV)/step+0.5))*step
	if target > maxV {
		target = maxV
	}
	if k := c.BindKey(); k != "" {
		a.store.Set(k, formatVal(target))
	}
	a.events.Emit(event.Event{
		Type:     event.InputChange,
		Name:     c.Attr("onChange", c.Attr("onActivate", "")),
		TargetID: c.ID,
		Value:    formatVal(target),
	})
}

// adjustSlider moves a slider by dir steps.
func (a *App) adjustSlider(c *component.Component, dir float64) {
	v, minV, maxV, step := a.sliderRange(c)
	a.setSliderValue(c, v+dir*step, minV, maxV, step)
}

// sliderBar renders a Slider track with a knob at frac in [0, 1].
func sliderBar(value, minV, maxV float64, width int, fill, empty, knob rune, brackets bool) string {
	if maxV <= minV {
		maxV = minV + 1
	}
	frac := (value - minV) / (maxV - minV)
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	suffix := " " + formatVal(value)
	trackW := width - len([]rune(suffix))
	if brackets {
		trackW -= 2
	}
	if trackW < 1 {
		trackW = 1
	}
	pos := int(frac * float64(trackW-1))
	var b strings.Builder
	if brackets {
		b.WriteByte('[')
	}
	for i := 0; i < trackW; i++ {
		switch {
		case i == pos:
			b.WriteRune(knob)
		case i < pos:
			b.WriteRune(fill)
		default:
			b.WriteRune(empty)
		}
	}
	if brackets {
		b.WriteByte(']')
	}
	b.WriteString(suffix)
	return b.String()
}

// barChart renders a numeric series as a multi-row bar chart of the
// given width and height, using the ramp for partial cells (ramp[0] is
// the lowest sliver, ramp[len-1] the full block). maxV <= 0 means the
// scale comes from the tallest value.
func barChart(series string, width, height int, ramp []rune, maxV float64) string {
	if width <= 0 || height <= 0 || len(ramp) < 2 {
		return ""
	}
	nums := parseSeries(series)
	if len(nums) == 0 {
		return ""
	}
	if maxV <= 0 {
		for _, v := range nums {
			if v > maxV {
				maxV = v
			}
		}
	}
	if maxV <= 0 {
		maxV = 1
	}
	if len(nums) > width {
		nums = nums[len(nums)-width:]
	}
	full := ramp[len(ramp)-1]
	rows := make([][]rune, height)
	for i := range rows {
		rows[i] = make([]rune, width)
		for j := range rows[i] {
			rows[i][j] = ' '
		}
	}
	for col, v := range nums {
		frac := v / maxV
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		for row := 0; row < height; row++ {
			fromBottom := height - 1 - row
			cell := frac*float64(height) - float64(fromBottom)
			switch {
			case cell >= 1:
				rows[row][col] = full
			case cell > 0:
				idx := int(cell * float64(len(ramp)))
				if idx > len(ramp)-1 {
					idx = len(ramp) - 1
				}
				if idx < 0 {
					idx = 0
				}
				rows[row][col] = ramp[idx]
			}
		}
	}
	lines := make([]string, height)
	for i, r := range rows {
		lines[i] = strings.TrimRight(string(r), " ")
	}
	return strings.Join(lines, "\n")
}

// tableLines renders a read-only table: an optional header row and
// rule, then up to maxRows data rows spread over total columns. When
// there are no data rows it shows the empty placeholder instead (below
// the header, when there is one). Column widths are clamped to at least
// one cell so bad markup cannot panic.
func tableLines(header []string, rows [][]string, widths []int, total, maxRows int, empty string) []string {
	if total <= 0 {
		return nil
	}
	ncols := len(header)
	for _, r := range rows {
		if len(r) > ncols {
			ncols = len(r)
		}
	}
	if ncols == 0 {
		return nil
	}
	if maxRows <= 0 {
		maxRows = 1
	}
	colW := make([]int, ncols)
	if len(widths) == ncols {
		for i, w := range widths {
			if w < 1 {
				w = 1
			}
			colW[i] = w
		}
	} else {
		base := (total - (ncols - 1)) / ncols
		if base < 1 {
			base = 1
		}
		for i := range colW {
			colW[i] = base
		}
	}
	pad := func(s string, w int) string {
		if w < 0 {
			w = 0
		}
		rs := []rune(s)
		if len(rs) > w {
			return string(rs[:w])
		}
		return s + strings.Repeat(" ", w-len(rs))
	}
	line := func(cells []string) string {
		parts := make([]string, ncols)
		for i := 0; i < ncols; i++ {
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			parts[i] = pad(c, colW[i])
		}
		out := []rune(strings.Join(parts, " "))
		if len(out) > total {
			out = out[:total]
		}
		return string(out)
	}
	var out []string
	if len(header) > 0 {
		out = append(out, line(header))
		out = append(out, strings.Repeat("-", total))
	}
	if len(rows) == 0 {
		if empty != "" {
			out = append(out, pad(empty, total))
		}
		return out
	}
	for i, r := range rows {
		if i >= maxRows {
			break
		}
		out = append(out, line(r))
	}
	return out
}

// parseInts parses a comma-separated list of integers.
func parseInts(s string) []int {
	var out []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if n, err := strconv.Atoi(part); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// tableData resolves a Table's header, rows and column widths from its
// attributes and the store.
func (a *App) tableData(c *component.Component) (header []string, rows [][]string, widths []int) {
	if cols := strings.TrimSpace(c.Attr("columns", "")); cols != "" {
		for _, s := range strings.Split(cols, ",") {
			header = append(header, a.store.Render(strings.TrimSpace(s)))
		}
	}
	widths = parseInts(c.Attr("widths", ""))
	sep := c.Attr("sep", "|")
	for _, line := range strings.Split(a.sourceValue(c), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		cells := strings.Split(line, sep)
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		rows = append(rows, cells)
	}
	return
}

// rowOf returns a row of width cells filled with ch, with label (if
// any) centered over it. Used by horizontal Dividers.
func rowOf(ch rune, label string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := make([]rune, width)
	for i := range runes {
		runes[i] = ch
	}
	if label != "" {
		mid := []rune(" " + label + " ")
		start := (width - len(mid)) / 2
		if start < 0 {
			start = 0
		}
		for i, m := range mid {
			if start+i >= width {
				break
			}
			runes[start+i] = m
		}
	}
	return string(runes)
}
