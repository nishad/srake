package search

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
)

// Manager coordinates between different search backends
type Manager struct {
	config   *config.Config
	sqlite   *database.DB
	bleve    SearchBackend
	embedder EmbedderInterface // Will be implemented later

	mu    sync.RWMutex
	cache *SearchCache
}

// EmbedderInterface will be implemented in embeddings package
type EmbedderInterface interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	IsEnabled() bool
}

// SearchCache provides simple caching for search results
type SearchCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	maxSize int
	ttl     time.Duration
}

type cacheEntry struct {
	result    *SearchResult
	timestamp time.Time
}

// NewManager creates a new search manager
func NewManager(cfg *config.Config, db *database.DB) (*Manager, error) {
	m := &Manager{
		config: cfg,
		sqlite: db,
	}

	// Initialize cache if enabled
	if cfg.Search.UseCache {
		m.cache = &SearchCache{
			entries: make(map[string]*cacheEntry),
			maxSize: 1000,
			ttl:     time.Duration(cfg.Search.CacheTTL) * time.Second,
		}
	}

	// Initialize Bleve if enabled
	if cfg.IsSearchEnabled() {
		// For now, use the existing BleveIndex
		// TODO: Switch to BleveBackend when ready
		bleveIndex, err := InitBleveIndex(cfg.DataDirectory)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Bleve: %w", err)
		}
		// Wrap it as a simple backend
		m.bleve = &bleveIndexWrapper{index: bleveIndex}
	}

	return m, nil
}

// Search performs a search using the appropriate backend
func (m *Manager) Search(query string, opts SearchOptions) (*SearchResult, error) {
	// Check cache first
	if m.cache != nil && !opts.NoCache {
		if cached := m.cache.get(m.cacheKey(query, opts)); cached != nil {
			return cached, nil
		}
	}

	// Determine search mode
	mode := m.determineSearchMode(opts)

	var result *SearchResult
	var err error

	switch mode {
	case "minimal":
		result, err = m.searchSQLite(query, opts)
	case "text":
		result, err = m.searchBleve(query, opts)
	case "vector":
		result, err = m.searchWithVector(query, opts)
	case "hybrid":
		result, err = m.searchHybrid(query, opts)
	default:
		return nil, fmt.Errorf("unknown search mode: %s", mode)
	}

	if err != nil {
		return nil, err
	}

	// Cache the result
	if m.cache != nil && !opts.NoCache {
		m.cache.set(m.cacheKey(query, opts), result)
	}

	return result, nil
}

// determineSearchMode decides which search mode to use
func (m *Manager) determineSearchMode(opts SearchOptions) string {
	if !m.config.IsSearchEnabled() {
		return "minimal" // SQLite FTS only
	}

	if opts.UseVectors && m.config.IsVectorEnabled() && m.embedder != nil && m.embedder.IsEnabled() {
		if opts.VectorWeight >= 1.0 {
			return "vector" // Pure vector search
		}
		return "hybrid" // Combined text + vector
	}

	return "text" // Bleve text search
}

// searchSQLite performs a search using SQLite FTS5
func (m *Manager) searchSQLite(query string, opts SearchOptions) (*SearchResult, error) {
	start := time.Now()

	// TODO: Implement SQLite FTS5 search
	// This is a placeholder that will query the database directly

	result := &SearchResult{
		Query:     query,
		TotalHits: 0,
		Hits:      []Hit{},
		TimeMs:    time.Since(start).Milliseconds(),
		Mode:      "minimal",
	}

	return result, nil
}

// searchBleve performs a text search using Bleve
func (m *Manager) searchBleve(query string, opts SearchOptions) (*SearchResult, error) {
	if m.bleve == nil || !m.bleve.IsEnabled() {
		return m.searchSQLite(query, opts)
	}

	return m.bleve.Search(query, opts)
}

