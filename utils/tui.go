package utils

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ColorEnabled reports whether ANSI color output should be produced.
// It honors NO_COLOR, FORCE_COLOR, and TTY detection, matching tui-kit.
func ColorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("FORCE_COLOR") == "0" {
		return false
	}
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func wrapANSI(code, text string) string {
	if !ColorEnabled() {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// Basic style helpers, ported from tui-kit/colors.js.
func Red(s string) string     { return wrapANSI("31", s) }
func Green(s string) string   { return wrapANSI("32", s) }
func Yellow(s string) string  { return wrapANSI("33", s) }
func Blue(s string) string    { return wrapANSI("34", s) }
func Magenta(s string) string { return wrapANSI("35", s) }
func Cyan(s string) string    { return wrapANSI("36", s) }
func White(s string) string   { return wrapANSI("37", s) }
func Gray(s string) string    { return wrapANSI("90", s) }
func Bold(s string) string    { return wrapANSI("1", s) }
func Dim(s string) string     { return wrapANSI("2", s) }

// TerminalWidth returns the current terminal width, or 80 when unavailable.
func TerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

// PrintBanner prints a small colored banner header followed by content.
func PrintBanner(name string, content string) {
	fmt.Println(Cyan(Bold("== " + name + " ==")))
	fmt.Print(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		fmt.Println()
	}
}
