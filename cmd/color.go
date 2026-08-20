package cmd

import (
	"os"
	"strings"

	"github.com/sebrandon1/succulent-cli/lib"
)

const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
	ansiGray   = "\033[90m"
)

var (
	noColor     bool
	stdoutIsTTY = stdoutIsTerminal
)

func stdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return fi.Mode()&os.ModeCharDevice != 0
}

func colorEnabled() bool {
	if noColor || os.Getenv("NO_COLOR") != "" || outputFormat == "json" {
		return false
	}

	return stdoutIsTTY()
}

func colorize(text, code string) string {
	if !colorEnabled() || code == "" {
		return text
	}

	return code + text + ansiReset
}

func colorStatus(status string) string {
	switch strings.ToLower(status) {
	case "active", "ready", lib.StatusUp:
		return colorize(status, ansiGreen)
	case "partial":
		return colorize(status, ansiYellow)
	case "empty":
		return colorize(status, ansiGray)
	case "down":
		return colorize(status, ansiRed)
	default:
		return status
	}
}
