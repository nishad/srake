package database

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// FTS5Manager manages SQLite FTS5 tables for fast text search
type FTS5Manager struct {
	db         *DB
	onProgress func(step string) // optional callback for progress reporting
}

// NewFTS5Manager creates a new FTS5 manager
func NewFTS5Manager(db *DB) *FTS5Manager {
	return &FTS5Manager{db: db}
}

// SetProgressCallback sets a callback invoked before each FTS5 table build step
func (f *FTS5Manager) SetProgressCallback(cb func(step string)) {
	f.onProgress = cb
}

func (f *FTS5Manager) reportProgress(step string) {
	if f.onProgress != nil {
		f.onProgress(step)
	}
}

// CreateFTSTables creates FTS5 tables for tier 3 search (samples and runs)
func (f *FTS5Manager) CreateFTSTables() error {
	log.Println("[FTS5] Creating FTS5 tables for fast search")
	start := time.Now()

	// Create FTS5 table for accessions (all types)
	f.reportProgress("FTS5: Building accession index (studies + experiments + samples + runs)...")
	err := f.createAccessionTable()
	if err != nil {
		return fmt.Errorf("failed to create accession FTS table: %w", err)
	}

	// Create FTS5 table for experiments (platform, strategy search)
	f.reportProgress("FTS5: Building experiment index (38M records)...")
	err = f.createExperimentFTSTable()
	if err != nil {
		return fmt.Errorf("failed to create experiment FTS table: %w", err)
	}

	// Create FTS5 table for samples
	f.reportProgress("FTS5: Building sample index...")
	err = f.createSampleFTSTable()
	if err != nil {
		return fmt.Errorf("failed to create sample FTS table: %w", err)
	}

	// Create FTS5 table for runs
	f.reportProgress("FTS5: Building run index...")
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

// createExperimentFTSTable creates an FTS5 table for experiment technical search
func (f *FTS5Manager) createExperimentFTSTable() error {
	_, err := f.db.DB.Exec(`DROP TABLE IF EXISTS fts_experiments`)
	if err != nil {
		return err
	}

	query := `
		CREATE VIRTUAL TABLE fts_experiments USING fts5(
			experiment_accession UNINDEXED,
			study_accession UNINDEXED,
			title,
			library_strategy,
			library_source,
			platform,
			instrument_model,
			tokenize='porter'
		)
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return err
	}

	log.Println("[FTS5] Populating experiment FTS table...")
	query = `
		INSERT INTO fts_experiments
		SELECT
			experiment_accession,
			study_accession,
			COALESCE(title, ''),
			COALESCE(library_strategy, ''),
			COALESCE(library_source, ''),
			COALESCE(platform, ''),
			COALESCE(instrument_model, '')
		FROM experiments
	`
	_, err = f.db.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to populate experiment FTS: %w", err)
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

// SearchExperiments searches experiments using FTS5
func (f *FTS5Manager) SearchExperiments(query string, limit int) ([]ExperimentResult, error) {
	ftsQuery := escapeFTSQuery(query)

	sqlQuery := `
		SELECT
			experiment_accession,
			study_accession,
			title,
			library_strategy,
			platform,
			instrument_model,
			bm25(fts_experiments) as score
		FROM fts_experiments
		WHERE fts_experiments MATCH ?
		ORDER BY score
		LIMIT ?
	`

	rows, err := f.db.DB.Query(sqlQuery, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("experiment FTS5 search failed: %w", err)
	}
	defer rows.Close()

	var results []ExperimentResult
	for rows.Next() {
		var r ExperimentResult
		err := rows.Scan(&r.ExperimentAccession, &r.StudyAccession, &r.Title,
			&r.LibraryStrategy, &r.Platform, &r.InstrumentModel, &r.Score)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

// ftsExperimentFilterFields enumerates the searchable columns in fts_experiments.
// Field names are interpolated into the FTS5 MATCH expression, so they are
// validated against this allowlist to prevent injection of FTS5 syntax.
var ftsExperimentFilterFields = map[string]bool{
	"title":            true,
	"library_strategy": true,
	"library_source":   true,
	"platform":         true,
	"instrument_model": true,
}

// SearchExperimentsByFilter searches experiments using FTS5 with specific field filters
func (f *FTS5Manager) SearchExperimentsByFilter(filters map[string]string, limit int) ([]ExperimentResult, error) {
	// Build FTS5 query from filters
	var parts []string
	for field, value := range filters {
		if !ftsExperimentFilterFields[field] {
			return nil, fmt.Errorf("unsupported filter field: %q", field)
		}
		parts = append(parts, fmt.Sprintf("%s:%s", field, escapeFTSQuery(value)))
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no filters provided")
	}
	ftsQuery := strings.Join(parts, " ")

	sqlQuery := `
		SELECT
			experiment_accession,
			study_accession,
			title,
			library_strategy,
			platform,
			instrument_model,
			bm25(fts_experiments) as score
		FROM fts_experiments
		WHERE fts_experiments MATCH ?
		ORDER BY score
		LIMIT ?
	`

	rows, err := f.db.DB.Query(sqlQuery, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("experiment filter FTS5 search failed: %w", err)
	}
	defer rows.Close()

	var results []ExperimentResult
	for rows.Next() {
		var r ExperimentResult
		err := rows.Scan(&r.ExperimentAccession, &r.StudyAccession, &r.Title,
			&r.LibraryStrategy, &r.Platform, &r.InstrumentModel, &r.Score)
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
	tables := []string{"fts_accessions", "fts_experiments", "fts_samples", "fts_runs"}

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
	tables := []string{"fts_accessions", "fts_experiments", "fts_samples", "fts_runs"}
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

// escapeFTSQuery turns arbitrary user input into a safe FTS5 MATCH expression.
//
// FTS5 has no backslash escape: the only way to treat text literally is to wrap
// it in double quotes, doubling any embedded double quote ("" inside a string).
// Each whitespace-separated token is quoted independently and joined with spaces,
// which FTS5 interprets as an implicit AND. This keeps special characters
// (-, +, *, ^, :, parentheses) literal while still matching multi-word queries as
// a conjunction of terms rather than a single rigid phrase.
func escapeFTSQuery(query string) string {
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return `""`
	}

	quoted := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		quoted = append(quoted, `"`+strings.ReplaceAll(tok, `"`, `""`)+`"`)
	}
	return strings.Join(quoted, " ")
}

// AccessionResult holds a single accession match from an FTS5 search, including its BM25 relevance score.
type AccessionResult struct {
	Accession string
	Type      string
	Title     string
	Metadata  string
	Score     float64
}

// SampleResult holds a single sample match from an FTS5 search, including its BM25 relevance score.
type SampleResult struct {
	SampleAccession string
	Description     string
	Organism        string
	ScientificName  string
	Score           float64
}

// ExperimentResult holds a single experiment match from an FTS5 search, including its BM25 relevance score.
type ExperimentResult struct {
	ExperimentAccession string
	StudyAccession      string
	Title               string
	LibraryStrategy     string
	Platform            string
	InstrumentModel     string
	Score               float64
}

// RunResult holds a single run match from an FTS5 search, including its BM25 relevance score.
type RunResult struct {
	RunAccession        string
	ExperimentAccession string
	TotalSpots          string
	TotalBases          string
	Score               float64
}
