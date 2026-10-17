//go:build windows

package ui

import (
	"os"

	"golang.org/x/sys/windows"
)

// vtSupported turns on ANSI escape processing for a Windows console and
// reports whether it is on. Older consoles refuse, and printing escape codes
// to them shows as garbage, so color stays off there. A handle that is not a
// console at all (mintty, Git Bash) is left to its terminal.
func vtSupported(f *os.File) bool {
	h := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return true
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}
	return windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}
