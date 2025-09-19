package search

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/paths"
)

// SearchIntent represents the type of search query
type SearchIntent int

const (
	IntentStudySearch SearchIntent = iota
	IntentAccessionLookup
	IntentTechnicalSearch
	IntentGeneralSearch
)

// TieredSearchBackend implements a tiered search strategy
// Tier 1: Studies (Bleve + Vectors) - ~500K records
// Tier 2: Experiments (Bleve) - ~2M records
// Tier 3: Samples/Runs (SQLite FTS5) - ~34M records
type TieredSearchBackend struct {
	db       *database.DB
	lazyIdx  *LazyIndex // Lazy-loaded Bleve index
	embedder EmbedderInterface
	config   *TieredConfig
	mu       sync.RWMutex

	// Cache for aggregated study data
	studyCache map[string]*StudySearchDoc
	cacheTTL   time.Duration
	cacheTime  time.Time
}

// TieredConfig configures the tiered search backend
type TieredConfig struct {
	// Indexing configuration
	IndexStudies     bool
	IndexExperiments bool
	UseEmbeddings    bool

	// Batch sizes for indexing
	StudyBatchSize int
	ExpBatchSize   int

	// Search configuration
	IdleTimeout      time.Duration
	CacheTTL         time.Duration
	MaxSearchResults int

	// Paths
	IndexPath      string
	EmbeddingsPath string
}

// StudySearchDoc represents an enriched study document with aggregated data
type StudySearchDoc struct {
	Type            string   `json:"type"`
	StudyAccession  string   `json:"study_accession"`
	StudyTitle      string   `json:"study_title"`
	StudyAbstract   string   `json:"study_abstract"`
	StudyType       string   `json:"study_type"`
	Organism        string   `json:"organism"`
	LibraryStrategies []string `json:"library_strategies"`
	Platforms       []string `json:"platforms"`
	ExperimentCount int      `json:"experiment_count"`
	SampleCount     int      `json:"sample_count"`
	RunCount        int      `json:"run_count"`
	EarliestRun     string   `json:"earliest_run"`
	LatestRun       string   `json:"latest_run"`
	Embedding       []float32 `json:"embedding,omitempty"`
}

// NewTieredSearchBackend creates a new tiered search backend
func NewTieredSearchBackend(db *database.DB, cfg *TieredConfig) (*TieredSearchBackend, error) {
	if cfg == nil {
		cfg = &TieredConfig{
			IndexStudies:     true,
			IndexExperiments: true,
			UseEmbeddings:    false,
			StudyBatchSize:   1000,
			ExpBatchSize:     5000,
			IdleTimeout:      5 * time.Minute,
			CacheTTL:         10 * time.Minute,
			MaxSearchResults: 100,
			IndexPath:        paths.GetIndexPath(),
			EmbeddingsPath:   paths.GetEmbeddingsPath(),
		}
	}

	// Create lazy index with optimized mapping
	lazyIdx := NewLazyIndex(cfg.IndexPath, cfg.IdleTimeout)

	return &TieredSearchBackend{
		db:         db,
		lazyIdx:    lazyIdx,
		config:     cfg,
		studyCache: make(map[string]*StudySearchDoc),
		cacheTTL:   cfg.CacheTTL,
	}, nil
}

// IsEnabled returns true if the backend is enabled
func (t *TieredSearchBackend) IsEnabled() bool {
	return true
}

// Search performs a tiered search based on query intent
func (t *TieredSearchBackend) Search(query string, opts SearchOptions) (*SearchResult, error) {
	start := time.Now()

	// Detect search intent
	intent := t.detectSearchIntent(query)

	log.Printf("[TIERED] Search query='%s' intent=%v", query, intent)

	var result *SearchResult
	var err error

	switch intent {
	case IntentAccessionLookup:
		// Fast accession lookup using FTS5
		result, err = t.searchByAccession(query, opts)

	case IntentStudySearch:
		// Search studies using Bleve (and optionally vectors)
		result, err = t.searchStudies(query, opts)

	case IntentTechnicalSearch:
		// Search technical metadata (experiments, platforms, etc.)
		result, err = t.searchTechnical(query, opts)

	default:
		// General search across all tiers
		result, err = t.searchAll(query, opts)
	}

	if err != nil {
		return nil, err
	}

	result.TimeMs = time.Since(start).Milliseconds()

	return result, nil
}

