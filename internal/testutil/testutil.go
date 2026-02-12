// Package testutil provides testing utilities for SRAKE packages.
// It includes helpers for creating test databases, fixtures, and mocks.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempDir creates a temporary directory for tests.
// It returns the path and a cleanup function.
func TempDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "srake-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	return dir, func() {
		os.RemoveAll(dir)
	}
}

// TempFile creates a temporary file with the given content.
// It returns the path and a cleanup function.
func TempFile(t *testing.T, name, content string) (string, func()) {
	t.Helper()
	dir, cleanup := TempDir(t)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		cleanup()
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path, cleanup
}

// RequireEnv skips the test if the environment variable is not set.
func RequireEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Skipf("skipping: %s not set", name)
	}
	return value
}

// SkipIfShort skips the test if running in short mode.
func SkipIfShort(t *testing.T, reason string) {
	t.Helper()
	if testing.Short() {
		t.Skipf("skipping in short mode: %s", reason)
	}
}

// SkipIfCI skips the test if running in CI environment.
func SkipIfCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("skipping in CI environment")
	}
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: %v", msg, err)
	}
}

// RequireNoError fails the test immediately if err is not nil.
func RequireNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertEqual fails the test if got != want.
func AssertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

// AssertTrue fails the test if condition is false.
func AssertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Errorf("assertion failed: %s", msg)
	}
}

// AssertFalse fails the test if condition is true.
func AssertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()
	if condition {
		t.Errorf("assertion failed (expected false): %s", msg)
	}
}

// AssertNotNil fails the test if value is nil.
func AssertNotNil(t *testing.T, value interface{}, msg string) {
	t.Helper()
	if value == nil {
		t.Errorf("expected non-nil: %s", msg)
	}
}

// AssertNil fails the test if value is not nil.
func AssertNil(t *testing.T, value interface{}, msg string) {
	t.Helper()
	if value != nil {
		t.Errorf("expected nil: %s (got %v)", msg, value)
	}
}

// AssertContains fails the test if substr is not in s.
func AssertContains(t *testing.T, s, substr, msg string) {
	t.Helper()
	if !contains(s, substr) {
		t.Errorf("%s: %q does not contain %q", msg, s, substr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
