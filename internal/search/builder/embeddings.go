package builder

import (
	"fmt"
	"strings"
)

// generateEmbedding generates embedding for a single text
func (b *IndexBuilder) generateEmbedding(text string) ([]float32, error) {
	if b.embedder == nil {
		return nil, nil // No embedder initialized
	}

	if text == "" {
		return nil, nil // Skip empty texts
	}

	// Truncate to max length if needed (512 tokens roughly 2000 chars)
	if len(text) > 2000 {
		text = text[:2000]
	}

	embedding, err := b.embedder.Embed(text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	return embedding, nil
}

// generateBatchEmbeddings generates embeddings for a batch of texts
func (b *IndexBuilder) generateBatchEmbeddings(texts []string) ([][]float32, error) {
	if b.embedder == nil {
		return make([][]float32, len(texts)), nil // Return empty embeddings
	}

	// Filter and truncate texts
	processedTexts := make([]string, len(texts))
	for i, text := range texts {
		if len(text) > 2000 {
			processedTexts[i] = text[:2000]
		} else {
			processedTexts[i] = text
		}
	}

	embeddings, err := b.embedder.EmbedBatch(processedTexts)
	if err != nil {
		// Return partial results or empty on error
		return make([][]float32, len(texts)), fmt.Errorf("batch embedding failed: %w", err)
	}

	return embeddings, nil
}

// combineTextFields combines multiple text fields for embedding
func combineTextFields(fields ...string) string {
	var nonEmpty []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" && field != "NULL" {
			nonEmpty = append(nonEmpty, field)
		}
	}
	return strings.Join(nonEmpty, " ")
}

// prepareStudyText prepares study text for embedding
func prepareStudyText(title, abstract string) string {
	text := combineTextFields(title, abstract)
	if text == "" {
		return ""
	}
	return text
}

// prepareExperimentText prepares experiment text for embedding
func prepareExperimentText(title, libraryStrategy, platform string) string {
	text := combineTextFields(title, libraryStrategy, platform)
	if text == "" {
		return ""
	}
	return text
}

// prepareSampleText prepares sample text for embedding
func prepareSampleText(title, organism, attributes string) string {
	text := combineTextFields(title, organism, attributes)
	if text == "" {
		return ""
	}
	return text
}

// Helper to check if embeddings are enabled
func (b *IndexBuilder) isEmbeddingEnabled() bool {
	return b.embedder != nil && b.options.WithEmbeddings
}
