package processor

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nishad/srake/internal/database"
)

// TestStreamProcessor tests the HTTP streaming processor
func TestStreamProcessor(t *testing.T) {
	// Create test tar.gz data
	testData := createTestTarGz(t)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testData)))
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	// Create mock database
	mockDB := newMockDatabase()

	// Create processor
	processor := NewStreamProcessor(mockDB)

	// Track progress
	progressUpdates := 0
	processor.SetProgressFunc(func(p Progress) {
		progressUpdates++

		// Validate progress
		if p.BytesProcessed < 0 || p.BytesProcessed > p.TotalBytes {
			t.Errorf("Invalid bytes processed: %d/%d", p.BytesProcessed, p.TotalBytes)
		}

		if p.PercentComplete < 0 || p.PercentComplete > 100 {
			t.Errorf("Invalid percent complete: %.2f", p.PercentComplete)
		}
	})

	// Process URL
	ctx := context.Background()
	err := processor.ProcessURL(ctx, server.URL)
	if err != nil {
		t.Fatalf("Failed to process URL: %v", err)
	}

	// Verify progress was reported
	if progressUpdates == 0 {
		t.Error("No progress updates received")
	}

	// Verify data was inserted
	if mockDB.insertedCount == 0 {
		t.Error("No data inserted into database")
	}

	// Get stats
	stats := processor.GetStats()
	if stats["bytes_processed"].(int64) != int64(len(testData)) {
		t.Errorf("Incorrect bytes processed: got %d, want %d",
			stats["bytes_processed"], len(testData))
	}
}

// TestContextCancellation tests that processing can be cancelled
func TestContextCancellation(t *testing.T) {
	// Create a test tar.gz that streams slowly
	testData := createTestTarGz(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testData)))

		// Send data slowly in chunks
		for i := 0; i < len(testData); i += 10 {
			end := i + 10
			if end > len(testData) {
				end = len(testData)
			}
			w.Write(testData[i:end])
			w.(http.Flusher).Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	mockDB := newMockDatabase()
	processor := NewStreamProcessor(mockDB)

	// Cancel context after short time
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Process should be cancelled due to context
	err := processor.ProcessURL(ctx, server.URL)
	if err == nil {
		t.Errorf("Expected context cancellation error, got nil")
	} else if err != context.DeadlineExceeded && !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected context-related error, got: %v", err)
	}
}

// TestHTTPError tests handling of HTTP errors
func TestHTTPError(t *testing.T) {
	// Create server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockDB := newMockDatabase()
	processor := NewStreamProcessor(mockDB)

	ctx := context.Background()
	err := processor.ProcessURL(ctx, server.URL)
	if err == nil {
		t.Error("Expected error for HTTP 404, got nil")
	}
}

// TestInvalidGzip tests handling of invalid gzip data
func TestInvalidGzip(t *testing.T) {
	// Create server with invalid gzip data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not gzip data"))
	}))
	defer server.Close()

	mockDB := newMockDatabase()
	processor := NewStreamProcessor(mockDB)

	ctx := context.Background()
	err := processor.ProcessURL(ctx, server.URL)
	if err == nil {
		t.Error("Expected error for invalid gzip, got nil")
	}
}

// Helper functions

