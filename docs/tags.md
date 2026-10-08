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

Wrapped text needs an explicit `height`;  layout sizes plain text to
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
