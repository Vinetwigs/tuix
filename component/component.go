// Package component builds the component tree from the AST and
// hosts the registry for custom components written in Go.
//
// Built-in primitives:
//
//	App, Column, Row, Box, Grid, Text, Button, Input, Spacer, List,
//	Divider, ProgressBar, Checkbox, Sparkline, Slider, BarChart, Table
//
// Custom components can be registered with Register and then used as
// tags in .tuix markup. A Factory receives the raw attributes and the
// already-built children and returns the component subtree root.
package component

import (
	"fmt"
	"strings"

	"github.com/Vinetwigs/tuix/ast"
	"github.com/Vinetwigs/tuix/style"
)

// Valid built-in component types.
const (
	TypeApp       = "App"
	TypeColumn    = "Column"
	TypeRow       = "Row"
	TypeBox       = "Box"
	TypeGrid      = "Grid"
	TextType      = "Text"
	TypeButton    = "Button"
	TypeInput     = "Input"
	TypeSpacer    = "Spacer"
	TypeList      = "List"
	TypeDivider   = "Divider"
	TypeProgress  = "ProgressBar"
	TypeCheckbox  = "Checkbox"
	TypeSparkline = "Sparkline"
	TypeSlider    = "Slider"
	TypeBarChart  = "BarChart"
	TypeTable     = "Table"
	TypeInclude   = "Include" // resolved by the parser, never reaches Build
)

// Component is a single UI node with resolved metadata.
type Component struct {
	Type      string
	ID        string
	Attrs     map[string]string
	Text      string // for Text: value attr or inner text
	Children  []*Component
	Focusable bool
	FocusIdx  int // order among focusables, -1 if not focusable
	Style     style.Style
	Sel       int // List selection index (runtime state, -1 = none)
	Cursor    int // Input cursor as rune offset (<0 = end, the default)
}

// Factory builds a custom component subtree.
type Factory func(attrs map[string]string, children []*Component) (*Component, error)

var registry = map[string]Factory{}

// Register adds a custom component type usable as a tag in markup.
// It returns an error if the name collides with a built-in type.
func Register(name string, f Factory) error {
	if name == "" || f == nil {
		return fmt.Errorf("component: invalid registration for %q", name)
	}
	switch name {
	case TypeApp, TypeColumn, TypeRow, TypeBox, TypeGrid, TextType, TypeButton, TypeInput, TypeSpacer, TypeList, TypeDivider, TypeProgress, TypeCheckbox, TypeSparkline, TypeSlider, TypeBarChart, TypeTable:
		return fmt.Errorf("component: %q is a built-in type", name)
	}
	registry[name] = f
	return nil
}

// Registered reports whether name is a custom registered type.
func Registered(name string) bool {
	_, ok := registry[name]
	return ok
}

// ResetRegistry clears custom registrations (mainly for tests).
func ResetRegistry() {
	registry = map[string]Factory{}
}

// Build converts an AST root into a Component tree.
func Build(root *ast.Node) (*Component, error) {
	if root == nil {
		return nil, fmt.Errorf("component: nil AST root")
	}
	return buildNode(root)
}

