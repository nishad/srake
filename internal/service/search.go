package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/search"
	"github.com/nishad/srake/internal/ui"
)

// SearchService handles all search-related operations
type SearchService struct {
	db         *database.DB
	index      search.Backend
	indexPath  string
	useVectors bool
}

// NewSearchService creates a new search service instance
func NewSearchService(db *database.DB, indexPath string) (*SearchService, error) {
	// Initialize search backend
	backend, err := search.NewBleveBackend(indexPath, db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize search backend: %w", err)
	}

	// Check if vectors are enabled
	useVectors := backend.HasVectorSupport()

	return &SearchService{
		db:         db,
		index:      backend,
		indexPath:  indexPath,
		useVectors: useVectors,
	}, nil
}

// Search performs a search with the given request parameters
func (s *SearchService) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	start := time.Now()

	// Convert request to search options
	opts := search.SearchOptions{
		Query:               req.Query,
		Limit:               req.Limit,
		Offset:              req.Offset,
		SimilarityThreshold: req.SimilarityThreshold,
		MinScore:           req.MinScore,
		TopPercentile:      req.TopPercentile,
		ShowConfidence:     req.ShowConfidence,
		HybridWeight:       req.HybridWeight,
		UseVectors:         req.UseVectors && s.useVectors,
		Fuzzy:              req.Fuzzy,
		Exact:              req.Exact,
	}

	// Apply filters
	if req.Filters != nil {
		opts.Filters = make(map[string]interface{})
		for k, v := range req.Filters {
			opts.Filters[k] = v
		}
	}

	// Apply field selections
	if len(req.Fields) > 0 {
		opts.Fields = req.Fields
	}

	// Determine search mode
	var results []search.SearchResult
	var err error

	switch req.SearchMode {
	case "database", "db":
		// Direct database search without index
		results, err = s.searchDatabase(ctx, opts)
	case "fts", "fulltext":
		// Full-text search only
		opts.UseVectors = false
		results, err = s.index.Search(opts)
	case "vector", "semantic":
		// Vector search only
		if !s.useVectors {
			return nil, fmt.Errorf("vector search not available - index was built without embeddings")
		}
		opts.UseVectors = true
		results, err = s.index.Search(opts)
	case "hybrid", "":
		// Default hybrid search
		results, err = s.index.Search(opts)
	default:
		return nil, fmt.Errorf("invalid search mode: %s", req.SearchMode)
	}

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results to response format
	response := &SearchResponse{
		Results: make([]SearchHit, 0, len(results)),
		Total:   len(results),
		Took:    time.Since(start),
	}

	for _, res := range results {
		hit := SearchHit{
			Score:      res.Score,
			Highlights: res.Highlights,
		}

		// Add confidence if requested
		if req.ShowConfidence && res.Similarity > 0 {
			hit.Confidence = s.calculateConfidence(res.Score, res.Similarity)
		}

		// Populate the appropriate entity type
		switch res.Type {
		case "study":
			if study, ok := res.Document.(*database.Study); ok {
				hit.Study = study
			}
		case "experiment":
			if exp, ok := res.Document.(*database.Experiment); ok {
				hit.Experiment = exp
			}
		case "sample":
			if sample, ok := res.Document.(*database.Sample); ok {
				hit.Sample = sample
			}
		case "run":
			if run, ok := res.Document.(*database.Run); ok {
				hit.Run = run
			}
		}

		response.Results = append(response.Results, hit)
	}

	return response, nil
}

// searchDatabase performs direct database search without index
func (s *SearchService) searchDatabase(ctx context.Context, opts search.SearchOptions) ([]search.SearchResult, error) {
	// Build SQL query based on filters
	query := "SELECT * FROM studies WHERE 1=1"
	args := []interface{}{}

	if opts.Query != "" {
		query += " AND (title LIKE ? OR abstract LIKE ? OR alias LIKE ?)"
		searchTerm := "%" + opts.Query + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// Add filters
	if filters, ok := opts.Filters["organism"].(string); ok && filters != "" {
		query += " AND organism LIKE ?"
		args = append(args, "%"+filters+"%")
	}

	if filters, ok := opts.Filters["library_strategy"].(string); ok && filters != "" {
		query += " AND accession IN (SELECT study_accession FROM experiments WHERE library_strategy = ?)"
		args = append(args, filters)
	}

	// Add limit and offset
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", opts.Offset)
		}
	}

	// Execute query
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []search.SearchResult
	for rows.Next() {
		var study database.Study
		if err := s.db.ScanStudy(rows, &study); err != nil {
			continue
		}

		results = append(results, search.SearchResult{
			Type:     "study",
			Document: &study,
			Score:    1.0, // Default score for database results
		})
	}

	return results, nil
}

