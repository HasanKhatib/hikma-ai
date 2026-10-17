package ui

import (
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

const (
	yellow = "\x1b[33m"
	red    = "\x1b[31m"
	green  = "\x1b[32m"
	cyan   = "\x1b[36m"
	bold   = "\x1b[1m"
	dim    = "\x1b[2m"
	reset  = "\x1b[0m"
)

// colorOK reports whether w is a terminal that should get color. NO_COLOR
// (https://no-color.org) and TERM=dumb turn it off; output that is piped or
// captured never gets escape codes.
func colorOK(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	return ok && (isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd()))
}

func paint(w io.Writer, code, s string) string {
	if !colorOK(w) {
		return s
	}
	return code + s + reset
}

// Yellow colors s for warnings when w is a color-capable terminal.
func Yellow(w io.Writer, s string) string { return paint(w, yellow, s) }

// Red colors s for errors and failures when w is a color-capable terminal.
func Red(w io.Writer, s string) string { return paint(w, red, s) }

// Green colors s for success when w is a color-capable terminal.
func Green(w io.Writer, s string) string { return paint(w, green, s) }

// Cyan colors s for commands and paths when w is a color-capable terminal.
func Cyan(w io.Writer, s string) string { return paint(w, cyan, s) }

// Bold makes s bold when w is a color-capable terminal.
func Bold(w io.Writer, s string) string { return paint(w, bold, s) }

// Dim fades s for secondary text when w is a color-capable terminal.
func Dim(w io.Writer, s string) string { return paint(w, dim, s) }

// Warning is the "warning:" prefix, in yellow on a terminal.
func Warning(w io.Writer) string { return Yellow(w, "warning:") }

// PrintWarning prints a line that starts with "warning:", coloring the prefix.
func PrintWarning(w io.Writer, line string) {
	if rest, ok := strings.CutPrefix(line, "warning:"); ok {
		line = Warning(w) + rest
	}
	io.WriteString(w, line+"\n")
}
