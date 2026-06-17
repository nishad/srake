//go:build !zvec
// +build !zvec

package search

// VectorIndexSupported reports whether this build includes a nearest-neighbor
// vector index (the "zvec" build tag). False here.
const VectorIndexSupported = false

// newVectorIndex is a no-op in builds without the "zvec" tag: it returns a nil
// index, signalling that vector search is unavailable and callers should fall
// back to text search. No native vector dependency is imported.
func newVectorIndex(path string, dim int) (VectorIndex, error) {
	return nil, nil
}
