package search

import (
	"context"
	"fmt"
	"time"
)

// bleveIndexWrapper wraps BleveIndex to implement SearchBackend interface
type bleveIndexWrapper struct {
	index *BleveIndex
}

// Index operations
func (w *bleveIndexWrapper) Index(doc interface{}) error {
	switch d := doc.(type) {
	case StudyDoc:
		return w.index.IndexStudy(d)
	case ExperimentDoc:
		return w.index.IndexExperiment(d)
	case SampleDoc:
		return w.index.IndexSample(d)
	case RunDoc:
		return w.index.IndexRun(d)
	case map[string]interface{}:
		// Generic document
		// For now, just skip generic documents
		return fmt.Errorf("generic documents not supported yet")
	default:
		return fmt.Errorf("unsupported document type: %T", doc)
	}
}

func (w *bleveIndexWrapper) IndexBatch(docs []interface{}) error {
	return w.index.BatchIndex(docs)
}

func (w *bleveIndexWrapper) Delete(id string) error {
	return w.index.Delete(id)
}

func (w *bleveIndexWrapper) DeleteBatch(ids []string) error {
	for _, id := range ids {
		if err := w.index.Delete(id); err != nil {
			return err
		}
	}
	return nil
}

// Search operations
func (w *bleveIndexWrapper) Search(query string, opts SearchOptions) (*SearchResult, error) {
	bleveResult, err := w.index.Search(query, opts.Limit)
	if err != nil {
		return nil, err
	}

	// Convert BleveSearchResult to SearchResult
	result := &SearchResult{
		Query:     query,
		TotalHits: int(bleveResult.Total),
		Hits:      make([]Hit, 0, len(bleveResult.Hits)),
		TimeMs:    int64(bleveResult.Took / time.Millisecond),
		Mode:      "text",
	}

	for _, hit := range bleveResult.Hits {
		h := Hit{
			ID:     hit.ID,
			Score:  hit.Score,
			Fields: hit.Fields,
		}

		// Extract type if available
		if typeField, ok := hit.Fields["type"].(string); ok {
			h.Type = typeField
		}

		result.Hits = append(result.Hits, h)
	}

	// Convert facets if any
	if len(bleveResult.Facets) > 0 {
		result.Facets = make(map[string][]FacetValue)
		for name, facet := range bleveResult.Facets {
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

	return result, nil
}

func (w *bleveIndexWrapper) SearchWithVector(query string, vector []float32, opts SearchOptions) (*SearchResult, error) {
	// Vectors not supported in basic Bleve wrapper
	return w.Search(query, opts)
}

func (w *bleveIndexWrapper) FindSimilar(id string, opts SearchOptions) (*SearchResult, error) {
	// Not implemented in basic wrapper
	return nil, fmt.Errorf("similar search not supported in basic mode")
}

// Management
func (w *bleveIndexWrapper) Close() error {
	return w.index.Close()
}

func (w *bleveIndexWrapper) Flush() error {
	// BleveIndex doesn't have a flush method, so we return nil
	// The underlying Bleve index handles persistence automatically
	return nil
}

func (w *bleveIndexWrapper) IsEnabled() bool {
	return true
}

func (w *bleveIndexWrapper) GetStats() (*IndexStats, error) {
	count, err := w.index.GetDocCount()
	if err != nil {
		return nil, err
	}

	return &IndexStats{
		DocumentCount:  count,
		IndexSize:      0, // TODO: Calculate index size
		LastModified:   time.Now(),
		IsHealthy:      true,
		Backend:        "bleve",
		VectorsEnabled: false,
	}, nil
}

func (w *bleveIndexWrapper) Rebuild(ctx context.Context) error {
	// For now, just return success since the index is already being built
	// The actual rebuild logic is handled by the calling code in search_index.go
	return nil
}