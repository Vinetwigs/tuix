package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMinimal(t *testing.T) {
	root, err := ParseString(`<App title="Demo"><Text value="hi"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	if root.Name != "App" {
		t.Fatalf("root = %q", root.Name)
	}
	if len(root.Children) != 1 || root.Children[0].Name != "Text" {
		t.Fatalf("unexpected children: %+v", root.Children)
	}
}

func TestParseRequiresAppRoot(t *testing.T) {
	if _, err := ParseString(`<Column></Column>`); err == nil {
		t.Fatal("expected error for non-App root")
	}
}

func TestParseUnbalanced(t *testing.T) {
	if _, err := ParseString(`<App><Text></App>`); err == nil {
		t.Fatal("expected error for unbalanced tags")
	}
}

func TestParseErrorLineNumbers(t *testing.T) {
	_, err := ParseString("<App>\n<Text/>\n</Nope>")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("missing line number: %q", err)
	}
	_, err = ParseString("<App><Text>")
	if err == nil {
		t.Fatal("expected error for unclosed tag")
	}
	if !strings.Contains(err.Error(), "line") {
		t.Fatalf("missing line number: %q", err)
	}
}

func TestParseRecordsNodeLines(t *testing.T) {
	root, err := ParseString("<App>\n<Column>\n<Text value=\"x\"/>\n</Column>\n</App>")
	if err != nil {
		t.Fatal(err)
	}
	if root.Line != 1 || root.Children[0].Line != 2 || root.Children[0].Children[0].Line != 3 {
		t.Fatalf("lines: %d %d %d", root.Line, root.Children[0].Line, root.Children[0].Children[0].Line)
	}
}

func TestParseInclude(t *testing.T) {
	dir := t.TempDir()
	inc := `<App><Text value="from-include"/></App>`
	if err := os.WriteFile(filepath.Join(dir, "part.tuix"), []byte(inc), 0644); err != nil {
		t.Fatal(err)
	}
	main := `<App><Include src="part.tuix"/><Text value="main"/></App>`
	p := filepath.Join(dir, "main.tuix")
	if err := os.WriteFile(p, []byte(main), 0644); err != nil {
		t.Fatal(err)
	}
	root, err := ParseFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 children after include, got %d", len(root.Children))
	}
	if root.Children[0].Attrs["value"] != "from-include" {
		t.Fatalf("include not expanded: %+v", root.Children[0])
	}
}
