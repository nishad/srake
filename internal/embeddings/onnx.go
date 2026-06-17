package embeddings

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

// The onnxruntime environment is a process-wide singleton. These guard a
// single initialization across all ONNXEmbedder instances.
var (
	onnxInitOnce       sync.Once
	onnxInitErr        error
	onnxRuntimeLibPath string
)

// initONNXRuntime sets the shared library path and initializes the onnxruntime
// environment exactly once per process. Subsequent calls return the cached
// result, so creating multiple embedders is safe.
func initONNXRuntime() error {
	onnxInitOnce.Do(func() {
		onnxRuntimeLibPath = resolveONNXRuntimeLibrary()
		if onnxRuntimeLibPath != "" {
			ort.SetSharedLibraryPath(onnxRuntimeLibPath)
		}
		// If something already initialized the environment, treat it as success.
		if ort.IsInitialized() {
			return
		}
		onnxInitErr = ort.InitializeEnvironment()
	})
	return onnxInitErr
}

// Embed the tokenizer.json file
//
//go:embed assets/tokenizer.json
var embeddedTokenizer []byte

// maxSequenceLength caps tokenized input to the SapBERT/BERT positional-embedding
// limit (config.json max_position_embeddings = 512). The tokenizer config does not
// enforce a limit, so inputs must be truncated here before ONNX inference.
const maxSequenceLength = 512

// ONNXEmbedder generates embeddings using ONNX Runtime
type ONNXEmbedder struct {
	session   *ort.DynamicAdvancedSession
	tokenizer *tokenizer.Tokenizer
	modelPath string
	enabled   bool
}

// NewONNXEmbedder creates a new ONNX embedder
func NewONNXEmbedder(modelPath string, cacheDir string) (*ONNXEmbedder, error) {
	embedder := &ONNXEmbedder{
		modelPath: modelPath,
	}

	// The onnxruntime environment is a process-global singleton: it must be
	// initialized exactly once, even when several embedders are created (e.g. a
	// search query embedder and an index-build embedder in the same process).
	// Initializing it more than once errors, so guard it behind a sync.Once.
	if err := initONNXRuntime(); err != nil {
		// Missing or incompatible onnxruntime is not fatal: degrade to text/FTS5
		// search rather than aborting.
		log.Printf("Warning: could not load ONNX Runtime%s: %v", describeLibSource(onnxRuntimeLibPath), err)
		log.Printf("Semantic (vector) search is disabled. Install onnxruntime, or set the")
		log.Printf("SRAKE_ONNXRUNTIME_LIB environment variable (or pass --onnxruntime-lib)")
		log.Printf("to the shared library, e.g. SRAKE_ONNXRUNTIME_LIB=/usr/local/lib/libonnxruntime.so")
		embedder.enabled = false
		return embedder, nil
	}

	// Get model variant from environment
	modelVariant := os.Getenv("SRAKE_MODEL_VARIANT")
	if modelVariant == "" {
		modelVariant = "quantized" // Default to quantized for better compatibility
	}

	// Download model if needed
	localModelPath, err := embedder.downloadModel(modelPath, cacheDir, modelVariant)
	if err != nil {
		log.Printf("Warning: Failed to download model: %v", err)
		log.Printf("Continuing without embeddings...")
		embedder.enabled = false
		return embedder, nil
	}

	// Load the model
	inputs := []string{"input_ids", "attention_mask", "token_type_ids"}
	outputs := []string{"last_hidden_state"}
	session, err := ort.NewDynamicAdvancedSession(localModelPath, inputs, outputs, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONNX session: %w", err)
	}
	embedder.session = session

	// Load tokenizer - try embedded first, then file
	var tokenizer *tokenizer.Tokenizer
	tokenizerPath := filepath.Join(filepath.Dir(localModelPath), "tokenizer.json")

	// First try to use embedded tokenizer
	if len(embeddedTokenizer) > 0 {
		// Write embedded tokenizer to temp file (sugarme/tokenizer needs file path)
		tempFile, err := os.CreateTemp("", "tokenizer-*.json")
		if err == nil {
			defer tempFile.Close()
			if _, err := tempFile.Write(embeddedTokenizer); err == nil {
				tokenizer, err = pretrained.FromFile(tempFile.Name())
				if err == nil {
					log.Printf("Using embedded tokenizer")
				}
			}
			os.Remove(tempFile.Name()) // Clean up temp file
		}
	}

	// Fall back to downloading if embedded failed
	if tokenizer == nil {
		if _, err := os.Stat(tokenizerPath); os.IsNotExist(err) {
			tokenizerURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/tokenizer.json", modelPath)
			log.Printf("Downloading tokenizer from %s...", tokenizerURL)
			if err := embedder.downloadFile(tokenizerURL, tokenizerPath); err != nil {
				return nil, fmt.Errorf("failed to download tokenizer: %w", err)
			}
		}
		tokenizer, err = pretrained.FromFile(tokenizerPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load tokenizer: %w", err)
		}
	}

	embedder.tokenizer = tokenizer
	embedder.enabled = true

	return embedder, nil
}