func copyAttrs(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func buildNode(n *ast.Node) (*Component, error) {
	// Custom component expansion (exact name first).
	if fn, ok := registry[n.Name]; ok {
		children := make([]*Component, 0, len(n.Children))
		for _, c := range n.Children {
			bc, err := buildNode(c)
			if err != nil {
				return nil, err
			}
			children = append(children, bc)
		}
		c, err := fn(copyAttrs(n.Attrs), children)
		if err != nil {
			return nil, fmt.Errorf("component: custom %q: %w", n.Name, err)
		}
		if c == nil {
			return nil, fmt.Errorf("component: custom %q returned nil", n.Name)
		}
		return c, nil
	}

	// Tolerate lowercase tags: <button> == <Button>.
	name := NormalizedType(n.Name)
	if fn, ok := registry[name]; ok {
		children := make([]*Component, 0, len(n.Children))
		for _, c := range n.Children {
			bc, err := buildNode(c)
			if err != nil {
				return nil, err
			}
			children = append(children, bc)
		}
		c, err := fn(copyAttrs(n.Attrs), children)
		if err != nil {
			return nil, fmt.Errorf("component: custom %q: %w", name, err)
		}
		if c == nil {
			return nil, fmt.Errorf("component: custom %q returned nil", name)
		}
		return c, nil
	}

	c := &Component{
		Type:     name,
		Attrs:    copyAttrs(n.Attrs),
		Style:    style.Parse(n.Attrs),
		FocusIdx: -1,
		Cursor:   -1,
	}
	c.ID = c.Attrs["id"]

	switch name {
	case TypeApp, TypeColumn, TypeRow, TypeBox, TypeGrid, TypeSpacer:
		// containers: nothing special
	case TypeList:
		c.Focusable = true
		c.Sel = -1
	case TypeCheckbox:
		c.Focusable = true
	case TypeSlider:
		c.Focusable = true
	case TypeDivider, TypeProgress, TypeSparkline, TypeBarChart, TypeTable:
		// visual leaves: nothing special
	case TextType:
		c.Text = c.Attrs["value"]
		if c.Text == "" {
			c.Text = n.Text
		}
	case TypeButton:
		c.Focusable = true
		if c.Attrs["label"] == "" && n.Text != "" {
			c.Attrs["label"] = n.Text
		}
	case TypeInput:
		c.Focusable = true
	default:
		if n.Line > 0 {
			return nil, fmt.Errorf("component: line %d: unknown type <%s> (register it with component.Register)", n.Line, name)
		}
		return nil, fmt.Errorf("component: unknown type <%s> (register it with component.Register)", name)
	}

	for _, child := range n.Children {
		// Text-only children of leaf components merge into Text.
		if (c.Type == TextType || c.Type == TypeButton) && child.Name == "" {
			c.Text += child.Text
			continue
		}
		bc, err := buildNode(child)
		if err != nil {
			return nil, err
		}
		c.Children = append(c.Children, bc)
	}
	// Leaf text fallback: inner text node stored in AST Text.
	if c.Type == TextType && c.Text == "" {
		c.Text = n.Text
	}
	if c.Type == TypeButton && c.Attrs["label"] == "" {
		c.Attrs["label"] = n.Text
	}
	return c, nil
}

// IsContainer reports whether the type can have children.
func IsContainer(t string) bool {
	switch t {
	case TypeApp, TypeColumn, TypeRow, TypeBox, TypeGrid:
		return true
	default:
		return Registered(t) // custom types decide themselves
	}
}

// IsFocusable reports whether the component can receive focus.
func (c *Component) IsFocusable() bool { return c != nil && c.Focusable }

// Label returns the display label for Button components.
func (c *Component) Label() string {
	if c == nil {
		return ""
	}
	return c.Attrs["label"]
}

// BindKey returns the state key an Input is bound to ("bind" attr).
func (c *Component) BindKey() string {
	if c == nil {
		return ""
	}
	return c.Attrs["bind"]
}

// Title returns the window/box title if any.
func (c *Component) Title() string {
	if c == nil {
		return ""
	}
	if t, ok := c.Attrs["title"]; ok {
		return t
	}
	return ""
}

// Walk visits the tree depth-first.
func Walk(c *Component, fn func(*Component)) {
	if c == nil || fn == nil {
		return
	}
	fn(c)
	for _, ch := range c.Children {
		Walk(ch, fn)
	}
}

// Focusables returns focusable components in document order.
func Focusables(root *Component) []*Component {
	var out []*Component
	Walk(root, func(c *Component) {
		if c.Focusable {
			out = append(out, c)
		}
	})
	for i, c := range out {
		c.FocusIdx = i
	}
	return out
}

// FindByID searches the tree for a component with the given id.
func FindByID(root *Component, id string) *Component {
	var found *Component
	Walk(root, func(c *Component) {
		if found == nil && c.ID == id && id != "" {
			found = c
		}
	})
	return found
}

// Describe returns a short human-readable tag like <Button id="ok">.
func (c *Component) Describe() string {
	if c == nil {
		return "<nil>"
	}
	if c.ID != "" {
		return "<" + c.Type + " id=\"" + c.ID + "\">"
	}
	return "<" + c.Type + ">"
}

// Attr gets an attribute with default, case-sensitive.
func (c *Component) Attr(key, def string) string {
	if c == nil || c.Attrs == nil {
		return def
	}
	if v, ok := c.Attrs[key]; ok {
		return v
	}
	return def
}

// NormalizedType returns the type with first letter upper-cased to be
// tolerant with markup like <button> vs <Button>.
func NormalizedType(t string) string {
	if t == "" {
		return t
	}
	up := strings.ToUpper(t[:1]) + t[1:]
	// Handle all-lowercase multi-letter names.
	lower := strings.ToLower(t)
	switch lower {
	case "app":
		return TypeApp
	case "column":
		return TypeColumn
	case "row":
		return TypeRow
	case "box":
		return TypeBox
	case "grid":
		return TypeGrid
	case "text":
		return TextType
	case "button":
		return TypeButton
	case "input":
		return TypeInput
	case "spacer":
		return TypeSpacer
	case "list":
		return TypeList
	case "divider":
		return TypeDivider
	case "progressbar", "progress", "bar":
		return TypeProgress
	case "checkbox", "check":
		return TypeCheckbox
	case "sparkline", "spark":
		return TypeSparkline
	case "slider", "range":
		return TypeSlider
	case "barchart", "bars":
		return TypeBarChart
	case "table", "grid-table":
		return TypeTable
	}
	return up
}
