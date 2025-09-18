package search

import (
	"fmt"

	"github.com/nishad/srake/internal/config"
)

// CreateSearchBackend creates the appropriate search backend based on build tags and configuration
func CreateSearchBackend(cfg *config.Config) (SearchBackend, error) {
	if !cfg.IsSearchEnabled() {
		return nil, fmt.Errorf("search is not enabled in configuration")
	}

	// Try to create the enhanced backend if available
	backend, err := tryCreateEnhancedBackend(cfg)
	if err == nil && backend != nil {
		return backend, nil
	}

	// Fall back to basic Bleve index wrapped as backend
	bleveIndex, err := InitBleveIndex(cfg.Search.IndexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Bleve index: %w", err)
	}

	return &bleveIndexWrapper{index: bleveIndex}, nil
}