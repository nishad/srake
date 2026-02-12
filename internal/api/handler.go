// Package api provides HTTP handlers for the srake REST API, exposing
// search, metadata retrieval, statistics, and export endpoints.
package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/search"
)

// Handler serves the srake REST API, routing requests to the appropriate
// database queries and search backend operations.
type Handler struct {
	db             *database.DB
	searchBackend  search.SearchBackend
	mux            *http.ServeMux
}

// NewHandler creates a new Handler with all API routes registered.
func NewHandler(db *database.DB, cfg *config.Config) (*Handler, error) {
	// Create search backend
	searchBackend, err := search.CreateSearchBackend(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create search backend: %w", err)
	}

	h := &Handler{
		db:            db,
		searchBackend: searchBackend,
		mux:           http.NewServeMux(),
	}

	// Set up API routes
	h.mux.HandleFunc("/api/v1/search", h.handleSearch)
	h.mux.HandleFunc("/api/v1/stats", h.handleStats)
	h.mux.HandleFunc("/api/v1/health", h.handleHealth)
	h.mux.HandleFunc("/api/v1/studies/", h.handleStudyDetails)
	h.mux.HandleFunc("/api/v1/samples/", h.handleSampleDetails)
	h.mux.HandleFunc("/api/v1/runs/", h.handleRunDetails)
	h.mux.HandleFunc("/api/v1/export", h.handleExport)
	h.mux.HandleFunc("/api/v1/aggregations/", h.handleAggregations)

	// Serve static files for the web app
	h.mux.Handle("/", http.FileServer(http.Dir("./web/build")))

	return h, nil
}

