// Package style handles component styles: colors and attributes.
//
// Supported markup attributes:
//
//	fg="red" bg="black" bold="true" dim="true" italic="true"
//	underline="true" blink="true" reverse="true"
//
// Named colors: black, red, green, yellow, blue, magenta, cyan,
// white, gray, default. Hex "#rrggbb" values are also accepted
// through tcell.
package style

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// Style describes how a component looks.
type Style struct {
	Fg        string
	Bg        string
	Bold      bool
	Dim       bool
	Italic    bool
	Underline bool
	Blink     bool
	Reverse   bool
}

// Parse reads style attributes from an XML attribute map.
func Parse(attrs map[string]string) Style {
	s := Style{}
	if attrs == nil {
		return s
	}
	s.Fg = strings.TrimSpace(attrs["fg"])
	s.Bg = strings.TrimSpace(attrs["bg"])
	s.Bold = boolAttr(attrs, "bold")
	s.Dim = boolAttr(attrs, "dim")
	s.Italic = boolAttr(attrs, "italic")
	s.Underline = boolAttr(attrs, "underline")
	s.Blink = boolAttr(attrs, "blink")
	s.Reverse = boolAttr(attrs, "reverse")
	return s
}

func boolAttr(attrs map[string]string, key string) bool {
	v, ok := attrs[key]
	if !ok {
		return false
	}
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return true
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return v == "1" || v == "yes" || v == "on"
	}
	return b
}

// ToTcell converts the Style into an actual tcell.Style.
func (s Style) ToTcell() tcell.Style {
	st := tcell.StyleDefault
	if s.Fg != "" {
		st = st.Foreground(Color(s.Fg))
	}
	if s.Bg != "" {
		st = st.Background(Color(s.Bg))
	}
	if s.Bold {
		st = st.Bold(true)
	}
	if s.Dim {
		st = st.Dim(true)
	}
	if s.Italic {
		st = st.Italic(true)
	}
	if s.Underline {
		st = st.Underline(true)
	}
	if s.Blink {
		st = st.Blink(true)
	}
	if s.Reverse {
		st = st.Reverse(true)
	}
	return st
}

// Merge returns a style where non-empty attributes of over
// override the base ones.
func (s Style) Merge(over Style) Style {
	out := s
	if over.Fg != "" {
		out.Fg = over.Fg
	}
	if over.Bg != "" {
		out.Bg = over.Bg
	}
	out.Bold = out.Bold || over.Bold
	out.Dim = out.Dim || over.Dim
	out.Italic = out.Italic || over.Italic
	out.Underline = out.Underline || over.Underline
	out.Blink = out.Blink || over.Blink
	out.Reverse = out.Reverse || over.Reverse
	return out
}

// Color converts a color name into a tcell.Color.
func Color(name string) tcell.Color {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "default", "none":
		return tcell.ColorDefault
	case "black":
		return tcell.ColorBlack
	case "red":
		return tcell.ColorRed
	case "green":
		return tcell.ColorGreen
	case "yellow":
		return tcell.ColorYellow
	case "blue":
		return tcell.ColorBlue
	case "magenta", "purple":
		return tcell.ColorPurple
	case "cyan", "teal":
		return tcell.ColorTeal
	case "white":
		return tcell.ColorWhite
	case "gray", "grey":
		return tcell.ColorGray
	default:
		return tcell.GetColor(name)
	}
}
