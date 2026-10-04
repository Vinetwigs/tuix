// Command demo runs a small interactive TUIX application.
//
// Usage (from the module root):
//
//	go run ./examples/demo
//
// It also works from inside examples/demo. Keys: Tab/Shift+Tab move
// focus, arrows move focus, Enter presses a button (or confirms an
// input), typing edits the focused input, Esc or the Quit button exits.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Vinetwigs/tuix"
	"github.com/Vinetwigs/tuix/event"
)

// findUI locates demo.tuix trying, in order: the current working
// directory layouts and the source file directory (so the demo works
// no matter where it is launched from). It returns the first path
// that loads successfully, or all encountered errors.
func findUI() (*tuix.App, string, []error) {
	cands := []string{"examples/demo/demo.tuix", "demo.tuix"}
	if _, file, _, ok := runtime.Caller(0); ok {
		cands = append(cands, filepath.Join(filepath.Dir(file), "demo.tuix"))
	}
	var errs []error
	for _, p := range cands {
		app, err := tuix.LoadFile(p)
		if err == nil {
			return app, p, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", p, err))
	}
	return nil, "", errs
}

func main() {
	app, _, errs := findUI()
	if app == nil {
		fmt.Fprintln(os.Stderr, "load: could not find a valid demo.tuix:")
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "  -", e)
		}
		os.Exit(1)
	}

	app.Computed("shout", func(get func(string) string) string {
		return strings.ToUpper(get("name")) + "!"
	})

	app.On("onClick", func(ev event.Event) {
		n, _ := strconv.Atoi(app.Get("clicks"))
		app.Set("clicks", strconv.Itoa(n+1))
	})
	app.On("onReset", func(ev event.Event) {
		app.Set("clicks", "0")
		app.Set("name", "ada")
	})

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
	fmt.Println("Bye! clicks =", app.Get("clicks"))
}
