package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nishad/srake/internal/database"
)

// MetadataService handles metadata access for studies, experiments, samples, and runs
type MetadataService struct {
	db *database.DB
}

// NewMetadataService creates a new metadata service instance
func NewMetadataService(db *database.DB) *MetadataService {
	return &MetadataService{
		db: db,
	}
}

// GetMetadata retrieves metadata for a specific accession
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
	query := `SELECT * FROM studies WHERE study_accession = ?`

	var study database.Study
	row := m.db.QueryRow(query, accession)
	if err := m.db.ScanStudy(row, &study); err != nil {
		return nil, fmt.Errorf("study not found: %s", accession)
	}

	return &study, nil
}

// GetStudies retrieves multiple studies with pagination
func (m *MetadataService) GetStudies(ctx context.Context, limit, offset int) ([]*database.Study, error) {
	query := `SELECT * FROM studies ORDER BY study_accession LIMIT ? OFFSET ?`

	rows, err := m.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var studies []*database.Study
	for rows.Next() {
		var study database.Study
		if err := m.db.ScanStudy(rows, &study); err != nil {
			continue
		}
		studies = append(studies, &study)
	}

	return studies, nil
}

// GetExperiment retrieves an experiment by accession
func (m *MetadataService) GetExperiment(ctx context.Context, accession string) (*database.Experiment, error) {
	query := `SELECT * FROM experiments WHERE experiment_accession = ?`

	var exp database.Experiment
	row := m.db.QueryRow(query, accession)
	if err := m.db.ScanExperiment(row, &exp); err != nil {
		return nil, fmt.Errorf("experiment not found: %s", accession)
	}

	return &exp, nil
}

// GetExperimentsByStudy retrieves all experiments for a study
func (m *MetadataService) GetExperimentsByStudy(ctx context.Context, studyAccession string) ([]*database.Experiment, error) {
	query := `SELECT * FROM experiments WHERE study_accession = ? ORDER BY experiment_accession`

	rows, err := m.db.Query(query, studyAccession)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var experiments []*database.Experiment
	for rows.Next() {
		var exp database.Experiment
		if err := m.db.ScanExperiment(rows, &exp); err != nil {
			continue
		}
		experiments = append(experiments, &exp)
	}

	return experiments, nil
}

// GetSample retrieves a sample by accession
func (m *MetadataService) GetSample(ctx context.Context, accession string) (*database.Sample, error) {
	query := `SELECT * FROM samples WHERE sample_accession = ?`

	var sample database.Sample
	row := m.db.QueryRow(query, accession)
	if err := m.db.ScanSample(row, &sample); err != nil {
		return nil, fmt.Errorf("sample not found: %s", accession)
	}

	return &sample, nil
}

// GetSamplesByStudy retrieves all samples for a study
func (m *MetadataService) GetSamplesByStudy(ctx context.Context, studyAccession string) ([]*database.Sample, error) {
	query := `
		SELECT s.* FROM samples s
		JOIN experiment_samples es ON s.sample_accession = es.sample_accession
		JOIN experiments e ON es.experiment_accession = e.experiment_accession
		WHERE e.study_accession = ?
		ORDER BY s.sample_accession
	`

	rows, err := m.db.Query(query, studyAccession)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []*database.Sample
	seenSamples := make(map[string]bool)

	for rows.Next() {
		var sample database.Sample
		if err := m.db.ScanSample(rows, &sample); err != nil {
			continue
		}
		// Deduplicate samples
		if !seenSamples[sample.SampleAccession] {
			samples = append(samples, &sample)
			seenSamples[sample.SampleAccession] = true
		}
	}

	return samples, nil
}

// GetRun retrieves a run by accession
func (m *MetadataService) GetRun(ctx context.Context, accession string) (*database.Run, error) {
	query := `SELECT * FROM runs WHERE run_accession = ?`

	var run database.Run
	row := m.db.QueryRow(query, accession)
	if err := m.db.ScanRun(row, &run); err != nil {
		return nil, fmt.Errorf("run not found: %s", accession)
	}

	return &run, nil
}

// GetRunsByExperiment retrieves all runs for an experiment
func (m *MetadataService) GetRunsByExperiment(ctx context.Context, experimentAccession string) ([]*database.Run, error) {
	query := `SELECT * FROM runs WHERE experiment_accession = ? ORDER BY run_accession`

	rows, err := m.db.Query(query, experimentAccession)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*database.Run
	for rows.Next() {
		var run database.Run
		if err := m.db.ScanRun(rows, &run); err != nil {
			continue
		}
		runs = append(runs, &run)
	}

	return runs, nil
}

// GetRunsByStudy retrieves all runs for a study
func (m *MetadataService) GetRunsByStudy(ctx context.Context, studyAccession string, limit int) ([]*database.Run, error) {
	query := `
		SELECT r.* FROM runs r
		JOIN experiments e ON r.experiment_accession = e.accession
		WHERE e.study_accession = ?
		ORDER BY r.accession
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := m.db.Query(query, studyAccession)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*database.Run
	for rows.Next() {
		var run database.Run
		if err := m.db.ScanRun(rows, &run); err != nil {
			continue
		}
		runs = append(runs, &run)
	}

	return runs, nil
}

// GetStudyMetadata retrieves complete metadata for a study including experiments, samples, and runs
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

// GetAccessionType determines the type of an accession (study, experiment, sample, or run)
func (m *MetadataService) GetAccessionType(ctx context.Context, accession string) (string, error) {
	// Check studies
	if exists, _ := m.existsInTable("studies", accession); exists {
		return "study", nil
	}

	// Check experiments
	if exists, _ := m.existsInTable("experiments", accession); exists {
		return "experiment", nil
	}

	// Check samples
	if exists, _ := m.existsInTable("samples", accession); exists {
		return "sample", nil
	}

	// Check runs
	if exists, _ := m.existsInTable("runs", accession); exists {
		return "run", nil
	}

	return "", fmt.Errorf("accession not found: %s", accession)
}

// existsInTable checks if an accession exists in a table
func (m *MetadataService) existsInTable(table, accession string) (bool, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE accession = ?", table)

	var count int
	err := m.db.QueryRow(query, accession).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Health checks if the service is operational
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

// Close releases resources
func (m *MetadataService) Close() error {
	// MetadataService doesn't hold any resources that need explicit cleanup
	// The database connection is managed elsewhere
	return nil
}