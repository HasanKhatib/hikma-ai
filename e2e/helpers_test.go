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

func downs(n int) string { return strings.Repeat(down, n) }

func skipShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping terminal tests in -short mode")
	}
}
