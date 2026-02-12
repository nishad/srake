package testutil

import (
	"github.com/nishad/srake/internal/database"
)

// Fixture data for tests

// TestStudy returns a test study with sensible defaults.
func TestStudy() *database.Study {
	return &database.Study{
		StudyAccession: "SRP999999",
		StudyTitle:     "Test Study for Unit Tests",
		StudyAbstract:  "A study created for testing purposes",
		StudyType:      "Other",
		Organism:       "Homo sapiens",
	}
}

// TestExperiment returns a test experiment with sensible defaults.
func TestExperiment() *database.Experiment {
	return &database.Experiment{
		ExperimentAccession: "SRX999999",
		Title:               "Test Experiment for Unit Tests",
		StudyAccession:      "SRP999999",
		Platform:            "ILLUMINA",
		LibraryStrategy:     "RNA-Seq",
		LibrarySource:       "TRANSCRIPTOMIC",
		LibrarySelection:    "cDNA",
		LibraryLayout:       "PAIRED",
		InstrumentModel:     "Illumina NovaSeq 6000",
	}
}

// TestSample returns a test sample with sensible defaults.
func TestSample() *database.Sample {
	return &database.Sample{
		SampleAccession: "SRS999999",
		Organism:        "Homo sapiens",
		ScientificName:  "Homo sapiens",
		TaxonID:         9606,
		Description:     "Test sample for unit tests",
		Tissue:          "blood",
		CellType:        "PBMC",
	}
}

// TestRun returns a test run with sensible defaults.
func TestRun() *database.Run {
	return &database.Run{
		RunAccession:        "SRR999999",
		ExperimentAccession: "SRX999999",
		TotalSpots:          1000000,
		TotalBases:          300000000,
		TotalSize:           50000000,
		Published:           "2024-01-01",
	}
}

// TestSubmission returns a test submission with sensible defaults.
func TestSubmission() *database.Submission {
	return &database.Submission{
		SubmissionAccession: "SRA999999",
		LabName:             "Test Lab",
		CenterName:          "Test Center",
	}
}

// StudyWithAccession returns a test study with a specific accession.
func StudyWithAccession(accession string) *database.Study {
	s := TestStudy()
	s.StudyAccession = accession
	return s
}

// ExperimentWithAccession returns a test experiment with a specific accession.
func ExperimentWithAccession(accession, studyAccession string) *database.Experiment {
	e := TestExperiment()
	e.ExperimentAccession = accession
	e.StudyAccession = studyAccession
	return e
}

// SampleWithOrganism returns a test sample with a specific organism.
func SampleWithOrganism(accession, organism string) *database.Sample {
	s := TestSample()
	s.SampleAccession = accession
	s.Organism = organism
	s.ScientificName = organism
	return s
}

// RunWithStats returns a test run with specific statistics.
func RunWithStats(accession, expAccession string, spots, bases int64) *database.Run {
	r := TestRun()
	r.RunAccession = accession
	r.ExperimentAccession = expAccession
	r.TotalSpots = spots
	r.TotalBases = bases
	return r
}

// Organisms commonly used in tests
var (
	OrganismHuman = "Homo sapiens"
	OrganismMouse = "Mus musculus"
	OrganismYeast = "Saccharomyces cerevisiae"
	OrganismEColi = "Escherichia coli"
	OrganismFly   = "Drosophila melanogaster"
	OrganismWorm  = "Caenorhabditis elegans"
)

// Library strategies commonly used in tests
var (
	StrategyRNASeq  = "RNA-Seq"
	StrategyWGS     = "WGS"
	StrategyChIPSeq = "ChIP-Seq"
	StrategyATACSeq = "ATAC-Seq"
	StrategyWXS     = "WXS"
)

// Platforms commonly used in tests
var (
	PlatformIllumina   = "ILLUMINA"
	PlatformPacBio     = "PACBIO_SMRT"
	PlatformOxford     = "OXFORD_NANOPORE"
	PlatformIONTorrent = "ION_TORRENT"
)