// searchWithVector performs a pure vector search
func (m *Manager) searchWithVector(query string, opts SearchOptions) (*SearchResult, error) {
	if m.embedder == nil || !m.embedder.IsEnabled() {
		return m.searchBleve(query, opts)
	}

	// Generate embedding for query
	vector, err := m.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	return m.bleve.SearchWithVector("", vector, opts)
}

// searchHybrid performs a hybrid text + vector search
func (m *Manager) searchHybrid(query string, opts SearchOptions) (*SearchResult, error) {
	if m.embedder == nil || !m.embedder.IsEnabled() {
		return m.searchBleve(query, opts)
	}

	// Generate embedding for query
	vector, err := m.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	return m.bleve.SearchWithVector(query, vector, opts)
}

// FindSimilar finds documents similar to the given ID
func (m *Manager) FindSimilar(id string, opts SearchOptions) (*SearchResult, error) {
	if !m.config.IsVectorEnabled() || m.bleve == nil {
		return nil, fmt.Errorf("vector search is not enabled")
	}

	return m.bleve.FindSimilar(id, opts)
}

// Index adds a document to the search index
func (m *Manager) Index(doc interface{}) error {
	if m.bleve == nil || !m.bleve.IsEnabled() {
		return nil // No-op if search is disabled
	}

	// Generate embedding if vectors are enabled
	if m.config.IsVectorEnabled() && m.embedder != nil && m.embedder.IsEnabled() {
		// TODO: Add embedding to document
	}

	return m.bleve.Index(doc)
}

// IndexBatch adds multiple documents to the search index
func (m *Manager) IndexBatch(docs []interface{}) error {
	if m.bleve == nil || !m.bleve.IsEnabled() {
		return nil // No-op if search is disabled
	}

	// Generate embeddings if vectors are enabled
	if m.config.IsVectorEnabled() && m.embedder != nil && m.embedder.IsEnabled() {
		// TODO: Add embeddings to documents
	}

	return m.bleve.IndexBatch(docs)
}

// RebuildIndex rebuilds the search index from SQLite
func (m *Manager) RebuildIndex(ctx context.Context) error {
	if m.bleve == nil || !m.bleve.IsEnabled() {
		return fmt.Errorf("search is not enabled")
	}

	return m.bleve.Rebuild(ctx)
}

// GetStats returns search index statistics
func (m *Manager) GetStats() (*IndexStats, error) {
	if m.bleve != nil && m.bleve.IsEnabled() {
		return m.bleve.GetStats()
	}

	// Return SQLite-only stats
	return &IndexStats{
		DocumentCount:  0, // TODO: Query from SQLite
		IndexSize:      0,
		LastModified:   time.Now(),
		IsHealthy:      true,
		Backend:        "sqlite",
		VectorsEnabled: false,
	}, nil
}

// Close closes all search backends
func (m *Manager) Close() error {
	if m.bleve != nil {
		return m.bleve.Close()
	}
	return nil
}

// GetBackend returns the search backend
func (m *Manager) GetBackend() SearchBackend {
	return m.bleve
}

// SetEmbedder sets the embedder interface
func (m *Manager) SetEmbedder(embedder EmbedderInterface) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embedder = embedder
}

// cacheKey generates a cache key for a search query
func (m *Manager) cacheKey(query string, opts SearchOptions) string {
	return fmt.Sprintf("%s:%+v", query, opts)
}

// Cache implementation

func (c *SearchCache) get(key string) *SearchResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil
	}

	// Check if entry has expired
	if time.Since(entry.timestamp) > c.ttl {
		return nil
	}

	return entry.result
}

func (c *SearchCache) set(key string, result *SearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: remove oldest entries if cache is full
	if len(c.entries) >= c.maxSize {
		// Find and remove oldest entry
		var oldestKey string
		var oldestTime time.Time
		for k, v := range c.entries {
			if oldestTime.IsZero() || v.timestamp.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.timestamp
			}
		}
		delete(c.entries, oldestKey)
	}

	c.entries[key] = &cacheEntry{
		result:    result,
		timestamp: time.Now(),
	}
}

func (c *SearchCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}