//go:build vectors
// +build vectors

package search

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

// Vector field mapping for builds with FAISS
func (b *BleveBackend) createVectorFieldMapping() *mapping.FieldMapping {
	fm := bleve.NewVectorFieldMapping()
	fm.Dims = 768 // SapBERT embedding dimensions
	fm.Similarity = "cosine"
	return fm
}

// AddKNN for builds with FAISS
func addKNNToRequest(req *bleve.SearchRequest, field string, vector []float32, k int, boost float64) error {
	req.AddKNN(field, vector, k, boost)
	return nil
}