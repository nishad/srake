package testutil

import (
	"testing"

	"github.com/nishad/srake/internal/database"
)

// TestDB creates a temporary in-memory database for testing.
// It returns the database and a cleanup function.
func TestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()

	// Create temp directory for the database
	dir, dirCleanup := TempDir(t)

	// Initialize database in temp directory
	db, err := database.Initialize(dir + "/test.db")
	if err != nil {
		dirCleanup()
		t.Fatalf("failed to create test database: %v", err)
	}

	return db, func() {
		db.Close()
		dirCleanup()
	}
}

// TestDBWithFixtures creates a test database and populates it with fixtures.
func TestDBWithFixtures(t *testing.T) (*database.DB, func()) {
	t.Helper()

	db, cleanup := TestDB(t)

	// Insert test fixtures
	if err := InsertTestStudies(db); err != nil {
		cleanup()
		t.Fatalf("failed to insert test studies: %v", err)
	}
	if err := InsertTestExperiments(db); err != nil {
		cleanup()
		t.Fatalf("failed to insert test experiments: %v", err)
	}
	if err := InsertTestSamples(db); err != nil {
		cleanup()
		t.Fatalf("failed to insert test samples: %v", err)
	}
	if err := InsertTestRuns(db); err != nil {
		cleanup()
		t.Fatalf("failed to insert test runs: %v", err)
	}

	return db, cleanup
}

// InsertTestStudies inserts test study records.
func InsertTestStudies(db *database.DB) error {
	studies := []database.Study{
		{
			StudyAccession: "SRP000001",
			StudyTitle:     "Human Cancer Genomics Study",
			StudyAbstract:  "A comprehensive study of cancer genomics in human tissue samples",
			StudyType:      "Other",
			Organism:       "Homo sapiens",
		},
		{
			StudyAccession: "SRP000002",
			StudyTitle:     "Mouse Brain Development",
			StudyAbstract:  "Study of neural development in mouse brain",
			StudyType:      "Transcriptome Analysis",
			Organism:       "Mus musculus",
		},
		{
			StudyAccession: "SRP000003",
			StudyTitle:     "COVID-19 Host Response",
			StudyAbstract:  "Analysis of host immune response to SARS-CoV-2 infection",
			StudyType:      "Other",
			Organism:       "Homo sapiens",
		},
	}

	for _, study := range studies {
		s := study // Create a copy to avoid pointer issues
		if err := db.InsertStudy(&s); err != nil {
			return err
		}
	}
	return nil
}

// InsertTestExperiments inserts test experiment records.
func InsertTestExperiments(db *database.DB) error {
	experiments := []database.Experiment{
		{
			ExperimentAccession: "SRX000001",
			Title:               "RNA-Seq of cancer cells",
			StudyAccession:      "SRP000001",
			Platform:            "ILLUMINA",
			LibraryStrategy:     "RNA-Seq",
			LibrarySource:       "TRANSCRIPTOMIC",
			LibrarySelection:    "cDNA",
			LibraryLayout:       "PAIRED",
			InstrumentModel:     "Illumina NovaSeq 6000",
		},
		{
			ExperimentAccession: "SRX000002",
			Title:               "Mouse brain single-cell RNA-seq",
			StudyAccession:      "SRP000002",
			Platform:            "ILLUMINA",
			LibraryStrategy:     "RNA-Seq",
			LibrarySource:       "TRANSCRIPTOMIC SINGLE CELL",
			LibrarySelection:    "cDNA",
			LibraryLayout:       "PAIRED",
			InstrumentModel:     "Illumina HiSeq 4000",
		},
		{
			ExperimentAccession: "SRX000003",
			Title:               "COVID-19 patient blood WGS",
			StudyAccession:      "SRP000003",
			Platform:            "ILLUMINA",
			LibraryStrategy:     "WGS",
			LibrarySource:       "GENOMIC",
			LibrarySelection:    "RANDOM",
			LibraryLayout:       "PAIRED",
			InstrumentModel:     "Illumina NovaSeq 6000",
		},
	}

	for _, exp := range experiments {
		e := exp
		if err := db.InsertExperiment(&e); err != nil {
			return err
		}
	}
	return nil
}

// InsertTestSamples inserts test sample records.
func InsertTestSamples(db *database.DB) error {
	samples := []database.Sample{
		{
			SampleAccession: "SRS000001",
			Organism:        "Homo sapiens",
			ScientificName:  "Homo sapiens",
			TaxonID:         9606,
			Description:     "Human breast cancer tissue",
			Tissue:          "breast",
			CellType:        "tumor",
		},
		{
			SampleAccession: "SRS000002",
			Organism:        "Mus musculus",
			ScientificName:  "Mus musculus",
			TaxonID:         10090,
			Description:     "Mouse brain cortex",
			Tissue:          "brain",
			CellType:        "neuron",
		},
		{
			SampleAccession: "SRS000003",
			Organism:        "Homo sapiens",
			ScientificName:  "Homo sapiens",
			TaxonID:         9606,
			Description:     "COVID-19 patient blood sample",
			Tissue:          "blood",
			CellType:        "PBMC",
		},
	}

	for _, sample := range samples {
		s := sample
		if err := db.InsertSample(&s); err != nil {
			return err
		}
	}
	return nil
}

// InsertTestRuns inserts test run records.
func InsertTestRuns(db *database.DB) error {
	runs := []database.Run{
		{
			RunAccession:        "SRR000001",
			ExperimentAccession: "SRX000001",
			TotalSpots:          100000000,
			TotalBases:          30000000000,
			TotalSize:           5000000000,
			Published:           "2023-01-15",
		},
		{
			RunAccession:        "SRR000002",
			ExperimentAccession: "SRX000002",
			TotalSpots:          50000000,
			TotalBases:          15000000000,
			TotalSize:           2500000000,
			Published:           "2023-02-20",
		},
		{
			RunAccession:        "SRR000003",
			ExperimentAccession: "SRX000003",
			TotalSpots:          200000000,
			TotalBases:          60000000000,
			TotalSize:           10000000000,
			Published:           "2023-03-10",
		},
	}

	for _, run := range runs {
		r := run
		if err := db.InsertRun(&r); err != nil {
			return err
		}
	}
	return nil
}
