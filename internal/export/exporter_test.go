package export

import (
	"database/sql"
	"testing"
	"time"
)

func TestNullString(t *testing.T) {
	tests := []struct {
		input    string
		expected sql.NullString
	}{
		{"", sql.NullString{Valid: false}},
		{"hello", sql.NullString{String: "hello", Valid: true}},
		{"test value", sql.NullString{String: "test value", Valid: true}},
	}

	for _, tt := range tests {
		result := nullString(tt.input)
		if result.Valid != tt.expected.Valid {
			t.Errorf("nullString(%q).Valid = %v, want %v", tt.input, result.Valid, tt.expected.Valid)
		}
		if result.Valid && result.String != tt.expected.String {
			t.Errorf("nullString(%q).String = %q, want %q", tt.input, result.String, tt.expected.String)
		}
	}
}

func TestNullInt64(t *testing.T) {
	tests := []struct {
		input    int64
		expected sql.NullInt64
	}{
		{0, sql.NullInt64{Valid: false}},
		{42, sql.NullInt64{Int64: 42, Valid: true}},
		{-1, sql.NullInt64{Int64: -1, Valid: true}},
	}

	for _, tt := range tests {
		result := nullInt64(tt.input)
		if result.Valid != tt.expected.Valid {
			t.Errorf("nullInt64(%d).Valid = %v, want %v", tt.input, result.Valid, tt.expected.Valid)
		}
		if result.Valid && result.Int64 != tt.expected.Int64 {
			t.Errorf("nullInt64(%d).Int64 = %d, want %d", tt.input, result.Int64, tt.expected.Int64)
		}
	}
}

func TestNullFloat64(t *testing.T) {
	tests := []struct {
		input    float64
		expected sql.NullFloat64
	}{
		{0, sql.NullFloat64{Valid: false}},
		{3.14, sql.NullFloat64{Float64: 3.14, Valid: true}},
		{-1.5, sql.NullFloat64{Float64: -1.5, Valid: true}},
	}

	for _, tt := range tests {
		result := nullFloat64(tt.input)
		if result.Valid != tt.expected.Valid {
			t.Errorf("nullFloat64(%f).Valid = %v, want %v", tt.input, result.Valid, tt.expected.Valid)
		}
		if result.Valid && result.Float64 != tt.expected.Float64 {
			t.Errorf("nullFloat64(%f).Float64 = %f, want %f", tt.input, result.Float64, tt.expected.Float64)
		}
	}
}

func TestFormatTime(t *testing.T) {
	// nil time
	result := formatTime(nil)
	if result != "" {
		t.Errorf("formatTime(nil) = %q, want empty string", result)
	}

	// Zero time
	zero := time.Time{}
	result = formatTime(&zero)
	if result != "" {
		t.Errorf("formatTime(zero) = %q, want empty string", result)
	}

	// Valid time
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result = formatTime(&tm)
	expected := "2024-01-15 10:30:00"
	if result != expected {
		t.Errorf("formatTime() = %q, want %q", result, expected)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := &Config{
		SourceDB:   "/path/to/source.db",
		OutputPath: "/path/to/output.db",
	}

	if cfg.BatchSize != 0 {
		t.Errorf("default BatchSize should be 0, got %d", cfg.BatchSize)
	}
	if cfg.Compress {
		t.Error("default Compress should be false")
	}
}

func TestStatsFields(t *testing.T) {
	stats := &Stats{
		Studies:     100,
		Experiments: 500,
		Samples:     200,
		Runs:        1000,
		Submissions: 50,
		SRARecords:  1850,
		Duration:    5 * time.Second,
	}

	if stats.Studies != 100 {
		t.Errorf("expected 100 studies, got %d", stats.Studies)
	}
	if stats.Duration != 5*time.Second {
		t.Errorf("expected 5s duration, got %v", stats.Duration)
	}
}
