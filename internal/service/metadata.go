// Package service provides high-level business logic for querying and managing
// SRA metadata, including study, experiment, sample, and run access.
package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nishad/srake/internal/database"
)

// MetadataService provides read access to SRA metadata records across
// studies, experiments, samples, and runs with pagination and relational lookups.
type MetadataService struct {
	db *database.DB
}

// NewMetadataService creates a new metadata service instance
func NewMetadataService(db *database.DB) *MetadataService {
	return &MetadataService{
		db: db,
	}
}

// GetMetadata retrieves metadata for a specific accession, dispatching to the
// appropriate record type (study, experiment, sample, or run) based on the request.
func (m *MetadataService) GetMetadata(ctx context.Context, req *MetadataRequest) (*MetadataResponse, error) {
	response := &MetadataResponse{
		Type:      req.Type,
		Retrieved: time.Now(),
	}

	var err error
	switch req.Type {
	case "study":
		response.Data, err = m.GetStudy(ctx, req.Accession)
	case "experiment":
		response.Data, err = m.GetExperiment(ctx, req.Accession)
	case "sample":
		response.Data, err = m.GetSample(ctx, req.Accession)
	case "run":
		response.Data, err = m.GetRun(ctx, req.Accession)
	default:
		return nil, fmt.Errorf("invalid type: %s", req.Type)
	}

	if err != nil {
		return nil, err
	}

	return response, nil
}

// GetStudy retrieves a study by accession
func (m *MetadataService) GetStudy(ctx context.Context, accession string) (*database.Study, error) {
	return m.db.GetStudy(accession)
}

// GetStudies retrieves multiple studies with pagination
func (m *MetadataService) GetStudies(ctx context.Context, limit, offset int) ([]*database.Study, error) {
	query := `SELECT study_accession, study_title, study_abstract, study_type,
			   organism, submission_date, COALESCE(metadata, '{}')
		FROM studies ORDER BY study_accession LIMIT ? OFFSET ?`

	rows, err := m.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var studies []*database.Study
	for rows.Next() {
		var study database.Study
		if err := rows.Scan(
			&study.StudyAccession, &study.StudyTitle, &study.StudyAbstract,
			&study.StudyType, &study.Organism, &study.SubmissionDate, &study.Metadata,
		); err != nil {
			continue
		}
		studies = append(studies, &study)
	}

	return studies, nil
}

// GetExperiment retrieves an experiment by accession
func (m *MetadataService) GetExperiment(ctx context.Context, accession string) (*database.Experiment, error) {
	return m.db.GetExperiment(accession)
}

// GetExperimentsByStudy retrieves all experiments for a study
func (m *MetadataService) GetExperimentsByStudy(ctx context.Context, studyAccession string) ([]*database.Experiment, error) {
	query := `SELECT experiment_accession, study_accession, title,
			   library_strategy, library_source, platform,
			   instrument_model, COALESCE(metadata, '{}')
		FROM experiments WHERE study_accession = ? ORDER BY experiment_accession`

	rows, err := m.db.Query(query, studyAccession)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var experiments []*database.Experiment
	for rows.Next() {
		var exp database.Experiment
		if err := rows.Scan(
			&exp.ExperimentAccession, &exp.StudyAccession, &exp.Title,
			&exp.LibraryStrategy, &exp.LibrarySource, &exp.Platform,
			&exp.InstrumentModel, &exp.Metadata,
		); err != nil {
			continue
		}
		experiments = append(experiments, &exp)
	}

	return experiments, nil
}

// GetSample retrieves a sample by accession
func (m *MetadataService) GetSample(ctx context.Context, accession string) (*database.Sample, error) {
	return m.db.GetSample(accession)
}

// GetSamplesByStudy retrieves all samples for a study via the experiment_samples junction table
func (m *MetadataService) GetSamplesByStudy(ctx context.Context, studyAccession string) ([]*database.Sample, error) {
	query := `
		SELECT DISTINCT s.sample_accession, s.organism, s.scientific_name,
			   s.taxon_id, s.tissue, s.cell_type, s.description,
			   COALESCE(s.metadata, '{}')
		FROM samples s
		JOIN experiment_samples es ON es.sample_accession = s.sample_accession
		JOIN experiments e ON e.experiment_accession = es.experiment_accession
		WHERE e.study_accession = ?
		ORDER BY s.sample_accession
	`

	rows, err := m.db.Query(query, studyAccession)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []*database.Sample
	for rows.Next() {
		var sample database.Sample
		if err := rows.Scan(
			&sample.SampleAccession, &sample.Organism, &sample.ScientificName,
			&sample.TaxonID, &sample.Tissue, &sample.CellType,
			&sample.Description, &sample.Metadata,
		); err != nil {
			continue
		}
		samples = append(samples, &sample)
	}

	return samples, nil
}

// GetRun retrieves a run by accession
func (m *MetadataService) GetRun(ctx context.Context, accession string) (*database.Run, error) {
	return m.db.GetRun(accession)
}

