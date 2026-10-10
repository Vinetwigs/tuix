// Package layout computes component rectangles for a given screen size.
//
// Supported containers: Row, Column, Box, Grid. Leaf widgets (Text,
// Button, Input, Spacer, List, Divider, ProgressBar, Checkbox,
// Sparkline, Slider, BarChart, Table) report a natural height;
// containers arrange children honoring gap, padding, align and flex.
//
// Attributes:
//
//	width, height: int, "fill" or "auto" (default auto)
//	gap: int spacing between children (default 0, Row/Column/Grid)
//	padding: int inner margin (default 0, containers)
//	align: start|center|end (cross-axis alignment, default start)
//	flex: int weight for sharing leftover space (default 0)
//	columns: int number of grid columns (Grid only, default 2)
//	border: true|false for Button/Input/Box (default true for Box/Button)
package layout

import (
	"strconv"
	"strings"

	"github.com/Vinetwigs/tuix/component"
)

// Rect is an integer cell rectangle on screen.
type Rect struct {
	X, Y int
	W, H int
}

// Right returns X+W, Bottom returns Y+H.
func (r Rect) Right() int  { return r.X + r.W }
func (r Rect) Bottom() int { return r.Y + r.H }

// Inset shrinks r by n cells on every side (clamped to >= 0).
func (r Rect) Inset(n int) Rect {
	r.X += n
	r.Y += n
	r.W -= 2 * n
	r.H -= 2 * n
	if r.W < 0 {
		r.W = 0
	}
	if r.H < 0 {
		r.H = 0
	}
	return r
}

// Map links each component to its rectangle.
type Map map[*component.Component]Rect

// Compute lays out the tree inside a width x height area at origin (0,0).
func Compute(root *component.Component, width, height int) Map {
	m := Map{}
	if root == nil || width <= 0 || height <= 0 {
		return m
	}
	layoutNode(root, Rect{X: 0, Y: 0, W: width, H: height}, m)
	return m
}

func atoi(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	if v < 0 {
		return 0
	}
	return v
}

// desiredHeight is the content-based height of a component: an explicit
// height attribute wins, leaves report their natural size, containers
// recursively sum their children (plus padding, gap and borders).
// It lets containers size to their content instead of collapsing.
func desiredHeight(c *component.Component) int {
	if h, ok := fixedSize(c, "h"); ok {
		return h
	}
	pad := atoi(c.Attr("padding", "0"), 0)
	gap := atoi(c.Attr("gap", "0"), 0)
	switch c.Type {
	case component.TextType, component.TypeSpacer, component.TypeDivider,
		component.TypeProgress, component.TypeCheckbox, component.TypeSparkline,
		component.TypeSlider:
		return 1
	case component.TypeBarChart:
		return 3
	case component.TypeTable:
		rows := atoi(c.Attr("rows", "5"), 5)
		if rows < 1 {
			rows = 1
		}
		h := rows
		if c.Attr("columns", "") != "" {
			h += 2 // header + rule
		}
		if c.Attr("border", "false") == "true" {
			h += 2
		}
		return h
	case component.TypeList:
		// Content comes from the store (unknown here): rows of visible
		// lines, scrollable. Explicit height wins via fixedSize above.
		rows := atoi(c.Attr("rows", "5"), 5)
		if rows < 1 {
			rows = 1
		}
		if c.Attr("border", "false") == "false" {
			return rows
		}
		return rows + 2
	case component.TypeButton, component.TypeInput:
		if c.Attr("border", "true") == "false" {
			return 1
		}
		return 3
	case component.TypeRow:
		m := 0
		for _, k := range c.Children {
			m = max(m, desiredHeight(k))
		}
		return m + 2*pad
	case component.TypeGrid:
		cols := atoi(c.Attr("columns", "2"), 2)
		if cols < 1 {
			cols = 1
		}
		if len(c.Children) == 0 {
			return max(2*pad, 1)
		}
		m := 0
		for _, k := range c.Children {
			m = max(m, desiredHeight(k))
		}
		rows := (len(c.Children) + cols - 1) / cols
		return rows*m + gap*(rows-1) + 2*pad
	case component.TypeApp, component.TypeColumn:
		if len(c.Children) == 0 {
			return max(2*pad, 1)
		}
		sum := 2 * pad
		for _, k := range c.Children {
			sum += desiredHeight(k)
		}
		return sum + gap*max(len(c.Children)-1, 0)
	case component.TypeBox:
		border := 0
		if c.Attr("border", "true") != "false" {
			border = 2 // top + bottom edge
		}
		if len(c.Children) == 0 {
			return border + 2*pad
		}
		sum := border + 2*pad
		for _, k := range c.Children {
			sum += desiredHeight(k)
		}
		return sum + gap*max(len(c.Children)-1, 0)
	default:
		// Custom components stack children vertically like a column.
		if len(c.Children) == 0 {
			return 1
		}
		sum := 0
		for _, k := range c.Children {
			sum += desiredHeight(k)
		}
		return sum + gap*max(len(c.Children)-1, 0)
	}
}

