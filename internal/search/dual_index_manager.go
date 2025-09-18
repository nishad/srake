package search

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/embeddings"
)

var accessionPattern = regexp.MustCompile(`^[SED]R[RSXP][0-9]+`)

// DualIndexManager manages both content and reference indexes
type DualIndexManager struct {
	contentIndex   bleve.Index
	referenceIndex bleve.Index
	embedder       *embeddings.ONNXEmbedder
	config         *config.Config
	enabled        bool
}

// NewDualIndexManager creates a new dual index manager
func NewDualIndexManager(cfg *config.Config) (*DualIndexManager, error) {
	manager := &DualIndexManager{
		config:  cfg,
		enabled: cfg.Search.Enabled,
	}

	if !manager.enabled {
		return manager, nil
	}

	// Initialize embedder if configured
	if cfg.Embeddings.Enabled {
		embedder, err := embeddings.NewONNXEmbedder(
			cfg.Embeddings.DefaultModel,
			cfg.Embeddings.ModelsDirectory,
		)
		if err != nil {
			fmt.Printf("Warning: Failed to initialize embedder: %v\n", err)
			// Continue without embeddings
		} else {
			manager.embedder = embedder
		}
	}

	// Open or create content index
	contentPath := filepath.Join(cfg.Search.IndexPath, "content.bleve")
	contentIndex, err := openOrCreateIndex(contentPath, createContentMapping())
	if err != nil {
		return nil, fmt.Errorf("failed to open content index: %w", err)
	}
	manager.contentIndex = contentIndex

	// Open or create reference index
	referencePath := filepath.Join(cfg.Search.IndexPath, "reference.bleve")
	referenceIndex, err := openOrCreateIndex(referencePath, createReferenceMapping())
	if err != nil {
		contentIndex.Close()
		return nil, fmt.Errorf("failed to open reference index: %w", err)
	}
	manager.referenceIndex = referenceIndex

	return manager, nil
}

// openOrCreateIndex opens an existing index or creates a new one
func openOrCreateIndex(path string, mapping mapping.IndexMapping) (bleve.Index, error) {
	// Try to open existing index
	index, err := bleve.Open(path)
	if err == nil {
		return index, nil
	}

	// Create new index if it doesn't exist
	index, err = bleve.New(path, mapping)
	if err != nil {
		return nil, err
	}

	return index, nil
}

// createContentMapping creates the mapping for the content index
func createContentMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	// Study document mapping
	studyMapping := bleve.NewDocumentMapping()

	// Text fields
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Analyzer = standard.Name

	studyMapping.AddFieldMappingsAt("id", bleve.NewTextFieldMapping())
	studyMapping.AddFieldMappingsAt("type", bleve.NewTextFieldMapping())
	studyMapping.AddFieldMappingsAt("title", textFieldMapping)
	studyMapping.AddFieldMappingsAt("abstract", textFieldMapping)
	studyMapping.AddFieldMappingsAt("organism", textFieldMapping)
	studyMapping.AddFieldMappingsAt("study_type", textFieldMapping)

	// Vector field for embeddings (768 dimensions for SapBERT)
	// For now, store as numeric field until we enable vector build tag
	vectorField := bleve.NewNumericFieldMapping()
	studyMapping.AddFieldMappingsAt("embedding", vectorField)

	indexMapping.AddDocumentMapping("study", studyMapping)

	// Sample document mapping
	sampleMapping := bleve.NewDocumentMapping()

	sampleMapping.AddFieldMappingsAt("id", bleve.NewTextFieldMapping())
	sampleMapping.AddFieldMappingsAt("type", bleve.NewTextFieldMapping())
	sampleMapping.AddFieldMappingsAt("description", textFieldMapping)
	sampleMapping.AddFieldMappingsAt("organism", textFieldMapping)
	sampleMapping.AddFieldMappingsAt("tissue", textFieldMapping)
	sampleMapping.AddFieldMappingsAt("cell_type", textFieldMapping)

	// Vector field for samples (reuse same numeric field mapping)
	sampleVectorField := bleve.NewNumericFieldMapping()
	sampleMapping.AddFieldMappingsAt("embedding", sampleVectorField)

	indexMapping.AddDocumentMapping("sample", sampleMapping)

	return indexMapping
}