// calculateConfidence determines confidence level based on scores
func (s *SearchService) calculateConfidence(score float64, similarity float32) string {
	// High confidence: high similarity or high text score
	if similarity > 0.8 || score > 10.0 {
		return "high"
	}
	// Medium confidence
	if similarity > 0.5 || score > 5.0 {
		return "medium"
	}
	// Low confidence
	return "low"
}

// GetStats returns database and index statistics
func (s *SearchService) GetStats(ctx context.Context) (*StatsResponse, error) {
	stats := &StatsResponse{
		LastUpdate: time.Now(),
	}

	// Get database counts
	var err error
	stats.TotalStudies, err = s.db.CountTable("studies")
	if err != nil {
		return nil, fmt.Errorf("failed to count studies: %w", err)
	}

	stats.TotalExperiments, err = s.db.CountTable("experiments")
	if err != nil {
		return nil, fmt.Errorf("failed to count experiments: %w", err)
	}

	stats.TotalSamples, err = s.db.CountTable("samples")
	if err != nil {
		return nil, fmt.Errorf("failed to count samples: %w", err)
	}

	stats.TotalRuns, err = s.db.CountTable("runs")
	if err != nil {
		return nil, fmt.Errorf("failed to count runs: %w", err)
	}

	// Get database size
	if dbInfo, err := s.db.GetInfo(); err == nil {
		stats.DatabaseSize = dbInfo.Size
	}

	// Get index stats if available
	if indexStats, err := s.index.Stats(); err == nil {
		stats.IndexSize = indexStats.DocumentCount
	}

	// Get top organisms
	topOrganisms, err := s.getTopItems("SELECT organism, COUNT(*) as cnt FROM studies GROUP BY organism ORDER BY cnt DESC LIMIT 10")
	if err == nil {
		stats.TopOrganisms = topOrganisms
	}

	// Get top platforms
	topPlatforms, err := s.getTopItems("SELECT platform, COUNT(*) as cnt FROM experiments GROUP BY platform ORDER BY cnt DESC LIMIT 10")
	if err == nil {
		stats.TopPlatforms = topPlatforms
	}

	// Get top strategies
	topStrategies, err := s.getTopItems("SELECT library_strategy, COUNT(*) as cnt FROM experiments GROUP BY library_strategy ORDER BY cnt DESC LIMIT 10")
	if err == nil {
		stats.TopStrategies = topStrategies
	}

	return stats, nil
}

// getTopItems executes a query and returns count items
func (s *SearchService) getTopItems(query string) ([]CountItem, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CountItem
	for rows.Next() {
		var name string
		var count int64
		if err := rows.Scan(&name, &count); err != nil {
			continue
		}
		items = append(items, CountItem{
			Name:  name,
			Count: count,
		})
	}
	return items, nil
}

// BuildIndex builds or rebuilds the search index
func (s *SearchService) BuildIndex(ctx context.Context, batchSize int, withVectors bool, showProgress bool) error {
	// Create progress spinner if requested
	var spinner *ui.Spinner
	if showProgress {
		spinner = ui.NewSpinner("Building search index...")
		spinner.Start()
		defer spinner.Stop()
	}

	// Get total document count
	total, err := s.db.CountTable("studies")
	if err != nil {
		return fmt.Errorf("failed to count documents: %w", err)
	}

	// Build index in batches
	processed := int64(0)
	for offset := int64(0); offset < total; offset += int64(batchSize) {
		// Fetch batch of studies
		studies, err := s.db.GetStudiesBatch(int(offset), batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch studies batch: %w", err)
		}

		// Index each study
		for _, study := range studies {
			if err := s.index.Index(study); err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to index study %s: %v\n", study.Accession, err)
				continue
			}
			processed++

			// Update progress
			if spinner != nil && processed%100 == 0 {
				spinner.UpdateMessage(fmt.Sprintf("Indexed %d/%d documents", processed, total))
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Commit changes
	if err := s.index.Close(); err != nil {
		return fmt.Errorf("failed to commit index: %w", err)
	}

	// Reopen index
	s.index, err = search.NewBleveBackend(s.indexPath, s.db)
	if err != nil {
		return fmt.Errorf("failed to reopen index: %w", err)
	}

	return nil
}

// Health checks if the service is operational
func (s *SearchService) Health(ctx context.Context) error {
	// Check database connection
	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("database unhealthy: %w", err)
	}

	// Check index availability
	if stats, err := s.index.Stats(); err != nil || stats.DocumentCount == 0 {
		return fmt.Errorf("index unhealthy or empty")
	}

	return nil
}

// Close releases resources
func (s *SearchService) Close() error {
	if s.index != nil {
		return s.index.Close()
	}
	return nil
}