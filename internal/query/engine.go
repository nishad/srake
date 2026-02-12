// Package query provides a unified search engine that combines full-text (Bleve),
// structured metadata (SQLite), and vector similarity search into a single interface.
package query

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/embeddings"
	"github.com/nishad/srake/internal/search"
	"github.com/nishad/srake/internal/vectors"
)

// QueryEngine provides a unified interface combining Bleve full-text search,
// SQLite metadata queries, and optional vector similarity search.
type QueryEngine struct {
	db       *database.DB
	bleve    *search.BleveIndex
	vectors  *vectors.VectorStore
	embedder *embeddings.Embedder
	cache    *Cache
	mu       sync.RWMutex
}

// SearchOptions configures search behavior including vector search, fuzzy matching,
// filters, and result limits.
type SearchOptions struct {
	UseVectors      bool              `json:"use_vectors"`
	UseFuzzy        bool              `json:"use_fuzzy"`
	Filters         map[string]string `json:"filters"`
	Limit           int               `json:"limit"`
	IncludeFacets   bool              `json:"include_facets"`
	VectorThreshold float32           `json:"vector_threshold"`
}

// SearchResults contains merged results from text, metadata, and vector search systems.
type SearchResults struct {
	TextMatches     []TextMatch                `json:"text_matches,omitempty"`
	MetadataMatches []MetadataMatch            `json:"metadata_matches,omitempty"`
	SimilarProjects []vectors.SimilarityResult `json:"similar_projects,omitempty"`
	Facets          map[string][]Facet         `json:"facets,omitempty"`
	TotalHits       int                        `json:"total_hits"`
	SearchTime      time.Duration              `json:"search_time"`
	Query           string                     `json:"query"`
}

// TextMatch represents a single full-text search hit with its relevance score and highlighted fragments.
type TextMatch struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Score     float64                `json:"score"`
	Title     string                 `json:"title"`
	Highlight map[string][]string    `json:"highlight,omitempty"`
	Fields    map[string]interface{} `json:"fields"`
}

// MetadataMatch represents a result from structured metadata filtering (e.g., organism or library strategy).
type MetadataMatch struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Fields map[string]interface{} `json:"fields"`
}

// Facet represents a term and its count within a search facet aggregation.
type Facet struct {
	Term  string `json:"term"`
	Count int    `json:"count"`
}

// NewQueryEngine creates a new QueryEngine with database, Bleve, and optional
// vector search backends initialized from the given data directory.
func NewQueryEngine(dataDir string) (*QueryEngine, error) {
	// Initialize database
	dbPath := filepath.Join(dataDir, "metadata.db")
	db, err := database.Initialize(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize Bleve
	bleveIndex, err := search.InitBleveIndex(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Bleve: %w", err)
	}

	// Initialize vector store
	vectorStore, err := vectors.NewVectorStore(dataDir)
	if err != nil {
		// Vector store is optional
		fmt.Printf("Warning: vector store not available: %v\n", err)
		vectorStore = nil
	}

	// Initialize embedder (optional)
	var embedder *embeddings.Embedder
	if vectorStore != nil {
		config := embeddings.DefaultEmbedderConfig()
		embedder, err = embeddings.NewEmbedder(config)
		if err != nil {
			fmt.Printf("Warning: embedder not available: %v\n", err)
			embedder = nil
		}
	}

	// Initialize cache
	cache := NewCache(100, 5*time.Minute)

	return &QueryEngine{
		db:       db,
		bleve:    bleveIndex,
		vectors:  vectorStore,
		embedder: embedder,
		cache:    cache,
	}, nil
}

// Search performs a hybrid search across Bleve, SQLite metadata, and optionally
// vector similarity, merging and ranking results from all systems.
func (qe *QueryEngine) Search(query string, opts SearchOptions) (*SearchResults, error) {
	startTime := time.Now()

	// Check cache
	cacheKey := fmt.Sprintf("%s:%+v", query, opts)
	if cached := qe.cache.Get(cacheKey); cached != nil {
		if results, ok := cached.(*SearchResults); ok {
			return results, nil
		}
	}

	// Set default limit
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	results := &SearchResults{
		Query:  query,
		Facets: make(map[string][]Facet),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	// Parallel search across all systems

	// 1. Bleve full-text search
	wg.Add(1)
	go func() {
		defer wg.Done()

		var searchResults interface{}
		var err error

		if opts.UseFuzzy {
			searchResults, err = qe.bleve.FuzzySearch(query, 2, opts.Limit)
		} else if len(opts.Filters) > 0 {
			searchResults, err = qe.bleve.SearchWithFilters(query, opts.Filters, opts.Limit)
		} else {
			searchResults, err = qe.bleve.Search(query, opts.Limit)
		}

		if err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("bleve search failed: %w", err))
			mu.Unlock()
			return
		}

		// Convert Bleve results
		if bleveResults, ok := searchResults.(*search.BleveSearchResult); ok {
			mu.Lock()
			results.TextMatches = convertBleveResults(bleveResults)
			if opts.IncludeFacets {
				results.Facets = convertBleveFacets(bleveResults)
			}
			results.TotalHits += int(bleveResults.Total)
			mu.Unlock()
		}
	}()

	// 2. SQLite metadata search
	if len(opts.Filters) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			metadataResults, err := qe.searchMetadata(opts.Filters, opts.Limit)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("metadata search failed: %w", err))
				mu.Unlock()
				return
			}

			mu.Lock()
			results.MetadataMatches = metadataResults
			results.TotalHits += len(metadataResults)
			mu.Unlock()
		}()
	}

	// 3. Vector similarity search (if available and requested)
	if qe.embedder != nil && qe.vectors != nil && opts.UseVectors {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Generate embedding for query
			embedding, err := qe.embedder.EmbedText(query)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("embedding generation failed: %w", err))
				mu.Unlock()
				return
			}

			// Search similar projects
			similarProjects, err := qe.vectors.SearchSimilar(embedding, 20)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("vector search failed: %w", err))
				mu.Unlock()
				return
			}

			// Filter by threshold
			if opts.VectorThreshold > 0 {
				filtered := make([]vectors.SimilarityResult, 0)
				for _, p := range similarProjects {
					if p.Score >= opts.VectorThreshold {
						filtered = append(filtered, p)
					}
				}
				similarProjects = filtered
			}

			mu.Lock()
			results.SimilarProjects = similarProjects
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Handle errors
	if len(errors) > 0 {
		// Continue with partial results but log errors
		for _, err := range errors {
			fmt.Printf("Search error: %v\n", err)
		}
	}

	// Merge and rank results
	results = qe.mergeAndRank(results, opts)

	results.SearchTime = time.Since(startTime)

	// Cache results
	qe.cache.Set(cacheKey, results)

	return results, nil
}

