package database

import (
	"errors"
	"testing"
)

func TestValidateTableName(t *testing.T) {
	tests := []struct {
		name    string
		table   string
		wantErr bool
	}{
		{"valid studies", "studies", false},
		{"valid experiments", "experiments", false},
		{"valid samples", "samples", false},
		{"valid runs", "runs", false},
		{"valid submissions", "submissions", false},
		{"valid analyses", "analyses", false},
		{"valid fts_accessions", "fts_accessions", false},
		{"valid statistics", "statistics", false},
		{"invalid table", "invalid_table", true},
		{"SQL injection attempt", "studies; DROP TABLE studies;--", true},
		{"empty string", "", true},
		{"table with spaces", "table name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTableName(tt.table)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTableName(%q) error = %v, wantErr %v", tt.table, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !errors.Is(err, ErrInvalidTableName) {
					t.Errorf("expected ErrInvalidTableName, got %v", err)
				}
			}
		})
	}
}

func TestValidateColumnName(t *testing.T) {
	tests := []struct {
		name    string
		column  string
		wantErr bool
	}{
		{"valid study_accession", "study_accession", false},
		{"valid experiment_accession", "experiment_accession", false},
		{"valid title", "title", false},
		{"valid organism", "organism", false},
		{"valid platform", "platform", false},
		{"invalid column", "invalid_column", true},
		{"SQL injection attempt", "title; DROP TABLE studies;--", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateColumnName(tt.column)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateColumnName(%q) error = %v, wantErr %v", tt.column, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !errors.Is(err, ErrInvalidColumnName) {
					t.Errorf("expected ErrInvalidColumnName, got %v", err)
				}
			}
		})
	}
}

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantErr    bool
	}{
		{"simple name", "table_name", false},
		{"starts with underscore", "_private", false},
		{"alphanumeric", "table123", false},
		{"uppercase", "TABLE_NAME", false},
		{"mixed case", "TableName", false},
		{"empty string", "", true},
		{"starts with number", "123table", true},
		{"has spaces", "table name", true},
		{"has dash", "table-name", true},
		{"has semicolon", "table;name", true},
		{"SQL injection", "'; DROP TABLE --", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIdentifier(tt.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIdentifier(%q) error = %v, wantErr %v", tt.identifier, err, tt.wantErr)
			}
		})
	}
}

func TestSafeTableName(t *testing.T) {
	// Valid table
	name, err := SafeTableName("studies")
	if err != nil {
		t.Errorf("SafeTableName(studies) unexpected error: %v", err)
	}
	if name != "studies" {
		t.Errorf("SafeTableName(studies) = %q, want studies", name)
	}

	// Invalid table
	_, err = SafeTableName("invalid")
	if err == nil {
		t.Error("SafeTableName(invalid) expected error, got nil")
	}
}

func TestSafeColumnName(t *testing.T) {
	// Valid column
	name, err := SafeColumnName("title")
	if err != nil {
		t.Errorf("SafeColumnName(title) unexpected error: %v", err)
	}
	if name != "title" {
		t.Errorf("SafeColumnName(title) = %q, want title", name)
	}

	// Invalid column
	_, err = SafeColumnName("invalid")
	if err == nil {
		t.Error("SafeColumnName(invalid) expected error, got nil")
	}
}

func TestMustTableName(t *testing.T) {
	// Valid table - should not panic
	name := MustTableName("studies")
	if name != "studies" {
		t.Errorf("MustTableName(studies) = %q, want studies", name)
	}

	// Invalid table - should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustTableName(invalid) should panic")
		}
	}()
	MustTableName("invalid")
}

func TestMustColumnName(t *testing.T) {
	// Valid column - should not panic
	name := MustColumnName("title")
	if name != "title" {
		t.Errorf("MustColumnName(title) = %q, want title", name)
	}

	// Invalid column - should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustColumnName(invalid) should panic")
		}
	}()
	MustColumnName("invalid")
}

func TestAllAllowedTables(t *testing.T) {
	// Verify all expected tables are in the whitelist
	expectedTables := []string{
		"studies", "experiments", "samples", "runs",
		"submissions", "analyses", "sample_pool",
		"identifiers", "links", "experiment_samples",
		"fts_accessions", "fts_samples", "fts_runs",
		"statistics", "sync_status", "progress", "index_progress",
	}

	for _, table := range expectedTables {
		if err := ValidateTableName(table); err != nil {
			t.Errorf("expected table %q to be allowed, got error: %v", table, err)
		}
	}
}

func TestAllAllowedColumns(t *testing.T) {
	// Verify common columns are in the whitelist
	expectedColumns := []string{
		"study_accession", "experiment_accession", "sample_accession", "run_accession",
		"title", "abstract", "description", "organism", "platform",
		"created_at", "updated_at",
	}

	for _, col := range expectedColumns {
		if err := ValidateColumnName(col); err != nil {
			t.Errorf("expected column %q to be allowed, got error: %v", col, err)
		}
	}
}
