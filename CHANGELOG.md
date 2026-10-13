# Changelog

All notable changes to TUIX are recorded here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [0.2.1] - 2026-10-10

Patch: fixes for the visual widgets added in 0.2.0, plus docs polish.
No new features and no breaking changes.

### Fixed

- `Table`: negative or zero `widths` no longer panic; each width is
  clamped to at least one cell.
- `Table`: the `empty` placeholder is now drawn when there are header
  columns but no data rows.
- `Sparkline`: a single `min` or `max` is honoured (the other bound
  still auto-scales) instead of being ignored unless both were set.

## [0.2.0] - 2026-10-10

Visual-widgets release. Adds a set of built-in visual elements and a
dashboard example. No existing markup or API changed.

### Added

- Visual tags: `Divider` (horizontal/vertical rule with centered
  label), `ProgressBar` (bound value with `max` and custom
  fill/empty characters), `Checkbox` (focusable state toggle with
  `onChange`), `Sparkline` (one-line chart with optional `min`/`max`
  scale), `Slider` (focusable numeric control driven by the keyboard
  and the mouse), `BarChart` (multi-row chart of a numeric series),
  and `Table` (read-only columns from a store key).
- `examples/dash`: live dashboard exercising all the
  visual tags with a background simulator and `App.Notify`.

## [0.1.0] - 2026-10-04

First public release. Early development status: the API works and is
tested, but may still change before v1.

### Added

- Declarative `.tuix` markup (XML) with `App`, `State`, `Column`,
  `Row`, `Box`, `Grid`, `Text`, `Button`, `Input`, `List`, `Spacer`,
  and `Include`.
- Reactive store: `{key}` bindings, `Computed` derived values,
  change watchers.
- Layout engine: desired-height sizing, `fill`/`flex` sharing,
  `gap`, `padding`, `align`, box borders, grid cells, clipping.
- Input: mid-text cursor (`Left`/`Right`/`Home`/`End`/`Delete`,
  click to position), `echo` password masking, `multiline` editing
  with 2D cursor movement.
- Text word wrapping with `maxLines`.
- Selectable, scrollable `List` with keyboard, mouse, and `Select`
  events.
- Mouse support: click to press buttons, focus inputs, select
  list rows.
- `App.Notify` for background refresh (timers, watchers).
- Markup errors with source line numbers.
- Headless rendering (`RenderCells`) for tests and screenshots.
- Custom components via `RegisterComponent`.
- Examples: demo, showcase, todo (persistent), studio (playground).
- Full tag reference in `docs/tags.md`.

[Unreleased]: https://github.com/Vinetwigs/tuix/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/Vinetwigs/tuix/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Vinetwigs/tuix/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Vinetwigs/tuix/releases/tag/v0.1.0
