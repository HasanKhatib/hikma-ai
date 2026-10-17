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
