package database

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper to create a temporary test database
func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "srake-db-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	db, err := Initialize(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to initialize database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}

	return db, cleanup
}

func TestInitialize(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("expected non-nil database")
	}

	// Test that we can ping the database
	if err := db.Ping(); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestInitializeInvalidPath(t *testing.T) {
	// Try to initialize in a non-existent directory
	_, err := Initialize("/nonexistent/path/test.db")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestStudyOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test InsertStudy
	study := &Study{
		StudyAccession: "SRP000001",
		StudyTitle:     "Test Study",
		StudyAbstract:  "A test study for unit testing",
		StudyType:      "Other",
		Organism:       "Homo sapiens",
	}

	err := db.InsertStudy(study)
	if err != nil {
		t.Fatalf("InsertStudy failed: %v", err)
	}

	// Test GetStudy
	retrieved, err := db.GetStudy("SRP000001")
	if err != nil {
		t.Fatalf("GetStudy failed: %v", err)
	}

	if retrieved.StudyAccession != study.StudyAccession {
		t.Errorf("got accession %q, want %q", retrieved.StudyAccession, study.StudyAccession)
	}
	if retrieved.StudyTitle != study.StudyTitle {
		t.Errorf("got title %q, want %q", retrieved.StudyTitle, study.StudyTitle)
	}
	if retrieved.Organism != study.Organism {
		t.Errorf("got organism %q, want %q", retrieved.Organism, study.Organism)
	}

	// Test GetStudy with non-existent accession
	_, err = db.GetStudy("NONEXISTENT")
	if err == nil {
		t.Error("expected error for non-existent study, got nil")
	}
}

func TestExperimentOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// First insert a study
	study := &Study{
		StudyAccession: "SRP000001",
		StudyTitle:     "Test Study",
	}
	if err := db.InsertStudy(study); err != nil {
		t.Fatalf("failed to insert study: %v", err)
	}

	// Test InsertExperiment
	exp := &Experiment{
		ExperimentAccession: "SRX000001",
		Title:               "Test Experiment",
		StudyAccession:      "SRP000001",
		Platform:            "ILLUMINA",
		LibraryStrategy:     "RNA-Seq",
		LibrarySource:       "TRANSCRIPTOMIC",
		InstrumentModel:     "Illumina NovaSeq 6000",
	}

	err := db.InsertExperiment(exp)
	if err != nil {
		t.Fatalf("InsertExperiment failed: %v", err)
	}

	// Test GetExperiment
	retrieved, err := db.GetExperiment("SRX000001")
	if err != nil {
		t.Fatalf("GetExperiment failed: %v", err)
	}

	if retrieved.ExperimentAccession != exp.ExperimentAccession {
		t.Errorf("got accession %q, want %q", retrieved.ExperimentAccession, exp.ExperimentAccession)
	}
	if retrieved.Platform != exp.Platform {
		t.Errorf("got platform %q, want %q", retrieved.Platform, exp.Platform)
	}
}

func TestSampleOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test InsertSample
	sample := &Sample{
		SampleAccession: "SRS000001",
		Organism:        "Homo sapiens",
		ScientificName:  "Homo sapiens",
		TaxonID:         9606,
		Description:     "Human blood sample",
		Tissue:          "blood",
		CellType:        "PBMC",
	}

	err := db.InsertSample(sample)
	if err != nil {
		t.Fatalf("InsertSample failed: %v", err)
	}

	// Test GetSample
	retrieved, err := db.GetSample("SRS000001")
	if err != nil {
		t.Fatalf("GetSample failed: %v", err)
	}

	if retrieved.SampleAccession != sample.SampleAccession {
		t.Errorf("got accession %q, want %q", retrieved.SampleAccession, sample.SampleAccession)
	}
	if retrieved.Organism != sample.Organism {
		t.Errorf("got organism %q, want %q", retrieved.Organism, sample.Organism)
	}
	if retrieved.TaxonID != sample.TaxonID {
		t.Errorf("got taxon_id %d, want %d", retrieved.TaxonID, sample.TaxonID)
	}
}

func TestRunOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert prerequisite experiment
	exp := &Experiment{
		ExperimentAccession: "SRX000001",
		Title:               "Test Experiment",
		Platform:            "ILLUMINA",
	}
	if err := db.InsertExperiment(exp); err != nil {
		t.Fatalf("failed to insert experiment: %v", err)
	}

	// Test InsertRun
	run := &Run{
		RunAccession:        "SRR000001",
		ExperimentAccession: "SRX000001",
		TotalSpots:         100000000,
		TotalBases:         30000000000,
		TotalSize:          5000000000,
		Published:          "2023-01-15",
	}

	err := db.InsertRun(run)
	if err != nil {
		t.Fatalf("InsertRun failed: %v", err)
	}

	// Test GetRun
	retrieved, err := db.GetRun("SRR000001")
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}

	if retrieved.RunAccession != run.RunAccession {
		t.Errorf("got accession %q, want %q", retrieved.RunAccession, run.RunAccession)
	}
	if retrieved.TotalSpots != run.TotalSpots {
		t.Errorf("got total_spots %d, want %d", retrieved.TotalSpots, run.TotalSpots)
	}
	if retrieved.TotalBases != run.TotalBases {
		t.Errorf("got total_bases %d, want %d", retrieved.TotalBases, run.TotalBases)
	}
}

func TestCountTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Initial count should be 0
	count, err := db.CountTable("studies")
	if err != nil {
		t.Fatalf("CountTable failed: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}

	// Insert some studies
	for i := 0; i < 5; i++ {
		study := &Study{
			StudyAccession: "SRP" + string(rune('0'+i)),
			StudyTitle:     "Test Study",
		}
		if err := db.InsertStudy(study); err != nil {
			t.Fatalf("failed to insert study: %v", err)
		}
	}

	// Count should now be 5
	count, err = db.CountTable("studies")
	if err != nil {
		t.Fatalf("CountTable failed: %v", err)
	}
	if count != 5 {
		t.Errorf("got count %d, want 5", count)
	}
}

func TestCountTableInvalidTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Try to count an invalid table
	_, err := db.CountTable("invalid_table")
	if err == nil {
		t.Error("expected error for invalid table, got nil")
	}
}

func TestSearchByOrganism(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert samples with different organisms
	samples := []*Sample{
		{SampleAccession: "SRS000001", Organism: "Homo sapiens", ScientificName: "Homo sapiens", TaxonID: 9606},
		{SampleAccession: "SRS000002", Organism: "Homo sapiens", ScientificName: "Homo sapiens", TaxonID: 9606},
		{SampleAccession: "SRS000003", Organism: "Mus musculus", ScientificName: "Mus musculus", TaxonID: 10090},
	}

	for _, s := range samples {
		if err := db.InsertSample(s); err != nil {
			t.Fatalf("failed to insert sample: %v", err)
		}
	}

	// Search for human samples
	results, err := db.SearchByOrganism("Homo sapiens", 10)
	if err != nil {
		t.Fatalf("SearchByOrganism failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}

	// Search for mouse samples
	results, err = db.SearchByOrganism("Mus musculus", 10)
	if err != nil {
		t.Fatalf("SearchByOrganism failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
}

func TestSearchByLibraryStrategy(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert experiments with different strategies
	experiments := []*Experiment{
		{ExperimentAccession: "SRX000001", Title: "RNA-Seq 1", LibraryStrategy: "RNA-Seq", Platform: "ILLUMINA"},
		{ExperimentAccession: "SRX000002", Title: "RNA-Seq 2", LibraryStrategy: "RNA-Seq", Platform: "ILLUMINA"},
		{ExperimentAccession: "SRX000003", Title: "WGS 1", LibraryStrategy: "WGS", Platform: "ILLUMINA"},
	}

	for _, e := range experiments {
		if err := db.InsertExperiment(e); err != nil {
			t.Fatalf("failed to insert experiment: %v", err)
		}
	}

	// Search for RNA-Seq experiments
	results, err := db.SearchByLibraryStrategy("RNA-Seq", 10)
	if err != nil {
		t.Fatalf("SearchByLibraryStrategy failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}

	// Search for WGS experiments
	results, err = db.SearchByLibraryStrategy("WGS", 10)
	if err != nil {
		t.Fatalf("SearchByLibraryStrategy failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
}

func TestGetStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert some data
	study := &Study{StudyAccession: "SRP000001", StudyTitle: "Test"}
	exp := &Experiment{ExperimentAccession: "SRX000001", Title: "Test", Platform: "ILLUMINA"}
	sample := &Sample{SampleAccession: "SRS000001", Organism: "Homo sapiens", TaxonID: 9606}
	run := &Run{RunAccession: "SRR000001", ExperimentAccession: "SRX000001", TotalSpots: 1000}

	db.InsertStudy(study)
	db.InsertExperiment(exp)
	db.InsertSample(sample)
	db.InsertRun(run)

	// Get stats
	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalStudies != 1 {
		t.Errorf("got %d studies, want 1", stats.TotalStudies)
	}
	if stats.TotalExperiments != 1 {
		t.Errorf("got %d experiments, want 1", stats.TotalExperiments)
	}
	if stats.TotalSamples != 1 {
		t.Errorf("got %d samples, want 1", stats.TotalSamples)
	}
	if stats.TotalRuns != 1 {
		t.Errorf("got %d runs, want 1", stats.TotalRuns)
	}
}

func TestBatchInsertExperiments(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	experiments := []Experiment{
		{ExperimentAccession: "SRX000001", Title: "Exp 1", Platform: "ILLUMINA"},
		{ExperimentAccession: "SRX000002", Title: "Exp 2", Platform: "ILLUMINA"},
		{ExperimentAccession: "SRX000003", Title: "Exp 3", Platform: "PACBIO"},
	}

	err := db.BatchInsertExperiments(experiments)
	if err != nil {
		t.Fatalf("BatchInsertExperiments failed: %v", err)
	}

	// Verify all were inserted
	count, err := db.CountTable("experiments")
	if err != nil {
		t.Fatalf("CountTable failed: %v", err)
	}
	if count != 3 {
		t.Errorf("got count %d, want 3", count)
	}
}

func TestInsertAndRetrieveSamplePool(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert sample pool
	pool := &SamplePool{
		ParentSample: "SRS000001",
		MemberSample: "SRS000002",
		MemberName:   "member1",
		Proportion:   0.5,
	}

	err := db.InsertSamplePool(pool)
	if err != nil {
		t.Fatalf("InsertSamplePool failed: %v", err)
	}

	// Retrieve pools
	pools, err := db.GetSamplePools("SRS000001")
	if err != nil {
		t.Fatalf("GetSamplePools failed: %v", err)
	}

	if len(pools) != 1 {
		t.Errorf("got %d pools, want 1", len(pools))
	}
	if pools[0].MemberSample != "SRS000002" {
		t.Errorf("got member %q, want SRS000002", pools[0].MemberSample)
	}
}

func TestInsertAndRetrieveIdentifier(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert identifier
	id := &Identifier{
		RecordType:      "study",
		RecordAccession: "SRP000001",
		IDType:          "GEO",
		IDValue:         "GSE123456",
	}

	err := db.InsertIdentifier(id)
	if err != nil {
		t.Fatalf("InsertIdentifier failed: %v", err)
	}

	// Retrieve identifiers
	ids, err := db.GetIdentifiers("study", "SRP000001")
	if err != nil {
		t.Fatalf("GetIdentifiers failed: %v", err)
	}

	if len(ids) != 1 {
		t.Errorf("got %d identifiers, want 1", len(ids))
	}
	if ids[0].IDValue != "GSE123456" {
		t.Errorf("got value %q, want GSE123456", ids[0].IDValue)
	}
}

func TestInsertAndRetrieveLink(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert link
	link := &Link{
		RecordType:      "study",
		RecordAccession: "SRP000001",
		LinkType:        "url",
		URL:             "https://example.com/study",
		Label:           "Study Website",
	}

	err := db.InsertLink(link)
	if err != nil {
		t.Fatalf("InsertLink failed: %v", err)
	}

	// Retrieve links
	links, err := db.GetLinks("study", "SRP000001")
	if err != nil {
		t.Fatalf("GetLinks failed: %v", err)
	}

	if len(links) != 1 {
		t.Errorf("got %d links, want 1", len(links))
	}
	if links[0].URL != "https://example.com/study" {
		t.Errorf("got URL %q, want https://example.com/study", links[0].URL)
	}
}

func TestSubmissionOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	submission := &Submission{
		SubmissionAccession: "SRA000001",
		CenterName:          "Test Center",
		LabName:             "Test Lab",
	}

	err := db.InsertSubmission(submission)
	if err != nil {
		t.Fatalf("InsertSubmission failed: %v", err)
	}

	retrieved, err := db.GetSubmission("SRA000001")
	if err != nil {
		t.Fatalf("GetSubmission failed: %v", err)
	}

	if retrieved.SubmissionAccession != submission.SubmissionAccession {
		t.Errorf("got accession %q, want %q", retrieved.SubmissionAccession, submission.SubmissionAccession)
	}
	if retrieved.CenterName != submission.CenterName {
		t.Errorf("got center %q, want %q", retrieved.CenterName, submission.CenterName)
	}
}

func TestAnalysisOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	analysis := &Analysis{
		AnalysisAccession: "SRZ000001",
		Title:             "Test Analysis",
		Description:       "Analysis for testing",
		AnalysisType:      "SEQUENCE_ASSEMBLY",
	}

	err := db.InsertAnalysis(analysis)
	if err != nil {
		t.Fatalf("InsertAnalysis failed: %v", err)
	}

	retrieved, err := db.GetAnalysis("SRZ000001")
	if err != nil {
		t.Fatalf("GetAnalysis failed: %v", err)
	}

	if retrieved.AnalysisAccession != analysis.AnalysisAccession {
		t.Errorf("got accession %q, want %q", retrieved.AnalysisAccession, analysis.AnalysisAccession)
	}
	if retrieved.AnalysisType != analysis.AnalysisType {
		t.Errorf("got type %q, want %q", retrieved.AnalysisType, analysis.AnalysisType)
	}
}

func TestGetStudiesBatch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert multiple studies
	for i := 0; i < 10; i++ {
		study := &Study{
			StudyAccession: "SRP00000" + string(rune('0'+i)),
			StudyTitle:     "Test Study " + string(rune('0'+i)),
		}
		if err := db.InsertStudy(study); err != nil {
			t.Fatalf("failed to insert study: %v", err)
		}
	}

	// Get first batch
	batch, err := db.GetStudiesBatch(0, 5)
	if err != nil {
		t.Fatalf("GetStudiesBatch failed: %v", err)
	}
	if len(batch) != 5 {
		t.Errorf("got %d studies, want 5", len(batch))
	}

	// Get second batch
	batch, err = db.GetStudiesBatch(5, 5)
	if err != nil {
		t.Fatalf("GetStudiesBatch failed: %v", err)
	}
	if len(batch) != 5 {
		t.Errorf("got %d studies, want 5", len(batch))
	}
}
