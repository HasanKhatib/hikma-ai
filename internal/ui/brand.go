package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// The banner draws the logo's frame and prompt in box characters, with a Latin
// wordmark beside it. The logo's calligraphy does not survive conversion to
// terminal art, so the Arabic is left to the docs site and the README.
var bannerArt = []string{
	"╭──────────╮",
	"│ >_       │",
	"│          │",
	"╰──────────╯",
}

var bannerText = []string{
	"",
	"hikma",
	"skills follow you",
	"",
}

// logo gradient: teal at the top to blue at the bottom, as in the logo.
var gradient = [][3]int{{94, 234, 212}, {70, 214, 200}, {56, 189, 248}, {56, 160, 248}}

func trueColor() bool {
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		return true
	}
	return false
}

func gradientPaint(w io.Writer, row int, s string) string {
	if !colorOK(w) {
		return s
	}
	if trueColor() {
		c := gradient[row%len(gradient)]
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s%s", c[0], c[1], c[2], s, reset)
	}
	if row < len(gradient)/2 {
		return green + s + reset
	}
	return cyan + s + reset
}

// Banner returns the logo and wordmark, one string with a trailing newline.
// It carries color only when w is a color-capable terminal.
func Banner(w io.Writer) string {
	var b strings.Builder
	for i, art := range bannerArt {
		b.WriteString(gradientPaint(w, i, art))
		switch text := bannerText[i]; {
		case i == 1:
			b.WriteString("  " + Bold(w, text))
		case text != "":
			b.WriteString("  " + Dim(w, text))
		}
		b.WriteString("\n")
	}
	return b.String()
}