// resolveONNXRuntimeLibrary returns the path to the onnxruntime shared library
// for the current platform, or "" to let onnxruntime_go use its default name
// (which the OS loader resolves via its standard search path).
//
// Resolution order:
//  1. SRAKE_ONNXRUNTIME_LIB / ONNXRUNTIME_LIB_PATH env override (any platform).
//  2. Well-known install locations for the current OS.
//  3. "" — fall back to the loader default (onnxruntime.so / onnxruntime.dll).
func resolveONNXRuntimeLibrary() string {
	// 1. Explicit override wins everywhere.
	for _, env := range []string{"SRAKE_ONNXRUNTIME_LIB", "ONNXRUNTIME_LIB_PATH"} {
		if p := os.Getenv(env); p != "" {
			return p
		}
	}

	// 2. Per-OS candidate paths.
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/opt/homebrew/lib/libonnxruntime.dylib", // Homebrew on Apple Silicon
			"/usr/local/lib/libonnxruntime.dylib",    // Homebrew on Intel / manual install
		}
	case "linux":
		candidates = []string{
			"/usr/local/lib/libonnxruntime.so", // MS release tarball (and our Docker image)
			"/usr/lib/libonnxruntime.so",
			"/usr/lib/x86_64-linux-gnu/libonnxruntime.so",
			"/usr/lib/aarch64-linux-gnu/libonnxruntime.so",
		}
	case "windows":
		candidates = []string{
			`C:\Program Files\onnxruntime\lib\onnxruntime.dll`,
			"onnxruntime.dll",
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// 3. Let onnxruntime_go use its platform default name.
	return ""
}

// describeLibSource returns a short human-readable description of where the
// onnxruntime library path came from, for use in warning messages.
func describeLibSource(libPath string) string {
	for _, env := range []string{"SRAKE_ONNXRUNTIME_LIB", "ONNXRUNTIME_LIB_PATH"} {
		if p := os.Getenv(env); p != "" {
			return fmt.Sprintf(" (from %s=%s)", env, p)
		}
	}
	if libPath != "" {
		return fmt.Sprintf(" (found at %s)", libPath)
	}
	return " (no library found; using the loader's default search path)"
}