// GetRunsByExperiment retrieves all runs for an experiment
func (m *MetadataService) GetRunsByExperiment(ctx context.Context, experimentAccession string) ([]*database.Run, error) {
	query := `SELECT run_accession, experiment_accession, total_spots,
			   total_bases, published, COALESCE(metadata, '{}')
		FROM runs WHERE experiment_accession = ? ORDER BY run_accession`

	rows, err := m.db.Query(query, experimentAccession)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*database.Run
	for rows.Next() {
		var run database.Run
		if err := rows.Scan(
			&run.RunAccession, &run.ExperimentAccession, &run.TotalSpots,
			&run.TotalBases, &run.Published, &run.Metadata,
		); err != nil {
			continue
		}
		runs = append(runs, &run)
	}

	return runs, nil
}

// GetRunsByStudy retrieves all runs for a study
func (m *MetadataService) GetRunsByStudy(ctx context.Context, studyAccession string, limit int) ([]*database.Run, error) {
	var rows *sql.Rows
	var err error

	if limit > 0 {
		query := `
			SELECT r.run_accession, r.experiment_accession, r.total_spots,
				   r.total_bases, r.published, COALESCE(r.metadata, '{}')
			FROM runs r
			JOIN experiments e ON r.experiment_accession = e.experiment_accession
			WHERE e.study_accession = ?
			ORDER BY r.run_accession
			LIMIT ?`
		rows, err = m.db.Query(query, studyAccession, limit)
	} else {
		query := `
			SELECT r.run_accession, r.experiment_accession, r.total_spots,
				   r.total_bases, r.published, COALESCE(r.metadata, '{}')
			FROM runs r
			JOIN experiments e ON r.experiment_accession = e.experiment_accession
			WHERE e.study_accession = ?
			ORDER BY r.run_accession`
		rows, err = m.db.Query(query, studyAccession)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*database.Run
	for rows.Next() {
		var run database.Run
		if err := rows.Scan(
			&run.RunAccession, &run.ExperimentAccession, &run.TotalSpots,
			&run.TotalBases, &run.Published, &run.Metadata,
		); err != nil {
			continue
		}
		runs = append(runs, &run)
	}

	return runs, nil
}

// GetStudyMetadata retrieves the complete metadata graph for a study,
// including its experiments, samples, and up to 100 runs.
func (m *MetadataService) GetStudyMetadata(ctx context.Context, studyAccession string) (map[string]interface{}, error) {
	// Get study
	study, err := m.GetStudy(ctx, studyAccession)
	if err != nil {
		return nil, err
	}

	// Get experiments
	experiments, err := m.GetExperimentsByStudy(ctx, studyAccession)
	if err != nil {
		return nil, err
	}

	// Get samples
	samples, err := m.GetSamplesByStudy(ctx, studyAccession)
	if err != nil {
		return nil, err
	}

	// Get runs (limited to 100 for performance)
	runs, err := m.GetRunsByStudy(ctx, studyAccession, 100)
	if err != nil {
		return nil, err
	}

	// Build response
	metadata := map[string]interface{}{
		"study":       study,
		"experiments": experiments,
		"samples":     samples,
		"runs":        runs,
		"summary": map[string]int{
			"total_experiments": len(experiments),
			"total_samples":     len(samples),
			"total_runs":        len(runs),
		},
	}

	return metadata, nil
}

// GetAccessionType determines whether an accession refers to a study, experiment,
// sample, or run by probing each table. Returns an error if the accession is not found.
func (m *MetadataService) GetAccessionType(ctx context.Context, accession string) (string, error) {
	// Check studies
	if exists, _ := m.existsInTable("studies", "study_accession", accession); exists {
		return "study", nil
	}

	// Check experiments
	if exists, _ := m.existsInTable("experiments", "experiment_accession", accession); exists {
		return "experiment", nil
	}

	// Check samples
	if exists, _ := m.existsInTable("samples", "sample_accession", accession); exists {
		return "sample", nil
	}

	// Check runs
	if exists, _ := m.existsInTable("runs", "run_accession", accession); exists {
		return "run", nil
	}

	return "", fmt.Errorf("accession not found: %s", accession)
}

// existsInTable checks if an accession exists in a table.
// The table and column names are validated against the AllowedTables and AllowedColumns
// whitelists to prevent SQL injection attacks.
func (m *MetadataService) existsInTable(table, column, accession string) (bool, error) {
	// Validate table and column names against whitelists to prevent SQL injection
	safeTable, err := database.SafeTableName(table)
	if err != nil {
		return false, fmt.Errorf("existsInTable: %w", err)
	}
	safeColumn, err := database.SafeColumnName(column)
	if err != nil {
		return false, fmt.Errorf("existsInTable: %w", err)
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", safeTable, safeColumn)

	var count int
	err = m.db.QueryRow(query, accession).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Health verifies the service is operational by checking the database connection
// and executing a basic query.
func (m *MetadataService) Health(ctx context.Context) error {
	// Check database connection
	if err := m.db.Ping(); err != nil {
		return fmt.Errorf("database unhealthy: %w", err)
	}

	// Check if we can query basic metadata
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM studies LIMIT 1").Scan(&count)
	if err != nil {
		return fmt.Errorf("cannot query metadata: %w", err)
	}

	return nil
}

// Close releases any resources held by the MetadataService.
func (m *MetadataService) Close() error {
	// MetadataService doesn't hold any resources that need explicit cleanup
	// The database connection is managed elsewhere
	return nil
}
