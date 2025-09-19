package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/search"
)

// SearchService handles search operations
type SearchService struct {
	db         *database.DB
	manager    *search.Manager
	useVectors bool
}

// NewSearchService creates a new search service
func NewSearchService(db *database.DB, indexPath string) (*SearchService, error) {
	// Create config for search
	cfg := &config.Config{
		Search: config.SearchConfig{
			IndexPath: indexPath,
			Enabled:   true,
			UseCache:  true,
			CacheTTL:  300,
		},
	}

	// Create search manager
	manager, err := search.NewManager(cfg, db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize search manager: %w", err)
	}

	return &SearchService{
		db:         db,
		manager:    manager,
		useVectors: false, // Will be enabled when vector support is added
	}, nil
}

// Search performs a search using the search manager
func (s *SearchService) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	// Validate request
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 1000 {
		req.Limit = 1000
	}

	// Convert request to search options
	opts := search.SearchOptions{
		Limit:               req.Limit,
		Offset:              req.Offset,
		SimilarityThreshold: req.SimilarityThreshold,
		MinScore:            float64(req.MinScore),
		TopPercentile:       req.TopPercentile,
		ShowConfidence:      req.ShowConfidence,
		UseVectors:          req.UseVectors && s.useVectors,
	}

	// Convert filters
	if len(req.Filters) > 0 {
		opts.Filters = make(map[string]interface{})
		for k, v := range req.Filters {
			opts.Filters[k] = v
		}
	}

	// Perform search
	result, err := s.manager.Search(req.Query, opts)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert search results to response
	response := &SearchResponse{
		Results:       make([]*SearchResult, 0, len(result.Hits)),
		TotalResults:  result.TotalHits,
		Query:         req.Query,
		TimeTaken:     result.TimeMs,
		SearchMode:    result.Mode,
	}

	// Convert hits to search results
	for _, hit := range result.Hits {
		sr := &SearchResult{
			ID:         hit.ID,
			Type:       hit.Type,
			Score:      float32(hit.Score),
			Similarity: hit.Similarity,
			Confidence: hit.Confidence,
			Fields:     hit.Fields,
			Highlights: hit.Highlights,
		}

		// Extract key fields
		if title, ok := hit.Fields["title"].(string); ok {
			sr.Title = title
		} else if title, ok := hit.Fields["study_title"].(string); ok {
			sr.Title = title
		}
		if desc, ok := hit.Fields["description"].(string); ok {
			sr.Description = desc
		} else if desc, ok := hit.Fields["study_abstract"].(string); ok {
			sr.Description = desc
		}
		if org, ok := hit.Fields["organism"].(string); ok {
			sr.Organism = org
		}
		if platform, ok := hit.Fields["platform"].(string); ok {
			sr.Platform = platform
		}
		if strategy, ok := hit.Fields["library_strategy"].(string); ok {
			sr.LibraryStrategy = strategy
		}

		response.Results = append(response.Results, sr)
	}

	return response, nil
}

// BuildIndex builds or rebuilds the search index
func (s *SearchService) BuildIndex(ctx context.Context, batchSize int, withEmbeddings bool) error {
	// Build index using manager
	if s.manager != nil {
		return fmt.Errorf("index building should be done through CLI commands")
	}
	return nil
}

// GetStats retrieves search statistics
func (s *SearchService) GetStats(ctx context.Context) (*SearchStats, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("search manager not initialized")
	}

	stats := &SearchStats{
		TotalDocuments: 0,
		IndexSize:      0,
		LastUpdated:    time.Now(),
	}

	// Get database statistics from cached values
	cachedStats, _ := s.db.GetStatistics()
	studyCount := cachedStats["studies"]
	experimentCount := cachedStats["experiments"]
	sampleCount := cachedStats["samples"]
	runCount := cachedStats["runs"]

	stats.TotalDocuments = studyCount + experimentCount + sampleCount + runCount

	// Get top organisms
	rows, err := s.db.Query(`
		SELECT organism, COUNT(*) as count
		FROM studies
		WHERE organism IS NOT NULL AND organism != ''
		GROUP BY organism
		ORDER BY count DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		stats.TopOrganisms = make([]StatItem, 0)
		for rows.Next() {
			var item StatItem
			if err := rows.Scan(&item.Name, &item.Count); err == nil {
				stats.TopOrganisms = append(stats.TopOrganisms, item)
			}
		}
	}

	// Get top platforms
	rows, err = s.db.Query(`
		SELECT platform, COUNT(*) as count
		FROM experiments
		WHERE platform IS NOT NULL AND platform != ''
		GROUP BY platform
		ORDER BY count DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		stats.TopPlatforms = make([]StatItem, 0)
		for rows.Next() {
			var item StatItem
			if err := rows.Scan(&item.Name, &item.Count); err == nil {
				stats.TopPlatforms = append(stats.TopPlatforms, item)
			}
		}
	}

	// Get top library strategies
	rows, err = s.db.Query(`
		SELECT library_strategy, COUNT(*) as count
		FROM experiments
		WHERE library_strategy IS NOT NULL AND library_strategy != ''
		GROUP BY library_strategy
		ORDER BY count DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		stats.TopStrategies = make([]StatItem, 0)
		for rows.Next() {
			var item StatItem
			if err := rows.Scan(&item.Name, &item.Count); err == nil {
				stats.TopStrategies = append(stats.TopStrategies, item)
			}
		}
	}

	// Index stats would come from manager if available

	return stats, nil
}

// Close cleans up the search service
func (s *SearchService) Close() error {
	if s.manager != nil {
		return s.manager.Close()
	}
	return nil
}

// Health checks if the search service is healthy
func (s *SearchService) Health(ctx context.Context) error {
	if s.manager != nil {
		// Simple ping to check if manager is working
		_, err := s.manager.Search("", search.SearchOptions{Limit: 1})
		if err != nil {
			return fmt.Errorf("search health check failed: %w", err)
		}
	}
	return nil
}