//go:build search || vectors
// +build search vectors

package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/nishad/srake/internal/config"
)

// BleveBackend implements SearchBackend with optional vector support
type BleveBackend struct {
	index  bleve.Index
	config *config.Config
	path   string
	mu     sync.RWMutex
}

// NewBleveBackend creates a new Bleve search backend
func NewBleveBackend(cfg *config.Config) (*BleveBackend, error) {
	b := &BleveBackend{
		config: cfg,
		path:   cfg.Search.IndexPath,
	}

	// Try to open existing index
	index, err := bleve.Open(b.path)
	if err == bleve.ErrorIndexPathDoesNotExist {
		// Create new index
		indexMapping := b.createIndexMapping()
		index, err = bleve.NewUsing(b.path, indexMapping, "scorch", "scorch", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to open index: %w", err)
	}

	b.index = index
	return b, nil
}

// createIndexMapping creates an index mapping with optional vector support
func (b *BleveBackend) createIndexMapping() mapping.IndexMapping {
	// Create a new index mapping
	indexMapping := bleve.NewIndexMapping()

	// Create document mappings
	indexMapping.DefaultMapping = b.createDocumentMapping()

	// Use standard analyzer
	indexMapping.DefaultAnalyzer = "standard"

	return indexMapping
}


// createDocumentMapping creates the document mapping with optional vector field
func (b *BleveBackend) createDocumentMapping() *mapping.DocumentMapping {
	docMapping := bleve.NewDocumentMapping()

	// Text fields
	docMapping.AddFieldMappingsAt("id", b.createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("type", b.createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("title", b.createTextFieldMapping())
	docMapping.AddFieldMappingsAt("abstract", b.createTextFieldMapping())
	docMapping.AddFieldMappingsAt("organism", b.createTextFieldMapping())
	docMapping.AddFieldMappingsAt("library_strategy", b.createTextFieldMapping())
	docMapping.AddFieldMappingsAt("platform", b.createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("instrument_model", b.createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("scientific_name", b.createTextFieldMapping())
	docMapping.AddFieldMappingsAt("tissue", b.createTextFieldMapping())
	docMapping.AddFieldMappingsAt("cell_type", b.createTextFieldMapping())
	docMapping.AddFieldMappingsAt("study_type", b.createKeywordFieldMapping())

	// Numeric fields
	docMapping.AddFieldMappingsAt("spots", b.createNumericFieldMapping())
	docMapping.AddFieldMappingsAt("bases", b.createNumericFieldMapping())
	docMapping.AddFieldMappingsAt("submission_date", b.createDateTimeFieldMapping())

	// Vector field if enabled
	if b.config.IsVectorEnabled() {
		vectorMapping := b.createVectorFieldMapping()
		docMapping.AddFieldMappingsAt("embedding", vectorMapping)
	}

	return docMapping
}

// Helper functions to create field mappings
func (b *BleveBackend) createKeywordFieldMapping() *mapping.FieldMapping {
	fm := bleve.NewTextFieldMapping()
	fm.Analyzer = "keyword"
	fm.Store = true
	fm.IncludeInAll = true
	return fm
}

func (b *BleveBackend) createTextFieldMapping() *mapping.FieldMapping {
	fm := bleve.NewTextFieldMapping()
	fm.Analyzer = "standard"
	fm.Store = true
	fm.IncludeInAll = true
	return fm
}

func (b *BleveBackend) createNumericFieldMapping() *mapping.FieldMapping {
	fm := bleve.NewNumericFieldMapping()
	fm.Store = true
	fm.IncludeInAll = false
	return fm
}

func (b *BleveBackend) createDateTimeFieldMapping() *mapping.FieldMapping {
	fm := bleve.NewDateTimeFieldMapping()
	fm.Store = true
	fm.IncludeInAll = false
	return fm
}

// Index adds a document to the index
func (b *BleveBackend) Index(doc interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Extract ID from document
	id := ""
	switch d := doc.(type) {
	case Document:
		id = d.GetID()
	case map[string]interface{}:
		if idVal, ok := d["id"].(string); ok {
			id = idVal
		}
	default:
		return fmt.Errorf("unsupported document type")
	}

	if id == "" {
		return fmt.Errorf("document has no ID")
	}

	return b.index.Index(id, doc)
}

// IndexBatch indexes multiple documents in a batch
func (b *BleveBackend) IndexBatch(docs []interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	batch := b.index.NewBatch()

	for _, doc := range docs {
		// Extract ID from document
		id := ""
		switch d := doc.(type) {
		case Document:
			id = d.GetID()
		case map[string]interface{}:
			if idVal, ok := d["id"].(string); ok {
				id = idVal
			}
		}

		if id == "" {
			continue // Skip documents without ID
		}

		if err := batch.Index(id, doc); err != nil {
			return fmt.Errorf("failed to add document %s to batch: %w", id, err)
		}
	}

	return b.index.Batch(batch)
}

// Delete removes a document from the index
func (b *BleveBackend) Delete(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.index.Delete(id)
}

// DeleteBatch removes multiple documents from the index
func (b *BleveBackend) DeleteBatch(ids []string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	batch := b.index.NewBatch()
	for _, id := range ids {
		batch.Delete(id)
	}

	return b.index.Batch(batch)
}

// Search performs a text search
func (b *BleveBackend) Search(queryStr string, opts SearchOptions) (*SearchResult, error) {
	start := time.Now()

	// Build query
	var q query.Query
	if queryStr != "" {
		q = bleve.NewQueryStringQuery(queryStr)
	} else {
		q = bleve.NewMatchAllQuery()
	}

	// Apply filters if any
	if len(opts.Filters) > 0 {
		queries := []query.Query{q}
		for field, value := range opts.Filters {
			termQuery := bleve.NewTermQuery(fmt.Sprintf("%v", value))
			termQuery.SetField(field)
			queries = append(queries, termQuery)
		}
		q = bleve.NewConjunctionQuery(queries...)
	}

	// Create search request
	searchRequest := bleve.NewSearchRequest(q)
	searchRequest.Size = opts.Limit
	searchRequest.From = opts.Offset

	// Add fields to retrieve
	searchRequest.Fields = []string{"*"}

	// Add facets if requested
	for _, facetField := range opts.Facets {
		searchRequest.AddFacet(facetField, bleve.NewFacetRequest(facetField, 10))
	}

	// Add highlight if requested
	if opts.Highlight {
		searchRequest.Highlight = bleve.NewHighlight()
	}

	// Set timeout if specified
	// Note: SearchContext is not available in Bleve v2.3
	// TODO: Add timeout support when upgrading to newer Bleve version

	// Execute search
	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	// Convert to our result format
	result := b.convertSearchResult(searchResult, queryStr, start)
	result.Mode = "text"

	return result, nil
}

// SearchWithVector performs a hybrid text + vector search
func (b *BleveBackend) SearchWithVector(queryStr string, vector []float32, opts SearchOptions) (*SearchResult, error) {
	if !b.config.IsVectorEnabled() {
		return b.Search(queryStr, opts)
	}

	start := time.Now()

	// Build text query if provided
	var textQuery query.Query
	if queryStr != "" {
		textQuery = bleve.NewQueryStringQuery(queryStr)
	}

	// Create search request
	var searchRequest *bleve.SearchRequest
	if textQuery != nil {
		searchRequest = bleve.NewSearchRequest(textQuery)
	} else {
		searchRequest = bleve.NewSearchRequest(bleve.NewMatchNoneQuery())
	}

	// Add kNN vector search if vector provided
	if len(vector) > 0 && b.config.IsVectorEnabled() {
		if err := addKNNToRequest(searchRequest, "embedding", vector, opts.Limit, 1.0); err != nil {
			// Fall back to text-only search if vectors not available
			fmt.Printf("Warning: %v\n", err)
		}
	}

	// Configure request
	searchRequest.Size = opts.Limit
	searchRequest.From = opts.Offset
	searchRequest.Fields = []string{"*"}

	// Add facets if requested
	for _, facetField := range opts.Facets {
		searchRequest.AddFacet(facetField, bleve.NewFacetRequest(facetField, 10))
	}

	// Execute search
	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	// Convert to our result format
	result := b.convertSearchResult(searchResult, queryStr, start)
	if queryStr != "" {
		result.Mode = "hybrid"
	} else {
		result.Mode = "vector"
	}

	return result, nil
}

// FindSimilar finds documents similar to the given ID
func (b *BleveBackend) FindSimilar(id string, opts SearchOptions) (*SearchResult, error) {
	if !b.config.IsVectorEnabled() {
		return nil, fmt.Errorf("vector search is not enabled")
	}

	// First, retrieve the document to get its embedding
	docQuery := bleve.NewDocIDQuery([]string{id})
	searchRequest := bleve.NewSearchRequest(docQuery)
	searchRequest.Fields = []string{"embedding"}

	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve document: %w", err)
	}

	if searchResult.Total == 0 {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	// Extract embedding
	hit := searchResult.Hits[0]
	embeddingField, ok := hit.Fields["embedding"]
	if !ok {
		return nil, fmt.Errorf("document has no embedding")
	}

	// Convert to []float32
	var embedding []float32
	switch v := embeddingField.(type) {
	case []interface{}:
		embedding = make([]float32, len(v))
		for i, val := range v {
			if f, ok := val.(float64); ok {
				embedding[i] = float32(f)
			}
		}
	case []float32:
		embedding = v
	default:
		return nil, fmt.Errorf("invalid embedding format")
	}

	// Search for similar documents
	return b.SearchWithVector("", embedding, opts)
}

// convertSearchResult converts Bleve results to our format
func (b *BleveBackend) convertSearchResult(sr *bleve.SearchResult, query string, start time.Time) *SearchResult {
	result := &SearchResult{
		Query:     query,
		TotalHits: int(sr.Total),
		Hits:      make([]Hit, 0, len(sr.Hits)),
		TimeMs:    time.Since(start).Milliseconds(),
	}

	// Convert hits
	for _, hit := range sr.Hits {
		h := Hit{
			ID:     hit.ID,
			Score:  hit.Score,
			Fields: hit.Fields,
		}

		// Extract type from fields
		if typeField, ok := hit.Fields["type"].(string); ok {
			h.Type = typeField
		}

		// Add highlights if available
		if len(hit.Fragments) > 0 {
			h.Highlights = hit.Fragments
		}

		result.Hits = append(result.Hits, h)
	}

	// Convert facets
	if len(sr.Facets) > 0 {
		result.Facets = make(map[string][]FacetValue)
		for name, facet := range sr.Facets {
			values := make([]FacetValue, 0)
			for _, term := range facet.Terms.Terms() {
				values = append(values, FacetValue{
					Value: term.Term,
					Count: term.Count,
				})
			}
			result.Facets[name] = values
		}
	}

	return result
}

// Close closes the Bleve index
func (b *BleveBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.index != nil {
		return b.index.Close()
	}
	return nil
}

// IsEnabled returns true if the backend is enabled
func (b *BleveBackend) IsEnabled() bool {
	return b.config.IsSearchEnabled()
}

// GetStats returns index statistics
func (b *BleveBackend) GetStats() (*IndexStats, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	docCount, err := b.index.DocCount()
	if err != nil {
		return nil, err
	}

	// Get index size
	info, err := os.Stat(b.path)
	var size int64
	if err == nil && info.IsDir() {
		size = b.getDirSize(b.path)
	}

	return &IndexStats{
		DocumentCount:  docCount,
		IndexSize:      size,
		LastModified:   time.Now(),
		IsHealthy:      true,
		Backend:        "bleve",
		VectorsEnabled: b.config.IsVectorEnabled(),
	}, nil
}

// getDirSize calculates the size of a directory
func (b *BleveBackend) getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size
}

// Rebuild rebuilds the index from scratch
func (b *BleveBackend) Rebuild(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Close existing index
	if b.index != nil {
		b.index.Close()
	}

	// Remove existing index directory
	os.RemoveAll(b.path)

	// Create new index
	indexMapping := b.createIndexMapping()
	index, err := bleve.NewUsing(b.path, indexMapping, "scorch", "scorch", nil)
	if err != nil {
		return fmt.Errorf("failed to create new index: %w", err)
	}

	b.index = index
	return nil
}