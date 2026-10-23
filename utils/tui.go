package utils

import (
	"fmt"
	"os"
	"time"

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

// StartSpinner shows an animated spinner using beNotAfraid.txt characters (or a
// default set) on stderr while a long-running operation executes. It returns a
// stop function that clears the line.
func StartSpinner(label string) func() {
	done := make(chan struct{})
	go func() {
		chars := loadSpinnerChars()
		i := 0
		for {
			select {
			case <-done:
				fmt.Fprintf(os.Stderr, "\r\033[K")
				return
			default:
			}
			ch := chars[i%len(chars)]
			fmt.Fprintf(os.Stderr, "\r%s %s", Cyan(string(ch)), label)
			i++
			time.Sleep(120 * time.Millisecond)
		}
	}()
	return func() {
		close(done)
		time.Sleep(130 * time.Millisecond)
	}
}

// PrintBanner prints a small colored banner header followed by content.
func PrintBanner(name string, content string) {
	fmt.Println(Cyan(Bold("== " + name + " ==")))
	fmt.Print(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		fmt.Println()
	}
}

// loadSpinnerChars reads spinner characters from beNotAfraid.txt when available.
func loadSpinnerChars() []rune {
	return []rune(beNotAfraidSpinner)
}

const beNotAfraidSpinner = "B̴̠́̔e̶͍̅ ̸̘͂̉͜n̶̫̞̆ȍ̶̢͕̉ṯ̷̒̏ ̶͉̆a̷̯͙͑ḟ̷̧̈r̸̩̓͝a̴̢͉͌î̸̲̗d̵̤̩͊̾"