// searchMetadata performs structured metadata search
func (qe *QueryEngine) searchMetadata(filters map[string]string, limit int) ([]MetadataMatch, error) {
	var results []MetadataMatch

	// Search by organism
	if organism, ok := filters["organism"]; ok {
		samples, err := qe.db.SearchByOrganism(organism, limit)
		if err != nil {
			return nil, err
		}

		for _, sample := range samples {
			results = append(results, MetadataMatch{
				ID:   sample.SampleAccession,
				Type: "sample",
				Fields: map[string]interface{}{
					"organism":        sample.Organism,
					"scientific_name": sample.ScientificName,
					"tissue":          sample.Tissue,
					"cell_type":       sample.CellType,
					"description":     sample.Description,
				},
			})
		}
	}

	// Search by library strategy
	if strategy, ok := filters["library_strategy"]; ok {
		experiments, err := qe.db.SearchByLibraryStrategy(strategy, limit)
		if err != nil {
			return nil, err
		}

		for _, exp := range experiments {
			results = append(results, MetadataMatch{
				ID:   exp.ExperimentAccession,
				Type: "experiment",
				Fields: map[string]interface{}{
					"title":            exp.Title,
					"library_strategy": exp.LibraryStrategy,
					"platform":         exp.Platform,
					"instrument_model": exp.InstrumentModel,
				},
			})
		}
	}

	return results, nil
}

// mergeAndRank merges results from different sources and ranks them
func (qe *QueryEngine) mergeAndRank(results *SearchResults, opts SearchOptions) *SearchResults {
	// Simple merge for now - can be enhanced with more sophisticated ranking
	// Priority: exact matches > text matches > metadata matches > vector matches

	// Limit total results
	totalLimit := opts.Limit
	currentCount := 0

	// Trim text matches
	if len(results.TextMatches) > 0 {
		remaining := totalLimit - currentCount
		if len(results.TextMatches) > remaining {
			results.TextMatches = results.TextMatches[:remaining]
		}
		currentCount += len(results.TextMatches)
	}

	// Trim metadata matches
	if len(results.MetadataMatches) > 0 && currentCount < totalLimit {
		remaining := totalLimit - currentCount
		if len(results.MetadataMatches) > remaining {
			results.MetadataMatches = results.MetadataMatches[:remaining]
		}
		currentCount += len(results.MetadataMatches)
	}

	// Trim similar projects
	if len(results.SimilarProjects) > 0 && currentCount < totalLimit {
		remaining := totalLimit - currentCount
		if len(results.SimilarProjects) > remaining {
			results.SimilarProjects = results.SimilarProjects[:remaining]
		}
	}

	return results
}

// FindSimilar finds studies or samples similar to the given accession using vector embeddings.
func (qe *QueryEngine) FindSimilar(entityType, accession string, limit int) ([]vectors.SimilarityResult, error) {
	if qe.vectors == nil {
		return nil, fmt.Errorf("vector store not available")
	}

	switch entityType {
	case "study", "project":
		return qe.vectors.FindSimilarProjects(accession, limit, nil)
	case "sample":
		return qe.vectors.FindSimilarSamples(accession, limit)
	default:
		return nil, fmt.Errorf("unsupported entity type: %s", entityType)
	}
}

