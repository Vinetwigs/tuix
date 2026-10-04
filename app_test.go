package tuix

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/Vinetwigs/tuix/component"
	"github.com/Vinetwigs/tuix/event"
)

const demoDoc = `<App title="Demo">
  <State key="name" value="ada"/>
  <Column gap="1" padding="1">
    <Text value="Hello {name}"/>
    <Input id="f1" bind="name" placeholder="name"/>
    <Button id="ok" label="OK" onActivate="onOk"/>
  </Column>
</App>`

func TestLoadAndBinding(t *testing.T) {
	app, err := LoadString(demoDoc)
	if err != nil {
		t.Fatal(err)
	}
	if app.Title() != "Demo" {
		t.Fatalf("title = %q", app.Title())
	}
	if got := app.Get("name"); got != "ada" {
		t.Fatalf("name = %q", got)
	}
	rows := app.RenderCells(30, 12)
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "Hello ada") {
		t.Fatalf("binding not rendered:\n%s", joined)
	}
	app.Set("name", "grace")
	rows = app.RenderCells(30, 12)
	if !strings.Contains(strings.Join(rows, "\n"), "Hello grace") {
		t.Fatalf("update not rendered:\n%s", strings.Join(rows, "\n"))
	}
}

func TestComputedRendering(t *testing.T) {
	app, err := LoadString(`<App><Text value="[{shout}]"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	app.Set("name", "ada")
	app.Computed("shout", func(get func(string) string) string {
		return strings.ToUpper(get("name"))
	})
	rows := app.RenderCells(30, 5)
	if !strings.Contains(strings.Join(rows, "\n"), "[ADA]") {
		t.Fatalf("computed not rendered: %q", rows)
	}
}

func TestButtonActivate(t *testing.T) {
	app, err := LoadString(demoDoc)
	if err != nil {
		t.Fatal(err)
	}
	var got []event.Event
	app.On("onOk", func(ev event.Event) { got = append(got, ev) })
	// Focus starts on first focusable (Input f1); move to Button ok.
	app.FocusNext()
	if f := app.Focused(); f == nil || f.ID != "ok" {
		t.Fatalf("focused = %+v", f)
	}
	app.HandleKey(tcell.KeyEnter, 0, 0)
	if len(got) != 1 || got[0].TargetID != "ok" || got[0].Type != event.Activate {
		t.Fatalf("events: %+v", got)
	}
}

func TestInputEditing(t *testing.T) {
	app, err := LoadString(`<App><Input id="n" bind="nick" value="ab"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	if got := app.Get("nick"); got != "ab" {
		t.Fatalf("seed = %q", got)
	}
	app.HandleKey(tcell.KeyRune, 'c', 0)
	if got := app.Get("nick"); got != "abc" {
		t.Fatalf("after rune = %q", got)
	}
	app.HandleKey(tcell.KeyBackspace, 0, 0)
	if got := app.Get("nick"); got != "ab" {
		t.Fatalf("after backspace = %q", got)
	}
}

func TestFocusWrap(t *testing.T) {
	app, err := LoadString(demoDoc)
	if err != nil {
		t.Fatal(err)
	}
	if len(app.focus) != 2 {
		t.Fatalf("focusables = %d", len(app.focus))
	}
	first := app.Focused().ID
	app.FocusPrev() // wraps to last
	if app.Focused().ID == first {
		t.Fatal("FocusPrev did not wrap")
	}
	app.FocusNext() // wraps back to first
	if app.Focused().ID != first {
		t.Fatal("FocusNext did not wrap")
	}
}

func TestUnknownComponentErrors(t *testing.T) {
	if _, err := LoadString(`<App><Nope/></App>`); err == nil {
		t.Fatal("expected error for unknown component")
	}
}

func clickCenter(t *testing.T, app *App, id string, w, h int) bool {
	t.Helper()
	target := component.FindByID(app.Tree(), id)
	if target == nil {
		t.Fatalf("no component %q", id)
	}
	r := app.Layout(w, h)[target]
	return app.ClickAt(r.X+r.W/2, r.Y+r.H/2, w, h)
}

func TestClickButtonActivates(t *testing.T) {
	app, err := LoadString(`<App><Column><Button id="b" label="Go" onActivate="go"/>
		<Button id="q" label="Quit" quit="true"/></Column></App>`)
	if err != nil {
		t.Fatal(err)
	}
	var got []event.Event
	app.On("go", func(ev event.Event) { got = append(got, ev) })
	if !clickCenter(t, app, "b", 40, 12) {
		t.Fatal("click not handled")
	}
	if len(got) != 1 || got[0].TargetID != "b" || got[0].Type != event.Activate {
		t.Fatalf("events: %+v", got)
	}
	if app.Quitting() {
		t.Fatal("should not quit on plain button")
	}
	if !clickCenter(t, app, "q", 40, 12) {
		t.Fatal("quit click not handled")
	}
	if !app.Quitting() {
		t.Fatal("quit button click should quit")
	}
}

func TestClickInputFocuses(t *testing.T) {
	app, err := LoadString(demoDoc)
	if err != nil {
		t.Fatal(err)
	}
	app.SetFocus(1) // Button ok
	if !clickCenter(t, app, "f1", 30, 12) {
		t.Fatal("input click not handled")
	}
	if f := app.Focused(); f == nil || f.ID != "f1" {
		t.Fatalf("focused = %+v", f)
	}
}

func TestClickEmptyArea(t *testing.T) {
	app, err := LoadString(`<App><Text value="hi"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	if app.ClickAt(39, 11, 40, 12) {
		t.Fatal("click on empty area should not be handled")
	}
}

func TestListKeyboardNav(t *testing.T) {
	app, err := LoadString(`<App><List id="l" source="items" onActivate="pick"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	app.Set("items", "a\nb\nc")
	var sel []event.Event
	var act []event.Event
	app.On("*", func(ev event.Event) {
		if ev.Type == event.Select {
			sel = append(sel, ev)
		}
	})
	app.On("pick", func(ev event.Event) { act = append(act, ev) })
	app.HandleKey(tcell.KeyDown, 0, 0) // sel 0
	app.HandleKey(tcell.KeyDown, 0, 0) // sel 1
	if got := component.FindByID(app.Tree(), "l").Sel; got != 1 {
		t.Fatalf("sel = %d", got)
	}
	if len(sel) != 2 || sel[1].Value != "b" {
		t.Fatalf("select events: %+v", sel)
	}
	app.HandleKey(tcell.KeyEnter, 0, 0)
	if len(act) != 1 || act[0].Value != "b" {
		t.Fatalf("activate events: %+v", act)
	}
	rows := strings.Join(app.RenderCells(20, 5), "\n")
	if !strings.Contains(rows, "> b") {
		t.Fatalf("selection not rendered:\n%s", rows)
	}
}

func TestListClickSelects(t *testing.T) {
	app, err := LoadString(`<App><List id="l" source="items"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	app.Set("items", "a\nb\nc")
	r := app.Layout(20, 5)[component.FindByID(app.Tree(), "l")]
	if !app.ClickAt(r.X, r.Y+2, 20, 5) {
		t.Fatal("list click not handled")
	}
	if got := component.FindByID(app.Tree(), "l").Sel; got != 2 {
		t.Fatalf("sel = %d", got)
	}
}

func typeRunes(t *testing.T, app *App, s string) {
	t.Helper()
	for _, r := range s {
		app.HandleKey(tcell.KeyRune, r, 0)
	}
}

func TestInputCursor(t *testing.T) {
	app, err := LoadString(`<App><Input id="n" bind="nick"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	inp := component.FindByID(app.Tree(), "n")
	typeRunes(t, app, "abc")
	if got := app.Get("nick"); got != "abc" {
		t.Fatalf("got %q", got)
	}
	app.HandleKey(tcell.KeyLeft, 0, 0)
	app.HandleKey(tcell.KeyLeft, 0, 0)
	typeRunes(t, app, "X")
	if got := app.Get("nick"); got != "aXbc" {
		t.Fatalf("mid insert: %q", got)
	}
	app.HandleKey(tcell.KeyBackspace, 0, 0)
	if got := app.Get("nick"); got != "abc" {
		t.Fatalf("mid backspace: %q", got)
	}
	app.HandleKey(tcell.KeyHome, 0, 0)
	typeRunes(t, app, "S")
	if got := app.Get("nick"); got != "Sabc" {
		t.Fatalf("home insert: %q", got)
	}
	app.HandleKey(tcell.KeyEnd, 0, 0)
	app.HandleKey(tcell.KeyLeft, 0, 0)
	app.HandleKey(tcell.KeyLeft, 0, 0)
	app.HandleKey(tcell.KeyDelete, 0, 0)
	if got := app.Get("nick"); got != "Sac" {
		t.Fatalf("delete: %q", got)
	}
	_ = inp
	rows := strings.Join(app.RenderCells(20, 5), "\n")
	if !strings.Contains(rows, "Sa_c") {
		t.Fatalf("cursor not rendered:\n%s", rows)
	}
}

func TestInputClickSetsCursor(t *testing.T) {
	app, err := LoadString(`<App><Input id="n" bind="nick" value="hello"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	r := app.Layout(30, 5)[component.FindByID(app.Tree(), "n")]
	if !app.ClickAt(r.X+2, r.Y+r.H/2, 30, 5) {
		t.Fatal("click not handled")
	}
	if got := component.FindByID(app.Tree(), "n").Cursor; got != 1 {
		t.Fatalf("cursor = %d", got)
	}
}

func TestTextWrap(t *testing.T) {
	app, err := LoadString(`<App><Column>
		<Text id="a" value="aa bb cc dd" height="2"/>
		<Text id="b" value="aa bb cc dd" wrap="false"/>
		<Text id="c" value="aa bb cc dd" maxLines="1"/>
	</Column></App>`)
	if err != nil {
		t.Fatal(err)
	}
	rows := app.RenderCells(6, 8)
	// Wrapped text occupies two rows; wrap=false and maxLines=1 stay
	// on a single clipped row each.
	want := []string{"aa bb", "cc dd", "aa bb", "aa bb"}
	for i, w := range want {
		if rows[i] != w {
			t.Fatalf("row %d = %q, want %q:\n%s", i, rows[i], w, strings.Join(rows, "\n"))
		}
	}
	for _, row := range rows[4:] {
		if strings.TrimSpace(row) != "" {
			t.Fatalf("overflow row %q:\n%s", row, strings.Join(rows, "\n"))
		}
	}
}

func TestInputEcho(t *testing.T) {
	app, err := LoadString(`<App><Input id="p" bind="pw" echo="*"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	typeRunes(t, app, "secret")
	if got := app.Get("pw"); got != "secret" {
		t.Fatalf("store = %q", got)
	}
	rows := strings.Join(app.RenderCells(20, 5), "\n")
	if !strings.Contains(rows, "******") {
		t.Fatalf("masked missing:\n%s", rows)
	}
	if strings.Contains(rows, "secret") {
		t.Fatalf("leaked secret:\n%s", rows)
	}
}

func TestMultilineInput(t *testing.T) {
	app, err := LoadString(`<App><Input id="m" bind="notes" multiline="true" height="5"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	typeRunes(t, app, "a")
	app.HandleKey(tcell.KeyEnter, 0, 0)
	typeRunes(t, app, "b")
	if got := app.Get("notes"); got != "a\nb" {
		t.Fatalf("notes = %q", got)
	}
	rows := strings.Join(app.RenderCells(20, 7), "\n")
	if !strings.Contains(rows, "a") || !strings.Contains(rows, "b") {
		t.Fatalf("lines missing:\n%s", rows)
	}
}
