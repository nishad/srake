//go:build !search && !vectors
// +build !search,!vectors

package search

import "github.com/nishad/srake/internal/config"

// VectorSearchSupported reports whether this build can perform vector (KNN)
// search. False here: this build has neither full-text search nor FAISS.
const VectorSearchSupported = false

// tryCreateEnhancedBackend returns nil when enhanced backend is not available
func tryCreateEnhancedBackend(cfg *config.Config) (SearchBackend, error) {
	return nil, nil
}
