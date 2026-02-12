// Package errors provides error handling utilities for SRAKE.
// It offers consistent error wrapping, logging, and handling patterns
// to improve error visibility throughout the codebase.
package errors

import (
	"fmt"
	"log"
	"runtime"
	"strings"
)

// Op represents an operation name for error context.
type Op string

// Error represents an application error with context.
type Error struct {
	Op   Op     // Operation that failed
	Kind Kind   // Category of error
	Err  error  // Underlying error
	Msg  string // Additional context message
}

// Kind represents the category of error.
type Kind uint8

const (
	KindUnknown Kind = iota
	KindDatabase
	KindSearch
	KindIO
	KindValidation
	KindConfig
	KindNetwork
	KindParse
)

// String returns the string representation of the error kind.
func (k Kind) String() string {
	switch k {
	case KindDatabase:
		return "database"
	case KindSearch:
		return "search"
	case KindIO:
		return "io"
	case KindValidation:
		return "validation"
	case KindConfig:
		return "config"
	case KindNetwork:
		return "network"
	case KindParse:
		return "parse"
	default:
		return "unknown"
	}
}

// Error implements the error interface.
func (e *Error) Error() string {
	var b strings.Builder
	if e.Op != "" {
		b.WriteString(string(e.Op))
		b.WriteString(": ")
	}
	if e.Msg != "" {
		b.WriteString(e.Msg)
		if e.Err != nil {
			b.WriteString(": ")
		}
	}
	if e.Err != nil {
		b.WriteString(e.Err.Error())
	}
	return b.String()
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

// E creates a new Error with the given arguments.
// Arguments can be: Op, Kind, error, string (message).
func E(args ...interface{}) *Error {
	e := &Error{}
	for _, arg := range args {
		switch a := arg.(type) {
		case Op:
			e.Op = a
		case Kind:
			e.Kind = a
		case error:
			e.Err = a
		case string:
			e.Msg = a
		}
	}
	return e
}

// Wrap wraps an error with an operation name for context.
func Wrap(op Op, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Err: err}
}

// WrapMsg wraps an error with an operation name and message.
func WrapMsg(op Op, msg string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Msg: msg, Err: err}
}

// SkipCounter tracks how many times operations have been skipped.
// Use this to provide visibility into silent error patterns.
type SkipCounter struct {
	Op         string
	Count      int
	LastErr    error
	LastDetail string
}

// NewSkipCounter creates a new skip counter for the given operation.
func NewSkipCounter(op string) *SkipCounter {
	return &SkipCounter{Op: op}
}

// Skip records a skipped operation due to an error.
func (s *SkipCounter) Skip(err error, detail string) {
	s.Count++
	s.LastErr = err
	s.LastDetail = detail
}

// Report logs a summary if any operations were skipped.
func (s *SkipCounter) Report() {
	if s.Count > 0 {
		log.Printf("Warning: %s skipped %d items (last error: %v, detail: %s)",
			s.Op, s.Count, s.LastErr, s.LastDetail)
	}
}

// ReportIfAny logs a summary only if the count exceeds threshold.
func (s *SkipCounter) ReportIfAny(threshold int) {
	if s.Count >= threshold {
		s.Report()
	}
}

// LogAndContinue logs an error and returns true (for use in continue patterns).
// This replaces silent continue statements with visible logging.
//
// Example:
//
//	if err != nil {
//	    errors.LogAndContinue("scanning row", err)
//	    continue
//	}
func LogAndContinue(operation string, err error) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		// Extract just the filename
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			file = file[idx+1:]
		}
		log.Printf("Warning [%s:%d]: %s failed: %v", file, line, operation, err)
	} else {
		log.Printf("Warning: %s failed: %v", operation, err)
	}
}

// LogAndContinueWith logs an error with additional context.
func LogAndContinueWith(operation string, err error, context string) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			file = file[idx+1:]
		}
		log.Printf("Warning [%s:%d]: %s failed for %s: %v", file, line, operation, context, err)
	} else {
		log.Printf("Warning: %s failed for %s: %v", operation, context, err)
	}
}

// MustHandle panics if the error is not nil.
// Use this only for errors that should never happen in normal operation.
func MustHandle(err error) {
	if err != nil {
		panic(fmt.Sprintf("unexpected error: %v", err))
	}
}

// Must panics if the error is not nil and returns the value otherwise.
// Use this only for initialization code where errors are unexpected.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("unexpected error: %v", err))
	}
	return v
}

// IgnoreError explicitly ignores an error with a reason.
// This documents that the error is intentionally ignored.
//
// Example:
//
//	errors.IgnoreError(file.Close(), "cleanup during error recovery")
func IgnoreError(err error, reason string) {
	if err != nil {
		log.Printf("Debug: ignoring error (%s): %v", reason, err)
	}
}

// IsKind checks if an error is of the given kind.
func IsKind(err error, kind Kind) bool {
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	return e.Kind == kind
}

// GetKind returns the kind of an error, or KindUnknown.
func GetKind(err error) Kind {
	e, ok := err.(*Error)
	if !ok {
		return KindUnknown
	}
	return e.Kind
}

// RowScanner provides utilities for database row scanning with error tracking.
type RowScanner struct {
	skipped *SkipCounter
	scanned int
}

// NewRowScanner creates a new row scanner with error tracking.
func NewRowScanner(operation string) *RowScanner {
	return &RowScanner{
		skipped: NewSkipCounter(operation),
	}
}

// RecordScan records a successful scan.
func (r *RowScanner) RecordScan() {
	r.scanned++
}

// RecordSkip records a skipped row due to scan error.
func (r *RowScanner) RecordSkip(err error, identifier string) {
	r.skipped.Skip(err, identifier)
}

// Report logs statistics about the scanning operation.
func (r *RowScanner) Report() {
	if r.skipped.Count > 0 {
		log.Printf("Row scan complete: %d scanned, %d skipped (%.1f%% success rate)",
			r.scanned, r.skipped.Count,
			float64(r.scanned)/float64(r.scanned+r.skipped.Count)*100)
		r.skipped.Report()
	}
}

// SkippedCount returns the number of skipped rows.
func (r *RowScanner) SkippedCount() int {
	return r.skipped.Count
}

// ScannedCount returns the number of successfully scanned rows.
func (r *RowScanner) ScannedCount() int {
	return r.scanned
}
