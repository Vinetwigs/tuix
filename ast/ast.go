// Package ast defines the AST of .tuix markup.
//
// Markup is intentionally separated from runtime and rendering:
// this package only holds the tree representation of the document,
// without any layout or drawing logic.
package ast

// Node is a single element of a .tuix document.
// Name is the tag name (e.g. "Column", "Text", "Button").
// Attrs holds XML attributes as key/value pairs.
// Children are nested child nodes.
// Text is free text between tags, already normalized.
// Line is the 1-based source line of the opening tag (0 if unknown);
// it is used to pinpoint markup errors.
type Node struct {
	Name     string
	Attrs    map[string]string
	Children []*Node
	Text     string
	Line     int
}

// Attr returns the value of attribute key or def if missing.
func (n *Node) Attr(key, def string) string {
	if n == nil || n.Attrs == nil {
		return def
	}
	if v, ok := n.Attrs[key]; ok {
		return v
	}
	return def
}

// HasAttr reports whether attribute key is present.
func (n *Node) HasAttr(key string) bool {
	if n == nil || n.Attrs == nil {
		return false
	}
	_, ok := n.Attrs[key]
	return ok
}
