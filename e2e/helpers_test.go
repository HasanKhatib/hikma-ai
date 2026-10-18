package e2e

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)`)

// plain strips terminal control sequences so tests can match visible text.
func plain(s string) string { return ansi.ReplaceAllString(s, "") }

func (tm *term) waitFor(substr string, d time.Duration) {
	tm.t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if strings.Contains(plain(tm.output()), substr) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	tm.t.Fatalf("timed out after %s waiting for %q; screen so far:\n%s", d, substr, plain(tm.output()))
}

const (
	down  = "\x1b[B"
	enter = "\r"
	ctrlC = "\x03"
)

// settle gives a freshly drawn prompt time to start reading input. On Linux a
// key sent the instant the text appears can be dropped while the program is
// still switching the terminal to raw mode.
func (tm *term) settle() { time.Sleep(500 * time.Millisecond) }

// press sends keys one at a time with a short gap, like a person typing, so no
// key arrives in the same read as the last.
func (tm *term) press(keys ...string) {
	tm.t.Helper()
	for _, k := range keys {
		tm.write(k)
		time.Sleep(40 * time.Millisecond)
	}
}

// colorCode matches the SGR codes hikma uses for color, bold, and dim. The
// terminal layer may add its own cursor and title sequences, so tests look for
// these specifically rather than for any escape code.
var colorCode = regexp.MustCompile(`\x1b\[(?:[0-9]+;)*(?:1|2|3[0-7]|9[0-7])m`)

func hasColor(s string) bool { return colorCode.MatchString(s) }

func skipShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping terminal tests in -short mode")
	}
}

// The menu redraws only the lines that change, so a moved cursor shows up as
// "> Option  Option" without the left bar. Both forms end the label at a double space or line end.
var cursorLine = regexp.MustCompile(`(?:┃ )?> ([A-Z][^\r\n]*?)(?:  |\r|\n|$)`)

// selected returns the option the menu cursor is on in the latest frame, or "".
func (tm *term) selected() string {
	m := cursorLine.FindAllStringSubmatch(plain(tm.output()), -1)
	if len(m) == 0 {
		return ""
	}
	return strings.TrimSpace(m[len(m)-1][1])
}

// choose moves down the list until label is under the cursor, then presses Enter.
// It checks the screen after each key instead of trusting that keys sent early
// reached the program: on a Linux pty the first ones can be dropped while the
// program is still setting the terminal up.
func (tm *term) choose(label string, d time.Duration) {
	tm.t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if tm.selected() == label {
			tm.press(enter)
			return
		}
		tm.press(down)
		time.Sleep(150 * time.Millisecond)
	}
	tm.t.Fatalf("could not move the cursor to %q in %s; it is on %q; screen:\n%s", label, d, tm.selected(), plain(tm.output()))
}

// typeText types s and waits until it shows on screen, so Enter does not race the echo.
func (tm *term) typeText(s string, d time.Duration) {
	tm.t.Helper()
	tm.write(s)
	tm.waitFor(s, d)
}
