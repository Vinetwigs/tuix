# TUIX tag reference

This is the canonical reference for `.tuix` markup. It is updated
whenever markup changes.

A document is XML with a mandatory `<App>` root:

```xml
<App title="My app">
  ...
</App>
```

Tag names are case-insensitive: `<button>` works like `<Button>`.
Unknown tags are a load error naming the source line.

## Shared attributes

These work on every component.

### Layout

| Attribute | Values | Default | Meaning |
|---|---|---|---|
| `width` | cells, `auto`, `fill` | `auto` | Fixed width, content size, or share of leftover space |
| `height` | cells, `auto`, `fill` | `auto` | Same vertically |
| `flex` | integer | `0` | Weight when sharing leftover space (implies fill-like behavior) |
| `align` | `start`, `center`, `end` | `start` | Cross-axis alignment inside `Row`/`Column` |
| `gap` | integer | `0` | Spacing between children (`Row`, `Column`, `Grid`, `Box`) |
| `padding` | integer | `0` | Inner margin on every side (containers) |
| `id` | string | — | Identifier: focus order, `FindByID`, event `TargetID` |

### Style

| Attribute | Values | Default | Meaning |
|---|---|---|---|
| `fg` | color | default | Foreground: `black red green yellow blue magenta cyan white gray default`, or `#rrggbb` |
| `bg` | color | default | Same for the background |
| `bold` `dim` `italic` `underline` `blink` `reverse` | `true`/`false` | `false` | Text attributes |

### Text interpolation

Any `value`, `label`, or `title` may embed `{key}` placeholders,
replaced with live store values (`{{` and `}}` for literal braces).

## App

```xml
<App title="My app"> ... </App>
```

The document root. `title` is informational (exposed as
`App.Title()`). Exactly one root element is required.

## State

```xml
<State key="name" value="ada"/>
<State key="bio">Free text works too.</State>
```

Seeds the reactive store. `<State>` nodes are consumed at load and
never rendered. A missing `key` is a load error with the line
number. Only direct children of `<App>` are treated as state.

## Column

```xml
<Column gap="1" padding="1">
  <Text value="one"/>
  <Text value="two"/>
</Column>
```

Stacks children vertically. Each child keeps its desired height
(explicit `height`, content size for leaves, recursive sum for
containers); `fill`/`flex` children split whatever space is left.
A lone container child fills the column.

## Row

```xml
<Row gap="2">
  <Button id="a" label="One" width="fill"/>
  <Button id="b" label="Two" width="fill"/>
</Row>
```

Places children side by side. Children default to equal shares;
`width` fixes a child, `flex` weights the sharing. Child height is
the row height unless a smaller fixed `height` is given.

## Box

```xml
<Box title="Profile {name}" padding="1" gap="0">
  ...
</Box>
```

A bordered box with an optional title (interpolated). The border
costs one cell per side; `border="false"` disables it. Children
stack vertically, each sized to content; a single child fills the
box. Overflowing content is clipped, never overlapping.

## Grid

```xml
<Grid columns="3" gap="1">
  ...
</Grid>
```

Distributes children into `columns` equal cells, row by row.
`columns` defaults to `2`; values below 1 count as 1. Desired
height is rows times the tallest child plus gaps and padding.

## Text

```xml
<Text value="Hello {name}!" fg="green" bold="true"/>
<Text>Free text between tags also works.</Text>
<Text value="long..." wrap="false"/>
<Text value="long..." maxLines="2" height="2"/>
```

| Attribute | Default | Meaning |
|---|---|---|
| `value` | inner text | The text (interpolated) |
| `wrap` | `true` | Word-wrap at the component width; `false` keeps one clipped line |
| `maxLines` | `0` (fit height) | Cap on rendered lines |

Wrapped text needs an explicit `height`; layout sizes plain text to
one row and will not guess wrapped height. Only the rectangle's rows
are drawn; the rest is clipped.

## Button

```xml
<Button id="ok" label="OK" onActivate="onOk"/>
<Button id="quit" label="Quit" quit="true"/>
```

Focusable. `Enter`/`Space` (or left click) emits `Activate` with
`Name` taken from `onActivate` (alias `onClick`) and the button id
as `TargetID`. Without a handler, `quit="true"` makes it exit the
app instead. A focused button renders reversed with `> label <`.

## Input

```xml
<Input id="user" bind="user" placeholder="your name"/>
<Input id="pw" bind="pw" echo="*" placeholder="password"/>
<Input id="bio" bind="bio" multiline="true" height="5"/>
<Input id="q" bind="q" onEnter="onGo" onChange="onType"/>
```

