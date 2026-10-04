// Package parser converts .tuix documents (XML) into an AST.
//
// Supported grammar (MVP):
//
//	<App title="..."> ... </App>        mandatory root
//	<Column gap="1" padding="1"> ... </Column>
//	<Row gap="1" padding="0"> ... </Row>
//	<Box title="..." padding="1"> ... </Box>
//	<Grid columns="2" gap="1"> ... </Grid>
//	<Text value="Hello {name}" />
//	<Button id="ok" label="OK" onActivate="onOk" />
//	<Input id="name" bind="name" placeholder="..." />
//	<Include src="other.tuix" />        inclusion of composed components
//
// Free text between tags is stored in Node.Text.
package parser

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vinetwigs/tuix/ast"
)

// Parse converts an in-memory .tuix document into its AST.
// Structural errors report 1-based line numbers; raw XML syntax
// errors already carry them via encoding/xml.
func Parse(data []byte) (*ast.Node, error) {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	dec.Strict = true

	lineOf := func(off int64) int {
		if off < 0 {
			off = 0
		}
		if off > int64(len(data)) {
			off = int64(len(data))
		}
		return 1 + strings.Count(string(data[:off]), "\n")
	}

	var stack []*ast.Node
	var root *ast.Node

	for {
		start := dec.InputOffset()
		tok, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("parser: XML error near line %d: %w", lineOf(start), err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &ast.Node{Name: t.Name.Local, Attrs: map[string]string{}}
			for _, a := range t.Attr {
				n.Attrs[a.Name.Local] = a.Value
			}
			n.Line = lineOf(start)
			if len(stack) == 0 {
				root = n
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, n)
			}
			stack = append(stack, n)
		case xml.EndElement:
			line := lineOf(start)
			if len(stack) == 0 {
				return nil, fmt.Errorf("parser: line %d: extra closing tag </%s>", line, t.Name.Local)
			}
			top := stack[len(stack)-1]
			if top.Name != t.Name.Local {
				return nil, fmt.Errorf("parser: line %d: unbalanced tag </%s> (opened <%s> at line %d)",
					line, t.Name.Local, top.Name, top.Line)
			}
			top.Text = strings.TrimSpace(top.Text)
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				text := strings.TrimSpace(string(t))
				if text != "" {
					top := stack[len(stack)-1]
					if top.Text != "" {
						top.Text += " "
					}
					top.Text += text
				}
			}
		case xml.Comment:
			// Ignored.
		}
	}
	if root == nil {
		return nil, fmt.Errorf("parser: empty document")
	}
	if root.Name != "App" {
		return nil, fmt.Errorf("parser: line %d: root must be <App>, found <%s>", root.Line, root.Name)
	}
	if len(stack) > 0 {
		top := stack[len(stack)-1]
		return nil, fmt.Errorf("parser: line %d: unclosed tag <%s> opened at line %d",
			lineOf(int64(len(data))), top.Name, top.Line)
	}
	return root, nil
}

// ParseFile reads and parses a .tuix file, resolving
// <Include src="..."/> relative to the file directory.
func ParseFile(path string) (*ast.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("parser: read %s: %w", path, err)
	}
	root, err := Parse(data)
	if err != nil {
		return nil, err
	}
	base := filepath.Dir(path)
	if err := expandIncludes(root, base, 0); err != nil {
		return nil, err
	}
	return root, nil
}

// ParseString is a convenience alias for Parse on strings.
func ParseString(src string) (*ast.Node, error) {
	return Parse([]byte(src))
}

// expandIncludes replaces every <Include src="..."/> with the children
// of the referenced file. It allows defining composed components in
// separate .tuix files and reusing them. Depth is limited to avoid cycles.
func expandIncludes(n *ast.Node, base string, depth int) error {
	if depth > 16 {
		return fmt.Errorf("parser: nested inclusions over the limit (possible cycle)")
	}
	var out []*ast.Node
	for _, c := range n.Children {
		if c.Name == "Include" {
			src := c.Attr("src", "")
			if src == "" {
				return fmt.Errorf("parser: <Include> without src attribute")
			}
			p := src
			if !filepath.IsAbs(p) {
				p = filepath.Join(base, src)
			}
			sub, err := ParseFile(p)
			if err != nil {
				return err
			}
			// Replace with the children of the included <App>.
			for _, sc := range sub.Children {
				if err := expandIncludes(sc, filepath.Dir(p), depth+1); err != nil {
					return err
				}
				out = append(out, sc)
			}
			continue
		}
		if err := expandIncludes(c, base, depth+1); err != nil {
			return err
		}
		out = append(out, c)
	}
	n.Children = out
	return nil
}
