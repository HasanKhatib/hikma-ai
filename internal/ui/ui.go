// Package ui provides shared output helpers for consistent CLI formatting.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Out is the default writer for normal output.
var Out io.Writer = os.Stdout

// Err is the default writer for error output.
var Err io.Writer = os.Stderr

// Print writes a line to Out.
func Print(format string, args ...any) {
	fmt.Fprintf(Out, format+"\n", args...)
}

// Error writes a formatted error message to Err and appends the doctor hint.
func Error(format string, args ...any) {
	fmt.Fprintf(Err, "error: "+format+"\n", args...)
	fmt.Fprintf(Err, "run 'hikma doctor' to check your environment\n")
}

// UserError writes a formatted error message to Err without the doctor hint.
// Use this for user-input errors (bad arguments, already installed, etc.)
// where the doctor command is not relevant.
func UserError(format string, args ...any) {
	fmt.Fprintf(Err, "error: "+format+"\n", args...)
}

// Fatal writes an error message to Err and exits with code 1.
func Fatal(format string, args ...any) {
	Error(format, args...)
	os.Exit(1)
}

// Width is how many terminal columns s takes. len counts bytes, which is wrong for
// accented and wide characters.
func Width(s string) int { return ansi.StringWidth(s) }

// Pad returns s padded with spaces to width columns.
func Pad(s string, width int) string {
	if n := width - Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// Truncate cuts s to at most width terminal columns, ending in "..." when it had to
// cut. It counts columns, not bytes or characters, so wide characters fit.
func Truncate(s string, width int) string {
	if Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "...")
}
