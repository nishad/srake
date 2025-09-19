package search

import (
	"fmt"
	"time"

	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/paths"
)

// CreateSearchBackend creates the appropriate search backend based on build tags and configuration
func CreateSearchBackend(cfg *config.Config) (SearchBackend, error) {
	if !cfg.IsSearchEnabled() {
		return nil, fmt.Errorf("search is not enabled in configuration")
	}

	// Open database connection
	db, err := database.Initialize(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Check if tiered backend is requested
	if cfg.Search.Backend == "tiered" {
		tieredCfg := &TieredConfig{
			IndexStudies:     true,
			IndexExperiments: true,
			UseEmbeddings:    cfg.IsVectorEnabled(),
			StudyBatchSize:   1000,
			ExpBatchSize:     5000,
			IdleTimeout:      5 * time.Minute,
			CacheTTL:         10 * time.Minute,
			MaxSearchResults: cfg.Search.DefaultLimit,
			IndexPath:        paths.GetIndexPath(),
			EmbeddingsPath:   paths.GetEmbeddingsPath(),
		}
		return NewTieredSearchBackend(db, tieredCfg)
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