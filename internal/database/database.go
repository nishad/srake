package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQL database connection
type DB struct {
	*sql.DB
	path string
}

// GetSQLDB returns the underlying SQL database connection
func (db *DB) GetSQLDB() *sql.DB {
	return db.DB
}

// Initialize creates and configures the database connection
func Initialize(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal=WAL&_timeout=5000&_sync=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set pragmas for performance - optimized based on performance testing
	pragmas := []string{
		"PRAGMA journal_mode = WAL",         // Write-ahead logging
		"PRAGMA synchronous = NORMAL",       // Balanced safety/speed
		"PRAGMA cache_size = 100000",        // ~400MB cache (10x increase)
		"PRAGMA temp_store = MEMORY",        // Use memory for temp tables
		"PRAGMA mmap_size = 1073741824",     // 1GB memory mapping (4x increase)
		"PRAGMA page_size = 32768",          // Larger page size for better I/O
		"PRAGMA wal_checkpoint = PASSIVE",   // Background checkpointing
		"PRAGMA wal_autocheckpoint = 10000", // Checkpoint every 10k pages
		"PRAGMA busy_timeout = 10000",       // 10 second timeout
		"PRAGMA foreign_keys = OFF",         // Disable FK checks during import
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &DB{
		DB:   db,
		path: path,
	}, nil
}

func createTables(db *sql.DB) error {
	// Core SRAmetadb-compatible schema from FINAL_IMPLEMENTATION_CONTEXT
	schema := `
	-- Core SRAmetadb-compatible schema
	CREATE TABLE IF NOT EXISTS studies (
		study_accession TEXT PRIMARY KEY,
		study_title TEXT,
		study_abstract TEXT,
		study_type TEXT,
		organism TEXT,
		submission_date DATE,
		metadata JSON
	);

	CREATE TABLE IF NOT EXISTS experiments (
		experiment_accession TEXT PRIMARY KEY,
		study_accession TEXT REFERENCES studies(study_accession),
		title TEXT,
		library_strategy TEXT,
		library_source TEXT,
		platform TEXT,
		instrument_model TEXT,
		metadata JSON
	);

	CREATE TABLE IF NOT EXISTS samples (
		sample_accession TEXT PRIMARY KEY,
		experiment_accession TEXT,
		organism TEXT,
		scientific_name TEXT,
		taxon_id INTEGER,
		tissue TEXT,
		cell_type TEXT,
		description TEXT,
		metadata JSON
	);

	CREATE TABLE IF NOT EXISTS runs (
		run_accession TEXT PRIMARY KEY,
		experiment_accession TEXT REFERENCES experiments(experiment_accession),
		total_spots INTEGER,
		total_bases INTEGER,
		published DATE,
		metadata JSON
	);

	-- Optimized indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_study_organism ON studies(organism);
	CREATE INDEX IF NOT EXISTS idx_study_date ON studies(submission_date);
	CREATE INDEX IF NOT EXISTS idx_exp_strategy ON experiments(library_strategy);
	CREATE INDEX IF NOT EXISTS idx_exp_study ON experiments(study_accession);
	CREATE INDEX IF NOT EXISTS idx_sample_organism ON samples(organism);
	CREATE INDEX IF NOT EXISTS idx_sample_tissue ON samples(tissue);
	CREATE INDEX IF NOT EXISTS idx_sample_experiment ON samples(experiment_accession);
	CREATE INDEX IF NOT EXISTS idx_run_experiment ON runs(experiment_accession);

	-- Submission table
	CREATE TABLE IF NOT EXISTS submissions (
		submission_accession TEXT PRIMARY KEY,
		alias TEXT,
		center_name TEXT,
		broker_name TEXT,
		lab_name TEXT,
		title TEXT,
		submission_date DATE,
		submission_comment TEXT,
		contacts JSON,
		actions JSON,
		submission_links JSON,
		submission_attributes JSON,
		metadata JSON
	);

	-- Analysis table
	CREATE TABLE IF NOT EXISTS analyses (
		analysis_accession TEXT PRIMARY KEY,
		alias TEXT,
		center_name TEXT,
		broker_name TEXT,
		analysis_center TEXT,
		analysis_date DATE,
		study_accession TEXT REFERENCES studies(study_accession),
		title TEXT,
		description TEXT,
		analysis_type TEXT,
		targets JSON,
		data_blocks JSON,
		assembly_ref JSON,
		run_labels JSON,
		seq_labels JSON,
		processing JSON,
		analysis_links JSON,
		analysis_attributes JSON,
		metadata JSON
	);

	-- Indexes for new tables
	CREATE INDEX IF NOT EXISTS idx_submission_date ON submissions(submission_date);
	CREATE INDEX IF NOT EXISTS idx_submission_center ON submissions(center_name);
	CREATE INDEX IF NOT EXISTS idx_analysis_study ON analyses(study_accession);
	CREATE INDEX IF NOT EXISTS idx_analysis_type ON analyses(analysis_type);
	CREATE INDEX IF NOT EXISTS idx_analysis_date ON analyses(analysis_date);

	-- Pool/multiplex relationships
	CREATE TABLE IF NOT EXISTS sample_pool (
		pool_id INTEGER PRIMARY KEY AUTOINCREMENT,
		parent_sample TEXT REFERENCES samples(sample_accession),
		member_sample TEXT,
		member_name TEXT,
		proportion REAL,
		read_label TEXT,
		UNIQUE(parent_sample, member_sample)
	);

	-- Structured identifiers
	CREATE TABLE IF NOT EXISTS identifiers (
		record_type TEXT NOT NULL,
		record_accession TEXT NOT NULL,
		id_type TEXT NOT NULL,
		id_namespace TEXT,
		id_value TEXT NOT NULL,
		id_label TEXT,
		PRIMARY KEY (record_type, record_accession, id_type, id_value)
	);

	-- External links
	CREATE TABLE IF NOT EXISTS links (
		link_id INTEGER PRIMARY KEY AUTOINCREMENT,
		record_type TEXT NOT NULL,
		record_accession TEXT NOT NULL,
		link_type TEXT,
		db TEXT,
		id TEXT,
		label TEXT,
		url TEXT
	);

	-- Junction table for experiment-sample many-to-many relationship
	CREATE TABLE IF NOT EXISTS experiment_samples (
		experiment_accession TEXT REFERENCES experiments(experiment_accession),
		sample_accession TEXT REFERENCES samples(sample_accession),
		PRIMARY KEY (experiment_accession, sample_accession)
	);

	-- Indexes for new tables
	CREATE INDEX IF NOT EXISTS idx_pool_parent ON sample_pool(parent_sample);
	CREATE INDEX IF NOT EXISTS idx_pool_member ON sample_pool(member_sample);
	CREATE INDEX IF NOT EXISTS idx_identifier_value ON identifiers(id_value);
	CREATE INDEX IF NOT EXISTS idx_identifier_record ON identifiers(record_type, record_accession);
	CREATE INDEX IF NOT EXISTS idx_link_record ON links(record_type, record_accession);
	CREATE INDEX IF NOT EXISTS idx_exp_sample_exp ON experiment_samples(experiment_accession);
	CREATE INDEX IF NOT EXISTS idx_exp_sample_sample ON experiment_samples(sample_accession);
	`

	_, err := db.Exec(schema)
	return err
}

// Study operations

func (db *DB) InsertStudy(study *Study) error {
	query := `
		INSERT OR REPLACE INTO studies (
			study_accession, study_title, study_abstract, study_type,
			organism, submission_date, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		study.StudyAccession, study.StudyTitle, study.StudyAbstract, study.StudyType,
		study.Organism, study.SubmissionDate, study.Metadata)
	return err
}

func (db *DB) GetStudy(accession string) (*Study, error) {
	study := &Study{}
	query := `
		SELECT study_accession, study_title, study_abstract, study_type,
			   organism, submission_date, COALESCE(metadata, '{}')
		FROM studies
		WHERE study_accession = ?
	`
	err := db.QueryRow(query, accession).Scan(
		&study.StudyAccession, &study.StudyTitle, &study.StudyAbstract, &study.StudyType,
		&study.Organism, &study.SubmissionDate, &study.Metadata)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("study not found: %s", accession)
	}
	return study, err
}

// Experiment operations

func (db *DB) InsertExperiment(exp *Experiment) error {
	query := `
		INSERT OR REPLACE INTO experiments (
			experiment_accession, study_accession, title,
			library_strategy, library_source, platform,
			instrument_model, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		exp.ExperimentAccession, exp.StudyAccession, exp.Title,
		exp.LibraryStrategy, exp.LibrarySource, exp.Platform,
		exp.InstrumentModel, exp.Metadata)
	return err
}

func (db *DB) GetExperiment(accession string) (*Experiment, error) {
	exp := &Experiment{}
	query := `
		SELECT experiment_accession, study_accession, title,
			   library_strategy, library_source, platform,
			   instrument_model, COALESCE(metadata, '{}')
		FROM experiments
		WHERE experiment_accession = ?
	`
	err := db.QueryRow(query, accession).Scan(
		&exp.ExperimentAccession, &exp.StudyAccession, &exp.Title,
		&exp.LibraryStrategy, &exp.LibrarySource, &exp.Platform,
		&exp.InstrumentModel, &exp.Metadata)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("experiment not found: %s", accession)
	}
	return exp, err
}