// IndexDocument indexes a document in Bleve and optionally generates and stores
// a vector embedding. The docType must be "study", "experiment", or "sample".
func (qe *QueryEngine) IndexDocument(docType string, doc interface{}) error {
	qe.mu.Lock()
	defer qe.mu.Unlock()

	var errors []error

	// Index in Bleve
	switch docType {
	case "study":
		if study, ok := doc.(*database.Study); ok {
			bleveDoc := search.StudyDoc{
				StudyAccession: study.StudyAccession,
				StudyTitle:     study.StudyTitle,
				StudyAbstract:  study.StudyAbstract,
				StudyType:      study.StudyType,
				Organism:       study.Organism,
			}
			if err := qe.bleve.IndexStudy(bleveDoc); err != nil {
				errors = append(errors, fmt.Errorf("bleve indexing failed: %w", err))
			}

			// Generate and store embedding if available
			if qe.embedder != nil && qe.vectors != nil {
				text := study.StudyTitle + " " + study.StudyAbstract + " " + study.Organism
				embedding, err := qe.embedder.EmbedText(text)
				if err == nil {
					pv := &vectors.ProjectVector{
						ProjectID: study.StudyAccession,
						Title:     study.StudyTitle,
						Abstract:  study.StudyAbstract,
						Organism:  study.Organism,
						StudyType: study.StudyType,
						Embedding: embedding,
					}
					if err := qe.vectors.InsertProjectVector(pv); err != nil {
						errors = append(errors, fmt.Errorf("vector indexing failed: %w", err))
					}
				}
			}
		}

	case "experiment":
		if exp, ok := doc.(*database.Experiment); ok {
			bleveDoc := search.ExperimentDoc{
				ExperimentAccession: exp.ExperimentAccession,
				Title:               exp.Title,
				LibraryStrategy:     exp.LibraryStrategy,
				Platform:            exp.Platform,
				InstrumentModel:     exp.InstrumentModel,
			}
			if err := qe.bleve.IndexExperiment(bleveDoc); err != nil {
				errors = append(errors, fmt.Errorf("bleve indexing failed: %w", err))
			}
		}

	case "sample":
		if sample, ok := doc.(*database.Sample); ok {
			bleveDoc := search.SampleDoc{
				SampleAccession: sample.SampleAccession,
				Organism:        sample.Organism,
				ScientificName:  sample.ScientificName,
				Tissue:          sample.Tissue,
				CellType:        sample.CellType,
				Description:     sample.Description,
			}
			if err := qe.bleve.IndexSample(bleveDoc); err != nil {
				errors = append(errors, fmt.Errorf("bleve indexing failed: %w", err))
			}

			// Generate and store embedding if available
			if qe.embedder != nil && qe.vectors != nil {
				text := sample.Description + " " + sample.Organism + " " + sample.Tissue
				embedding, err := qe.embedder.EmbedText(text)
				if err == nil {
					sv := &vectors.SampleVector{
						SampleID:    sample.SampleAccession,
						Description: sample.Description,
						Organism:    sample.Organism,
						Tissue:      sample.Tissue,
						CellType:    sample.CellType,
						Embedding:   embedding,
					}
					if err := qe.vectors.InsertSampleVector(sv); err != nil {
						errors = append(errors, fmt.Errorf("vector indexing failed: %w", err))
					}
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("indexing errors: %v", errors)
	}

	return nil
}

// GetStats returns statistics from all query engine subsystems including
// database counts, Bleve document counts, vector store stats, and cache metrics.
func (qe *QueryEngine) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Database stats
	dbStats, err := qe.db.GetStats()
	if err == nil {
		stats["database"] = dbStats
	}

	// Bleve stats
	docCount, err := qe.bleve.GetDocCount()
	if err == nil {
		stats["bleve_docs"] = docCount
	}

	// Vector stats
	if qe.vectors != nil {
		vectorStats, err := qe.vectors.GetStats()
		if err == nil {
			stats["vectors"] = vectorStats
		}
	}

	// Embedder status
	stats["embedder_available"] = qe.embedder != nil

	// Cache stats
	stats["cache"] = qe.cache.GetStats()

	return stats, nil
}

// Close releases all resources held by the query engine, including database
// connections, search indexes, vector store, and embedder.
func (qe *QueryEngine) Close() error {
	var errors []error

	if err := qe.db.Close(); err != nil {
		errors = append(errors, fmt.Errorf("database close error: %w", err))
	}

	if err := qe.bleve.Close(); err != nil {
		errors = append(errors, fmt.Errorf("bleve close error: %w", err))
	}

	if qe.vectors != nil {
		if err := qe.vectors.Close(); err != nil {
			errors = append(errors, fmt.Errorf("vector store close error: %w", err))
		}
	}

	if qe.embedder != nil {
		qe.embedder.Close()
	}

	if len(errors) > 0 {
		return fmt.Errorf("close errors: %v", errors)
	}

	return nil
}

// Helper functions

func convertBleveResults(bleveResults *search.BleveSearchResult) []TextMatch {
	// This is a placeholder - implement based on actual Bleve result structure
	return []TextMatch{}
}

func convertBleveFacets(bleveResults *search.BleveSearchResult) map[string][]Facet {
	// This is a placeholder - implement based on actual Bleve facet structure
	return map[string][]Facet{}
}
