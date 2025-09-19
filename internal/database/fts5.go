package database

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// FTS5Manager manages SQLite FTS5 tables for fast text search
type FTS5Manager struct {
	db *DB
}

// NewFTS5Manager creates a new FTS5 manager
func NewFTS5Manager(db *DB) *FTS5Manager {
	return &FTS5Manager{db: db}
}

// CreateFTSTables creates FTS5 tables for tier 3 search (samples and runs)
func (f *FTS5Manager) CreateFTSTables() error {
	log.Println("[FTS5] Creating FTS5 tables for fast search")
	start := time.Now()

	// Create FTS5 table for accessions (all types)
	err := f.createAccessionTable()
	if err != nil {
		return fmt.Errorf("failed to create accession FTS table: %w", err)
	}

	// Create FTS5 table for samples
	err = f.createSampleFTSTable()
	if err != nil {
		return fmt.Errorf("failed to create sample FTS table: %w", err)
	}

	// Create FTS5 table for runs
	err = f.createRunFTSTable()
	if err != nil {
		return fmt.Errorf("failed to create run FTS table: %w", err)
	}

	log.Printf("[FTS5] FTS5 tables created in %v", time.Since(start))
	return nil
}

// createAccessionTable creates an FTS5 table for fast accession lookups
func (f *FTS5Manager) createAccessionTable() error {
	// Drop existing table if it exists
	_, err := f.db.DB.Exec(`DROP TABLE IF EXISTS fts_accessions`)
	if err != nil {
		return err
	}

	// Create FTS5 table
	query := `
		CREATE VIRTUAL TABLE fts_accessions USING fts5(
			accession,
			type,
			title,
			metadata,
			tokenize='porter'
		)
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return err
	}

	// Populate with all accessions
	log.Println("[FTS5] Populating accession FTS table...")

	// Insert studies
	query = `
		INSERT INTO fts_accessions (accession, type, title, metadata)
		SELECT
			study_accession,
			'study',
			study_title,
			study_abstract || ' ' || COALESCE(study_type, '')
		FROM studies
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to insert studies: %w", err)
	}

	// Insert experiments
	query = `
		INSERT INTO fts_accessions (accession, type, title, metadata)
		SELECT
			experiment_accession,
			'experiment',
			COALESCE(title, ''),
			COALESCE(library_strategy, '') || ' ' || COALESCE(platform, '') || ' ' || COALESCE(instrument_model, '')
		FROM experiments
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to insert experiments: %w", err)
	}

	// Insert samples (limit metadata to avoid bloat)
	query = `
		INSERT INTO fts_accessions (accession, type, title, metadata)
		SELECT
			sample_accession,
			'sample',
			COALESCE(description, ''),
			COALESCE(organism, '') || ' ' || COALESCE(scientific_name, '') || ' ' || COALESCE(tissue, '') || ' ' || COALESCE(cell_type, '')
		FROM samples
		LIMIT 1000000
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to insert samples: %w", err)
	}

	// Insert runs (minimal metadata)
	query = `
		INSERT INTO fts_accessions (accession, type, title, metadata)
		SELECT
			run_accession,
			'run',
			COALESCE(run_accession, ''),
			experiment_accession
		FROM runs
		LIMIT 1000000
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to insert runs: %w", err)
	}

	return nil
}

// createSampleFTSTable creates an FTS5 table for sample search
func (f *FTS5Manager) createSampleFTSTable() error {
	// Drop existing table if it exists
	_, err := f.db.DB.Exec(`DROP TABLE IF EXISTS fts_samples`)
	if err != nil {
		return err
	}

	// Create FTS5 table for samples
	query := `
		CREATE VIRTUAL TABLE fts_samples USING fts5(
			sample_accession UNINDEXED,
			description,
			organism,
			scientific_name,
			tissue,
			tokenize='porter'
		)
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return err
	}

	// Populate with sample data (batch insert for performance)
	log.Println("[FTS5] Populating sample FTS table...")
	query = `
		INSERT INTO fts_samples
		SELECT
			sample_accession,
			COALESCE(description, ''),
			COALESCE(organism, ''),
			COALESCE(scientific_name, ''),
			COALESCE(tissue, '')
		FROM samples
		LIMIT 1000000
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to populate sample FTS: %w", err)
	}

	return nil
}

// createRunFTSTable creates an FTS5 table for run search
func (f *FTS5Manager) createRunFTSTable() error {
	// Drop existing table if it exists
	_, err := f.db.DB.Exec(`DROP TABLE IF EXISTS fts_runs`)
	if err != nil {
		return err
	}

	// Create FTS5 table for runs
	query := `
		CREATE VIRTUAL TABLE fts_runs USING fts5(
			run_accession UNINDEXED,
			experiment_accession UNINDEXED,
			total_spots,
			total_bases,
			tokenize='porter'
		)
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return err
	}

	// Populate with run data (batch insert for performance)
	log.Println("[FTS5] Populating run FTS table...")
	query = `
		INSERT INTO fts_runs
		SELECT
			run_accession,
			experiment_accession,
			CAST(total_spots AS TEXT),
			CAST(total_bases AS TEXT)
		FROM runs
		LIMIT 1000000
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to populate run FTS: %w", err)
	}

	return nil
}

// SearchAccessions searches for accessions using FTS5
func (f *FTS5Manager) SearchAccessions(query string, limit int) ([]AccessionResult, error) {
	// Escape special characters in FTS5 query
	ftsQuery := escapeFTSQuery(query)

	sqlQuery := `
		SELECT
			accession,
			type,
			title,
			metadata,
			bm25(fts_accessions) as score
		FROM fts_accessions
		WHERE fts_accessions MATCH ?
		ORDER BY score
		LIMIT ?
	`

	rows, err := f.db.DB.Query(sqlQuery, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("FTS5 search failed: %w", err)
	}
	defer rows.Close()

	var results []AccessionResult
	for rows.Next() {
		var r AccessionResult
		err := rows.Scan(&r.Accession, &r.Type, &r.Title, &r.Metadata, &r.Score)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

// SearchSamples searches samples using FTS5
func (f *FTS5Manager) SearchSamples(query string, limit int) ([]SampleResult, error) {
	ftsQuery := escapeFTSQuery(query)

	sqlQuery := `
		SELECT
			sample_accession,
			description,
			organism,
			scientific_name,
			bm25(fts_samples) as score
		FROM fts_samples
		WHERE fts_samples MATCH ?
		ORDER BY score
		LIMIT ?
	`

	rows, err := f.db.DB.Query(sqlQuery, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("sample FTS5 search failed: %w", err)
	}
	defer rows.Close()

	var results []SampleResult
	for rows.Next() {
		var r SampleResult
		err := rows.Scan(&r.SampleAccession, &r.Description, &r.Organism, &r.ScientificName, &r.Score)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

// SearchRuns searches runs using FTS5
func (f *FTS5Manager) SearchRuns(query string, limit int) ([]RunResult, error) {
	ftsQuery := escapeFTSQuery(query)

	sqlQuery := `
		SELECT
			run_accession,
			experiment_accession,
			total_spots,
			total_bases,
			bm25(fts_runs) as score
		FROM fts_runs
		WHERE fts_runs MATCH ?
		ORDER BY score
		LIMIT ?
	`

	rows, err := f.db.DB.Query(sqlQuery, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("run FTS5 search failed: %w", err)
	}
	defer rows.Close()

	var results []RunResult
	for rows.Next() {
		var r RunResult
		err := rows.Scan(&r.RunAccession, &r.ExperimentAccession, &r.TotalSpots, &r.TotalBases, &r.Score)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

// OptimizeFTSTables optimizes FTS5 tables for better performance
func (f *FTS5Manager) OptimizeFTSTables() error {
	tables := []string{"fts_accessions", "fts_samples", "fts_runs"}

	for _, table := range tables {
		// Optimize the FTS table
		// #nosec G201 - table names are from a fixed list, not user input
		query := fmt.Sprintf("INSERT INTO %s(%s) VALUES('optimize')", table, table)
		_, err := f.db.DB.Exec(query)
		if err != nil {
			log.Printf("[FTS5] Warning: failed to optimize %s: %v", table, err)
		}
	}

	return nil
}

// GetFTSStats returns statistics about FTS5 tables
func (f *FTS5Manager) GetFTSStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	// Get row counts for each FTS table
	tables := []string{"fts_accessions", "fts_samples", "fts_runs"}
	for _, table := range tables {
		var count int64
		// #nosec G201 - table names are from a fixed list, not user input
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		err := f.db.DB.QueryRow(query).Scan(&count)
		if err != nil {
			// Table might not exist
			stats[table] = 0
		} else {
			stats[table] = count
		}
	}

	return stats, nil
}

// escapeFTSQuery escapes special characters in FTS5 queries
func escapeFTSQuery(query string) string {
	// FTS5 special characters that need escaping
	specialChars := []string{"\"", "*", "-", "+", "^"}

	result := query
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}

	// If query contains spaces, wrap in quotes for phrase search
	if strings.Contains(result, " ") {
		result = "\"" + result + "\""
	}

	return result
}

// Result types

type AccessionResult struct {
	Accession string
	Type      string
	Title     string
	Metadata  string
	Score     float64
}

type SampleResult struct {
	SampleAccession string
	Description     string
	Organism        string
	ScientificName  string
	Score           float64
}

type RunResult struct {
	RunAccession        string
	ExperimentAccession string
	TotalSpots          string
	TotalBases          string
	Score               float64
}