// Sample operations

func (db *DB) InsertSample(sample *Sample) error {
	query := `
		INSERT OR REPLACE INTO samples (
			sample_accession, experiment_accession, organism,
			scientific_name, taxon_id, tissue, cell_type,
			description, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		sample.SampleAccession, "", sample.Organism,
		sample.ScientificName, sample.TaxonID, sample.Tissue,
		sample.CellType, sample.Description, sample.Metadata)
	return err
}

func (db *DB) GetSample(accession string) (*Sample, error) {
	sample := &Sample{}
	query := `
		SELECT sample_accession, experiment_accession, organism,
			   scientific_name, taxon_id, tissue, cell_type,
			   description, COALESCE(metadata, '{}')
		FROM samples
		WHERE sample_accession = ?
	`
	var expAccession string
	err := db.QueryRow(query, accession).Scan(
		&sample.SampleAccession, &expAccession, &sample.Organism,
		&sample.ScientificName, &sample.TaxonID, &sample.Tissue,
		&sample.CellType, &sample.Description, &sample.Metadata)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sample not found: %s", accession)
	}
	return sample, err
}

// Run operations

func (db *DB) InsertRun(run *Run) error {
	query := `
		INSERT OR REPLACE INTO runs (
			run_accession, experiment_accession, total_spots, total_bases,
			published, metadata
		) VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		run.RunAccession, run.ExperimentAccession, run.TotalSpots,
		run.TotalBases, run.Published, run.Metadata)
	return err
}

