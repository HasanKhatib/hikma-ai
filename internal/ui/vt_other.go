//go:build !windows

package ui

import "os"

// vtSupported reports whether f understands ANSI escape sequences. Terminals
// on macOS and Linux do; the TTY check has already been made.
func vtSupported(*os.File) bool { return true }
