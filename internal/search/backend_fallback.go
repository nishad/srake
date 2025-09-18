//go:build !search && !vectors
// +build !search,!vectors

package search

import "github.com/nishad/srake/internal/config"

// tryCreateEnhancedBackend returns nil when enhanced backend is not available
func tryCreateEnhancedBackend(cfg *config.Config) (SearchBackend, error) {
	return nil, nil
}