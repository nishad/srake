package search

// VectorIndex is an optional nearest-neighbor index over study embeddings. The
// concrete implementation is build-tag gated: the "zvec" tag provides a real
// zvec-backed index; all other builds use a no-op that reports unavailable, so
// the default build carries no native vector dependency.
type VectorIndex interface {
	// Add stores a vector under the given primary key (a study accession).
	Add(pk string, vector []float32) error
	// Search returns up to topK nearest neighbors for the query vector.
	Search(vector []float32, topK int) ([]VectorHit, error)
	// Flush persists buffered writes.
	Flush() error
	// Close releases the index.
	Close() error
}

// VectorHit is a single nearest-neighbor result.
type VectorHit struct {
	PK    string  // study accession
	Score float32 // similarity score (higher = more similar)
}

// newVectorIndex opens (or creates) the vector index at path with the given
// dimensionality (768 for SapBERT). It is implemented per build tag in
// vector_index_zvec.go ("zvec") and vector_index_nozvec.go ("!zvec"). A nil
// VectorIndex means vector search is unavailable in this build and callers
// fall back to text search.
