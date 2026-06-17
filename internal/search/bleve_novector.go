//go:build search && !vectors
// +build search,!vectors

package search

import (
	"fmt"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

// VectorSearchSupported reports whether this build can perform vector (KNN)
// search. False here: without the "vectors" tag (FAISS) there is no working
// vector search, so embeddings are not generated or stored.
const VectorSearchSupported = false

// Vector field mapping stub for builds without FAISS
func (b *BleveBackend) createVectorFieldMapping() *mapping.FieldMapping {
	// When vectors are disabled, store embedding as non-indexed field
	fm := bleve.NewTextFieldMapping()
	fm.Store = true
	fm.IncludeInAll = false
	fm.Index = false
	return fm
}

// AddKNN stub for builds without FAISS
func addKNNToRequest(req *bleve.SearchRequest, field string, vector []float32, k int, boost float64) error {
	// Vector search not supported without FAISS
	return fmt.Errorf("vector search requires building with -tags vectors")
}