func (db *DB) GetRun(accession string) (*Run, error) {
	run := &Run{}
	query := `
		SELECT run_accession, experiment_accession, total_spots, total_bases,
			   published, COALESCE(metadata, '{}')
		FROM runs
		WHERE run_accession = ?
	`
	err := db.QueryRow(query, accession).Scan(
		&run.RunAccession, &run.ExperimentAccession, &run.TotalSpots,
		&run.TotalBases, &run.Published, &run.Metadata)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found: %s", accession)
	}
	return run, err
}

// Submission operations

func (db *DB) InsertSubmission(submission *Submission) error {
	query := `
		INSERT OR REPLACE INTO submissions (
			submission_accession, alias, center_name, broker_name,
			lab_name, title, submission_date, submission_comment,
			contacts, actions, submission_links, submission_attributes, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		submission.SubmissionAccession, submission.Alias, submission.CenterName,
		submission.BrokerName, submission.LabName, submission.Title,
		submission.SubmissionDate, submission.SubmissionComment,
		submission.Contacts, submission.Actions, submission.SubmissionLinks,
		submission.SubmissionAttributes, submission.Metadata)
	return err
}

func (db *DB) GetSubmission(accession string) (*Submission, error) {
	submission := &Submission{}
	query := `
		SELECT submission_accession, alias, center_name, broker_name,
			   lab_name, title, submission_date, submission_comment,
			   contacts, actions, submission_links, submission_attributes,
			   COALESCE(metadata, '{}')
		FROM submissions WHERE submission_accession = ?
	`
	err := db.QueryRow(query, accession).Scan(
		&submission.SubmissionAccession, &submission.Alias, &submission.CenterName,
		&submission.BrokerName, &submission.LabName, &submission.Title,
		&submission.SubmissionDate, &submission.SubmissionComment,
		&submission.Contacts, &submission.Actions, &submission.SubmissionLinks,
		&submission.SubmissionAttributes, &submission.Metadata)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("submission not found: %s", accession)
	}
	return submission, err
}

// Analysis operations

func (db *DB) InsertAnalysis(analysis *Analysis) error {
	query := `
		INSERT OR REPLACE INTO analyses (
			analysis_accession, alias, center_name, broker_name,
			analysis_center, analysis_date, study_accession,
			title, description, analysis_type, targets, data_blocks,
			assembly_ref, run_labels, seq_labels, processing,
			analysis_links, analysis_attributes, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		analysis.AnalysisAccession, analysis.Alias, analysis.CenterName,
		analysis.BrokerName, analysis.AnalysisCenter, analysis.AnalysisDate,
		analysis.StudyAccession, analysis.Title, analysis.Description,
		analysis.AnalysisType, analysis.Targets, analysis.DataBlocks,
		analysis.AssemblyRef, analysis.RunLabels, analysis.SeqLabels,
		analysis.Processing, analysis.AnalysisLinks, analysis.AnalysisAttributes,
		analysis.Metadata)
	return err
}