// detectSearchIntent analyzes the query to determine search intent
func (t *TieredSearchBackend) detectSearchIntent(query string) SearchIntent {
	q := strings.ToUpper(query)

	// Check for accession patterns
	if strings.HasPrefix(q, "SR") || strings.HasPrefix(q, "ER") ||
		strings.HasPrefix(q, "DR") || strings.HasPrefix(q, "PR") {
		// Likely an accession lookup
		if strings.Contains(q, " ") {
			// Multiple terms, not just accession
			return IntentGeneralSearch
		}
		return IntentAccessionLookup
	}

	// Check for technical terms
	technicalTerms := []string{"ILLUMINA", "PACBIO", "RNA-SEQ", "WGS", "CHIP-SEQ",
		"ATAC-SEQ", "SINGLE", "PAIRED", "TRANSCRIPTOME", "GENOME"}
	for _, term := range technicalTerms {
		if strings.Contains(q, term) {
			return IntentTechnicalSearch
		}
	}

	// Check for biological/study terms
	studyTerms := []string{"CANCER", "DISEASE", "PATIENT", "TREATMENT", "STUDY",
		"CLINICAL", "COHORT", "HUMAN", "MOUSE", "CELL"}
	for _, term := range studyTerms {
		if strings.Contains(q, term) {
			return IntentStudySearch
		}
	}

	return IntentGeneralSearch
}

// searchByAccession performs fast accession lookup
func (t *TieredSearchBackend) searchByAccession(accession string, opts SearchOptions) (*SearchResult, error) {
	// Use FTS5 for fast accession lookup
	ftsManager := database.NewFTS5Manager(t.db)
	results, err := ftsManager.SearchAccessions(accession, opts.Limit)
	if err != nil {
		return nil, err
	}

	result := &SearchResult{
		Query:     accession,
		TotalHits: len(results),
		Hits:      make([]Hit, 0, len(results)),
		Mode:      "accession",
	}

	for _, r := range results {
		hit := Hit{
			ID:    r.Accession,
			Score: r.Score,
			Fields: map[string]interface{}{
				"type":     r.Type,
				"title":    r.Title,
				"metadata": r.Metadata,
			},
		}
		result.Hits = append(result.Hits, hit)
	}

	return result, nil
}

// searchStudies searches only study documents
func (t *TieredSearchBackend) searchStudies(query string, opts SearchOptions) (*SearchResult, error) {
	// Check if we should use cached aggregated data
	if t.shouldUseCache() {
		return t.searchCachedStudies(query, opts)
	}

	// Search using Bleve
	bleveResult, err := t.lazyIdx.Search(query, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("bleve search failed: %w", err)
	}

	// Convert Bleve results to our format
	result := &SearchResult{
		Query:     query,
		TotalHits: int(bleveResult.Total),
		Hits:      make([]Hit, 0, len(bleveResult.Hits)),
		Mode:      "studies",
		Facets:    make(map[string][]FacetValue),
	}

	// Convert hits
	for _, hit := range bleveResult.Hits {
		h := Hit{
			ID:     hit.ID,
			Score:  hit.Score,
			Fields: hit.Fields,
		}
		result.Hits = append(result.Hits, h)
	}

	// Convert facets
	for name, facet := range bleveResult.Facets {
		facetValues := make([]FacetValue, 0)
		for _, term := range facet.Terms.Terms() {
			facetValues = append(facetValues, FacetValue{
				Value: term.Term,
				Count: term.Count,
			})
		}
		result.Facets[name] = facetValues
	}

	return result, nil
}

