package search

import (
	"fmt"
	"log"
	"time"

	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/embeddings"
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
			IndexExperiments: false,
			// Only generate/store embeddings when this build can actually use
			// them for vector search — either Bleve KNN ("vectors"/FAISS) or the
			// zvec nearest-neighbor index ("zvec").
			UseEmbeddings:    cfg.IsVectorEnabled() && (VectorSearchSupported || VectorIndexSupported),
			StudyBatchSize:   1000,
			ExpBatchSize:     5000,
			IdleTimeout:      5 * time.Minute,
			CacheTTL:         10 * time.Minute,
			MaxSearchResults: cfg.Search.DefaultLimit,
			IndexPath:        paths.GetIndexPath(),
			EmbeddingsPath:   paths.GetEmbeddingsPath(),
		}

		backend, err := NewTieredSearchBackend(db, tieredCfg)
		if err != nil {
			return nil, err
		}

		// Initialize embedder only if vector search is enabled and supported by
		// this build (FAISS Bleve KNN or the zvec index). Otherwise the embedder
		// is unused dead weight.
		if cfg.IsVectorEnabled() && (VectorSearchSupported || VectorIndexSupported) {
			embedderConfig := embeddings.DefaultEmbedderConfig()
			embedderConfig.ModelsDir = paths.GetModelsPath()

			embedder, err := embeddings.NewEmbedder(embedderConfig)
			if err != nil {
				log.Printf("[FACTORY] Warning: failed to create embedder: %v", err)
				// Continue without embeddings
			} else {
				// Load the default model
				if err := embedder.LoadDefaultModel(); err != nil {
					log.Printf("[FACTORY] Warning: failed to load default embedding model: %v", err)
					// Continue without embeddings
				} else {
					// Backend is already the correct type (*TieredSearchBackend)
					backend.SetEmbedder(embedder)
					log.Printf("[FACTORY] Successfully initialized embedder for tiered backend")
				}
			}
		}

		return backend, nil
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
