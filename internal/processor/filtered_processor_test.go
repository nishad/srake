package processor

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/nishad/srake/internal/database"
)

// filterMockDatabase tracks per-type insert counts for filter test assertions.
type filterMockDatabase struct {
	studiesInserted     int
	experimentsInserted int
	samplesInserted     int
	runsInserted        int
}

func newFilterMockDatabase() *filterMockDatabase { return &filterMockDatabase{} }

func (m *filterMockDatabase) InsertStudy(*database.Study) error {
	m.studiesInserted++
	return nil
}
func (m *filterMockDatabase) InsertExperiment(*database.Experiment) error {
	m.experimentsInserted++
	return nil
}
func (m *filterMockDatabase) InsertSample(*database.Sample) error {
	m.samplesInserted++
	return nil
}
func (m *filterMockDatabase) InsertRun(*database.Run) error {
	m.runsInserted++
	return nil
}
func (m *filterMockDatabase) BatchInsertExperiments(exps []database.Experiment) error {
	m.experimentsInserted += len(exps)
	return nil
}
func (m *filterMockDatabase) InsertSubmission(*database.Submission) error { return nil }
func (m *filterMockDatabase) InsertAnalysis(*database.Analysis) error     { return nil }
func (m *filterMockDatabase) InsertSamplePool(*database.SamplePool) error { return nil }
func (m *filterMockDatabase) GetSamplePools(string) ([]database.SamplePool, error) {
	return nil, nil
}
func (m *filterMockDatabase) CountSamplePools() (int, error)       { return 0, nil }
func (m *filterMockDatabase) GetAveragePoolSize() (float64, error) { return 0, nil }
func (m *filterMockDatabase) GetMaxPoolSize() (int, error)         { return 0, nil }
func (m *filterMockDatabase) InsertIdentifier(*database.Identifier) error {
	return nil
}
func (m *filterMockDatabase) GetIdentifiers(string, string) ([]database.Identifier, error) {
	return nil, nil
}
func (m *filterMockDatabase) FindRecordsByIdentifier(string) ([]database.Identifier, error) {
	return nil, nil
}
func (m *filterMockDatabase) InsertLink(*database.Link) error                  { return nil }
func (m *filterMockDatabase) GetLinks(string, string) ([]database.Link, error) { return nil, nil }

func (m *filterMockDatabase) totalInserted() int {
	return m.studiesInserted + m.experimentsInserted + m.samplesInserted + m.runsInserted
}

