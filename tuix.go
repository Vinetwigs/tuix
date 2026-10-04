// Package tuix is the public API of TUIX, a declarative TUI framework.
//
// A UI is declared in a .tuix XML file and loaded with LoadFile or
// LoadString. Application logic lives in Go: state via Set/Get,
// derived values via Computed, and handlers via On. Run starts the
// terminal event loop backed by tcell.
//
// Minimal example:
//
//	app, err := tuix.LoadString(`<App><Text value="Hi {name}"/></App>`)
//	if err != nil { ... }
//	app.Set("name", "ada")
//	app.On("save", func(ev event.Event) { ... })
//	if err := app.Run(); err != nil { ... }
package tuix

import (
	"github.com/Vinetwigs/tuix/component"
)

// RegisterComponent registers a custom component type usable as a tag
// in .tuix markup. See component.Register for details.
func RegisterComponent(name string, f component.Factory) error {
	return component.Register(name, f)
}
