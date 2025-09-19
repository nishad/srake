package search

import (
	"context"
	"time"
)

// SearchBackend defines the interface for search backends
type SearchBackend interface {
	// Index operations
	Index(doc interface{}) error
	IndexBatch(docs []interface{}) error
	Delete(id string) error
	DeleteBatch(ids []string) error

	// Search operations
	Search(query string, opts SearchOptions) (*SearchResult, error)
	SearchWithVector(query string, vector []float32, opts SearchOptions) (*SearchResult, error)
	FindSimilar(id string, opts SearchOptions) (*SearchResult, error)

	// Management
	Close() error
	Flush() error // Flush pending changes to disk
	IsEnabled() bool
	GetStats() (*IndexStats, error)
	Rebuild(ctx context.Context) error
}

// SearchOptions contains search parameters
type SearchOptions struct {
	Limit        int                    // Maximum results to return
	Offset       int                    // Pagination offset
	Filters      map[string]interface{} // Field filters
	Facets       []string               // Facet fields to return
	Highlight    bool                   // Enable highlighting
	IncludeScore bool                   // Include relevance scores

	// Vector search options
	UseVectors   bool    // Enable vector search
	VectorWeight float64 // Weight for vector scoring (0-1)
	KNN          int     // Number of nearest neighbors
	MinScore     float64 // Minimum score threshold

	// Quality control options
	SimilarityThreshold float32 // Minimum cosine similarity for vector results (0-1)
	TopPercentile       int     // Only return top N percentile of results
	ShowConfidence      bool    // Include confidence levels in results
	HybridWeight        float32 // Weight for hybrid scoring (0=text only, 1=vector only)

	// Performance options
	TimeoutMs int  // Query timeout in milliseconds
	NoCache   bool // Bypass cache
}

// SearchResult represents search results
type SearchResult struct {
	Query     string                  `json:"query"`
	TotalHits int                     `json:"total_hits"`
	Hits      []Hit                   `json:"hits"`
	Facets    map[string][]FacetValue `json:"facets,omitempty"`
	TimeMs    int64                   `json:"time_ms"`
	Mode      string                  `json:"mode"` // "text", "vector", "hybrid"
}

// Hit represents a single search result
type Hit struct {
	ID         string                 `json:"id"`
	Score      float64                `json:"score,omitempty"`
	Similarity float32                `json:"similarity,omitempty"` // Cosine similarity for vector search
	Confidence string                 `json:"confidence,omitempty"` // "high", "medium", "low"
	Fields     map[string]interface{} `json:"fields"`
	Highlights map[string][]string    `json:"highlights,omitempty"`
	Type       string                 `json:"type"` // study, experiment, sample, run
}

// FacetValue represents a facet value and count
type FacetValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// IndexStats contains index statistics
type IndexStats struct {
	DocumentCount  uint64    `json:"document_count"`
	IndexSize      int64     `json:"index_size"`
	LastModified   time.Time `json:"last_modified"`
	IsHealthy      bool      `json:"is_healthy"`
	Backend        string    `json:"backend"`
	VectorsEnabled bool      `json:"vectors_enabled"`
}

// Document represents a generic document for indexing
type Document interface {
	GetID() string
	GetType() string
	GetFields() map[string]interface{}
}
