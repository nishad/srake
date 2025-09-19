package service

import (
	"context"
	"time"

	"github.com/nishad/srake/internal/database"
)

// SearchRequest represents a search request with all parameters
type SearchRequest struct {
	Query   string            `json:"query"`
	Filters map[string]string `json:"filters,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Output control
	Format string   `json:"format,omitempty"`
	Fields []string `json:"fields,omitempty"`

	// Search options
	Fuzzy      bool   `json:"fuzzy,omitempty"`
	Exact      bool   `json:"exact,omitempty"`
	UseVectors bool   `json:"use_vectors,omitempty"`
	SearchMode string `json:"search_mode,omitempty"`

	// Quality control
	SimilarityThreshold float32 `json:"similarity_threshold,omitempty"`
	MinScore            float32 `json:"min_score,omitempty"`
	TopPercentile       int     `json:"top_percentile,omitempty"`
	ShowConfidence      bool    `json:"show_confidence,omitempty"`
	HybridWeight        float32 `json:"hybrid_weight,omitempty"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Results      []*SearchResult        `json:"results"`
	TotalResults int                    `json:"total_results"`
	Query        string                 `json:"query"`
	TimeTaken    int64                  `json:"time_taken_ms"`
	SearchMode   string                 `json:"search_mode,omitempty"`
	Facets       map[string]interface{} `json:"facets,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Title           string                 `json:"title,omitempty"`
	Description     string                 `json:"description,omitempty"`
	Organism        string                 `json:"organism,omitempty"`
	Platform        string                 `json:"platform,omitempty"`
	LibraryStrategy string                 `json:"library_strategy,omitempty"`
	Score           float32                `json:"score,omitempty"`
	Similarity      float32                `json:"similarity,omitempty"`
	Confidence      string                 `json:"confidence,omitempty"`
	Fields          map[string]interface{} `json:"fields,omitempty"`
	Highlights      map[string][]string    `json:"highlights,omitempty"`
}

// SearchHit represents a single search result with full entities (deprecated, use SearchResult)
type SearchHit struct {
	Study      *database.Study      `json:"study,omitempty"`
	Experiment *database.Experiment `json:"experiment,omitempty"`
	Sample     *database.Sample     `json:"sample,omitempty"`
	Run        *database.Run        `json:"run,omitempty"`
	Score      float64              `json:"score,omitempty"`
	Confidence string               `json:"confidence,omitempty"`
	Highlights map[string][]string  `json:"highlights,omitempty"`
}

// MetadataRequest for fetching specific records
type MetadataRequest struct {
	Accession string `json:"accession"`
	Type      string `json:"type"` // study, experiment, sample, run
}

// MetadataResponse for metadata queries
type MetadataResponse struct {
	Data      interface{} `json:"data"`
	Type      string      `json:"type"`
	Retrieved time.Time   `json:"retrieved"`
}

// SearchStats for search service statistics
type SearchStats struct {
	TotalDocuments   int64      `json:"total_documents"`
	IndexedDocuments int64      `json:"indexed_documents"`
	IndexSize        int64      `json:"index_size"`
	LastIndexed      time.Time  `json:"last_indexed,omitempty"`
	LastUpdated      time.Time  `json:"last_updated"`
	TopOrganisms     []StatItem `json:"top_organisms,omitempty"`
	TopPlatforms     []StatItem `json:"top_platforms,omitempty"`
	TopStrategies    []StatItem `json:"top_strategies,omitempty"`
}

// StatItem for statistical data
type StatItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// StatsResponse for database statistics (alias for API compatibility)
type StatsResponse struct {
	TotalStudies     int64       `json:"total_studies"`
	TotalExperiments int64       `json:"total_experiments"`
	TotalSamples     int64       `json:"total_samples"`
	TotalRuns        int64       `json:"total_runs"`
	LastUpdate       time.Time   `json:"last_update"`
	IndexSize        int64       `json:"index_size"`
	DatabaseSize     int64       `json:"database_size"`
	TopOrganisms     []CountItem `json:"top_organisms,omitempty"`
	TopPlatforms     []CountItem `json:"top_platforms,omitempty"`
	TopStrategies    []CountItem `json:"top_strategies,omitempty"`
}

// CountItem for statistical counts
type CountItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// ExportRequest for data export
type ExportRequest struct {
	Query   string            `json:"query"`
	Filters map[string]string `json:"filters,omitempty"`
	Format  string            `json:"format"` // json, csv, tsv, xml
	Limit   int               `json:"limit,omitempty"`
	Fields  []string          `json:"fields,omitempty"`
}

// IngestRequest for data ingestion
type IngestRequest struct {
	Source    string        `json:"source"` // file path or URL
	BatchSize int           `json:"batch_size,omitempty"`
	Filters   IngestFilters `json:"filters,omitempty"`
}

// IngestFilters for selective ingestion
type IngestFilters struct {
	TaxonIDs   []string  `json:"taxon_ids,omitempty"`
	Platforms  []string  `json:"platforms,omitempty"`
	Strategies []string  `json:"strategies,omitempty"`
	DateFrom   time.Time `json:"date_from,omitempty"`
	DateTo     time.Time `json:"date_to,omitempty"`
}

// IngestResponse for ingestion results
type IngestResponse struct {
	Processed int64         `json:"processed"`
	Inserted  int64         `json:"inserted"`
	Updated   int64         `json:"updated"`
	Errors    int64         `json:"errors"`
	Duration  time.Duration `json:"duration"`
}

// IndexRequest for index operations
type IndexRequest struct {
	Operation   string `json:"operation"` // build, rebuild, update
	BatchSize   int    `json:"batch_size,omitempty"`
	WithVectors bool   `json:"with_vectors,omitempty"`
}

// IndexResponse for index operations
type IndexResponse struct {
	Status    string        `json:"status"`
	Documents int64         `json:"documents"`
	Duration  time.Duration `json:"duration"`
}

// ServiceError represents a service-level error
type ServiceError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e *ServiceError) Error() string {
	return e.Message
}

// Common service interfaces that all services should implement
type BaseService interface {
	// Health checks if the service is operational
	Health(ctx context.Context) error

	// Close releases any resources
	Close() error
}
