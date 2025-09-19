package embeddings

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Embedder is the main service for generating embeddings
type Embedder struct {
	manager   *Manager
	model     *Model
	tokenizer *Tokenizer
	modelID   string
	config    *EmbedderConfig
	mu        sync.RWMutex
}

// EmbedderConfig contains configuration for the embedder
type EmbedderConfig struct {
	ModelsDir    string `yaml:"models_dir"`
	DefaultModel string `yaml:"default_model"`
	BatchSize    int    `yaml:"batch_size"`
	MaxLength    int    `yaml:"max_length"`
	NumThreads   int    `yaml:"num_threads"`
	CacheEnabled bool   `yaml:"cache_enabled"`
}

// DefaultEmbedderConfig returns the default configuration
func DefaultEmbedderConfig() *EmbedderConfig {
	return &EmbedderConfig{
		ModelsDir:    "~/.sra-tool/models",
		DefaultModel: "Xenova/SapBERT-from-PubMedBERT-fulltext",
		BatchSize:    32,
		MaxLength:    512,
		NumThreads:   4,
		CacheEnabled: true,
	}
}

// NewEmbedder creates a new embedder service
func NewEmbedder(config *EmbedderConfig) (*Embedder, error) {
	if config == nil {
		config = DefaultEmbedderConfig()
	}

	// Create model manager
	manager, err := NewManager(config.ModelsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create model manager: %w", err)
	}

	return &Embedder{
		manager: manager,
		config:  config,
	}, nil
}

// LoadModel loads a specific model for embedding
func (e *Embedder) LoadModel(modelID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Close existing model if any
	if e.model != nil {
		e.model.Close()
		e.model = nil
	}

	// Try to get model info from manager first
	var modelPath string
	var variantName string
	var tokenizerPath string

	modelInfo, err := e.manager.GetModel(modelID)
	if err != nil {
		// If manager doesn't have the model, try direct paths
		// This supports both managed models and direct file paths
		modelPath = e.findModelDirectly(modelID)
		if modelPath == "" {
			return fmt.Errorf("model %s not found: %w", modelID, err)
		}
		variantName = e.getVariantFromEnv()
		tokenizerPath = e.findTokenizerPath(modelID)
	} else {
		// Use manager's model info
		variantPath, err := e.manager.GetActiveVariantPath(modelID)
		if err != nil {
			// Try direct path as fallback
			modelPath = e.findModelDirectly(modelID)
			if modelPath == "" {
				return fmt.Errorf("failed to get active variant: %w", err)
			}
			variantName = e.getVariantFromEnv()
		} else {
			modelPath = variantPath
			variantName = modelInfo.ActiveVariant
		}
		tokenizerPath = filepath.Join(modelInfo.Path, "tokenizer.json")
	}

	// Get model config
	config, err := GetModelConfig(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model config: %w", err)
	}

	// Load ONNX model
	model, err := LoadModel(modelPath, config, variantName)
	if err != nil {
		return fmt.Errorf("failed to load ONNX model from %s: %w", modelPath, err)
	}

	// Load tokenizer - try different locations
	var tokenizer *Tokenizer
	if tokenizerPath != "" && fileExists(tokenizerPath) {
		tokenizer, err = LoadTokenizer(filepath.Dir(tokenizerPath))
	}
	if err != nil || tokenizer == nil {
		// Use embedded tokenizer as fallback
		tokenizer, err = LoadEmbeddedTokenizer()
	}
	if err != nil || tokenizer == nil {
		model.Close()
		return fmt.Errorf("failed to load tokenizer: %w", err)
	}

	e.model = model
	e.tokenizer = tokenizer
	e.modelID = modelID

	return nil
}

// findModelDirectly searches for model files in standard locations
func (e *Embedder) findModelDirectly(modelID string) string {
	// Get variant from environment or use default
	variant := e.getVariantFromEnv()

	// Define possible model file names based on variant
	modelFiles := []string{
		fmt.Sprintf("model_%s.onnx", variant),
		"model.onnx",
	}
	if variant == "quantized" {
		modelFiles = []string{"model_quantized.onnx", "model.onnx"}
	}

	// Convert modelID to underscore format as alternative
	modelIDUnderscore := strings.Replace(modelID, "/", "_", -1)

	// Search paths to check (include both slash and underscore formats)
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".local/share/srake/models", modelID, "onnx"),
		filepath.Join(os.Getenv("HOME"), ".local/share/srake/models", modelIDUnderscore, "onnx"),
		filepath.Join(os.Getenv("HOME"), ".srake/models", modelID, "onnx"),
		filepath.Join(os.Getenv("HOME"), ".srake/models", modelIDUnderscore, "onnx"),
		filepath.Join(e.config.ModelsDir, modelID, "onnx"),
		filepath.Join(e.config.ModelsDir, modelIDUnderscore, "onnx"),
	}

	// Check each combination
	for _, basePath := range paths {
		for _, modelFile := range modelFiles {
			fullPath := filepath.Join(basePath, modelFile)
			if fileExists(fullPath) {
				return fullPath
			}
		}
	}

	return ""
}

// findTokenizerPath searches for tokenizer files in standard locations
func (e *Embedder) findTokenizerPath(modelID string) string {
	// Convert modelID to underscore format as alternative
	modelIDUnderscore := strings.Replace(modelID, "/", "_", -1)

	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".local/share/srake/models", modelID, "tokenizer.json"),
		filepath.Join(os.Getenv("HOME"), ".local/share/srake/models", modelIDUnderscore, "tokenizer.json"),
		filepath.Join(os.Getenv("HOME"), ".srake/models", modelID, "tokenizer.json"),
		filepath.Join(os.Getenv("HOME"), ".srake/models", modelIDUnderscore, "tokenizer.json"),
		filepath.Join(e.config.ModelsDir, modelID, "tokenizer.json"),
		filepath.Join(e.config.ModelsDir, modelIDUnderscore, "tokenizer.json"),
	}

	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}

	return ""
}

