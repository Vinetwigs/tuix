package layout

import (
	"testing"

	"github.com/Vinetwigs/tuix/ast"
	"github.com/Vinetwigs/tuix/component"
)

func build(t *testing.T, root *ast.Node) *component.Component {
	t.Helper()
	c, err := component.Build(root)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestColumnSplitsRows(t *testing.T) {
	c := build(t, &ast.Node{Name: "App", Children: []*ast.Node{
		{Name: "Column", Children: []*ast.Node{
			{Name: "Text", Attrs: map[string]string{"value": "a"}},
			{Name: "Text", Attrs: map[string]string{"value": "b"}},
		}},
	}})
	m := Compute(c, 20, 10)
	col := c.Children[0]
	r := m[col]
	if r.W != 20 || r.H != 10 {
		t.Fatalf("column rect = %+v", r)
	}
	a := m[col.Children[0]]
	b := m[col.Children[1]]
	if a.Y != 0 || b.Y <= a.Y {
		t.Fatalf("rows not stacked: %+v %+v", a, b)
	}
	if a.W != 20 || b.W != 20 {
		t.Fatalf("widths: %+v %+v", a, b)
	}
}

func TestRowSplitsColumns(t *testing.T) {
	c := build(t, &ast.Node{Name: "App", Children: []*ast.Node{
		{Name: "Row", Children: []*ast.Node{
			{Name: "Text", Attrs: map[string]string{"value": "a"}},
			{Name: "Text", Attrs: map[string]string{"value": "b"}},
		}},
	}})
	m := Compute(c, 20, 6)
	a := m[c.Children[0].Children[0]]
	b := m[c.Children[0].Children[1]]
	if a.X != 0 || b.X <= a.X {
		t.Fatalf("cols not side by side: %+v %+v", a, b)
	}
	if a.W+b.W > 20 {
		t.Fatalf("overflow: %+v %+v", a, b)
	}
}

func TestGridCells(t *testing.T) {
	c := build(t, &ast.Node{Name: "App", Children: []*ast.Node{
		{Name: "Grid", Attrs: map[string]string{"columns": "2"}, Children: []*ast.Node{
			{Name: "Text", Attrs: map[string]string{"value": "a"}},
			{Name: "Text", Attrs: map[string]string{"value": "b"}},
			{Name: "Text", Attrs: map[string]string{"value": "c"}},
		}},
	}})
	m := Compute(c, 20, 10)
	kids := c.Children[0].Children
	r0, r1, r2 := m[kids[0]], m[kids[1]], m[kids[2]]
	if r1.X <= r0.X {
		t.Fatalf("grid row: %+v %+v", r0, r1)
	}
	if r2.Y <= r0.Y {
		t.Fatalf("grid wrap: %+v %+v", r0, r2)
	}
}

func TestBoxInset(t *testing.T) {
	c := build(t, &ast.Node{Name: "App", Children: []*ast.Node{
		{Name: "Box", Attrs: map[string]string{"title": "T"}, Children: []*ast.Node{
			{Name: "Text", Attrs: map[string]string{"value": "hi"}},
		}},
	}})
	m := Compute(c, 20, 10)
	inner := m[c.Children[0].Children[0]]
	outer := m[c.Children[0]]
	if inner.X <= outer.X || inner.Y <= outer.Y {
		t.Fatalf("box did not inset: outer=%+v inner=%+v", outer, inner)
	}
}
