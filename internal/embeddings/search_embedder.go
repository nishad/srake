package embeddings

import (
	"fmt"
	"github.com/nishad/srake/internal/config"
)

// SearchEmbedder implements the search.EmbedderInterface
type SearchEmbedder struct {
	onnx    *ONNXEmbedder
	enabled bool
}

// NewSearchEmbedder creates an embedder for search integration
func NewSearchEmbedder(cfg *config.Config) (*SearchEmbedder, error) {
	if !cfg.Embeddings.Enabled {
		return &SearchEmbedder{enabled: false}, nil
	}

	onnx, err := NewONNXEmbedder(
		cfg.Embeddings.DefaultModel,
		cfg.Embeddings.ModelsDirectory,
	)
	if err != nil {
		// Log warning but don't fail completely
		// The ONNXEmbedder might have set enabled=false internally
		if onnx != nil && !onnx.enabled {
			return &SearchEmbedder{
				onnx:    onnx,
				enabled: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to initialize ONNX embedder: %w", err)
	}

	return &SearchEmbedder{
		onnx:    onnx,
		enabled: onnx != nil && onnx.enabled,
	}, nil
}

// Embed generates an embedding for a single text
func (s *SearchEmbedder) Embed(text string) ([]float32, error) {
	if !s.enabled || s.onnx == nil {
		return nil, fmt.Errorf("embedder is not enabled")
	}
	return s.onnx.Embed(text)
}

// EmbedBatch generates embeddings for multiple texts
func (s *SearchEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	if !s.enabled || s.onnx == nil {
		return nil, fmt.Errorf("embedder is not enabled")
	}
	return s.onnx.EmbedBatch(texts)
}

// IsEnabled returns whether the embedder is enabled
func (s *SearchEmbedder) IsEnabled() bool {
	return s.enabled
}

// Close cleans up resources
func (s *SearchEmbedder) Close() error {
	if s.onnx != nil {
		return s.onnx.Close()
	}
	return nil
}