// getVariantFromEnv gets the model variant from environment or returns default
func (e *Embedder) getVariantFromEnv() string {
	variant := os.Getenv("SRAKE_MODEL_VARIANT")
	if variant == "" {
		variant = "quantized" // Default to quantized for better compatibility
	}
	return variant
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// LoadDefaultModel loads the default model
func (e *Embedder) LoadDefaultModel() error {
	return e.LoadModel(e.config.DefaultModel)
}

// IsModelLoaded checks if a model is currently loaded
func (e *Embedder) IsModelLoaded() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.model != nil && e.tokenizer != nil
}

// GetLoadedModel returns the ID of the currently loaded model
func (e *Embedder) GetLoadedModel() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.modelID
}

// EmbedText generates an embedding for a text string
func (e *Embedder) EmbedText(text string) ([]float32, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.model == nil || e.tokenizer == nil {
		return nil, fmt.Errorf("no model loaded")
	}

	// Tokenize the text
	encoding, err := e.tokenizer.Encode(text, e.config.MaxLength)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize: %w", err)
	}

	// Generate embedding
	embedding, err := e.model.EmbedSingle(encoding.InputIDs, encoding.AttentionMask)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	return embedding, nil
}

// EmbedTexts generates embeddings for multiple texts
func (e *Embedder) EmbedTexts(texts []string) ([][]float32, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.model == nil || e.tokenizer == nil {
		return nil, fmt.Errorf("no model loaded")
	}

	// Process in batches
	var allEmbeddings [][]float32
	batchSize := e.config.BatchSize

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		batchEmbeddings, err := e.embedBatch(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to embed batch %d-%d: %w", i, end, err)
		}

		allEmbeddings = append(allEmbeddings, batchEmbeddings...)
	}

	return allEmbeddings, nil
}

// embedBatch generates embeddings for a batch of texts
func (e *Embedder) embedBatch(texts []string) ([][]float32, error) {
	// Tokenize all texts
	var inputIDs [][]int64
	var attentionMasks [][]int64

	for _, text := range texts {
		encoding, err := e.tokenizer.Encode(text, e.config.MaxLength)
		if err != nil {
			return nil, fmt.Errorf("failed to tokenize: %w", err)
		}
		inputIDs = append(inputIDs, encoding.InputIDs)
		attentionMasks = append(attentionMasks, encoding.AttentionMask)
	}

	// Pad sequences to same length
	maxLen := 0
	for _, ids := range inputIDs {
		if len(ids) > maxLen {
			maxLen = len(ids)
		}
	}

	// Pad all sequences
	for i := range inputIDs {
		for len(inputIDs[i]) < maxLen {
			inputIDs[i] = append(inputIDs[i], 0) // PAD token ID
			attentionMasks[i] = append(attentionMasks[i], 0)
		}
	}

	// Generate embeddings
	return e.model.Embed(inputIDs, attentionMasks)
}

// PrepareTextForEmbedding prepares SRA metadata for embedding
func PrepareTextForEmbedding(organism, libraryStrategy, title, abstract string) string {
	// Combine relevant fields with some structure
	parts := []string{}

	if organism != "" {
		parts = append(parts, "Organism: "+organism)
	}
	if libraryStrategy != "" {
		parts = append(parts, "Method: "+libraryStrategy)
	}
	if title != "" {
		parts = append(parts, title)
	}
	if abstract != "" {
		// Truncate abstract if too long
		if len(abstract) > 200 {
			abstract = abstract[:200] + "..."
		}
		parts = append(parts, abstract)
	}

	return strings.Join(parts, " ")
}

// Close releases all resources
func (e *Embedder) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.model != nil {
		e.model.Close()
		e.model = nil
	}

	return nil
}

// GetManager returns the model manager
func (e *Embedder) GetManager() *Manager {
	return e.manager
}

// DownloadModel downloads a model using the embedded downloader
func (e *Embedder) DownloadModel(modelID string, variantName string, progress chan<- DownloadProgress) error {
	downloader := NewDownloader(e.manager, progress)
	return downloader.DownloadModel(modelID, variantName)
}

// TestModel tests a model by generating an embedding for sample text
func (e *Embedder) TestModel(modelID string, text string) ([]float32, error) {
	// Save current model
	currentModel := e.modelID

	// Load test model
	if err := e.LoadModel(modelID); err != nil {
		return nil, err
	}

	// Generate embedding
	embedding, err := e.EmbedText(text)

	// Restore previous model if different
	if currentModel != "" && currentModel != modelID {
		e.LoadModel(currentModel)
	}

	return embedding, err
}

// ComputeSimilarity computes cosine similarity between two embeddings
func ComputeSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("embeddings have different dimensions: %d vs %d", len(a), len(b))
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, nil
	}

	// Cosine similarity
	similarity := dotProduct / (sqrt(normA) * sqrt(normB))
	return similarity, nil
}

// sqrt computes square root of float32
func sqrt(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

// NormalizeEmbedding normalizes an embedding to unit length
func NormalizeEmbedding(embedding []float32) []float32 {
	var norm float32
	for _, val := range embedding {
		norm += val * val
	}

	if norm == 0 {
		return embedding
	}

	norm = sqrt(norm)
	normalized := make([]float32, len(embedding))
	for i, val := range embedding {
		normalized[i] = val / norm
	}

	return normalized
}