// ServeHTTP dispatches incoming requests, adding CORS headers and handling OPTIONS preflight.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for development
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	q := r.URL.Query()

	searchQuery := q.Get("query")
	libraryStrategy := q.Get("library_strategy")
	platform := q.Get("platform")
	organism := q.Get("organism")
	searchMode := q.Get("search_mode")

	// Validate query length to prevent abuse
	const maxQueryLength = 1000
	if len(searchQuery) > maxQueryLength {
		http.Error(w, "Query too long (max 1000 characters)", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	similarityThreshold, _ := strconv.ParseFloat(q.Get("similarity_threshold"), 32)
	if similarityThreshold == 0 {
		similarityThreshold = 0.7
	}

	minScore, _ := strconv.ParseFloat(q.Get("min_score"), 64)
	showConfidence := q.Get("show_confidence") == "true"

	// Build search options
	opts := search.SearchOptions{
		Limit:               limit,
		Offset:              offset,
		SimilarityThreshold: float32(similarityThreshold),
		MinScore:            minScore,
		ShowConfidence:      showConfidence,
		IncludeScore:        true,
		Filters:             make(map[string]interface{}),
	}

	// Add filters
	if libraryStrategy != "" {
		opts.Filters["library_strategy"] = libraryStrategy
	}
	if platform != "" {
		opts.Filters["platform"] = platform
	}
	if organism != "" {
		opts.Filters["organism"] = organism
	}
	if searchMode == "vector" {
		opts.UseVectors = true
	}

	// Execute search
	result, err := h.searchBackend.Search(searchQuery, opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Transform results for API response
	results := result.Hits
	apiResults := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		apiResult := map[string]interface{}{
			"id":    r.ID,
			"type":  r.Type,
			"score": r.Score,
		}

		// Extract fields from the hit
		if r.Fields != nil {
			if title, ok := r.Fields["title"].(string); ok {
				apiResult["title"] = title
			}
			if abstract, ok := r.Fields["abstract"].(string); ok {
				apiResult["abstract"] = abstract
			}
			if organism, ok := r.Fields["organism"].(string); ok {
				apiResult["organism"] = organism
			}
			if libraryStrategy, ok := r.Fields["library_strategy"].(string); ok {
				apiResult["library_strategy"] = libraryStrategy
			}
			if platform, ok := r.Fields["platform"].(string); ok {
				apiResult["platform"] = platform
			}
			if submissionDate, ok := r.Fields["submission_date"].(string); ok {
				apiResult["submission_date"] = submissionDate
			}
			if sampleCount, ok := r.Fields["sample_count"].(float64); ok {
				apiResult["sample_count"] = int(sampleCount)
			}
			if runCount, ok := r.Fields["run_count"].(float64); ok {
				apiResult["run_count"] = int(runCount)
			}
		}

		// Add confidence if requested
		if showConfidence && r.Confidence != "" {
			apiResult["confidence"] = r.Confidence
		}

		apiResults = append(apiResults, apiResult)
	}

	// Get total count from result
	totalCount := result.TotalHits
	if totalCount == 0 {
		totalCount = len(results)
	}

	response := map[string]interface{}{
		"results":       apiResults,
		"total":         totalCount,
		"offset":        offset,
		"limit":         limit,
		"total_results": totalCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var stats struct {
		TotalStudies     int64 `json:"total_studies"`
		TotalSamples     int64 `json:"total_samples"`
		TotalRuns        int64 `json:"total_runs"`
		TotalExperiments int64 `json:"total_experiments"`
	}

	// Get counts from database using raw SQL with proper error handling
	if err := h.db.QueryRow("SELECT COUNT(*) FROM studies").Scan(&stats.TotalStudies); err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	if err := h.db.QueryRow("SELECT COUNT(*) FROM samples").Scan(&stats.TotalSamples); err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	if err := h.db.QueryRow("SELECT COUNT(*) FROM runs").Scan(&stats.TotalRuns); err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	if err := h.db.QueryRow("SELECT COUNT(*) FROM experiments").Scan(&stats.TotalExperiments); err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"total_documents":    stats.TotalStudies,
		"indexed_documents":  stats.TotalSamples,
		"total_studies":      stats.TotalStudies,
		"total_samples":      stats.TotalSamples,
		"total_runs":         stats.TotalRuns,
		"total_experiments":  stats.TotalExperiments,
		"last_updated":       time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check database connection
	dbStatus := "healthy"
	if err := h.db.Ping(); err != nil {
		dbStatus = "unhealthy"
	}

	// Check search index
	searchStatus := "healthy"
	if h.searchBackend == nil {
		searchStatus = "unavailable"
	}

	response := map[string]interface{}{
		"status":       "ok",
		"database":     dbStatus,
		"search_index": searchStatus,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleStudyDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract study ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/studies/")
	studyID := strings.TrimSuffix(path, "/")

	if studyID == "" {
		http.Error(w, "Study ID required", http.StatusBadRequest)
		return
	}

	var study database.Study
	row := h.db.QueryRow("SELECT study_accession, study_title, study_abstract, study_type, organism, submission_date, metadata FROM studies WHERE study_accession = ?", studyID)
	if err := row.Scan(&study.StudyAccession, &study.StudyTitle, &study.StudyAbstract, &study.StudyType, &study.Organism, &study.SubmissionDate, &study.Metadata); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Study not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(study)
}

func (h *Handler) handleSampleDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract sample ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/samples/")
	sampleID := strings.TrimSuffix(path, "/")

	if sampleID == "" {
		http.Error(w, "Sample ID required", http.StatusBadRequest)
		return
	}

	var sample database.Sample
	row := h.db.QueryRow("SELECT sample_accession, title, description, scientific_name, taxon_id, metadata FROM samples WHERE sample_accession = ?", sampleID)
	if err := row.Scan(&sample.SampleAccession, &sample.Title, &sample.Description, &sample.ScientificName, &sample.TaxonID, &sample.Metadata); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Sample not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sample)
}

func (h *Handler) handleRunDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract run ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/runs/")
	runID := strings.TrimSuffix(path, "/")

	if runID == "" {
		http.Error(w, "Run ID required", http.StatusBadRequest)
		return
	}

	var run database.Run
	row := h.db.QueryRow("SELECT run_accession, experiment_accession, title, run_date, run_center, total_spots, total_bases, metadata FROM runs WHERE run_accession = ?", runID)
	if err := row.Scan(&run.RunAccession, &run.ExperimentAccession, &run.Title, &run.RunDate, &run.RunCenter, &run.TotalSpots, &run.TotalBases, &run.Metadata); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Run not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

func (h *Handler) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For now, return a simple CSV response
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"export.csv\"")

	w.Write([]byte("id,title,organism,library_strategy,platform\n"))
	w.Write([]byte("SRP001,Example Study,Homo sapiens,RNA-Seq,ILLUMINA\n"))
}

func (h *Handler) handleAggregations(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract field from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/aggregations/")
	field := strings.TrimSuffix(path, "/")

	if field == "" {
		http.Error(w, "Field required", http.StatusBadRequest)
		return
	}

	// Simple aggregation query
	type AggResult struct {
		Value string
		Count int64
	}

	var results []AggResult

	var query string
	switch field {
	case "organism":
		query = `SELECT organism as value, COUNT(*) as count
				 FROM studies
				 WHERE organism IS NOT NULL
				 GROUP BY organism
				 ORDER BY count DESC
				 LIMIT 20`
	case "library_strategy":
		query = `SELECT library_strategy as value, COUNT(*) as count
				 FROM experiments
				 WHERE library_strategy IS NOT NULL
				 GROUP BY library_strategy
				 ORDER BY count DESC
				 LIMIT 20`
	case "platform":
		query = `SELECT platform as value, COUNT(*) as count
				 FROM experiments
				 WHERE platform IS NOT NULL
				 GROUP BY platform
				 ORDER BY count DESC
				 LIMIT 20`
	default:
		http.Error(w, "Invalid aggregation field", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var result AggResult
		if err := rows.Scan(&result.Value, &result.Count); err != nil {
			continue
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Query error: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"field": field,
		"values": results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}