// Command dash runs tuicd, a live CI/CD pipeline dashboard.
//
// It exercises the visual tags end to end: ProgressBar for the running
// stages and the overall build, Sparkline for the build-time trend,
// BarChart for the recent builds, Table for the run history,
// Checkbox for runtime toggles, Slider for the alert threshold, and
// Divider to separate sections. A background simulator advances the
// pipeline and refreshes the UI through App.Notify.
//
// Usage (from the module root):
//
//	go run ./examples/dash
//
// Mouse: click the checkboxes and the slider track. Keyboard: Tab
// moves focus, Space/Enter toggles a checkbox, Left/Right (and
// Home/End) move a slider, Esc quits.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Vinetwigs/tuix"
	"github.com/Vinetwigs/tuix/event"
)

const tickRate = 200 * time.Millisecond
const historyCap = 24
const runCap = 6

// pipeline is the pure simulation state; step advances it by one tick.
type pipeline struct {
	build, test, deploy float64
	duration            float64
	lastDuration        float64
	done                int
	history             []float64
	runs                []string
}

// step advances the pipeline by one tick: build, then test, then
// deploy, then it records the cycle duration and starts over. Each
// cycle gets a deterministic jitter so the trend line has shape.
func (p *pipeline) step() {
	const speed = 12.0
	switch {
	case p.build < 100:
		p.build = min(100, p.build+speed)
	case p.test < 100:
		p.test = min(100, p.test+speed)
	case p.deploy < 100:
		p.deploy = min(100, p.deploy+speed)
	default:
		d := p.duration + float64((p.done*7)%11)
		p.lastDuration = d
		p.history = append(p.history, d)
		if len(p.history) > historyCap {
			p.history = p.history[len(p.history)-historyCap:]
		}
		p.build, p.test, p.deploy = 0, 0, 0
		p.duration = 0
		p.done++
	}
	p.duration += 1.4
}

// overall returns the average stage progress in [0, 100].
func (p *pipeline) overall() float64 {
	return (p.build + p.test + p.deploy) / 3
}

// historyText renders the history as a space-separated series.
func (p *pipeline) historyText() string {
	parts := make([]string, len(p.history))
	for i, v := range p.history {
		parts[i] = strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strings.Join(parts, " ")
}

func findUI() (*tuix.App, []error) {
	cands := []string{"examples/dash/dash.tuix", "dash.tuix"}
	if _, file, _, ok := runtime.Caller(0); ok {
		cands = append(cands, filepath.Join(filepath.Dir(file), "dash.tuix"))
	}
	var errs []error
	for _, p := range cands {
		if app, err := tuix.LoadFile(p); err == nil {
			return app, nil
		} else {
			errs = append(errs, fmt.Errorf("%s: %w", p, err))
		}
	}
	return nil, errs
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func number(s string, def float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return v
}

// appendLog pushes a timestamped line, keeping the last few.
func appendLog(app *tuix.App, line string) {
	lines := strings.Split(strings.TrimSpace(app.Get("log")), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	lines = append(lines, time.Now().Format("15:04:05")+" "+line)
	if len(lines) > 2 {
		lines = lines[len(lines)-2:]
	}
	app.Set("log", strings.Join(lines, "\n"))
}

// publish copies the pipeline state into the store. When compact is on
// the series are withheld so the Sparkline and BarChart collapse.
func publish(app *tuix.App, p *pipeline) {
	app.Set("build", strconv.FormatFloat(p.build, 'f', 0, 64))
	app.Set("test", strconv.FormatFloat(p.test, 'f', 0, 64))
	app.Set("deploy", strconv.FormatFloat(p.deploy, 'f', 0, 64))
	app.Set("overall", strconv.FormatFloat(p.overall(), 'f', 0, 64))
	if truthy(app.Get("compact")) {
		app.Set("history", "")
		app.Set("bars", "")
	} else {
		h := p.historyText()
		app.Set("history", h)
		app.Set("bars", h)
	}
}

// recordRun appends the finished run to the table when a cycle ends.
func recordRun(app *tuix.App, p *pipeline) {
	thr := number(app.Get("threshold"), 30)
	status := "ok"
	if p.lastDuration > thr {
		status = "SLOW"
		appendLog(app, fmt.Sprintf("build %.0fs over threshold %.0fs", p.lastDuration, thr))
	}
	line := fmt.Sprintf("%d|%s|%.0fs", p.done, status, p.lastDuration)
	p.runs = append([]string{line}, p.runs...)
	if len(p.runs) > runCap {
		p.runs = p.runs[:runCap]
	}
	app.Set("runs", strings.Join(p.runs, "\n"))
}

// simulate runs the pipeline forward, pausing while the Pause box is on.
func simulate(app *tuix.App, p *pipeline) {
	t := time.NewTicker(tickRate)
	go func() {
		for now := range t.C {
			prev := p.done
			if !truthy(app.Get("pause")) {
				p.step()
			}
			publish(app, p)
			if p.done != prev {
				recordRun(app, p)
			}
			app.Set("clock", now.Format("15:04:05"))
			app.Set("status", fmt.Sprintf("cycle %d - %d runs recorded", p.done+1, len(p.history)))
			app.Notify()
		}
	}()
}

func main() {
	app, errs := findUI()
	if app == nil {
		fmt.Fprintln(os.Stderr, "load: could not find a valid dash.tuix:")
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "  -", e)
		}
		os.Exit(1)
	}
	p := &pipeline{}
	publish(app, p)

	app.On("onToggle", func(ev event.Event) {
		on := truthy(ev.Value)
		state := "off"
		if on {
			state = "on"
		}
		switch ev.TargetID {
		case "pause":
			if on {
				app.Set("status", "paused")
			} else {
				app.Set("status", "resumed")
			}
			appendLog(app, "pause "+state)
		case "compact":
			appendLog(app, "compact view "+state)
		case "notify":
			appendLog(app, "notifications "+state)
		}
	})
	app.On("onThreshold", func(ev event.Event) {
		app.Set("status", "alert threshold "+ev.Value+"s")
	})

	simulate(app, p)
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
	fmt.Println("Bye! completed cycles:", app.Get("status"))
}