// searchTechnical searches technical metadata
func (t *TieredSearchBackend) searchTechnical(query string, opts SearchOptions) (*SearchResult, error) {
	// Build filters for technical search
	filters := make(map[string]string)

	// Parse query for known technical terms
	q := strings.ToUpper(query)
	if strings.Contains(q, "ILLUMINA") {
		filters["platform"] = "ILLUMINA"
	}
	if strings.Contains(q, "RNA-SEQ") {
		filters["library_strategy"] = "RNA-Seq"
	}

	// Search with filters
	bleveResult, err := t.lazyIdx.SearchWithFilters(query, filters, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("filtered search failed: %w", err)
	}

	// Convert results
	result := &SearchResult{
		Query:     query,
		TotalHits: int(bleveResult.Total),
		Hits:      make([]Hit, 0, len(bleveResult.Hits)),
		Mode:      "technical",
	}

	for _, hit := range bleveResult.Hits {
		h := Hit{
			ID:     hit.ID,
			Score:  hit.Score,
			Fields: hit.Fields,
		}
		result.Hits = append(result.Hits, h)
	}

	return result, nil
}

// searchAll performs a general search across all tiers
func (t *TieredSearchBackend) searchAll(query string, opts SearchOptions) (*SearchResult, error) {
	// Start with Bleve search for studies and experiments
	bleveResult, err := t.lazyIdx.Search(query, opts.Limit)
	if err != nil {
		// Fall back to SQLite FTS5 if Bleve fails
		return t.searchSQLiteFTS(query, opts)
	}

	// Convert results
	result := &SearchResult{
		Query:     query,
		TotalHits: int(bleveResult.Total),
		Hits:      make([]Hit, 0, len(bleveResult.Hits)),
		Mode:      "hybrid",
	}

	for _, hit := range bleveResult.Hits {
		h := Hit{
			ID:     hit.ID,
			Score:  hit.Score,
			Fields: hit.Fields,
		}
		result.Hits = append(result.Hits, h)
	}

	return result, nil
}

// searchSQLiteFTS performs a fallback search using SQLite FTS5
func (t *TieredSearchBackend) searchSQLiteFTS(query string, opts SearchOptions) (*SearchResult, error) {
	// Placeholder for SQLite FTS5 implementation
	result := &SearchResult{
		Query:     query,
		TotalHits: 0,
		Hits:      []Hit{},
		Mode:      "fts5",
	}

	// TODO: Implement SQLite FTS5 search

	return result, nil
}

// searchCachedStudies searches using cached aggregated study data
func (t *TieredSearchBackend) searchCachedStudies(query string, opts SearchOptions) (*SearchResult, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Simple string matching on cached studies
	result := &SearchResult{
		Query:     query,
		TotalHits: 0,
		Hits:      []Hit{},
		Mode:      "cached",
	}

	q := strings.ToLower(query)
	for _, study := range t.studyCache {
		// Simple matching logic
		if strings.Contains(strings.ToLower(study.StudyTitle), q) ||
			strings.Contains(strings.ToLower(study.StudyAbstract), q) {
			hit := Hit{
				ID:    study.StudyAccession,
				Score: 1.0, // Simple scoring
				Fields: map[string]interface{}{
					"study_title":      study.StudyTitle,
					"study_abstract":   study.StudyAbstract,
					"experiment_count": study.ExperimentCount,
					"sample_count":     study.SampleCount,
					"run_count":        study.RunCount,
				},
			}
			result.Hits = append(result.Hits, hit)
			result.TotalHits++

			if len(result.Hits) >= opts.Limit {
				break
			}
		}
	}

	return result, nil
}

// shouldUseCache determines if cached data should be used
func (t *TieredSearchBackend) shouldUseCache() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.studyCache) == 0 {
		return false
	}

	return time.Since(t.cacheTime) < t.cacheTTL
}

// SearchWithVector performs a search with vector similarity
func (t *TieredSearchBackend) SearchWithVector(query string, vector []float32, opts SearchOptions) (*SearchResult, error) {
	// TODO: Implement vector search
	return t.Search(query, opts)
}

// FindSimilar finds documents similar to the given ID
func (t *TieredSearchBackend) FindSimilar(id string, opts SearchOptions) (*SearchResult, error) {
	// TODO: Implement similarity search
	return nil, fmt.Errorf("similarity search not yet implemented")
}

