// Package database provides safe SQL utilities to prevent SQL injection.
package database

import (
	"fmt"
	"regexp"
)

// AllowedTables is the whitelist of valid table names in SRAKE database.
// Any table name not in this list will be rejected to prevent SQL injection.
var AllowedTables = map[string]bool{
	// Core entity tables
	"studies":      true,
	"experiments":  true,
	"samples":      true,
	"runs":         true,
	"submissions":  true,
	"analyses":     true,

	// Relationship tables
	"sample_pool":         true,
	"identifiers":         true,
	"links":               true,
	"experiment_samples":  true,

	// FTS5 virtual tables
	"fts_accessions": true,
	"fts_samples":    true,
	"fts_runs":       true,

	// System tables
	"statistics":     true,
	"sync_status":    true,
	"progress":       true,
	"index_progress": true,
}

// AllowedColumns is the whitelist of valid column names.
// This is used for dynamic column selection in queries.
var AllowedColumns = map[string]bool{
	// Common identifier columns
	"study_accession":      true,
	"experiment_accession": true,
	"sample_accession":     true,
	"run_accession":        true,
	"submission_accession": true,
	"analysis_accession":   true,

	// Common metadata columns
	"title":          true,
	"abstract":       true,
	"description":    true,
	"organism":        true,
	"scientific_name": true,
	"taxon_id":        true,
	"platform":        true,
	"instrument_model": true,
	"library_strategy": true,
	"library_source":   true,
	"library_selection": true,
	"library_layout":   true,

	// Timestamp columns
	"created_at":     true,
	"updated_at":     true,
	"submission_date": true,
	"first_public":    true,
	"last_update":     true,

	// Statistics columns
	"table_name": true,
	"row_count":  true,
}

// ErrInvalidTableName is returned when a table name is not in the whitelist.
var ErrInvalidTableName = fmt.Errorf("invalid table name")

// ErrInvalidColumnName is returned when a column name is not in the whitelist.
var ErrInvalidColumnName = fmt.Errorf("invalid column name")

// validIdentifierPattern matches valid SQL identifiers (alphanumeric and underscore).
var validIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateTableName checks if a table name is in the allowed list.
// Returns nil if valid, ErrInvalidTableName otherwise.
func ValidateTableName(table string) error {
	if !AllowedTables[table] {
		return fmt.Errorf("%w: %q", ErrInvalidTableName, table)
	}
	return nil
}

// ValidateColumnName checks if a column name is in the allowed list.
// Returns nil if valid, ErrInvalidColumnName otherwise.
func ValidateColumnName(column string) error {
	if !AllowedColumns[column] {
		return fmt.Errorf("%w: %q", ErrInvalidColumnName, column)
	}
	return nil
}

// ValidateIdentifier checks if a string is a valid SQL identifier format.
// This is a fallback for dynamic identifiers not in the whitelists.
// Valid format: starts with letter or underscore, followed by alphanumeric or underscore.
func ValidateIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("empty identifier")
	}
	if !validIdentifierPattern.MatchString(identifier) {
		return fmt.Errorf("invalid identifier format: %q", identifier)
	}
	return nil
}

// SafeTableName returns the table name if valid, otherwise returns an error.
// Use this when you need the table name for SQL construction.
func SafeTableName(table string) (string, error) {
	if err := ValidateTableName(table); err != nil {
		return "", err
	}
	return table, nil
}

// SafeColumnName returns the column name if valid, otherwise returns an error.
// Use this when you need the column name for SQL construction.
func SafeColumnName(column string) (string, error) {
	if err := ValidateColumnName(column); err != nil {
		return "", err
	}
	return column, nil
}

// MustTableName returns the table name if valid, panics otherwise.
// Use this only for hardcoded table names that are known to be valid.
func MustTableName(table string) string {
	if err := ValidateTableName(table); err != nil {
		panic(fmt.Sprintf("invalid table name in code: %s", table))
	}
	return table
}

// MustColumnName returns the column name if valid, panics otherwise.
// Use this only for hardcoded column names that are known to be valid.
func MustColumnName(column string) string {
	if err := ValidateColumnName(column); err != nil {
		panic(fmt.Sprintf("invalid column name in code: %s", column))
	}
	return column
}
