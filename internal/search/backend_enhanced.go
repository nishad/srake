//go:build search || vectors
// +build search vectors

package search

import "github.com/nishad/srake/internal/config"

// tryCreateEnhancedBackend attempts to create the enhanced Bleve backend
func tryCreateEnhancedBackend(cfg *config.Config) (SearchBackend, error) {
	return NewBleveBackend(cfg)
}
