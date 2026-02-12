package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/service"
)

// testServer is a simplified server for testing handlers
type testServer struct {
	*Server
}

// setupTestServer creates a minimal test server with only the database
func setupTestServer(t *testing.T) (*testServer, func()) {
	t.Helper()

	// Create temp directory for database
	dir, err := os.MkdirTemp("", "srake-api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	db, err := database.Initialize(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to initialize database: %v", err)
	}

	// Create services
	metadataService := service.NewMetadataService(db)

	// Create minimal server
	s := &Server{
		router:          mux.NewRouter(),
		metadataService: metadataService,
		db:              db,
	}

	// Setup routes manually - only metadata endpoints that don't require search service
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/study/{accession}", s.handleGetStudy).Methods("GET")
	api.HandleFunc("/experiment/{accession}", s.handleGetExperiment).Methods("GET")
	api.HandleFunc("/sample/{accession}", s.handleGetSample).Methods("GET")
	api.HandleFunc("/run/{accession}", s.handleGetRun).Methods("GET")

	// Add middleware
	s.router.Use(corsMiddleware)
	s.router.Use(jsonMiddleware)

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}

	return &testServer{s}, cleanup
}

func TestStudyEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Insert test study
	study := &database.Study{
		StudyAccession: "SRP000001",
		StudyTitle:     "Test Study",
		StudyAbstract:  "A test study",
		Organism:       "Homo sapiens",
	}
	if err := server.db.InsertStudy(study); err != nil {
		t.Fatalf("failed to insert test study: %v", err)
	}

	// Test get study
	req := httptest.NewRequest("GET", "/api/study/SRP000001", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if acc, ok := resp["study_accession"].(string); !ok || acc != "SRP000001" {
		t.Errorf("expected study_accession 'SRP000001', got %v", resp["study_accession"])
	}
}

func TestStudyEndpointNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/study/NONEXISTENT", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestExperimentEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Insert test experiment
	exp := &database.Experiment{
		ExperimentAccession: "SRX000001",
		Title:               "Test Experiment",
		Platform:            "ILLUMINA",
		LibraryStrategy:     "RNA-Seq",
	}
	if err := server.db.InsertExperiment(exp); err != nil {
		t.Fatalf("failed to insert test experiment: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/experiment/SRX000001", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestExperimentEndpointNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/experiment/NONEXISTENT", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestSampleEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Insert test sample
	sample := &database.Sample{
		SampleAccession: "SRS000001",
		Organism:        "Homo sapiens",
		TaxonID:         9606,
	}
	if err := server.db.InsertSample(sample); err != nil {
		t.Fatalf("failed to insert test sample: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/sample/SRS000001", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestSampleEndpointNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/sample/NONEXISTENT", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestRunEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Insert prerequisite experiment
	exp := &database.Experiment{
		ExperimentAccession: "SRX000001",
		Title:               "Test Experiment",
		Platform:            "ILLUMINA",
	}
	if err := server.db.InsertExperiment(exp); err != nil {
		t.Fatalf("failed to insert experiment: %v", err)
	}

	// Insert test run
	run := &database.Run{
		RunAccession:        "SRR000001",
		ExperimentAccession: "SRX000001",
		TotalSpots:          1000000,
		TotalBases:          300000000,
	}
	if err := server.db.InsertRun(run); err != nil {
		t.Fatalf("failed to insert test run: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/run/SRR000001", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRunEndpointNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/run/NONEXISTENT", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestCORSHeaders(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Insert test study for a valid endpoint
	study := &database.Study{StudyAccession: "SRP000001", StudyTitle: "Test"}
	server.db.InsertStudy(study)

	// Test that GET request includes CORS headers
	req := httptest.NewRequest("GET", "/api/study/SRP000001", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin == "" {
		t.Error("expected Access-Control-Allow-Origin header to be set")
	}
}

func TestContentTypeJSON(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Insert test study
	study := &database.Study{StudyAccession: "SRP000001", StudyTitle: "Test"}
	server.db.InsertStudy(study)

	req := httptest.NewRequest("GET", "/api/study/SRP000001", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestWriteError(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Manually test writeError function
	w := httptest.NewRecorder()
	server.writeError(w, http.StatusBadRequest, "Test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// writeError returns {"error": true, "message": ..., "status": ...}
	if msg, ok := resp["message"].(string); !ok || msg != "Test error message" {
		t.Errorf("expected message 'Test error message', got %v", resp["message"])
	}
	if errFlag, ok := resp["error"].(bool); !ok || !errFlag {
		t.Errorf("expected error flag to be true, got %v", resp["error"])
	}
}

func TestWriteJSON(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	w := httptest.NewRecorder()
	data := map[string]string{"test": "value"}
	server.writeJSON(w, http.StatusOK, data)

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["test"] != "value" {
		t.Errorf("expected test='value', got %v", resp["test"])
	}
}

// Benchmark tests

func BenchmarkStudyEndpoint(b *testing.B) {
	dir, _ := os.MkdirTemp("", "srake-api-bench-*")
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")
	db, _ := database.Initialize(dbPath)
	defer db.Close()

	metadataService := service.NewMetadataService(db)

	// Insert test data
	study := &database.Study{StudyAccession: "SRP000001", StudyTitle: "Test"}
	db.InsertStudy(study)

	s := &Server{
		router:          mux.NewRouter(),
		metadataService: metadataService,
		db:              db,
	}

	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/study/{accession}", s.handleGetStudy).Methods("GET")
	s.router.Use(jsonMiddleware)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/study/SRP000001", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
	}
}