// downloadModel downloads the ONNX model from HuggingFace if not cached
func (e *ONNXEmbedder) downloadModel(modelPath string, cacheDir string, variant string) (string, error) {
	// Expand cache directory
	if strings.HasPrefix(cacheDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cacheDir = filepath.Join(home, cacheDir[2:])
	}

	log.Printf("Looking for model in cache directory: %s", cacheDir)

	// Check multiple possible locations for the model
	// Try both underscore and slash separators
	modelDirs := []string{
		filepath.Join(cacheDir, strings.Replace(modelPath, "/", "_", -1)),
		filepath.Join(cacheDir, modelPath),
	}

	// Also check common user directories
	home, _ := os.UserHomeDir()
	if home != "" {
		modelDirs = append(modelDirs,
			filepath.Join(home, ".local/share/srake/models", strings.Replace(modelPath, "/", "_", -1)),
			filepath.Join(home, ".srake/models", strings.Replace(modelPath, "/", "_", -1)),
		)
	}

	// Determine model filename based on variant
	modelFile := "model.onnx"
	switch variant {
	case "quantized":
		modelFile = "model_quantized.onnx"
	case "int8":
		modelFile = "model_int8.onnx"
	case "fp16":
		modelFile = "model_fp16.onnx"
	case "bnb4":
		modelFile = "model_bnb4.onnx"
	}

	// Try to find existing model
	for _, modelDir := range modelDirs {
		onnxPath := filepath.Join(modelDir, "onnx", modelFile)
		log.Printf("Checking for model at: %s", onnxPath)

		if info, err := os.Stat(onnxPath); err == nil {
			// Verify size is reasonable
			if info.Size() > 50*1024*1024 { // At least 50MB
				log.Printf("Found existing %s model: %s (%.2f MB)", variant, onnxPath, float64(info.Size())/(1024*1024))
				if e.verifyONNXFile(onnxPath) {
					return onnxPath, nil
				}
			}
		}
	}

	// If not found, prepare to download
	// Use the first model dir for download
	modelDir := filepath.Join(cacheDir, strings.Replace(modelPath, "/", "_", -1))
	onnxDir := filepath.Join(modelDir, "onnx")
	if err := os.MkdirAll(onnxDir, 0755); err != nil {
		return "", err
	}

	onnxPath := filepath.Join(onnxDir, modelFile)
	log.Printf("Model not found locally, will download to: %s", onnxPath)

	// Download ONNX model from HuggingFace
	modelURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/onnx/%s", modelPath, modelFile)
	log.Printf("Embedding model not found locally")
	log.Printf("Downloading SapBERT %s model (%.0f MB) from HuggingFace...",
		strings.ToUpper(variant), getModelSize(variant))

	// Download with retry logic
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			log.Printf("Retry attempt %d/3...", attempt)
		}

		if err := e.downloadFileWithProgress(modelURL, onnxPath); err != nil {
			lastErr = err
			continue
		}

		// Verify downloaded file
		if e.verifyONNXFile(onnxPath) {
			if info, err := os.Stat(onnxPath); err == nil {
				log.Printf("Model downloaded and verified successfully (size: %.2f MB)", float64(info.Size())/(1024*1024))
			}
			return onnxPath, nil
		}

		// Download succeeded but verification failed
		os.Remove(onnxPath)
		lastErr = fmt.Errorf("downloaded file failed verification")
	}

	return "", fmt.Errorf("failed to download model after 3 attempts: %w", lastErr)
}

// downloadFile downloads a file from URL to destination (legacy, kept for tokenizer download)
func (e *ONNXEmbedder) downloadFile(url string, dest string) error {
	return e.downloadFileWithProgress(url, dest)
}

// downloadFileWithProgress downloads a file with progress reporting
func (e *ONNXEmbedder) downloadFileWithProgress(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	// Create temporary file first
	tempFile := dest + ".tmp"
	out, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer func() {
		out.Close()
		// Clean up temp file if it still exists
		os.Remove(tempFile)
	}()

	// Get content length for progress reporting
	contentLength := resp.ContentLength
	if contentLength > 0 {
		fmt.Printf("Downloading %.2f MB...\n", float64(contentLength)/(1024*1024))
	}

	// Download with progress tracking
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// Close file before rename
	out.Close()

	// Verify size if content length was provided
	if contentLength > 0 && written != contentLength {
		return fmt.Errorf("incomplete download: expected %d bytes, got %d", contentLength, written)
	}

	// Rename temp file to final destination
	if err := os.Rename(tempFile, dest); err != nil {
		return err
	}

	return nil
}