Single-line text field bound to a store key (`bind`). Typing
inserts at the cursor; `Backspace`/`Delete` remove around it;
`Left`/`Right`/`Home`/`End` move it; clicking positions it.
`Up`/`Down` (`Enter` too) hop to the next/previous field; `Tab`
always moves.

| Attribute | Default | Meaning |
|---|---|---|
| `bind` | — | Store key holding the value |
| `value` | — | Seeds the key when it is still empty |
| `placeholder` | — | Shown while the value is empty |
| `onEnter` (`onActivate`) | — | `Activate` handler on `Enter` (single-line) |
| `onChange` | — | `InputChange` handler on every edit, `Value` = new content |
| `echo` | — | Password mask, e.g. `echo="*"` (`true` also means `*`); the stored value stays real |
| `multiline` | `false` | `Enter` inserts a newline instead of confirming; all four arrows move inside the text (`Up`/`Down` leave the field past the first/last line); `Home`/`End` are line-wise; the tail of long texts stays visible |

## List

```xml
<List id="tasks" source="tasks" rows="8"
      onSelect="onMove" onActivate="onPick"
      empty="(nothing here)"/>
```

Selectable, scrollable list. Rows come from the store key named by
`source`, one row per line. `Up`/`Down` move the selection (emits
`Select` with the row in `Value`), `Enter`/`Space` emits `Activate`
the same way, clicking a row selects it (and focuses the list).
The view scrolls so the selection stays visible; `> ` marks it,
reversed while focused.

| Attribute | Default | Meaning |
|---|---|---|
| `source` | — | Store key with `\n`-separated rows |
| `rows` | `5` | Visible rows (plus 2 with `border="true"`) |
| `border` | `false` | Draw a box around the list (`title` supported) |
| `onSelect` | — | `Select` handler on selection change |
| `onActivate` (`onClick`) | — | `Activate` handler on confirm |
| `empty` | `(empty)` | Shown when there are no rows |

## Spacer

```xml
<Spacer/>
```

One empty row. Mostly useful with `flex`/`fill` siblings to push
content around.

## Divider

```xml
<Divider/>
<Divider char="#" label="Section two"/>
<Row height="5">
  <Text value="left" width="fill"/>
  <Divider vertical="true"/>
  <Text value="right" width="fill"/>
</Row>
```

A rule between sections. Horizontal by default (one row, full
width); `vertical="true"` draws a column instead, one cell wide.
The optional `label` is interpolated and centered over the line.

| Attribute | Default | Meaning |
|---|---|---|
| `char` | `-`, or pipe when vertical | Line character |
| `label` | — | Text centered on the rule (horizontal only) |
| `vertical` | `false` | Draw a vertical rule, one cell wide |

## ProgressBar

```xml
<ProgressBar bind="percent" max="100"/>
<ProgressBar value="3" max="11" fill="#" empty="." showValue="false"/>
```

A filled bar driven by state. The value comes from `bind` (a store
key) or the literal `value` attribute; it is clamped to `[0, max]`.
One row tall, spans the component width.

| Attribute | Default | Meaning |
|---|---|---|
| `bind` | — | Store key holding the numeric value |
| `value` | `0` | Literal value when there is no `bind` |
| `max` | `100` | Value that fills the bar |
| `fill` | `#` | Filled cell character |
| `empty` | `-` | Empty cell character |
| `brackets` | `true` | Wrap the bar in `[` `]` |
| `showValue` | `true` | Append the ` 42%` label |

## Checkbox

```xml
<Checkbox id="done" bind="done" label="Mark as done" onChange="onDone"/>
```

Focusable boolean toggle bound to a store key. `Enter`/`Space` (or a
left click) flips it, stores `"true"`/`"false"` and emits `Activate`
with `Value` set to the new state and `Name` taken from `onChange`
(alias `onActivate`). Renders `[x] label` / `[ ] label`; when focused
it is highlighted.

| Attribute | Default | Meaning |
|---|---|---|
| `bind` | — | Store key holding `"true"`/`"false"` |
| `value` | — | Seeds the key when it is still empty |
| `label` | — | Text after the checkbox |
| `onChange` (`onActivate`) | — | `Activate` handler on toggle |

## Sparkline

```xml
<Sparkline source="load" height="1"/>
<Sparkline bind="samples" chars=".:-=+*#"/>
```

