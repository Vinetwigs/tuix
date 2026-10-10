package tuix

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/Vinetwigs/tuix/component"
	"github.com/Vinetwigs/tuix/event"
)

func TestRuneHelpers(t *testing.T) {
	if got := firstRune("", 'x'); got != 'x' {
		t.Fatalf("empty = %q", got)
	}
	if got := firstRune("ab", 'x'); got != 'a' {
		t.Fatalf("first = %q", got)
	}
	if atof("2.5", 0) != 2.5 || atof("nope", 7) != 7 {
		t.Fatal("atof")
	}
	for s, want := range map[string]bool{
		"true": true, "TRUE": true, "1": true, "yes": true, "on": true,
		"false": false, "0": false, "": false, "nope": false,
	} {
		if boolValue(s) != want {
			t.Errorf("boolValue(%q) = %v", s, !want)
		}
	}
}

func TestProgressBarFormat(t *testing.T) {
	// 14 cells of bar, brackets and a " 50%" suffix -> width 20.
	if got := progressBar(50, 100, 20, '#', '-', true, true); got != "[#######-------] 50%" {
		t.Fatalf("got %q", got)
	}
	if got := progressBar(0, 100, 20, '#', '-', true, true); got != "[---------------] 0%" {
		t.Fatalf("got %q", got)
	}
	if got := progressBar(100, 100, 20, '#', '-', true, true); got != "[#############] 100%" {
		t.Fatalf("got %q", got)
	}
	// Out-of-range values clamp.
	if got := progressBar(150, 100, 10, '#', '-', false, false); got != "##########" {
		t.Fatalf("clamp high: %q", got)
	}
	if got := progressBar(-5, 100, 10, '#', '-', false, false); got != "----------" {
		t.Fatalf("clamp low: %q", got)
	}
}

func TestSparklineFormat(t *testing.T) {
	ramp := []rune("\u2581\u2582\u2583\u2584\u2585\u2586\u2587\u2588")
	if got := sparkline("0 1 2 3 4 5", 6, ramp, 0, 0, true); got != "\u2581\u2582\u2583\u2585\u2586\u2588" {
		t.Fatalf("got %q", got)
	}
	if got := sparkline("", 6, ramp, 0, 0, true); got != "" {
		t.Fatalf("empty = %q", got)
	}
	// A flat series sits on the middle level.
	if got := sparkline("5 5 5", 6, ramp, 0, 0, true); got != "\u2584\u2584\u2584" {
		t.Fatalf("flat = %q", got)
	}
	// Fixed scale: 0..10 maps 5 to the middle level.
	if got := sparkline("5 5 5", 3, ramp, 0, 10, false); got != "\u2584\u2584\u2584" {
		t.Fatalf("fixed = %q", got)
	}
	// Fixed scale: values above max clamp to the top level.
	if got := sparkline("20", 1, ramp, 0, 10, false); got != "\u2588" {
		t.Fatalf("clamp = %q", got)
	}
	// More values than width: keep the most recent ones.
	if got := sparkline("1 2 3 4 5 6", 2, ramp, 0, 0, true); len([]rune(got)) != 2 {
		t.Fatalf("trim = %q", got)
	}
}

