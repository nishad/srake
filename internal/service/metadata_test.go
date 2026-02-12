package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nishad/srake/internal/database"
)

func setupTestMetadataService(t *testing.T) (*MetadataService, *database.DB, func()) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := database.Initialize(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	svc := NewMetadataService(db)
	cleanup := func() {
		svc.Close()
		db.Close()
	}

	return svc, db, cleanup
}

func seedTestData(t *testing.T, db *database.DB) {
	t.Helper()

	// Insert studies
	studies := []*database.Study{
		{StudyAccession: "SRP000001", StudyTitle: "Human Genome Study", StudyAbstract: "Whole genome sequencing", StudyType: "WGS", Organism: "Homo sapiens"},
		{StudyAccession: "SRP000002", StudyTitle: "Mouse Transcriptome", StudyAbstract: "RNA-Seq analysis", StudyType: "Transcriptome Analysis", Organism: "Mus musculus"},
	}
	for _, s := range studies {
		if err := db.InsertStudy(s); err != nil {
			t.Fatalf("failed to insert study: %v", err)
		}
	}

	// Insert experiments
	experiments := []*database.Experiment{
		{ExperimentAccession: "SRX000001", StudyAccession: "SRP000001", Title: "WGS Exp 1", LibraryStrategy: "WGS", Platform: "ILLUMINA"},
		{ExperimentAccession: "SRX000002", StudyAccession: "SRP000001", Title: "WGS Exp 2", LibraryStrategy: "WGS", Platform: "PACBIO_SMRT"},
		{ExperimentAccession: "SRX000003", StudyAccession: "SRP000002", Title: "RNA-Seq Exp", LibraryStrategy: "RNA-Seq", Platform: "ILLUMINA"},
	}
	for _, e := range experiments {
		if err := db.InsertExperiment(e); err != nil {
			t.Fatalf("failed to insert experiment: %v", err)
		}
	}

	// Insert runs
	runs := []*database.Run{
		{RunAccession: "SRR000001", ExperimentAccession: "SRX000001", TotalSpots: 1000000, TotalBases: 300000000},
		{RunAccession: "SRR000002", ExperimentAccession: "SRX000001", TotalSpots: 2000000, TotalBases: 600000000},
		{RunAccession: "SRR000003", ExperimentAccession: "SRX000003", TotalSpots: 500000, TotalBases: 150000000},
	}
	for _, r := range runs {
		if err := db.InsertRun(r); err != nil {
			t.Fatalf("failed to insert run: %v", err)
		}
	}

	// Insert samples
	samples := []*database.Sample{
		{SampleAccession: "SRS000001", Organism: "Homo sapiens", ScientificName: "Homo sapiens", TaxonID: 9606, Tissue: "blood"},
		{SampleAccession: "SRS000002", Organism: "Mus musculus", ScientificName: "Mus musculus", TaxonID: 10090, Tissue: "liver"},
	}
	for _, s := range samples {
		if err := db.InsertSample(s); err != nil {
			t.Fatalf("failed to insert sample: %v", err)
		}
	}
}

func TestNewMetadataService(t *testing.T) {
	svc, _, cleanup := setupTestMetadataService(t)
	defer cleanup()

	if svc == nil {
		t.Fatal("NewMetadataService returned nil")
	}
}

func TestGetStudy(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()
	study, err := svc.GetStudy(ctx, "SRP000001")
	if err != nil {
		t.Fatalf("GetStudy failed: %v", err)
	}
	if study.StudyAccession != "SRP000001" {
		t.Errorf("expected SRP000001, got %q", study.StudyAccession)
	}
	if study.StudyTitle != "Human Genome Study" {
		t.Errorf("expected 'Human Genome Study', got %q", study.StudyTitle)
	}
}