// createReferenceMapping creates the mapping for the reference index
func createReferenceMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	// Generic document mapping for all entity types
	docMapping := bleve.NewDocumentMapping()

	// Only index accession and type fields
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Index = true
	keywordFieldMapping.Store = true
	keywordFieldMapping.IncludeTermVectors = false

	docMapping.AddFieldMappingsAt("id", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("accession", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("type", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("library_strategy", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("platform", keywordFieldMapping)

	// Apply to all document types
	indexMapping.AddDocumentMapping("study", docMapping)
	indexMapping.AddDocumentMapping("experiment", docMapping)
	indexMapping.AddDocumentMapping("sample", docMapping)
	indexMapping.AddDocumentMapping("run", docMapping)

	return indexMapping
}

// RouteQuery determines which index to use based on the query
func (dm *DualIndexManager) RouteQuery(query string) bleve.Index {
	if !dm.enabled {
		return nil
	}

	// Check if query matches accession pattern
	if accessionPattern.MatchString(strings.TrimSpace(query)) {
		return dm.referenceIndex
	}

	// Check for technical queries (could be expanded)
	lowerQuery := strings.ToLower(query)
	technicalTerms := []string{"illumina", "pacbio", "rna-seq", "wgs", "wxs", "chip-seq"}
	for _, term := range technicalTerms {
		if strings.Contains(lowerQuery, term) {
			// Still use content index for technical terms as they may benefit from context
			return dm.contentIndex
		}
	}

	// Default to content index for semantic queries
	return dm.contentIndex
}

// Search performs a search on the appropriate index
func (dm *DualIndexManager) Search(query string, opts SearchOptions) (*SearchResult, error) {
	if !dm.enabled {
		return nil, fmt.Errorf("search is disabled")
	}

	// Route to appropriate index
	index := dm.RouteQuery(query)
	if index == nil {
		return nil, fmt.Errorf("no index available")
	}

	// Create search request
	searchRequest := bleve.NewSearchRequest(bleve.NewQueryStringQuery(query))
	searchRequest.Size = opts.Limit
	searchRequest.From = opts.Offset

	// Execute search
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to our result format
	result := &SearchResult{
		Query:     query,
		TotalHits: int(searchResult.Total),
		Hits:      make([]Hit, 0, len(searchResult.Hits)),
		Mode:      "text",
	}

	for _, hit := range searchResult.Hits {
		result.Hits = append(result.Hits, Hit{
			ID:     hit.ID,
			Score:  hit.Score,
			Fields: hit.Fields,
		})
	}

	return result, nil
}

// SearchWithVector performs a hybrid search with text and vector components
func (dm *DualIndexManager) SearchWithVector(query string, opts SearchOptions) (*SearchResult, error) {
	if !dm.enabled {
		return nil, fmt.Errorf("search is disabled")
	}

	if dm.embedder == nil {
		// Fall back to text-only search
		return dm.Search(query, opts)
	}

	// Generate query embedding
	queryEmbedding, err := dm.embedder.Embed(query)
	if err != nil {
		fmt.Printf("Warning: Failed to generate embedding: %v\n", err)
		// Fall back to text-only search
		return dm.Search(query, opts)
	}

	// Use content index for vector search
	index := dm.contentIndex

	// Create hybrid search request
	searchRequest := bleve.NewSearchRequest(bleve.NewQueryStringQuery(query))
	searchRequest.Size = opts.Limit
	searchRequest.From = opts.Offset

	// Add KNN vector search (commented until vector support is enabled)
	// TODO: Enable when Bleve vector support is available
	// knn := opts.KNN
	// if knn == 0 {
	// 	knn = 10 // Default K
	// }
	// searchRequest.AddKNN("embedding", queryEmbedding, knn, opts.VectorWeight)

	// For now, just ignore the embedding
	_ = queryEmbedding

	// Execute search
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}

	// Convert to our result format
	result := &SearchResult{
		Query:     query,
		TotalHits: int(searchResult.Total),
		Hits:      make([]Hit, 0, len(searchResult.Hits)),
		Mode:      "hybrid",
	}

	for _, hit := range searchResult.Hits {
		result.Hits = append(result.Hits, Hit{
			ID:     hit.ID,
			Score:  hit.Score,
			Fields: hit.Fields,
		})
	}

	return result, nil
}

// IndexDocument indexes a document in the appropriate index
func (dm *DualIndexManager) IndexDocument(doc map[string]interface{}) error {
	if !dm.enabled {
		return nil
	}

	docType, ok := doc["type"].(string)
	if !ok {
		return fmt.Errorf("document missing type field")
	}

	docID, ok := doc["id"].(string)
	if !ok {
		return fmt.Errorf("document missing id field")
	}

	// Always index in reference index (minimal fields)
	referenceDoc := map[string]interface{}{
		"id":   docID,
		"type": docType,
	}

	// Add accession if present
	if accession, ok := doc["accession"]; ok {
		referenceDoc["accession"] = accession
	}

	// Add technical fields for experiments
	if docType == "experiment" {
		if strategy, ok := doc["library_strategy"]; ok {
			referenceDoc["library_strategy"] = strategy
		}
		if platform, ok := doc["platform"]; ok {
			referenceDoc["platform"] = platform
		}
	}

	if err := dm.referenceIndex.Index(docID, referenceDoc); err != nil {
		return fmt.Errorf("failed to index in reference: %w", err)
	}

	// Index in content index if it's a study or sample with sufficient text
	if docType == "study" || docType == "sample" {
		// Check if document has enough text content
		textLength := 0
		if title, ok := doc["title"].(string); ok {
			textLength += len(title)
		}
		if abstract, ok := doc["abstract"].(string); ok {
			textLength += len(abstract)
		}
		if description, ok := doc["description"].(string); ok {
			textLength += len(description)
		}

		minLength := 50
		if docType == "study" {
			minLength = 100
		}

		if textLength >= minLength {
			// Generate embedding if embedder is available
			if dm.embedder != nil {
				text := dm.prepareTextForEmbedding(doc, docType)
				if len(text) > 0 {
					embedding, err := dm.embedder.Embed(text)
					if err == nil {
						doc["embedding"] = embedding
					}
				}
			}

			// Index in content index
			if err := dm.contentIndex.Index(docID, doc); err != nil {
				return fmt.Errorf("failed to index in content: %w", err)
			}
		}
	}

	return nil
}

// prepareTextForEmbedding prepares document text for embedding generation
func (dm *DualIndexManager) prepareTextForEmbedding(doc map[string]interface{}, docType string) string {
	var parts []string

	if docType == "study" {
		// Title gets double weight
		if title, ok := doc["title"].(string); ok && title != "" {
			parts = append(parts, title, title)
		}
		if abstract, ok := doc["abstract"].(string); ok && abstract != "" {
			// Limit abstract to 300 characters
			if len(abstract) > 300 {
				abstract = abstract[:300]
			}
			parts = append(parts, abstract)
		}
		if organism, ok := doc["organism"].(string); ok && organism != "" {
			parts = append(parts, "Organism: "+organism)
		}
	} else if docType == "sample" {
		if description, ok := doc["description"].(string); ok && description != "" {
			// Limit description to 200 characters
			if len(description) > 200 {
				description = description[:200]
			}
			parts = append(parts, description)
		}
		// Add biological context
		var context []string
		if organism, ok := doc["organism"].(string); ok && organism != "" {
			context = append(context, "organism:"+organism)
		}
		if tissue, ok := doc["tissue"].(string); ok && tissue != "" {
			context = append(context, "tissue:"+tissue)
		}
		if cellType, ok := doc["cell_type"].(string); ok && cellType != "" {
			context = append(context, "cell:"+cellType)
		}
		if len(context) > 0 {
			parts = append(parts, strings.Join(context, " "))
		}
	}

	text := strings.Join(parts, " ")
	// Limit total length to 512 characters (will be further limited by tokenizer)
	if len(text) > 512 {
		text = text[:512]
	}

	return text
}

// Close closes both indexes
func (dm *DualIndexManager) Close() error {
	if !dm.enabled {
		return nil
	}

	var lastErr error
	if dm.contentIndex != nil {
		if err := dm.contentIndex.Close(); err != nil {
			lastErr = err
		}
	}
	if dm.referenceIndex != nil {
		if err := dm.referenceIndex.Close(); err != nil {
			lastErr = err
		}
	}
	if dm.embedder != nil {
		dm.embedder.Close()
	}

	return lastErr
}

// GetStats returns statistics for both indexes
func (dm *DualIndexManager) GetStats() (*IndexStats, error) {
	if !dm.enabled {
		return &IndexStats{IsHealthy: false}, nil
	}

	stats := &IndexStats{
		IsHealthy:      true,
		Backend:        "dual-bleve",
		VectorsEnabled: dm.embedder != nil,
	}

	// Get content index stats
	if dm.contentIndex != nil {
		contentCount, err := dm.contentIndex.DocCount()
		if err == nil {
			stats.DocumentCount += contentCount
		}
	}

	// Get reference index stats
	if dm.referenceIndex != nil {
		refCount, err := dm.referenceIndex.DocCount()
		if err == nil {
			stats.DocumentCount += refCount
		}
	}

	return stats, nil
}

// Flush flushes pending changes in both indexes
func (dm *DualIndexManager) Flush() error {
	if !dm.enabled {
		return nil
	}

	// Bleve indexes auto-flush, but we can trigger it
	// by doing a dummy search to ensure consistency
	if dm.contentIndex != nil {
		_, _ = dm.contentIndex.DocCount()
	}
	if dm.referenceIndex != nil {
		_, _ = dm.referenceIndex.DocCount()
	}

	return nil
}