// buildFilterTestArchive creates a tar.gz in memory with known records for
// filter integration tests.
//
// Contents:
//   - 4 samples:  2x Homo sapiens (9606), 1x Mus musculus (10090), 1x Drosophila (7227)
//   - 4 experiments: 2x ILLUMINA+RNA-Seq, 1x OXFORD_NANOPORE+WGS, 1x ILLUMINA+WGS
//   - 3 studies:  1x date 2024-06-15, 1x date 2020-01-01, 1x no date
//   - 3 runs:     2x high reads (1M, 5M spots), 1x low reads (100 spots)
func buildFilterTestArchive(t *testing.T) []byte {
	t.Helper()

	files := []struct {
		name    string
		content string
	}{
		{
			name: "test_sample.xml",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<SAMPLE_SET>
  <SAMPLE accession="SRS0001">
    <SAMPLE_NAME>
      <TAXON_ID>9606</TAXON_ID>
      <SCIENTIFIC_NAME>Homo sapiens</SCIENTIFIC_NAME>
    </SAMPLE_NAME>
    <DESCRIPTION>Human sample 1</DESCRIPTION>
  </SAMPLE>
  <SAMPLE accession="SRS0002">
    <SAMPLE_NAME>
      <TAXON_ID>9606</TAXON_ID>
      <SCIENTIFIC_NAME>Homo sapiens</SCIENTIFIC_NAME>
    </SAMPLE_NAME>
    <DESCRIPTION>Human sample 2</DESCRIPTION>
  </SAMPLE>
  <SAMPLE accession="SRS0003">
    <SAMPLE_NAME>
      <TAXON_ID>10090</TAXON_ID>
      <SCIENTIFIC_NAME>Mus musculus</SCIENTIFIC_NAME>
    </SAMPLE_NAME>
    <DESCRIPTION>Mouse sample</DESCRIPTION>
  </SAMPLE>
  <SAMPLE accession="SRS0004">
    <SAMPLE_NAME>
      <TAXON_ID>7227</TAXON_ID>
      <SCIENTIFIC_NAME>Drosophila melanogaster</SCIENTIFIC_NAME>
    </SAMPLE_NAME>
    <DESCRIPTION>Fly sample</DESCRIPTION>
  </SAMPLE>
</SAMPLE_SET>`,
		},
		{
			name: "test_experiment.xml",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<EXPERIMENT_SET>
  <EXPERIMENT accession="SRX0001">
    <TITLE>Illumina RNA-Seq 1</TITLE>
    <STUDY_REF accession="SRP0001"/>
    <DESIGN>
      <SAMPLE_DESCRIPTOR accession="SRS0001"/>
      <LIBRARY_DESCRIPTOR>
        <LIBRARY_STRATEGY>RNA-Seq</LIBRARY_STRATEGY>
        <LIBRARY_SOURCE>TRANSCRIPTOMIC</LIBRARY_SOURCE>
        <LIBRARY_SELECTION>cDNA</LIBRARY_SELECTION>
      </LIBRARY_DESCRIPTOR>
    </DESIGN>
    <PLATFORM>
      <ILLUMINA><INSTRUMENT_MODEL>HiSeq 2500</INSTRUMENT_MODEL></ILLUMINA>
    </PLATFORM>
  </EXPERIMENT>
  <EXPERIMENT accession="SRX0002">
    <TITLE>Illumina RNA-Seq 2</TITLE>
    <STUDY_REF accession="SRP0001"/>
    <DESIGN>
      <SAMPLE_DESCRIPTOR accession="SRS0002"/>
      <LIBRARY_DESCRIPTOR>
        <LIBRARY_STRATEGY>RNA-Seq</LIBRARY_STRATEGY>
        <LIBRARY_SOURCE>TRANSCRIPTOMIC</LIBRARY_SOURCE>
        <LIBRARY_SELECTION>cDNA</LIBRARY_SELECTION>
      </LIBRARY_DESCRIPTOR>
    </DESIGN>
    <PLATFORM>
      <ILLUMINA><INSTRUMENT_MODEL>NovaSeq 6000</INSTRUMENT_MODEL></ILLUMINA>
    </PLATFORM>
  </EXPERIMENT>
  <EXPERIMENT accession="SRX0003">
    <TITLE>Nanopore WGS</TITLE>
    <STUDY_REF accession="SRP0002"/>
    <DESIGN>
      <SAMPLE_DESCRIPTOR accession="SRS0003"/>
      <LIBRARY_DESCRIPTOR>
        <LIBRARY_STRATEGY>WGS</LIBRARY_STRATEGY>
        <LIBRARY_SOURCE>GENOMIC</LIBRARY_SOURCE>
        <LIBRARY_SELECTION>RANDOM</LIBRARY_SELECTION>
      </LIBRARY_DESCRIPTOR>
    </DESIGN>
    <PLATFORM>
      <OXFORD_NANOPORE><INSTRUMENT_MODEL>PromethION</INSTRUMENT_MODEL></OXFORD_NANOPORE>
    </PLATFORM>
  </EXPERIMENT>
  <EXPERIMENT accession="SRX0004">
    <TITLE>Illumina WGS</TITLE>
    <STUDY_REF accession="SRP0003"/>
    <DESIGN>
      <SAMPLE_DESCRIPTOR accession="SRS0004"/>
      <LIBRARY_DESCRIPTOR>
        <LIBRARY_STRATEGY>WGS</LIBRARY_STRATEGY>
        <LIBRARY_SOURCE>GENOMIC</LIBRARY_SOURCE>
        <LIBRARY_SELECTION>RANDOM</LIBRARY_SELECTION>
      </LIBRARY_DESCRIPTOR>
    </DESIGN>
    <PLATFORM>
      <ILLUMINA><INSTRUMENT_MODEL>HiSeq X Ten</INSTRUMENT_MODEL></ILLUMINA>
    </PLATFORM>
  </EXPERIMENT>
</EXPERIMENT_SET>`,
		},
		{
			name: "test_study.xml",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<STUDY_SET>
  <STUDY accession="SRP0001">
    <DESCRIPTOR>
      <STUDY_TITLE>Recent RNA-Seq study</STUDY_TITLE>
      <STUDY_ABSTRACT>Recent transcriptomic work</STUDY_ABSTRACT>
      <STUDY_TYPE existing_study_type="Transcriptome Analysis"/>
    </DESCRIPTOR>
    <STUDY_ATTRIBUTES>
      <STUDY_ATTRIBUTE><TAG>ENA-FIRST-PUBLIC</TAG><VALUE>2024-06-15</VALUE></STUDY_ATTRIBUTE>
    </STUDY_ATTRIBUTES>
  </STUDY>
  <STUDY accession="SRP0002">
    <DESCRIPTOR>
      <STUDY_TITLE>Older WGS study</STUDY_TITLE>
      <STUDY_ABSTRACT>Older genomic survey</STUDY_ABSTRACT>
      <STUDY_TYPE existing_study_type="Whole Genome Sequencing"/>
    </DESCRIPTOR>
    <STUDY_ATTRIBUTES>
      <STUDY_ATTRIBUTE><TAG>ENA-FIRST-PUBLIC</TAG><VALUE>2020-01-01</VALUE></STUDY_ATTRIBUTE>
    </STUDY_ATTRIBUTES>
  </STUDY>
  <STUDY accession="SRP0003">
    <DESCRIPTOR>
      <STUDY_TITLE>No-date study</STUDY_TITLE>
      <STUDY_ABSTRACT>Study without publication date</STUDY_ABSTRACT>
      <STUDY_TYPE existing_study_type="Other"/>
    </DESCRIPTOR>
  </STUDY>
</STUDY_SET>`,
		},
		{
			name: "test_run.xml",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<RUN_SET>
  <RUN accession="SRR0001">
    <EXPERIMENT_REF accession="SRX0001"/>
    <Statistics total_spots="1000000" total_bases="150000000"/>
  </RUN>
  <RUN accession="SRR0002">
    <EXPERIMENT_REF accession="SRX0002"/>
    <Statistics total_spots="5000000" total_bases="750000000"/>
  </RUN>
  <RUN accession="SRR0003">
    <EXPERIMENT_REF accession="SRX0003"/>
    <Statistics total_spots="100" total_bases="15000"/>
  </RUN>
</RUN_SET>`,
		},
	}

	var buf bytes.Buffer
	gzW := gzip.NewWriter(&buf)
	tarW := tar.NewWriter(gzW)

	for _, f := range files {
		hdr := &tar.Header{
			Name:     f.name,
			Mode:     0644,
			Size:     int64(len(f.content)),
			Typeflag: tar.TypeReg,
			ModTime:  time.Now(),
		}
		if err := tarW.WriteHeader(hdr); err != nil {
			t.Fatalf("tar header: %v", err)
		}
		if _, err := io.WriteString(tarW, f.content); err != nil {
			t.Fatalf("tar write: %v", err)
		}
	}
	tarW.Close()
	gzW.Close()
	return buf.Bytes()
}