// verifyONNXFile checks if the ONNX file has a valid header
func (e *ONNXEmbedder) verifyONNXFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first 16 bytes to check for ONNX/protobuf markers
	header := make([]byte, 16)
	if _, err := file.Read(header); err != nil {
		return false
	}

	// ONNX files typically start with protobuf markers
	// Looking for common patterns in ONNX files
	// Check for "pytorch" string which appears in PyTorch-exported ONNX models
	if len(header) >= 16 {
		// Convert to string to check for readable markers
		headerStr := string(header)
		// Check for common ONNX file markers
		if strings.Contains(headerStr, "pytorch") ||
			strings.Contains(headerStr, "onnx") ||
			// Check for protobuf field markers (0x08, 0x12 are common field tags)
			(header[0] == 0x08 && (header[2] == 0x12 || header[1] == 0x12)) {
			return true
		}
	}

	return false
}

// getModelSize returns the approximate size in MB for a model variant
func getModelSize(variant string) float64 {
	switch variant {
	case "int8":
		return 110
	case "fp16":
		return 218
	case "bnb4":
		return 144
	default:
		return 436
	}
}

// Embed generates an embedding for a single text
func (e *ONNXEmbedder) Embed(text string) ([]float32, error) {
	if !e.enabled {
		return nil, fmt.Errorf("embedder is not enabled")
	}

	// Tokenize the text using sugarme tokenizer
	// The tokenizer automatically adds special tokens ([CLS] and [SEP] for BERT)
	encoding, err := e.tokenizer.EncodeSingle(text)
	if err != nil {
		return nil, fmt.Errorf("failed to encode text: %w", err)
	}

	// BERT models have a fixed positional-embedding table (max_position_embeddings,
	// 512 for SapBERT). The tokenizer config does not enforce a limit, so long
	// abstracts produce sequences > 512 tokens, which makes ONNX inference fail
	// ("broadcast an axis ... 512 by N") and can trip a slice-bounds panic in the
	// tokenizer. Cap the encoding to the model's limit before inference.
	if len(encoding.Ids) > maxSequenceLength {
		if truncated, terr := encoding.Truncate(maxSequenceLength, 0); terr == nil && truncated != nil {
			encoding = truncated
		} else {
			// Fall back to a manual slice if the tokenizer's truncate fails.
			encoding.Ids = encoding.Ids[:maxSequenceLength]
			encoding.AttentionMask = encoding.AttentionMask[:maxSequenceLength]
		}
	}

	// Get token IDs and attention mask
	tokenIDs := encoding.Ids
	attentionMask := encoding.AttentionMask

	// Convert to int64 for ONNX
	inputIDs := make([]int64, len(tokenIDs))
	maskIDs := make([]int64, len(attentionMask))
	typeIDs := make([]int64, len(tokenIDs)) // All zeros for single sequence
	for i := range tokenIDs {
		inputIDs[i] = int64(tokenIDs[i])
		maskIDs[i] = int64(attentionMask[i])
		typeIDs[i] = 0 // Single sequence, so all zeros
	}

	// Create tensors
	inputShape := ort.NewShape(1, int64(len(tokenIDs)))
	inputIDsTensor, err := ort.NewTensor[int64](inputShape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	maskTensor, err := ort.NewTensor[int64](inputShape, maskIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create mask tensor: %w", err)
	}
	defer maskTensor.Destroy()

	typeIDsTensor, err := ort.NewTensor[int64](inputShape, typeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create type IDs tensor: %w", err)
	}
	defer typeIDsTensor.Destroy()

	// Prepare outputs (will be allocated by Run)
	outputs := []ort.Value{nil} // Model has 1 output

	// Run inference with all 3 inputs
	err = e.session.Run(
		[]ort.Value{inputIDsTensor, maskTensor, typeIDsTensor},
		outputs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to run inference: %w", err)
	}
	defer outputs[0].Destroy()

	// Get output tensor
	outputTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output type")
	}

	// Extract embeddings (mean pooling)
	embeddings := outputTensor.GetData()

	// For BERT models, we typically use the [CLS] token representation
	// which is the first token's embedding
	// The output shape is [batch_size, sequence_length, hidden_size]
	// We want the [CLS] token (index 0) from the sequence

	seqLen := len(tokenIDs)
	embDim := len(embeddings) / seqLen

	// Extract [CLS] token embedding (first token)
	result := make([]float32, embDim)
	for i := 0; i < embDim; i++ {
		result[i] = embeddings[i]
	}

	return result, nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *ONNXEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))

	for i, text := range texts {
		embedding, err := e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		results[i] = embedding
	}

	return results, nil
}