func isFill(c *component.Component, axis string) bool {
	v := ""
	if axis == "w" {
		v = strings.ToLower(strings.TrimSpace(c.Attr("width", "")))
	} else {
		v = strings.ToLower(strings.TrimSpace(c.Attr("height", "")))
	}
	return v == "fill"
}

func fixedSize(c *component.Component, axis string) (int, bool) {
	v := ""
	if axis == "w" {
		v = strings.TrimSpace(c.Attr("width", ""))
	} else {
		v = strings.TrimSpace(c.Attr("height", ""))
	}
	if v == "" || v == "auto" || v == "fill" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func layoutNode(c *component.Component, r Rect, m Map) {
	m[c] = r
	if len(c.Children) == 0 {
		return
	}
	switch c.Type {
	case component.TypeApp, component.TypeColumn:
		layoutColumn(c, r, m)
	case component.TypeRow:
		layoutRow(c, r, m)
	case component.TypeBox:
		layoutBox(c, r, m)
	case component.TypeGrid:
		layoutGrid(c, r, m)
	default:
		// Custom or leaf with children: stack vertically like a column.
		layoutColumn(c, r, m)
	}
}

func layoutColumn(c *component.Component, r Rect, m Map) {
	pad := atoi(c.Attr("padding", "0"), 0)
	gap := atoi(c.Attr("gap", "0"), 0)
	inner := r.Inset(pad)
	x := inner.X
	y := inner.Y
	w := inner.W

	kids := c.Children
	// First pass: fixed heights + natural heights.
	heights := make([]int, len(kids))
	totalFlex := 0
	used := 0
	for i, k := range kids {
		if h, ok := fixedSize(k, "h"); ok {
			heights[i] = h
		} else if isFill(k, "h") {
			heights[i] = -1
			totalFlex += max(atoi(k.Attr("flex", "1"), 1), 1)
		} else if f := atoi(k.Attr("flex", "0"), 0); f > 0 {
			heights[i] = -1
			totalFlex += f
		} else {
			heights[i] = desiredHeight(k)
			// A lone container child claims the remaining space.
			switch k.Type {
			case component.TypeColumn, component.TypeRow, component.TypeBox, component.TypeGrid:
				if len(kids) == 1 {
					heights[i] = -1
					totalFlex++
				}
			}
		}
		if heights[i] >= 0 {
			used += heights[i]
		}
	}
	used += gap * max(len(kids)-1, 0)
	leftover := inner.H - used
	if leftover < 0 {
		leftover = 0
	}
	for i, k := range kids {
		h := heights[i]
		if h < 0 {
			f := atoi(k.Attr("flex", "1"), 1)
			if f <= 0 {
				f = 1
			}
			if totalFlex > 0 {
				h = leftover * f / totalFlex
			} else {
				h = leftover
			}
		}
		cw := w
		if fw, ok := fixedSize(k, "w"); ok && fw < cw {
			cw = fw
		}
		// Cross-axis align.
		cx := x
		switch strings.ToLower(strings.TrimSpace(k.Attr("align", ""))) {
		case "center":
			cx = x + (w-cw)/2
		case "end", "right":
			cx = x + w - cw
		}
		layoutNode(k, Rect{X: cx, Y: y, W: cw, H: h}, m)
		y += h + gap
	}
}

func layoutRow(c *component.Component, r Rect, m Map) {
	pad := atoi(c.Attr("padding", "0"), 0)
	gap := atoi(c.Attr("gap", "0"), 0)
	inner := r.Inset(pad)
	x := inner.X
	y := inner.Y
	h := inner.H

	kids := c.Children
	widths := make([]int, len(kids))
	totalFlex := 0
	used := 0
	for i, k := range kids {
		if k.Type == component.TypeDivider && k.Attr("vertical", "false") == "true" {
			if _, ok := fixedSize(k, "w"); !ok {
				widths[i] = 1
				used++
				continue
			}
		}
		if w, ok := fixedSize(k, "w"); ok {
			widths[i] = w
		} else if isFill(k, "w") {
			widths[i] = -1
			totalFlex += max(atoi(k.Attr("flex", "1"), 1), 1)
		} else if f := atoi(k.Attr("flex", "0"), 0); f > 0 {
			widths[i] = -1
			totalFlex += f
		} else {
			// Natural width: leaf widgets size to content, containers share.
			switch k.Type {
			case component.TextType:
				widths[i] = -1 // fill share; rendering clips
				totalFlex++
			case component.TypeButton, component.TypeInput, component.TypeSpacer:
				widths[i] = -1
				totalFlex++
			default:
				widths[i] = -1
				totalFlex++
			}
		}
		if widths[i] >= 0 {
			used += widths[i]
		}
	}
	used += gap * max(len(kids)-1, 0)
	leftover := inner.W - used
	if leftover < 0 {
		leftover = 0
	}
	// If nobody asked to flex, share equally.
	if totalFlex == 0 && len(kids) > 0 {
		share := leftover / len(kids)
		for i := range widths {
			if widths[i] < 0 {
				widths[i] = share
			}
		}
	} else {
		for i, k := range kids {
			if widths[i] < 0 {
				f := atoi(k.Attr("flex", "1"), 1)
				if f <= 0 {
					f = 1
				}
				widths[i] = leftover * f / totalFlex
			}
		}
	}
	for i, k := range kids {
		ch := h
		if fh, ok := fixedSize(k, "h"); ok && fh < ch {
			ch = fh
		}
		layoutNode(k, Rect{X: x, Y: y, W: widths[i], H: ch}, m)
		x += widths[i] + gap
	}
}

func layoutBox(c *component.Component, r Rect, m Map) {
	pad := atoi(c.Attr("padding", "0"), 0)
	border := 0
	if c.Attr("border", "true") != "false" {
		border = 1
	}
	inner := r
	inner.X += border
	inner.Y += border
	inner.W -= 2 * border
	inner.H -= 2 * border
	inner = inner.Inset(pad)
	if inner.W < 0 {
		inner.W = 0
	}
	if inner.H < 0 {
		inner.H = 0
	}
	if len(c.Children) == 0 {
		return
	}
	// Box stacks its children vertically, each sized to content.
	// A single child fills the whole inner area. Children are
	// clamped to the remaining space so overflowing content is
	// clipped instead of overlapping.
	y := inner.Y
	gap := atoi(c.Attr("gap", "0"), 0)
	for _, k := range c.Children {
		h := desiredHeight(k)
		if len(c.Children) == 1 {
			h = inner.H
		}
		if fh, ok := fixedSize(k, "h"); ok {
			h = fh
		}
		if y+h > inner.Bottom() {
			h = inner.Bottom() - y
		}
		if h < 0 {
			h = 0
		}
		layoutNode(k, Rect{X: inner.X, Y: y, W: inner.W, H: h}, m)
		y += h + gap
	}
}

func layoutGrid(c *component.Component, r Rect, m Map) {
	pad := atoi(c.Attr("padding", "0"), 0)
	gap := atoi(c.Attr("gap", "0"), 0)
	cols := atoi(c.Attr("columns", "2"), 2)
	if cols < 1 {
		cols = 1
	}
	inner := r.Inset(pad)
	n := len(c.Children)
	if n == 0 {
		return
	}
	rows := (n + cols - 1) / cols
	cellW := (inner.W - gap*(cols-1)) / cols
	cellH := (inner.H - gap*(rows-1)) / rows
	if cellW < 0 {
		cellW = 0
	}
	if cellH < 0 {
		cellH = 0
	}
	for i, k := range c.Children {
		col := i % cols
		row := i / cols
		layoutNode(k, Rect{
			X: inner.X + col*(cellW+gap),
			Y: inner.Y + row*(cellH+gap),
			W: cellW,
			H: cellH,
		}, m)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
