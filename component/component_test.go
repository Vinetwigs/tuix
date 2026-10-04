package component

import (
	"testing"

	"github.com/Vinetwigs/tuix/ast"
)

func node(name string, attrs map[string]string, kids ...*ast.Node) *ast.Node {
	return &ast.Node{Name: name, Attrs: attrs, Children: kids}
}

func TestBuildTree(t *testing.T) {
	root := node("App", nil,
		node("Column", nil,
			node("Text", map[string]string{"value": "hi"}),
			node("Button", map[string]string{"id": "ok", "label": "OK"}),
		),
	)
	c, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	if c.Type != "App" || len(c.Children) != 1 {
		t.Fatalf("unexpected tree: %+v", c)
	}
	f := Focusables(c)
	if len(f) != 1 || f[0].ID != "ok" {
		t.Fatalf("focusables: %+v", f)
	}
}

func TestBuildUnknownType(t *testing.T) {
	root := node("App", nil, node("Nope", nil))
	if _, err := Build(root); err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestCustomRegistry(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()
	err := Register("Badge", func(attrs map[string]string, kids []*Component) (*Component, error) {
		return &Component{Type: "Badge", Attrs: attrs, Children: kids, Text: attrs["label"]}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	root := node("App", nil, node("Badge", map[string]string{"label": "B"}))
	c, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Children) != 1 || c.Children[0].Type != "Badge" {
		t.Fatalf("custom not expanded: %+v", c)
	}
	if err := Register("Text", func(map[string]string, []*Component) (*Component, error) {
		return nil, nil
	}); err == nil {
		t.Fatal("expected error when overriding built-in")
	}
}

func TestLowercaseTags(t *testing.T) {
	root := node("App", nil, node("button", map[string]string{"id": "b", "label": "x"}))
	c, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	if c.Children[0].Type != TypeButton {
		t.Fatalf("type = %q", c.Children[0].Type)
	}
}

func TestListBuild(t *testing.T) {
	root := node("App", nil, node("List", map[string]string{"id": "l", "source": "items"}))
	c, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	l := c.Children[0]
	if l.Type != TypeList || !l.Focusable || l.Sel != -1 {
		t.Fatalf("list = %+v", l)
	}
	if got := Focusables(c); len(got) != 1 {
		t.Fatalf("focusables = %d", len(got))
	}
}