func (db *DB) GetAnalysis(accession string) (*Analysis, error) {
	analysis := &Analysis{}
	query := `
		SELECT analysis_accession, alias, center_name, broker_name,
			   analysis_center, analysis_date, study_accession,
			   title, description, analysis_type, targets, data_blocks,
			   assembly_ref, run_labels, seq_labels, processing,
			   analysis_links, analysis_attributes, COALESCE(metadata, '{}')
		FROM analyses WHERE analysis_accession = ?
	`
	err := db.QueryRow(query, accession).Scan(
		&analysis.AnalysisAccession, &analysis.Alias, &analysis.CenterName,
		&analysis.BrokerName, &analysis.AnalysisCenter, &analysis.AnalysisDate,
		&analysis.StudyAccession, &analysis.Title, &analysis.Description,
		&analysis.AnalysisType, &analysis.Targets, &analysis.DataBlocks,
		&analysis.AssemblyRef, &analysis.RunLabels, &analysis.SeqLabels,
		&analysis.Processing, &analysis.AnalysisLinks, &analysis.AnalysisAttributes,
		&analysis.Metadata)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("analysis not found: %s", accession)
	}
	return analysis, err
}

// Search operations
func (db *DB) SearchByOrganism(organism string, limit int) ([]Sample, error) {
	query := `
		SELECT sample_accession, experiment_accession, organism,
			   scientific_name, taxon_id, tissue, cell_type,
			   description, COALESCE(metadata, '{}')
		FROM samples
		WHERE organism LIKE ? OR scientific_name LIKE ?
		LIMIT ?
	`

	rows, err := db.Query(query, "%"+organism+"%", "%"+organism+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []Sample
	for rows.Next() {
		var sample Sample
		var expAccession string
		err := rows.Scan(
			&sample.SampleAccession, &expAccession, &sample.Organism,
			&sample.ScientificName, &sample.TaxonID, &sample.Tissue,
			&sample.CellType, &sample.Description, &sample.Metadata)
		if err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}

	return samples, nil
}

func (db *DB) SearchByLibraryStrategy(strategy string, limit int) ([]Experiment, error) {
	query := `
		SELECT experiment_accession, study_accession, title,
			   library_strategy, library_source, platform,
			   instrument_model, COALESCE(metadata, '{}')
		FROM experiments
		WHERE library_strategy LIKE ?
		LIMIT ?
	`

	rows, err := db.Query(query, "%"+strategy+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var experiments []Experiment
	for rows.Next() {
		var exp Experiment
		err := rows.Scan(
			&exp.ExperimentAccession, &exp.StudyAccession, &exp.Title,
			&exp.LibraryStrategy, &exp.LibrarySource, &exp.Platform,
			&exp.InstrumentModel, &exp.Metadata)
		if err != nil {
			return nil, err
		}
		experiments = append(experiments, exp)
	}

	return experiments, nil
}

// Simple full-text search across multiple tables
func (db *DB) FullTextSearch(query string) (interface{}, error) {
	searchTerm := "%" + query + "%"

	type SearchResult struct {
		Type      string `json:"type"`
		Accession string `json:"accession"`
		Title     string `json:"title"`
		Organism  string `json:"organism,omitempty"`
		Platform  string `json:"platform,omitempty"`
		Strategy  string `json:"strategy,omitempty"`
	}

	var results []SearchResult

	// Search studies
	studyQuery := `
		SELECT 'study', study_accession, study_title, organism
		FROM studies
		WHERE study_title LIKE ? OR study_abstract LIKE ? OR organism LIKE ?
		LIMIT 10
	`
	rows, err := db.Query(studyQuery, searchTerm, searchTerm, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var result SearchResult
		err := rows.Scan(&result.Type, &result.Accession, &result.Title, &result.Organism)
		if err != nil {
			continue
		}
		results = append(results, result)
	}

	// Search experiments
	expQuery := `
		SELECT 'experiment', experiment_accession, title, platform, library_strategy
		FROM experiments
		WHERE title LIKE ? OR library_strategy LIKE ? OR platform LIKE ?
		LIMIT 10
	`
	rows2, err := db.Query(expQuery, searchTerm, searchTerm, searchTerm)
	if err != nil {
		return results, nil
	}
	defer rows2.Close()

	for rows2.Next() {
		var result SearchResult
		err := rows2.Scan(&result.Type, &result.Accession, &result.Title, &result.Platform, &result.Strategy)
		if err != nil {
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// Statistics
type DatabaseStats struct {
	TotalStudies     int       `json:"total_studies"`
	TotalExperiments int       `json:"total_experiments"`
	TotalSamples     int       `json:"total_samples"`
	TotalRuns        int       `json:"total_runs"`
	LastUpdate       time.Time `json:"last_update"`
}

func (db *DB) GetStats() (*DatabaseStats, error) {
	stats := &DatabaseStats{}

	// Get counts
	db.QueryRow("SELECT COUNT(*) FROM studies").Scan(&stats.TotalStudies)
	db.QueryRow("SELECT COUNT(*) FROM experiments").Scan(&stats.TotalExperiments)
	db.QueryRow("SELECT COUNT(*) FROM runs").Scan(&stats.TotalRuns)
	db.QueryRow("SELECT COUNT(*) FROM samples").Scan(&stats.TotalSamples)

	stats.LastUpdate = time.Now()

	return stats, nil
}

// Batch operations for performance
func (db *DB) BatchInsertExperiments(experiments []Experiment) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO experiments (
			experiment_accession, study_accession, title,
			library_strategy, library_source, platform,
			instrument_model, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, exp := range experiments {
		_, err = stmt.Exec(
			exp.ExperimentAccession, exp.StudyAccession, exp.Title,
			exp.LibraryStrategy, exp.LibrarySource, exp.Platform,
			exp.InstrumentModel, exp.Metadata)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// InsertSamplePool inserts a pool relationship
func (db *DB) InsertSamplePool(pool *SamplePool) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO sample_pool (
			parent_sample, member_sample, member_name,
			proportion, read_label
		) VALUES (?, ?, ?, ?, ?)
	`, pool.ParentSample, pool.MemberSample, pool.MemberName,
		pool.Proportion, pool.ReadLabel)
	return err
}

// GetSamplePools retrieves pool relationships for a parent sample
func (db *DB) GetSamplePools(parentSample string) ([]SamplePool, error) {
	rows, err := db.Query(`
		SELECT pool_id, parent_sample, member_sample, member_name,
			proportion, read_label
		FROM sample_pool
		WHERE parent_sample = ?
	`, parentSample)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []SamplePool
	for rows.Next() {
		var p SamplePool
		err := rows.Scan(&p.PoolID, &p.ParentSample, &p.MemberSample,
			&p.MemberName, &p.Proportion, &p.ReadLabel)
		if err != nil {
			return nil, err
		}
		pools = append(pools, p)
	}

	return pools, rows.Err()
}

// CountSamplePools counts total number of pool relationships
func (db *DB) CountSamplePools() (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(DISTINCT parent_sample)
		FROM sample_pool
	`).Scan(&count)
	return count, err
}

// GetAveragePoolSize returns the average pool size
func (db *DB) GetAveragePoolSize() (float64, error) {
	var avg sql.NullFloat64
	err := db.QueryRow(`
		SELECT AVG(member_count) FROM (
			SELECT parent_sample, COUNT(*) as member_count
			FROM sample_pool
			GROUP BY parent_sample
		)
	`).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if avg.Valid {
		return avg.Float64, nil
	}
	return 0, nil
}

// GetMaxPoolSize returns the maximum pool size
func (db *DB) GetMaxPoolSize() (int, error) {
	var max sql.NullInt64
	err := db.QueryRow(`
		SELECT MAX(member_count) FROM (
			SELECT parent_sample, COUNT(*) as member_count
			FROM sample_pool
			GROUP BY parent_sample
		)
	`).Scan(&max)
	if err != nil {
		return 0, err
	}
	if max.Valid {
		return int(max.Int64), nil
	}
	return 0, nil
}

// InsertIdentifier inserts a structured identifier
func (db *DB) InsertIdentifier(identifier *Identifier) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO identifiers (
			record_type, record_accession, id_type,
			id_namespace, id_value, id_label
		) VALUES (?, ?, ?, ?, ?, ?)
	`, identifier.RecordType, identifier.RecordAccession, identifier.IDType,
		identifier.IDNamespace, identifier.IDValue, identifier.IDLabel)
	return err
}

// GetIdentifiers retrieves identifiers for a record
func (db *DB) GetIdentifiers(recordType, recordAccession string) ([]Identifier, error) {
	rows, err := db.Query(`
		SELECT record_type, record_accession, id_type,
			id_namespace, id_value, id_label
		FROM identifiers
		WHERE record_type = ? AND record_accession = ?
	`, recordType, recordAccession)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identifiers []Identifier
	for rows.Next() {
		var id Identifier
		err := rows.Scan(&id.RecordType, &id.RecordAccession, &id.IDType,
			&id.IDNamespace, &id.IDValue, &id.IDLabel)
		if err != nil {
			return nil, err
		}
		identifiers = append(identifiers, id)
	}

	return identifiers, rows.Err()
}

// FindRecordsByIdentifier finds records with a specific identifier value
func (db *DB) FindRecordsByIdentifier(idValue string) ([]Identifier, error) {
	rows, err := db.Query(`
		SELECT record_type, record_accession, id_type,
			id_namespace, id_value, id_label
		FROM identifiers
		WHERE id_value = ?
	`, idValue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identifiers []Identifier
	for rows.Next() {
		var id Identifier
		err := rows.Scan(&id.RecordType, &id.RecordAccession, &id.IDType,
			&id.IDNamespace, &id.IDValue, &id.IDLabel)
		if err != nil {
			return nil, err
		}
		identifiers = append(identifiers, id)
	}

	return identifiers, rows.Err()
}

// InsertLink inserts a structured link
func (db *DB) InsertLink(link *Link) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO links (
			record_type, record_accession, link_type,
			db, id, label, url
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, link.RecordType, link.RecordAccession, link.LinkType,
		link.DB, link.ID, link.Label, link.URL)
	return err
}

// GetLinks retrieves links for a record
func (db *DB) GetLinks(recordType, recordAccession string) ([]Link, error) {
	rows, err := db.Query(`
		SELECT record_type, record_accession, link_type,
			db, id, label, url
		FROM links
		WHERE record_type = ? AND record_accession = ?
	`, recordType, recordAccession)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		err := rows.Scan(&link.RecordType, &link.RecordAccession, &link.LinkType,
			&link.DB, &link.ID, &link.Label, &link.URL)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, rows.Err()
}
