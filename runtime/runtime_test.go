package runtime

import (
	"testing"
)

func TestSetGet(t *testing.T) {
	s := New(nil)
	s.Set("name", "ada")
	if got := s.Get("name"); got != "ada" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderInterpolation(t *testing.T) {
	s := New(map[string]string{"name": "ada"})
	if got := s.Render("hi {name}!"); got != "hi ada!" {
		t.Fatalf("got %q", got)
	}
	if got := s.Render("hi {missing}!"); got != "hi !" {
		t.Fatalf("got %q", got)
	}
	if got := s.Render("{{name}}"); got != "{name}" {
		t.Fatalf("escape failed: %q", got)
	}
}

func TestComputed(t *testing.T) {
	s := New(map[string]string{"a": "x"})
	s.Computed("b", func(get func(string) string) string {
		return get("a") + "!"
	})
	if got := s.Get("b"); got != "x!" {
		t.Fatalf("got %q", got)
	}
	s.Set("a", "y")
	if got := s.Get("b"); got != "y!" {
		t.Fatalf("got %q", got)
	}
	if got := s.Render("{b}"); got != "y!" {
		t.Fatalf("got %q", got)
	}
}

func TestOnChangeAndVersion(t *testing.T) {
	s := New(nil)
	var seen []string
	s.OnChange(func(k, v string) { seen = append(seen, k+"="+v) })
	v0 := s.Version()
	s.Set("k", "v")
	if len(seen) != 1 || seen[0] != "k=v" {
		t.Fatalf("watchers: %v", seen)
	}
	if s.Version() <= v0 {
		t.Fatal("version did not increase")
	}
	// Setting the same value is a no-op.
	v1 := s.Version()
	s.Set("k", "v")
	if s.Version() != v1 {
		t.Fatal("same-value Set should not bump version")
	}
}
