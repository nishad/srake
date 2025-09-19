package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/nishad/srake/internal/service"
)

// Search handlers

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Build search request from query parameters or JSON body
	var req service.SearchRequest

	if r.Method == "POST" {
		// Parse JSON body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	} else {
		// Parse query parameters
		q := r.URL.Query()
		req.Query = q.Get("q")
		if req.Query == "" {
			req.Query = q.Get("query")
		}

		// Pagination
		if limit := q.Get("limit"); limit != "" {
			if l, err := strconv.Atoi(limit); err == nil {
				req.Limit = l
			}
		}
		if req.Limit <= 0 {
			req.Limit = 20
		}
		if req.Limit > 1000 {
			req.Limit = 1000
		}

		if offset := q.Get("offset"); offset != "" {
			if o, err := strconv.Atoi(offset); err == nil {
				req.Offset = o
			}
		}

		// Quality control parameters
		if threshold := q.Get("similarity_threshold"); threshold != "" {
			if t, err := strconv.ParseFloat(threshold, 32); err == nil {
				req.SimilarityThreshold = float32(t)
			}
		}
		if minScore := q.Get("min_score"); minScore != "" {
			if s, err := strconv.ParseFloat(minScore, 32); err == nil {
				req.MinScore = float32(s)
			}
		}
		if topPercentile := q.Get("top_percentile"); topPercentile != "" {
			if p, err := strconv.Atoi(topPercentile); err == nil {
				req.TopPercentile = p
			}
		}
		req.ShowConfidence = q.Get("show_confidence") == "true"

		// Search mode
		req.SearchMode = q.Get("mode")
		if req.SearchMode == "" {
			req.SearchMode = q.Get("search_mode")
		}

		// Format
		req.Format = q.Get("format")

		// Filters
		if organism := q.Get("organism"); organism != "" {
			if req.Filters == nil {
				req.Filters = make(map[string]string)
			}
			req.Filters["organism"] = organism
		}
		if strategy := q.Get("library_strategy"); strategy != "" {
			if req.Filters == nil {
				req.Filters = make(map[string]string)
			}
			req.Filters["library_strategy"] = strategy
		}
		if platform := q.Get("platform"); platform != "" {
			if req.Filters == nil {
				req.Filters = make(map[string]string)
			}
			req.Filters["platform"] = platform
		}
	}

	// Perform search
	response, err := s.searchService.Search(ctx, &req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleAdvancedSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req service.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate advanced search parameters
	if req.Query == "" && len(req.Filters) == 0 {
		s.writeError(w, http.StatusBadRequest, "Query or filters required")
		return
	}

	// Perform search
	response, err := s.searchService.Search(ctx, &req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, response)
}

// Metadata handlers

func (s *Server) handleGetStudy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	accession := vars["accession"]

	study, err := s.metadataService.GetStudy(ctx, accession)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "Study not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	s.writeJSON(w, http.StatusOK, study)
}

func (s *Server) handleGetExperiment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	accession := vars["accession"]

	experiment, err := s.metadataService.GetExperiment(ctx, accession)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "Experiment not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	s.writeJSON(w, http.StatusOK, experiment)
}

func (s *Server) handleGetSample(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	accession := vars["accession"]

	sample, err := s.metadataService.GetSample(ctx, accession)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "Sample not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	s.writeJSON(w, http.StatusOK, sample)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	accession := vars["accession"]

	run, err := s.metadataService.GetRun(ctx, accession)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "Run not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	s.writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleListStudies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	// Parse pagination
	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 {
				limit = 100
			}
		}
	}

	offset := 0
	if o := q.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	studies, err := s.metadataService.GetStudies(ctx, limit, offset)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"studies": studies,
		"limit":   limit,
		"offset":  offset,
	}

	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetStudyMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	accession := vars["accession"]

	metadata, err := s.metadataService.GetStudyMetadata(ctx, accession)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "Study not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	s.writeJSON(w, http.StatusOK, metadata)
}

func (s *Server) handleGetStudyExperiments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	accession := vars["accession"]

	experiments, err := s.metadataService.GetExperimentsByStudy(ctx, accession)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"study_accession": accession,
		"experiments":     experiments,
		"total":           len(experiments),
	})
}

func (s *Server) handleGetStudySamples(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	accession := vars["accession"]

	samples, err := s.metadataService.GetSamplesByStudy(ctx, accession)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"study_accession": accession,
		"samples":         samples,
		"total":           len(samples),
	})
}

func (s *Server) handleGetStudyRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	accession := vars["accession"]

	q := r.URL.Query()
	limit := 100
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	runs, err := s.metadataService.GetRunsByStudy(ctx, accession, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"study_accession": accession,
		"runs":            runs,
		"total":           len(runs),
		"limit":           limit,
	})
}

// Statistics handlers

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := s.searchService.GetStats(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleGetOrganismStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := s.searchService.GetStats(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"organisms": stats.TopOrganisms,
		"total":     len(stats.TopOrganisms),
	})
}

func (s *Server) handleGetPlatformStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := s.searchService.GetStats(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"platforms": stats.TopPlatforms,
		"total":     len(stats.TopPlatforms),
	})
}

func (s *Server) handleGetStrategyStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := s.searchService.GetStats(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"strategies": stats.TopStrategies,
		"total":      len(stats.TopStrategies),
	})
}

// Export handler

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req service.ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate format
	validFormats := map[string]bool{
		"json": true, "csv": true, "tsv": true, "xml": true, "jsonl": true,
	}
	if !validFormats[strings.ToLower(req.Format)] {
		s.writeError(w, http.StatusBadRequest, "Invalid format. Supported: json, csv, tsv, xml, jsonl")
		return
	}

	// Set appropriate content type
	contentTypes := map[string]string{
		"json":  "application/json",
		"jsonl": "application/x-ndjson",
		"csv":   "text/csv",
		"tsv":   "text/tab-separated-values",
		"xml":   "application/xml",
	}
	w.Header().Set("Content-Type", contentTypes[strings.ToLower(req.Format)])
	w.Header().Set("Content-Disposition", "attachment; filename=export."+strings.ToLower(req.Format))

	// Perform export
	if err := s.exportService.Export(ctx, &req, w); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
}
