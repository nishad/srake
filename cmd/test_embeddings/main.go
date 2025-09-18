package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/nishad/srake/internal/embeddings"
)

func main() {
	fmt.Println("Testing ONNX Embedder...")
	
	// Create cache directory
	cacheDir := filepath.Join(os.Getenv("HOME"), ".srake", "models")
	
	// Initialize embedder with SapBERT model
	embedder, err := embeddings.NewONNXEmbedder("Xenova/SapBERT-from-PubMedBERT-fulltext", cacheDir)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	defer embedder.Close()
	
	// Test texts
	texts := []string{
		"human cancer RNA sequencing",
		"mouse brain tissue analysis", 
		"protein structure prediction",
		"BRCA1 gene mutation",
		"COVID-19 spike protein",
	}
	
	fmt.Println("\nGenerating embeddings for biological texts:")
	for _, text := range texts {
		fmt.Printf("\nText: %s\n", text)
		embedding, err := embedder.Embed(text)
		if err != nil {
			log.Printf("Failed to generate embedding: %v", err)
			continue
		}
		fmt.Printf("  Embedding dimensions: %d\n", len(embedding))
		fmt.Printf("  First 5 values: [%.4f, %.4f, %.4f, %.4f, %.4f]\n", 
			embedding[0], embedding[1], embedding[2], embedding[3], embedding[4])
	}
	
	// Test batch embedding
	fmt.Println("\n\nTesting batch embeddings...")
	batchEmbeddings, err := embedder.EmbedBatch(texts)
	if err != nil {
		log.Fatalf("Failed to generate batch embeddings: %v", err)
	}
	fmt.Printf("Generated %d embeddings in batch\n", len(batchEmbeddings))
	
	// Calculate similarity between first two texts (cosine similarity)
	if len(batchEmbeddings) >= 2 {
		similarity := cosineSimilarity(batchEmbeddings[0], batchEmbeddings[1])
		fmt.Printf("\nCosine similarity between '%s' and '%s': %.4f\n",
			texts[0], texts[1], similarity)
	}
	
	fmt.Println("\n✅ ONNX embedding test completed successfully!")
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	
	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (sqrt32(normA) * sqrt32(normB))
}

func sqrt32(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}
