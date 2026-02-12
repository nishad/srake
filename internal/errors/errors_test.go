package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestErrorCreation(t *testing.T) {
	err := E(Op("test.operation"), KindDatabase, "something failed")

	if err.Op != "test.operation" {
		t.Errorf("expected Op 'test.operation', got %q", err.Op)
	}
	if err.Kind != KindDatabase {
		t.Errorf("expected Kind KindDatabase, got %v", err.Kind)
	}
	if err.Msg != "something failed" {
		t.Errorf("expected Msg 'something failed', got %q", err.Msg)
	}
}

func TestErrorWithWrappedError(t *testing.T) {
	underlying := fmt.Errorf("connection refused")
	err := E(Op("db.connect"), KindDatabase, underlying, "failed to connect")

	if err.Err != underlying {
		t.Error("expected underlying error to be set")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "db.connect") {
		t.Errorf("error string should contain operation, got %q", errStr)
	}
	if !strings.Contains(errStr, "failed to connect") {
		t.Errorf("error string should contain message, got %q", errStr)
	}
	if !strings.Contains(errStr, "connection refused") {
		t.Errorf("error string should contain underlying error, got %q", errStr)
	}
}

func TestErrorUnwrap(t *testing.T) {
	underlying := fmt.Errorf("root cause")
	err := E(Op("test"), underlying)

	unwrapped := err.Unwrap()
	if unwrapped != underlying {
		t.Error("Unwrap should return the underlying error")
	}
}

