package main

import (
	"fmt"
	"os"
)

// Color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// Check if output is to terminal
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Apply color if terminal output and color enabled
func colorize(color, text string) string {
	if !noColor && isTerminal() && os.Getenv("NO_COLOR") == "" {
		return color + text + colorReset
	}
	return text
}

// Print error message in user-friendly format
func printError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", colorize(colorRed, "✗"), msg)
}

// Print success message
func printSuccess(format string, args ...interface{}) {
	if !quiet {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", colorize(colorGreen, "✓"), msg)
	}
}

// Print info message
func printInfo(format string, args ...interface{}) {
	if !quiet {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s\n", colorize(colorCyan, msg))
	}
}