func TestDividerRender(t *testing.T) {
	app, err := LoadString(`<App><Divider char="#" label="hi"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	rows := app.RenderCells(11, 4)
	if rows[0] != "### hi ####" {
		t.Fatalf("divider = %q", rows[0])
	}
}

func TestDividerVertical(t *testing.T) {
	app, err := LoadString(`<App><Row height="3"><Text value="ab"/><Divider vertical="true"/><Text value="cd"/></Row></App>`)
	if err != nil {
		t.Fatal(err)
	}
	rows := app.RenderCells(6, 3)
	if !strings.Contains(rows[1], "|") {
		t.Fatalf("no vertical rule:\n%s", strings.Join(rows, "\n"))
	}
}

func TestProgressBarRender(t *testing.T) {
	app, err := LoadString(`<App><ProgressBar bind="p" max="100"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	app.Set("p", "50")
	rows := app.RenderCells(20, 3)
	if !strings.Contains(rows[0], "50%") || !strings.HasPrefix(rows[0], "[") {
		t.Fatalf("bar = %q", rows[0])
	}
	if rows[1] != "" {
		t.Fatalf("progress bar is not one row: %q", rows[1])
	}
}

func TestCheckboxToggle(t *testing.T) {
	app, err := LoadString(`<App><Checkbox id="c" bind="done" label="Done" onChange="changed"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	app.On("changed", func(ev event.Event) { seen = append(seen, ev.Value) })

	// Focused (only focusable) and unchecked.
	if rows := app.RenderCells(20, 3); !strings.Contains(rows[0], "[ ] Done") {
		t.Fatalf("initial = %q", rows[0])
	}
	app.HandleKey(tcell.KeyEnter, 0, 0)
	if app.Get("done") != "true" {
		t.Fatalf("store = %q", app.Get("done"))
	}
	if rows := app.RenderCells(20, 3); !strings.Contains(rows[0], "[x] Done") {
		t.Fatalf("after toggle = %q", rows[0])
	}
	// Click toggles back.
	r := app.Layout(20, 3)[app.Tree().Children[0]]
	if !app.ClickAt(r.X, r.Y, 20, 3) {
		t.Fatal("click not handled")
	}
	if app.Get("done") != "false" {
		t.Fatalf("after click = %q", app.Get("done"))
	}
	if len(seen) != 2 || seen[0] != "true" || seen[1] != "false" {
		t.Fatalf("events = %v", seen)
	}
}

func TestSparklineRender(t *testing.T) {
	app, err := LoadString(`<App><Sparkline source="series" width="6"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	app.Set("series", "0 1 2 3 4 5")
	rows := app.RenderCells(6, 2)
	if !strings.Contains(rows[0], "\u2581") || !strings.Contains(rows[0], "\u2588") {
		t.Fatalf("sparkline = %q", rows[0])
	}
}

func TestVisualWidgetsStack(t *testing.T) {
	app, err := LoadString(`<App>
		<ProgressBar bind="p"/>
		<Checkbox id="c" bind="done" label="Done"/>
		<Sparkline source="series"/>
		<Divider/>
	</App>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(app.focus) != 1 {
		t.Fatalf("focusables = %d", len(app.focus))
	}
	rects := app.Layout(20, 10)
	root := app.Tree()
	for i, child := range root.Children {
		r, ok := rects[child]
		if !ok {
			t.Fatalf("child %d has no rect", i)
		}
		if r.H != 1 {
			t.Fatalf("child %d (%s) height = %d, want 1", i, child.Type, r.H)
		}
		if r.Y != i {
			t.Fatalf("child %d Y = %d, want %d", i, r.Y, i)
		}
	}
}

func TestSliderFormat(t *testing.T) {
	got := sliderBar(50, 0, 100, 20, '=', '-', 'O', true)
	if got != "[=======O-------] 50" {
		t.Fatalf("slider = %q", got)
	}
	if got := sliderBar(0, 0, 100, 20, '=', '-', 'O', false); got != "O----------------- 0" {
		t.Fatalf("min = %q", got)
	}
}

func TestSliderKeyboardAndSteps(t *testing.T) {
	app, err := LoadString(`<App><Slider id="s" bind="lvl" min="0" max="10" step="2"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	if app.Focused() == nil || app.Focused().Type != component.TypeSlider {
		t.Fatal("slider not focused")
	}
	app.HandleKey(tcell.KeyRight, 0, 0)
	if app.Get("lvl") != "2" {
		t.Fatalf("after right = %q", app.Get("lvl"))
	}
	app.HandleKey(tcell.KeyRight, 0, 0)
	if app.Get("lvl") != "4" {
		t.Fatalf("after right2 = %q", app.Get("lvl"))
	}
	app.HandleKey(tcell.KeyLeft, 0, 0)
	if app.Get("lvl") != "2" {
		t.Fatalf("after left = %q", app.Get("lvl"))
	}
	app.HandleKey(tcell.KeyEnd, 0, 0)
	if app.Get("lvl") != "10" {
		t.Fatalf("after end = %q", app.Get("lvl"))
	}
	app.HandleKey(tcell.KeyHome, 0, 0)
	if app.Get("lvl") != "0" {
		t.Fatalf("after home = %q", app.Get("lvl"))
	}
	// Cannot go below min.
	app.HandleKey(tcell.KeyLeft, 0, 0)
	if app.Get("lvl") != "0" {
		t.Fatalf("clamped = %q", app.Get("lvl"))
	}
}

func TestSliderClick(t *testing.T) {
	app, err := LoadString(`<App><Slider id="s" bind="lvl" min="0" max="100"/></App>`)
	if err != nil {
		t.Fatal(err)
	}
	r := app.Layout(40, 3)[app.Tree().Children[0]]
	if !app.ClickAt(r.Right()-1, r.Y, 40, 3) {
		t.Fatal("click not handled")
	}
	if app.Get("lvl") != "100" {
		t.Fatalf("click far right = %q", app.Get("lvl"))
	}
}

func TestBarChartFormat(t *testing.T) {
	ramp := []rune("\u2581\u2582\u2583\u2584\u2585\u2586\u2587\u2588")
	got := barChart("1 2 3 4", 4, 3, ramp, 0)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("lines = %d: %q", len(lines), got)
	}
	if !strings.Contains(got, "\u2588") {
		t.Fatalf("no full block:\n%q", got)
	}
	// The tallest column reaches the chart height.
	if strings.Count(lines[0], " ") == len([]rune(lines[0])) {
		t.Fatalf("top row empty:\n%q", got)
	}
	if got := barChart("", 4, 3, ramp, 0); got != "" {
		t.Fatalf("empty = %q", got)
	}
}

func TestTableFormat(t *testing.T) {
	header := []string{"NAME", "STATUS"}
	rows := [][]string{{"nginx", "up"}, {"redis", "down"}}
	got := tableLines(header, rows, nil, 20, 5)
	if len(got) != 4 { // header + rule + 2 rows
		t.Fatalf("lines = %d: %q", len(got), got)
	}
	if !strings.HasPrefix(got[1], "----") {
		t.Fatalf("rule = %q", got[1])
	}
	if !strings.Contains(got[2], "nginx") || !strings.Contains(got[3], "redis") {
		t.Fatalf("rows = %q", got)
	}
}

func TestNewVisualTagsRender(t *testing.T) {
	app, err := LoadString(`<App>
		<Slider id="lvl" bind="lvl" min="0" max="10"/>
		<BarChart source="bars" width="8" height="3"/>
		<Table columns="NAME,STATUS" source="svc" widths="8,6"/>
		<Sparkline source="series" min="0" max="10"/>
	</App>`)
	if err != nil {
		t.Fatal(err)
	}
	app.Set("lvl", "5")
	app.Set("bars", "1 3 5 7")
	app.Set("svc", "nginx|up\nredis|down")
	app.Set("series", "2 4 6 8 10 8 6 4")
	rows := strings.Join(app.RenderCells(20, 12), "\n")
	for _, want := range []string{"O", "NAME", "nginx", "redis", "\u2588"} {
		if !strings.Contains(rows, want) {
			t.Fatalf("missing %q:\n%s", want, rows)
		}
	}
}
