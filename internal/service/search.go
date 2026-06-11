package service

import (
	"context"
	"fmt"
	"sync"
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

	// Stats cache
	statsMu        sync.Mutex
	statsCache     *SearchStats
	statsTime      time.Time
	statsComputing bool // true while a background computeSlowStats is in flight
}

// NewSearchService creates a new search service
func NewSearchService(db *database.DB, indexPath string) (*SearchService, error) {
	// Create config for search. The backend factory re-opens the database from
	// cfg.Database.Path, so it must point at the same file as the already-open
	// db; otherwise an empty path opens a junk/empty database.
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: db.Path(),
		},
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
		Results:      make([]*SearchResult, 0, len(result.Hits)),
		TotalResults: result.TotalHits,
		Query:        req.Query,
		TimeTaken:    result.TimeMs,
		SearchMode:   result.Mode,
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

// GetStats retrieves search statistics.
// Returns fast counts immediately; slow GROUP BY results are computed in background and cached.
func (s *SearchService) GetStats(ctx context.Context) (*SearchStats, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("search manager not initialized")
	}

	// Return cached stats if available (within 10 minutes)
	s.statsMu.Lock()
	if s.statsCache != nil && time.Since(s.statsTime) < 10*time.Minute {
		cached := *s.statsCache
		s.statsMu.Unlock()
		return &cached, nil
	}
	s.statsMu.Unlock()

	// Always compute fast counts synchronously
	stats := s.computeFastStats()

	// Reuse the previous cache's top lists while fresh ones are computed, and
	// launch exactly one background computation even under concurrent callers.
	s.statsMu.Lock()
	if s.statsCache != nil {
		stats.TopOrganisms = s.statsCache.TopOrganisms
		stats.TopPlatforms = s.statsCache.TopPlatforms
		stats.TopStrategies = s.statsCache.TopStrategies
	}
	if !s.statsComputing {
		s.statsComputing = true
		go s.computeSlowStats()
	}
	s.statsMu.Unlock()

	return stats, nil
}

// computeFastStats returns entity counts from the cached statistics table,
// falling back to a direct COUNT(*) only when the cache has not been populated.
func (s *SearchService) computeFastStats() *SearchStats {
	stats := &SearchStats{
		LastUpdated: time.Now(),
	}

	cachedStats, _ := s.db.GetStatistics()
	studyCount := cachedStats["studies"]
	experimentCount := cachedStats["experiments"]
	sampleCount := cachedStats["samples"]
	runCount := cachedStats["runs"]

	if studyCount+experimentCount+sampleCount+runCount == 0 {
		for _, tbl := range []struct {
			name string
			dest *int64
		}{
			{"studies", &studyCount},
			{"experiments", &experimentCount},
			{"samples", &sampleCount},
			{"runs", &runCount},
		} {
			// #nosec G201 - table names come from this fixed list, not user input
			_ = s.db.QueryRow("SELECT COUNT(*) FROM " + tbl.name).Scan(tbl.dest)
		}
	}

	stats.TotalStudies = studyCount
	stats.TotalExperiments = experimentCount
	stats.TotalSamples = sampleCount
	stats.TotalRuns = runCount
	stats.TotalDocuments = studyCount + experimentCount + sampleCount + runCount

	return stats
}

// computeSlowStats computes GROUP BY queries in the background and caches the
// result. GetStats serializes invocations via the statsComputing flag, so at
// most one runs at a time; this function clears that flag when it returns.
func (s *SearchService) computeSlowStats() {
	defer func() {
		s.statsMu.Lock()
		s.statsComputing = false
		s.statsMu.Unlock()
	}()

	stats := s.computeFastStats()

	// Top organisms: try samples table first (has organism data), fall back to studies
	sampleFilter := ""
	if stats.TotalSamples > 500000 {
		sampleFilter = fmt.Sprintf(" AND rowid > %d", stats.TotalSamples-500000)
	}
	stats.TopOrganisms = s.queryTopItems(fmt.Sprintf(`
		SELECT organism, COUNT(*) as count
		FROM samples
		WHERE organism IS NOT NULL AND organism != ''%s
		GROUP BY organism
		ORDER BY count DESC
		LIMIT 10
	`, sampleFilter))
	if len(stats.TopOrganisms) == 0 {
		stats.TopOrganisms = s.queryTopItems(`
			SELECT organism, COUNT(*) as count
			FROM studies
			WHERE organism IS NOT NULL AND organism != ''
			GROUP BY organism
			ORDER BY count DESC
			LIMIT 10
		`)
	}

	// For experiments: sample recent rows for fast GROUP BY
	expFilter := ""
	if stats.TotalExperiments > 500000 {
		expFilter = fmt.Sprintf(" AND rowid > %d", stats.TotalExperiments-500000)
	}

	stats.TopPlatforms = s.queryTopItems(fmt.Sprintf(`
		SELECT platform, COUNT(*) as count
		FROM experiments
		WHERE platform IS NOT NULL AND platform != ''%s
		GROUP BY platform
		ORDER BY count DESC
		LIMIT 10
	`, expFilter))

	stats.TopStrategies = s.queryTopItems(fmt.Sprintf(`
		SELECT library_strategy, COUNT(*) as count
		FROM experiments
		WHERE library_strategy IS NOT NULL AND library_strategy != ''%s
		GROUP BY library_strategy
		ORDER BY count DESC
		LIMIT 10
	`, expFilter))

	// Cache the full result
	s.statsMu.Lock()
	s.statsCache = stats
	s.statsTime = time.Now()
	s.statsMu.Unlock()
}

// queryTopItems runs a GROUP BY query and returns the results as StatItems.
func (s *SearchService) queryTopItems(query string) []StatItem {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var items []StatItem
	for rows.Next() {
		var item StatItem
		if err := rows.Scan(&item.Name, &item.Count); err == nil {
			items = append(items, item)
		}
	}
	return items
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
