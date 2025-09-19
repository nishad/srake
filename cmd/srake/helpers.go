package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
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

// Print warning message
func printWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", colorize(colorYellow, "⚠"), msg)
}

// Print debug message
func printDebug(format string, args ...interface{}) {
	if debug {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(os.Stderr, "%s %s\n", colorize(colorGray, "[DEBUG]"), msg)
	}
}

// Helper function to read accessions from file or stdin
func readAccessionsFromReader(r io.Reader) ([]string, error) {
	accessions := make([]string, 0)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			accessions = append(accessions, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return accessions, nil
}

// Helper function to read accessions from file
func readAccessionFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return readAccessionsFromReader(file)
}

// Spinner represents a simple command-line spinner
type Spinner struct {
	message string
	stop    chan bool
	done    chan bool
}

// StartSpinner creates and starts a new spinner
func StartSpinner(message string) *Spinner {
	if quiet || !isTerminal() {
		// Just print the message without spinner in quiet mode or non-terminal
		fmt.Printf("%s...", message)
		return &Spinner{
			message: message,
			stop:    make(chan bool),
			done:    make(chan bool),
		}
	}

	s := &Spinner{
		message: message,
		stop:    make(chan bool),
		done:    make(chan bool),
	}

	go func() {
		spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.stop:
				s.done <- true
				return
			default:
				fmt.Printf("\r%s %s %s", colorize(colorCyan, spinChars[i]), message, colorize(colorGray, "..."))
				i = (i + 1) % len(spinChars)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return s
}

// StopSpinner stops the spinner and prints a result
func (s *Spinner) Stop(success bool, resultMsg string) {
	if quiet || !isTerminal() {
		if success {
			fmt.Println(" ✓")
		} else {
			fmt.Println(" ✗")
		}
		return
	}

	s.stop <- true
	<-s.done

	// Clear the line
	fmt.Printf("\r%s", strings.Repeat(" ", len(s.message)+20))

	// Print the final result
	if success {
		fmt.Printf("\r%s %s %s\n", colorize(colorGreen, "✓"), s.message, colorize(colorGray, resultMsg))
	} else {
		fmt.Printf("\r%s %s %s\n", colorize(colorRed, "✗"), s.message, colorize(colorGray, resultMsg))
	}
}

// PrintPhase prints a phase header
func printPhase(phase string) {
	if !quiet {
		fmt.Printf("\n%s %s\n", colorize(colorBlue, "▶"), colorize(colorBold, phase))
	}
}