func TestGetStudyNotFound(t *testing.T) {
	svc, _, cleanup := setupTestMetadataService(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.GetStudy(ctx, "NONEXISTENT")
	if err == nil {
		t.Error("expected error for non-existent study")
	}
}

func TestGetStudies(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()
	studies, err := svc.GetStudies(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetStudies failed: %v", err)
	}
	if len(studies) != 2 {
		t.Errorf("expected 2 studies, got %d", len(studies))
	}
}

func TestGetStudiesPagination(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()

	// First page
	page1, err := svc.GetStudies(ctx, 1, 0)
	if err != nil {
		t.Fatalf("GetStudies page 1 failed: %v", err)
	}
	if len(page1) != 1 {
		t.Errorf("expected 1 study in page 1, got %d", len(page1))
	}

	// Second page
	page2, err := svc.GetStudies(ctx, 1, 1)
	if err != nil {
		t.Fatalf("GetStudies page 2 failed: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("expected 1 study in page 2, got %d", len(page2))
	}

	// Pages should have different studies
	if len(page1) > 0 && len(page2) > 0 && page1[0].StudyAccession == page2[0].StudyAccession {
		t.Error("paginated results should return different studies")
	}
}

func TestGetExperiment(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()
	exp, err := svc.GetExperiment(ctx, "SRX000001")
	if err != nil {
		t.Fatalf("GetExperiment failed: %v", err)
	}
	if exp.ExperimentAccession != "SRX000001" {
		t.Errorf("expected SRX000001, got %q", exp.ExperimentAccession)
	}
}

func TestGetExperimentsByStudy(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()
	exps, err := svc.GetExperimentsByStudy(ctx, "SRP000001")
	if err != nil {
		t.Fatalf("GetExperimentsByStudy failed: %v", err)
	}
	if len(exps) != 2 {
		t.Errorf("expected 2 experiments, got %d", len(exps))
	}
}

func TestGetSample(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()
	sample, err := svc.GetSample(ctx, "SRS000001")
	if err != nil {
		t.Fatalf("GetSample failed: %v", err)
	}
	if sample.SampleAccession != "SRS000001" {
		t.Errorf("expected SRS000001, got %q", sample.SampleAccession)
	}
}

func TestGetRun(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()
	run, err := svc.GetRun(ctx, "SRR000001")
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if run.RunAccession != "SRR000001" {
		t.Errorf("expected SRR000001, got %q", run.RunAccession)
	}
}

func TestGetRunsByExperiment(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()
	runs, err := svc.GetRunsByExperiment(ctx, "SRX000001")
	if err != nil {
		t.Fatalf("GetRunsByExperiment failed: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}

func TestGetRunsByStudy(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()

	// With limit
	runs, err := svc.GetRunsByStudy(ctx, "SRP000001", 1)
	if err != nil {
		t.Fatalf("GetRunsByStudy failed: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("expected 1 run with limit 1, got %d", len(runs))
	}

	// Without limit (0)
	allRuns, err := svc.GetRunsByStudy(ctx, "SRP000001", 0)
	if err != nil {
		t.Fatalf("GetRunsByStudy (no limit) failed: %v", err)
	}
	if len(allRuns) != 2 {
		t.Errorf("expected 2 runs without limit, got %d", len(allRuns))
	}
}

func TestGetMetadata(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()

	tests := []struct {
		reqType   string
		accession string
	}{
		{"study", "SRP000001"},
		{"experiment", "SRX000001"},
		{"sample", "SRS000001"},
		{"run", "SRR000001"},
	}

	for _, tt := range tests {
		t.Run(tt.reqType, func(t *testing.T) {
			req := &MetadataRequest{
				Type:      tt.reqType,
				Accession: tt.accession,
			}
			resp, err := svc.GetMetadata(ctx, req)
			if err != nil {
				t.Fatalf("GetMetadata(%s) failed: %v", tt.reqType, err)
			}
			if resp.Type != tt.reqType {
				t.Errorf("expected type %q, got %q", tt.reqType, resp.Type)
			}
			if resp.Data == nil {
				t.Error("expected non-nil data")
			}
		})
	}
}

func TestGetMetadataInvalidType(t *testing.T) {
	svc, _, cleanup := setupTestMetadataService(t)
	defer cleanup()

	ctx := context.Background()
	req := &MetadataRequest{
		Type:      "invalid",
		Accession: "SRP000001",
	}
	_, err := svc.GetMetadata(ctx, req)
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestGetAccessionType(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()

	tests := []struct {
		accession    string
		expectedType string
	}{
		{"SRP000001", "study"},
		{"SRX000001", "experiment"},
		{"SRS000001", "sample"},
		{"SRR000001", "run"},
	}

	for _, tt := range tests {
		t.Run(tt.accession, func(t *testing.T) {
			accType, err := svc.GetAccessionType(ctx, tt.accession)
			if err != nil {
				t.Fatalf("GetAccessionType(%s) failed: %v", tt.accession, err)
			}
			if accType != tt.expectedType {
				t.Errorf("expected type %q, got %q", tt.expectedType, accType)
			}
		})
	}
}

func TestGetAccessionTypeNotFound(t *testing.T) {
	svc, _, cleanup := setupTestMetadataService(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.GetAccessionType(ctx, "NONEXISTENT")
	if err == nil {
		t.Error("expected error for non-existent accession")
	}
}

func TestGetStudyMetadata(t *testing.T) {
	svc, db, cleanup := setupTestMetadataService(t)
	defer cleanup()
	seedTestData(t, db)

	ctx := context.Background()
	metadata, err := svc.GetStudyMetadata(ctx, "SRP000001")
	if err != nil {
		t.Fatalf("GetStudyMetadata failed: %v", err)
	}

	if metadata["study"] == nil {
		t.Error("expected study in metadata")
	}
	if metadata["experiments"] == nil {
		t.Error("expected experiments in metadata")
	}

	summary, ok := metadata["summary"].(map[string]int)
	if !ok {
		t.Fatal("expected summary in metadata")
	}
	if summary["total_experiments"] != 2 {
		t.Errorf("expected 2 experiments in summary, got %d", summary["total_experiments"])
	}
	if summary["total_runs"] != 2 {
		t.Errorf("expected 2 runs in summary, got %d", summary["total_runs"])
	}
}

func TestHealth(t *testing.T) {
	svc, _, cleanup := setupTestMetadataService(t)
	defer cleanup()

	ctx := context.Background()
	err := svc.Health(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
}

func TestClose(t *testing.T) {
	svc, _, cleanup := setupTestMetadataService(t)
	defer cleanup()

	err := svc.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
