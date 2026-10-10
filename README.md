# tuix

[![CI](https://github.com/Vinetwigs/tuix/actions/workflows/go.yml/badge.svg)](https://github.com/Vinetwigs/tuix/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Vinetwigs/tuix.svg)](https://pkg.go.dev/github.com/Vinetwigs/tuix)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

TUIX is a small Go library for building terminal user interfaces
declaratively. You describe the UI in `.tuix` files (plain XML),
write the application logic in Go, and the library handles layout,
state updates, and input. Rendering and keyboard/mouse handling are
built on [tcell](https://github.com/gdamore/tcell).

```xml
<App title="Demo">
  <State key="name" value="ada"/>
  <Column gap="1" padding="1">
    <Text value="Hello {name}!" fg="green" bold="true"/>
    <Input id="name-in" bind="name" placeholder="type your name"/>
    <Button id="ok" label="OK" onActivate="onOk"/>
  </Column>
</App>
```

```go
app, err := tuix.LoadFile("ui.tuix")
if err != nil {
    log.Fatal(err)
}
app.On("onOk", func(ev event.Event) {
    fmt.Println("hello,", app.Get("name"))
})
if err := app.Run(); err != nil {
    log.Fatal(err)
}
```

> Status: early development (`v0.2.0`). It builds, it runs, and it is
> tested but the API has only been used by its own examples so far
> and may still change before `v1`. Tested on Windows; Linux and macOS
> should work through tcell but have not been tried yet.

## Requirements

Go 1.24 or newer. One dependency:

```sh
go get github.com/Vinetwigs/tuix
```

## Quick start

1. Write a `ui.tuix` file with an `<App>` root (see above).
2. Load it with `tuix.LoadFile` (or `tuix.LoadString` for inline markup).
3. Seed state with `app.Set`, derive values with `app.Computed`,
   attach behavior with `app.On`.
4. Call `app.Run()` to start the terminal loop.

Run the bundled demo to see it in action. It needs a real terminal;
an IDE output pane will not do, tcell takes over the screen:

```sh
go run ./examples/demo
```

Keys: `Tab` / `Shift+Tab` move focus, arrows navigate, `Enter`
confirms, typing edits the focused field, `Esc` or `Ctrl+C` quits.
Inputs support mid-text cursor movement (`Left`/`Right`,
`Home`/`End`, `Delete`, click to position). The left mouse button
presses buttons and focuses inputs and list rows.

## Examples

| Directory | What it shows | Run |
|---|---|---|
| `examples/demo` | Form with bindings, includes, quit button | `go run ./examples/demo` |
| `examples/showcase` | Dashboard: stats grid, login form, clickable keypad | `go run ./examples/showcase` |
| `examples/todo` | Persistent todo list, autosaved to `tasks.json` | `go run ./examples/todo` |
| `examples/studio` | `.tuix` playground: edit snippets, live validation with line numbers, AST outline, password-gated publishing | `go run ./examples/studio` |
| `examples/dash` | Live CI/CD dashboard: progress bars, sparkline and bar chart, run table, threshold slider, toggles | `go run ./examples/dash` |

## Markup

The full tag reference lives in [`docs/tags.md`](docs/tags.md).
The short version:

- **Containers:** `Column`, `Row` (with `gap`, `padding`), `Box`
  (border + `title`), `Grid` (`columns`). Sizing via `width`,
  `height` (`auto`, `fill`, or cells), `flex` weights, and `align`.
- **Widgets:** `Text` (`value`, word `wrap`, `maxLines`), `Button`
  (`label`, `onActivate`), `Input` (`bind`, `placeholder`,
  `onChange`/`onEnter`, `echo` for passwords, `multiline`),
  `List` (`source` key with one row per line, `rows`, `onSelect`,
  `empty` placeholder), `Spacer`.
- **Visual elements:** `Divider` (rule with optional `label`),
  `ProgressBar` (`bind`, `max`), `Checkbox` (`bind`, `label`),
  `Slider` (`bind`, `min`/`max`/`step`), `Sparkline` and `BarChart`
  (numeric series charts), `Table` (`columns`, `source`).
- **State:** `<State key="..." value="..."/>` seeds the store;
  `{key}` anywhere in text interpolates the current value and
  re-renders on change. `{{` and `}}` escape braces.
- **Reuse:** `<Include src="other.tuix"/>` splices another file in;
  Go code can register fully custom components with
  `tuix.RegisterComponent`.
- **Style:** `fg`/`bg` (named colors or `#rrggbb`) plus `bold`,
  `dim`, `italic`, `underline`, `blink`, `reverse`.

Markup errors point at the offending line (`line N`), both for bad
XML and for unknown component types.

## Go API

```go
// Loading
app, err := tuix.LoadFile("ui.tuix")
app, err := tuix.LoadString(`<App>...</App>`)

// Reactive state
app.Set("name", "grace")
v := app.Get("name") // computed keys included
app.Computed("shout", func(get func(string) string) string {
    return strings.ToUpper(get("name"))
})

// Events: named handlers from markup, "*" observes everything
app.On("onOk", func(ev event.Event) { ... })
app.On("*", func(ev event.Event) { ... })
app.Emit(event.Event{Type: event.Custom, Name: "tick"})

// Focus, mouse, lifecycle
app.Focused()
app.FocusNext()
app.FocusPrev()
app.SetFocus(0)
app.ClickAt(x, y, width, height) // headless/testing clicks
app.Notify() // wake Run loop after background Set calls
app.Run()    // blocking tcell loop (keyboard + mouse)
app.Quit()
```

`event.Event` carries `Type` (`KeyPress`, `Activate`, `InputChange`,
`FocusChange`, `Select`, `Custom`, `Quit`), the handler `Name`,
`TargetID`, an optional `Value`, and raw `Key`/`Rune`/`Mod` for
keyboard input.

```go
rows := app.RenderCells(80, 24) // one string per row
```

## Layout

`layout.Compute` assigns every component a cell rectangle. Columns
stack, rows place side by side, boxes reserve a border plus padding,
grids split into equal cells. `gap` separates siblings, `fill` and
`flex` share leftover space, and containers size to their content
where the content is known (`Text` that wraps needs an explicit
`height`). Anything that does not fit is clipped, never wrapped by
the layout itself.

## Project structure

```text
tuix.go, app.go   public API
ast/              syntax tree, no dependencies
parser/           XML to AST, <Include> resolution
component/        component tree + custom registry
runtime/          reactive store (bindings, computed values)
layout/           rectangles for row/column/box/grid
style/            colors and attributes to tcell.Style
event/            event types + dispatcher
backend/          thin tcell wrapper (drawing, mouse, wakeup)
docs/tags.md      complete tag reference (updated with every change)
examples/         demo, showcase, todo, studio
```

Markup, runtime, and rendering are kept separate on purpose: the
parser never imports tcell, layout only sees the component tree,
and the backend only draws rectangles and text.

## Developing

```sh
go build ./...
go vet ./...
go test ./...          # or: go test -count=1 ./...
gofmt -l .             # must print nothing
```

Conventions: exported identifiers get doc comments, every markup or
behavior change updates `docs/tags.md` and adds a test.

## License

GPL-3.0-or-later; see [LICENSE](LICENSE).
