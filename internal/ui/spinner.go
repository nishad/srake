package ui

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Spinner provides a simple command-line spinner for long-running operations
type Spinner struct {
	chars   []string
	message string
	active  bool
	mu      sync.Mutex
	done    chan struct{}
}

// NewSpinner creates a new spinner instance
func NewSpinner(message string) *Spinner {
	return &Spinner{
		chars:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		message: message,
		done:    make(chan struct{}),
	}
}

// Start begins spinning, showing feedback within 100ms
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	// Check if output is to terminal and NO_COLOR is not set
	if !isTerminal() || os.Getenv("NO_COLOR") != "" {
		fmt.Fprintf(os.Stderr, "%s...\n", s.message)
		return
	}

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond) // Start within 100ms
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-s.done:
				// Clear the spinner line
				fmt.Fprintf(os.Stderr, "\r\033[K")
				return
			case <-ticker.C:
				s.mu.Lock()
				if s.active {
					fmt.Fprintf(os.Stderr, "\r%s %s", s.chars[i], s.message)
					i = (i + 1) % len(s.chars)
				}
				s.mu.Unlock()
			}
		}
	}()
}

// Stop stops the spinner and optionally shows a final message
func (s *Spinner) Stop(finalMessage string) {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	close(s.done)
	time.Sleep(100 * time.Millisecond) // Allow goroutine to clean up

	if finalMessage != "" {
		fmt.Fprintf(os.Stderr, "\r\033[K%s\n", finalMessage)
	}
}

// Update changes the spinner message while it's running
func (s *Spinner) Update(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// isTerminal checks if output is to a terminal
func isTerminal() bool {
	fileInfo, _ := os.Stderr.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ShowSpinner is a convenience function for simple spinner usage
func ShowSpinner(message string, fn func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()
	err := fn()
	if err != nil {
		spinner.Stop(fmt.Sprintf("✗ %s", err.Error()))
	} else {
		spinner.Stop("✓ Done")
	}
	return err
}