// createTestTarGz creates a test tar.gz file with sample XML data
func createTestTarGz(t *testing.T) []byte {
	var buf bytes.Buffer

	// Create gzip writer
	gzWriter := gzip.NewWriter(&buf)

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)

	// Add test XML files
	files := []struct {
		name    string
		content string
	}{
		{
			name: "test_experiment.xml",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<EXPERIMENT_PACKAGE accession="SRX001">
	<EXPERIMENT>
		<TITLE>Test Experiment</TITLE>
		<STUDY_REF accession="SRP001"/>
		<DESIGN>
			<SAMPLE_DESCRIPTOR accession="SRS001"/>
			<LIBRARY_DESCRIPTOR>
				<LIBRARY_STRATEGY>RNA-Seq</LIBRARY_STRATEGY>
				<LIBRARY_SOURCE>TRANSCRIPTOMIC</LIBRARY_SOURCE>
				<LIBRARY_SELECTION>cDNA</LIBRARY_SELECTION>
			</LIBRARY_DESCRIPTOR>
		</DESIGN>
		<PLATFORM>
			<ILLUMINA>HiSeq 2000</ILLUMINA>
		</PLATFORM>
	</EXPERIMENT>
</EXPERIMENT_PACKAGE>`,
		},
		{
			name: "test_study.xml",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<STUDY_PACKAGE accession="SRP001">
	<STUDY>
		<DESCRIPTOR>
			<STUDY_TITLE>Test Study</STUDY_TITLE>
			<STUDY_ABSTRACT>This is a test study</STUDY_ABSTRACT>
			<STUDY_TYPE existing_study_type="Transcriptome Analysis"/>
		</DESCRIPTOR>
	</STUDY>
</STUDY_PACKAGE>`,
		},
		{
			name:    "test.txt",
			content: "This should be skipped",
		},
	}

	for _, file := range files {
		header := &tar.Header{
			Name:     file.name,
			Mode:     0644,
			Size:     int64(len(file.content)),
			Typeflag: tar.TypeReg,
			ModTime:  time.Now(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("Failed to write tar header: %v", err)
		}

		if _, err := io.WriteString(tarWriter, file.content); err != nil {
			t.Fatalf("Failed to write tar content: %v", err)
		}
	}

	// Close writers
	tarWriter.Close()
	gzWriter.Close()

	return buf.Bytes()
}

// mockDatabase is a mock database for testing
type mockDatabase struct {
	insertedCount int
}

func newMockDatabase() *mockDatabase {
	return &mockDatabase{}
}

func (m *mockDatabase) InsertStudy(study *database.Study) error {
	m.insertedCount++
	return nil
}

func (m *mockDatabase) InsertExperiment(exp *database.Experiment) error {
	m.insertedCount++
	return nil
}

func (m *mockDatabase) InsertSample(sample *database.Sample) error {
	m.insertedCount++
	return nil
}

func (m *mockDatabase) InsertRun(run *database.Run) error {
	m.insertedCount++
	return nil
}

func (m *mockDatabase) BatchInsertExperiments(experiments []database.Experiment) error {
	m.insertedCount += len(experiments)
	return nil
}

func (m *mockDatabase) InsertSubmission(submission *database.Submission) error {
	m.insertedCount++
	return nil
}

func (m *mockDatabase) InsertAnalysis(analysis *database.Analysis) error {
	m.insertedCount++
	return nil
}

// Pool/multiplex support
func (m *mockDatabase) InsertSamplePool(pool *database.SamplePool) error {
	return nil
}

func (m *mockDatabase) GetSamplePools(parentSample string) ([]database.SamplePool, error) {
	return nil, nil
}

func (m *mockDatabase) CountSamplePools() (int, error) {
	return 0, nil
}

func (m *mockDatabase) GetAveragePoolSize() (float64, error) {
	return 0, nil
}

func (m *mockDatabase) GetMaxPoolSize() (int, error) {
	return 0, nil
}

// Identifier and link support
func (m *mockDatabase) InsertIdentifier(identifier *database.Identifier) error {
	return nil
}

func (m *mockDatabase) GetIdentifiers(recordType, recordAccession string) ([]database.Identifier, error) {
	return nil, nil
}

func (m *mockDatabase) FindRecordsByIdentifier(idValue string) ([]database.Identifier, error) {
	return nil, nil
}

func (m *mockDatabase) InsertLink(link *database.Link) error {
	return nil
}

func (m *mockDatabase) GetLinks(recordType, recordAccession string) ([]database.Link, error) {
	return nil, nil
}