// writeTempArchive writes data to a temp file and returns the path.
func writeTempArchive(t *testing.T, data []byte) string {
	t.Helper()
	path := t.TempDir() + "/test.tar.gz"
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

// processFilterTestArchive creates a FilteredProcessor, feeds it the test
// archive, and returns the mock database + filter stats.
func processFilterTestArchive(t *testing.T, opts FilterOptions) (*filterMockDatabase, *FilterStats) {
	t.Helper()

	data := buildFilterTestArchive(t)
	tmp := writeTempArchive(t, data)

	db := newFilterMockDatabase()
	fp, err := NewFilteredProcessor(db, opts)
	if err != nil {
		t.Fatalf("NewFilteredProcessor: %v", err)
	}

	if err := fp.ProcessWithFilters(context.Background(), tmp); err != nil {
		t.Fatalf("ProcessWithFilters: %v", err)
	}

	return db, fp.GetStats()
}

// ---- Tests ----------------------------------------------------------------

func TestTaxonomyFilter(t *testing.T) {
	db, stats := processFilterTestArchive(t, FilterOptions{
		TaxonomyIDs: []int{9606},
	})

	// 2 of 4 samples should match (Homo sapiens), 2 skipped by taxonomy
	if stats.SkippedByTaxonomy != 2 {
		t.Errorf("SkippedByTaxonomy: got %d, want 2", stats.SkippedByTaxonomy)
	}
	if db.samplesInserted != 2 {
		t.Errorf("samplesInserted: got %d, want 2", db.samplesInserted)
	}

	// TotalMatched counts all record types that pass. Taxonomy filter only
	// applies to samples, so studies (3), experiments (4), and runs (3) all
	// pass = 10, plus 2 matched samples = 12.
	if stats.TotalMatched != 12 {
		t.Errorf("TotalMatched: got %d, want 12", stats.TotalMatched)
	}

	// Studies, experiments, and runs have no taxonomy filter — all pass
	if db.studiesInserted != 3 {
		t.Errorf("studiesInserted: got %d, want 3", db.studiesInserted)
	}
	if db.experimentsInserted != 4 {
		t.Errorf("experimentsInserted: got %d, want 4", db.experimentsInserted)
	}
	if db.runsInserted != 3 {
		t.Errorf("runsInserted: got %d, want 3", db.runsInserted)
	}
}

func TestPlatformAndStrategyFilter(t *testing.T) {
	db, stats := processFilterTestArchive(t, FilterOptions{
		Platforms:  []string{"ILLUMINA"},
		Strategies: []string{"RNA-Seq"},
	})

	// Experiments: SRX0001 (ILLUMINA+RNA-Seq) pass, SRX0002 (ILLUMINA+RNA-Seq) pass,
	//   SRX0003 (OXFORD_NANOPORE+WGS) skip, SRX0004 (ILLUMINA+WGS) skip
	if db.experimentsInserted != 2 {
		t.Errorf("experimentsInserted: got %d, want 2", db.experimentsInserted)
	}
	if stats.SkippedByPlatform+stats.SkippedByStrategy != 2 {
		t.Errorf("total skipped experiments: got %d, want 2",
			stats.SkippedByPlatform+stats.SkippedByStrategy)
	}
}

func TestMinReadsFilter(t *testing.T) {
	db, stats := processFilterTestArchive(t, FilterOptions{
		MinReads: 500000,
	})

	// Runs: SRR0001 (1M) pass, SRR0002 (5M) pass, SRR0003 (100) skip
	if db.runsInserted != 2 {
		t.Errorf("runsInserted: got %d, want 2", db.runsInserted)
	}
	if stats.SkippedByReads != 1 {
		t.Errorf("SkippedByReads: got %d, want 1", stats.SkippedByReads)
	}
}

func TestStatsOnlyMode(t *testing.T) {
	db, stats := processFilterTestArchive(t, FilterOptions{
		TaxonomyIDs: []int{9606},
		StatsOnly:   true,
	})

	// Should count matched records but insert nothing
	if stats.TotalMatched < 2 {
		t.Errorf("TotalMatched: got %d, want >=2", stats.TotalMatched)
	}
	if db.totalInserted() != 0 {
		t.Errorf("totalInserted: got %d, want 0 (stats-only mode)", db.totalInserted())
	}
}

func TestNoFiltersPassAll(t *testing.T) {
	db, stats := processFilterTestArchive(t, FilterOptions{})

	// No filters: everything passes, nothing skipped
	if stats.TotalSkipped != 0 {
		t.Errorf("TotalSkipped: got %d, want 0", stats.TotalSkipped)
	}
	if db.samplesInserted != 4 {
		t.Errorf("samplesInserted: got %d, want 4", db.samplesInserted)
	}
	if db.experimentsInserted != 4 {
		t.Errorf("experimentsInserted: got %d, want 4", db.experimentsInserted)
	}
	if db.studiesInserted != 3 {
		t.Errorf("studiesInserted: got %d, want 3", db.studiesInserted)
	}
	if db.runsInserted != 3 {
		t.Errorf("runsInserted: got %d, want 3", db.runsInserted)
	}
}

func TestProgressCallback(t *testing.T) {
	data := buildFilterTestArchive(t)
	tmp := writeTempArchive(t, data)

	db := newFilterMockDatabase()
	fp, err := NewFilteredProcessor(db, FilterOptions{
		TaxonomyIDs: []int{9606},
	})
	if err != nil {
		t.Fatalf("NewFilteredProcessor: %v", err)
	}

	callbackCount := 0
	fp.SetProgressFunc(func(p Progress) {
		callbackCount++
	})

	if err := fp.ProcessWithFilters(context.Background(), tmp); err != nil {
		t.Fatalf("ProcessWithFilters: %v", err)
	}

	if callbackCount == 0 {
		t.Error("progress callback was never called")
	}

	stats := fp.GetStats()
	if stats.TotalMatched == 0 {
		t.Error("expected non-zero TotalMatched in callback test")
	}
}