func TestErrorStringFormats(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "op only",
			err:      &Error{Op: "test"},
			expected: "test: ",
		},
		{
			name:     "msg only",
			err:      &Error{Msg: "failed"},
			expected: "failed",
		},
		{
			name:     "err only",
			err:      &Error{Err: fmt.Errorf("root")},
			expected: "root",
		},
		{
			name:     "op and msg",
			err:      &Error{Op: "test", Msg: "failed"},
			expected: "test: failed",
		},
		{
			name:     "all fields",
			err:      &Error{Op: "test", Msg: "failed", Err: fmt.Errorf("root")},
			expected: "test: failed: root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestKindString(t *testing.T) {
	tests := []struct {
		kind     Kind
		expected string
	}{
		{KindUnknown, "unknown"},
		{KindDatabase, "database"},
		{KindSearch, "search"},
		{KindIO, "io"},
		{KindValidation, "validation"},
		{KindConfig, "config"},
		{KindNetwork, "network"},
		{KindParse, "parse"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.kind.String()
			if got != tt.expected {
				t.Errorf("Kind.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	// Wrap nil error
	wrapped := Wrap("test", nil)
	if wrapped != nil {
		t.Error("Wrap(nil) should return nil")
	}

	// Wrap non-nil error
	underlying := fmt.Errorf("test error")
	wrapped = Wrap("db.query", underlying)
	if wrapped == nil {
		t.Fatal("Wrap should return non-nil for non-nil error")
	}

	appErr, ok := wrapped.(*Error)
	if !ok {
		t.Fatal("Wrap should return *Error")
	}
	if appErr.Op != "db.query" {
		t.Errorf("expected Op 'db.query', got %q", appErr.Op)
	}
}

func TestWrapMsg(t *testing.T) {
	// Wrap nil error
	wrapped := WrapMsg("test", "msg", nil)
	if wrapped != nil {
		t.Error("WrapMsg(nil) should return nil")
	}

	// Wrap non-nil error
	underlying := fmt.Errorf("test error")
	wrapped = WrapMsg("db.query", "query failed", underlying)
	if wrapped == nil {
		t.Fatal("WrapMsg should return non-nil for non-nil error")
	}

	errStr := wrapped.Error()
	if !strings.Contains(errStr, "query failed") {
		t.Errorf("error should contain message, got %q", errStr)
	}
}

func TestIsKind(t *testing.T) {
	err := E(KindDatabase, "test")
	if !IsKind(err, KindDatabase) {
		t.Error("expected IsKind to return true for matching kind")
	}
	if IsKind(err, KindSearch) {
		t.Error("expected IsKind to return false for non-matching kind")
	}

	// Non-Error type
	stdErr := fmt.Errorf("standard error")
	if IsKind(stdErr, KindDatabase) {
		t.Error("expected IsKind to return false for non-Error type")
	}
}

func TestGetKind(t *testing.T) {
	err := E(KindNetwork, "test")
	kind := GetKind(err)
	if kind != KindNetwork {
		t.Errorf("expected KindNetwork, got %v", kind)
	}

	// Non-Error type
	stdErr := fmt.Errorf("standard error")
	kind = GetKind(stdErr)
	if kind != KindUnknown {
		t.Errorf("expected KindUnknown for non-Error, got %v", kind)
	}
}

func TestSkipCounter(t *testing.T) {
	sc := NewSkipCounter("test_operation")

	if sc.Count != 0 {
		t.Errorf("initial count should be 0, got %d", sc.Count)
	}

	// Skip some items
	sc.Skip(fmt.Errorf("error 1"), "item1")
	sc.Skip(fmt.Errorf("error 2"), "item2")
	sc.Skip(fmt.Errorf("error 3"), "item3")

	if sc.Count != 3 {
		t.Errorf("expected count 3, got %d", sc.Count)
	}

	if sc.LastErr == nil || sc.LastErr.Error() != "error 3" {
		t.Errorf("LastErr should be last error, got %v", sc.LastErr)
	}

	if sc.LastDetail != "item3" {
		t.Errorf("LastDetail should be 'item3', got %q", sc.LastDetail)
	}
}

func TestSkipCounterReport(t *testing.T) {
	sc := NewSkipCounter("test")

	// Report with no skips should not panic
	sc.Report()

	// Report with skips
	sc.Skip(fmt.Errorf("err"), "detail")
	sc.Report() // should log
}

func TestSkipCounterReportIfAny(t *testing.T) {
	sc := NewSkipCounter("test")

	// Below threshold - no report
	sc.Skip(fmt.Errorf("err"), "detail")
	sc.ReportIfAny(5) // count=1 < threshold=5, no report

	// At threshold
	for i := 0; i < 4; i++ {
		sc.Skip(fmt.Errorf("err %d", i), fmt.Sprintf("detail%d", i))
	}
	sc.ReportIfAny(5) // count=5 >= threshold=5, should report
}

func TestRowScanner(t *testing.T) {
	rs := NewRowScanner("test_scan")

	// Record some scans and skips
	rs.RecordScan()
	rs.RecordScan()
	rs.RecordScan()
	rs.RecordSkip(fmt.Errorf("scan error"), "row1")

	if rs.ScannedCount() != 3 {
		t.Errorf("expected 3 scanned, got %d", rs.ScannedCount())
	}
	if rs.SkippedCount() != 1 {
		t.Errorf("expected 1 skipped, got %d", rs.SkippedCount())
	}

	// Report should not panic
	rs.Report()
}

func TestRowScannerNoSkips(t *testing.T) {
	rs := NewRowScanner("test_scan")
	rs.RecordScan()
	rs.RecordScan()

	// Report with no skips should not panic
	rs.Report()

	if rs.SkippedCount() != 0 {
		t.Errorf("expected 0 skipped, got %d", rs.SkippedCount())
	}
}

func TestLogAndContinue(t *testing.T) {
	// Should not panic
	LogAndContinue("test operation", fmt.Errorf("test error"))
}

func TestLogAndContinueWith(t *testing.T) {
	// Should not panic
	LogAndContinueWith("test operation", fmt.Errorf("test error"), "context info")
}

func TestMustHandle(t *testing.T) {
	// nil error - should not panic
	MustHandle(nil)

	// Non-nil error - should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustHandle should panic on non-nil error")
		}
	}()
	MustHandle(fmt.Errorf("fatal error"))
}

func TestMust(t *testing.T) {
	// Success case
	result := Must(42, nil)
	if result != 42 {
		t.Errorf("Must should return value, got %d", result)
	}

	// Error case - should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Must should panic on error")
		}
	}()
	Must(0, fmt.Errorf("error"))
}

func TestIgnoreError(t *testing.T) {
	// Should not panic for nil error
	IgnoreError(nil, "test")

	// Should not panic for non-nil error
	IgnoreError(fmt.Errorf("test"), "test reason")
}