A one-line chart of a numeric series. The series comes from `source`
(a store key) or `bind`/`value`, and is a list of numbers separated
by spaces, commas, semicolons or newlines. Values are scaled across
the whole series; when there are more values than columns, the most
recent ones win. One row tall.

| Attribute | Default | Meaning |
|---|---|---|
| `source` | — | Store key holding the numeric series |
| `bind` / `value` | — | Fallback source when `source` is absent |
| `chars` | `▁▂▃▄▅▆▇█` | Ramp of characters, low to high |
| `min` / `max` | auto | Pin the scale so several sparklines are comparable |

With no `min`/`max` the scale comes from the series itself, so a flat
series sits on the middle level. Pin both to compare several
sparklines on the same scale, or just one to lock a single side while
the other keeps auto-scaling; values outside a pinned bound are
clamped to the ends.

## Slider

```xml
<Slider id="lvl" bind="level" min="0" max="10" step="1"
        fill="=" empty="-" knob="O" onChange="onLevel"/>
```

Focusable numeric control bound to a store key. One row tall.
`Left`/`Right` change the value by `step`; `Home`/`End` jump to the
bounds; clicking the track jumps to the clicked position. Every change
stores the new value and emits `InputChange` with `Value` set to it and
`Name` taken from `onChange` (alias `onActivate`).

| Attribute | Default | Meaning |
|---|---|---|
| `bind` | — | Store key holding the numeric value |
| `value` | `min` | Seeds the key when it is still empty |
| `min` / `max` | `0` / `100` | Range of the slider |
| `step` | `1` | Increment per key press |
| `fill` / `empty` | `=` / `-` | Track characters before/after the knob |
| `knob` | `O` | Knob character |
| `brackets` | `true` | Wrap the track in `[` `]` |
| `onChange` (`onActivate`) | — | `InputChange` handler on every change |

## BarChart

```xml
<BarChart source="load" height="3" max="100" chars="▁▂▃▄▅▆▇█"/>
```

A multi-row bar chart of a numeric series (`source`, or `bind`/`value`,
numbers separated by spaces, commas, semicolons or newlines). Bars grow
from the bottom row; the ramp provides the partial cell above the last
full block. Height defaults to 3 rows.

| Attribute | Default | Meaning |
|---|---|---|
| `source` | — | Store key holding the numeric series |
| `bind` / `value` | — | Fallback source when `source` is absent |
| `height` | `3` | Chart height in rows |
| `max` | auto (tallest) | Value that fills a column |
| `chars` | `▁▂▃▄▅▆▇█` | Ramp used for partial cells, low to high |

## Table

```xml
<Table columns="NAME,STATUS,TIME" source="servers" widths="10,8,6"
       sep="|" rows="6" empty="(no rows)"/>
```

A read-only table. `columns` is a comma-separated header (interpolated);
rows come from the store key `source`, one row per line, with cells
separated by `sep`. Column widths are shared equally unless `widths`
(a comma-separated list, each clamped to at least one cell) is given.
`rows` caps how many data rows are drawn; `border="true"` adds a frame.
When there are no rows the `empty` placeholder is drawn below the
header.

| Attribute | Default | Meaning |
|---|---|---|
| `columns` | — | Comma-separated header titles |
| `source` | — | Store key with one row per line |
| `sep` | `|` | Cell separator inside each row |
| `widths` | equal split | Comma-separated column widths |
| `rows` | `5` | Maximum data rows drawn |
| `border` | `false` | Draw a frame around the table |
| `empty` | `(no rows)` | Shown when there are no rows |

## Include

```xml
<Include src="card.tuix"/>
```

Replaced at load with the children of the included file's `<App>`.
Paths resolve against the including file's directory; nesting is
capped (cycles fail loudly). The included file must itself be a
valid document with an `<App>` root.

## Custom components

```go
tuix.RegisterComponent("Badge", func(
    attrs map[string]string,
    kids []*component.Component,
) (*component.Component, error) {
    return &component.Component{
        Type: "Badge", Attrs: attrs, Children: kids,
    }, nil
})
```

```xml
<Badge label="new"/>
```

Registered names become usable tags. Built-in names cannot be
overridden. The renderer draws unknown custom types by rendering
their children; a custom type that wants its own visuals needs
renderer support (see `App.RenderCells` / `renderToScreen`).

## Errors

Load failures name the source line: unbalanced or unclosed tags,
extra closing tags, a non-`<App>` root, `<State>` without a key,
and unknown component types. Raw XML syntax errors already carry
their line via `encoding/xml`.