// IsEnabled returns whether the embedder is enabled
func (e *ONNXEmbedder) IsEnabled() bool {
	return e.enabled
}

// Close cleans up resources
func (e *ONNXEmbedder) Close() error {
	if e.session != nil {
		e.session.Destroy()
	}
	// Tokenizer doesn't need explicit cleanup with sugarme
	return nil
}

// BertTokenizer handles BERT-style tokenization
type BertTokenizer struct {
	vocab     map[string]int
	invVocab  map[int]string
	unkToken  string
	sepToken  string
	padToken  string
	clsToken  string
	maskToken string
}

// NewBertTokenizer creates a new BERT tokenizer
func NewBertTokenizer(vocabPath string) (*BertTokenizer, error) {
	// Load vocabulary from tokenizer.json
	data, err := os.ReadFile(vocabPath)
	if err != nil {
		return nil, err
	}

	var tokenizerData map[string]interface{}
	if err := json.Unmarshal(data, &tokenizerData); err != nil {
		return nil, err
	}

	// Extract vocabulary
	vocab := make(map[string]int)
	invVocab := make(map[int]string)

	if model, ok := tokenizerData["model"].(map[string]interface{}); ok {
		if v, ok := model["vocab"].(map[string]interface{}); ok {
			for token, id := range v {
				if fid, ok := id.(float64); ok {
					vocab[token] = int(fid)
					invVocab[int(fid)] = token
				}
			}
		}
	}

	return &BertTokenizer{
		vocab:     vocab,
		invVocab:  invVocab,
		unkToken:  "[UNK]",
		sepToken:  "[SEP]",
		padToken:  "[PAD]",
		clsToken:  "[CLS]",
		maskToken: "[MASK]",
	}, nil
}

// Encode converts text to token IDs
func (t *BertTokenizer) Encode(text string, maxLength int) []int {
	// Simple tokenization (word-level + subword)
	tokens := []int{t.vocab[t.clsToken]} // Start with [CLS]

	// Lowercase and split
	words := strings.Fields(strings.ToLower(text))

	for _, word := range words {
		if len(tokens) >= maxLength-1 {
			break
		}

		// Check if word exists in vocab
		if id, ok := t.vocab[word]; ok {
			tokens = append(tokens, id)
		} else {
			// Try WordPiece tokenization
			subwords := t.wordPieceTokenize(word)
			for _, sw := range subwords {
				if len(tokens) >= maxLength-1 {
					break
				}
				tokens = append(tokens, sw)
			}
		}
	}

	// Add [SEP] token
	tokens = append(tokens, t.vocab[t.sepToken])

	// Pad if necessary
	for len(tokens) < maxLength {
		tokens = append(tokens, t.vocab[t.padToken])
	}

	return tokens
}

// wordPieceTokenize performs WordPiece tokenization
func (t *BertTokenizer) wordPieceTokenize(word string) []int {
	tokens := []int{}
	start := 0

	for start < len(word) {
		end := len(word)
		found := false

		for end > start {
			substr := word[start:end]
			if start > 0 {
				substr = "##" + substr
			}

			if id, ok := t.vocab[substr]; ok {
				tokens = append(tokens, id)
				found = true
				break
			}
			end--
		}

		if !found {
			tokens = append(tokens, t.vocab[t.unkToken])
			break
		}
		start = end
	}

	return tokens
}