// Index adds a document to the search index
func (t *TieredSearchBackend) Index(doc interface{}) error {
	switch d := doc.(type) {
	case StudyDoc:
		return t.lazyIdx.IndexStudy(d)
	case ExperimentDoc:
		return t.lazyIdx.IndexExperiment(d)
	default:
		return fmt.Errorf("unsupported document type: %T", doc)
	}
}

// IndexBatch adds multiple documents to the search index
func (t *TieredSearchBackend) IndexBatch(docs []interface{}) error {
	return t.lazyIdx.BatchIndex(docs)
}

// Rebuild rebuilds the entire index from the database
func (t *TieredSearchBackend) Rebuild(ctx context.Context) error {
	log.Printf("[TIERED] Starting index rebuild")
	start := time.Now()

	// Create new index with optimized mapping
	if err := t.lazyIdx.ForceClose(); err != nil {
		log.Printf("[TIERED] Warning: failed to close existing index: %v", err)
	}

	// TODO: Implement full rebuild logic
	// 1. Create new index with optimized mapping
	// 2. Index studies (Tier 1)
	// 3. Index experiments (Tier 2)
	// 4. Create FTS5 tables for samples/runs (Tier 3)
	// 5. Generate embeddings if enabled

	log.Printf("[TIERED] Index rebuild completed in %v", time.Since(start))
	return nil
}

// GetStats returns index statistics
func (t *TieredSearchBackend) GetStats() (*IndexStats, error) {
	docCount, err := t.lazyIdx.GetDocCount()
	if err != nil {
		docCount = 0
	}

	return &IndexStats{
		DocumentCount:  docCount,
		IndexSize:      0, // TODO: Calculate index size
		LastModified:   time.Now(),
		IsHealthy:      true,
		Backend:        "tiered",
		VectorsEnabled: t.config.UseEmbeddings,
	}, nil
}

// Delete removes a document from the search index
func (t *TieredSearchBackend) Delete(id string) error {
	return t.lazyIdx.Delete(id)
}

// DeleteBatch removes multiple documents from the search index
func (t *TieredSearchBackend) DeleteBatch(ids []string) error {
	for _, id := range ids {
		if err := t.lazyIdx.Delete(id); err != nil {
			return err
		}
	}
	return nil
}

// Flush flushes pending changes to disk
func (t *TieredSearchBackend) Flush() error {
	// Bleve index automatically handles flushing
	// Nothing to do for FTS5 as it's transactional
	return nil
}

// Close closes the backend
func (t *TieredSearchBackend) Close() error {
	return t.lazyIdx.Close()
}

// SetEmbedder sets the embedder for vector search
func (t *TieredSearchBackend) SetEmbedder(embedder EmbedderInterface) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.embedder = embedder
}

// RefreshCache refreshes the study cache
func (t *TieredSearchBackend) RefreshCache() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	log.Printf("[TIERED] Refreshing study cache")
	start := time.Now()

	// Query aggregated study data
	_ = `
		SELECT
			s.study_accession,
			s.study_title,
			s.study_abstract,
			s.study_type,
			GROUP_CONCAT(DISTINCT e.library_strategy) as library_strategies,
			GROUP_CONCAT(DISTINCT e.platform) as platforms,
			COUNT(DISTINCT e.experiment_accession) as experiment_count,
			COUNT(DISTINCT sa.sample_accession) as sample_count,
			COUNT(DISTINCT r.run_accession) as run_count
		FROM studies s
		LEFT JOIN experiments e ON s.study_accession = e.study_accession
		LEFT JOIN samples sa ON e.sample_accession = sa.sample_accession
		LEFT JOIN runs r ON e.experiment_accession = r.experiment_accession
		GROUP BY s.study_accession
		LIMIT 10000
	`

	// TODO: Execute query and populate cache

	t.cacheTime = time.Now()
	log.Printf("[TIERED] Study cache refreshed with %d studies in %v",
		len(t.studyCache), time.Since(start))

	return nil
}