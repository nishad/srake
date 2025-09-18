package embeddings

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	ort "github.com/yalue/onnxruntime_go"
)

// ONNXEmbedder generates embeddings using ONNX Runtime
type ONNXEmbedder struct {
	session   *ort.DynamicAdvancedSession
	tokenizer *BertTokenizer
	modelPath string
	enabled   bool
}

// NewONNXEmbedder creates a new ONNX embedder
func NewONNXEmbedder(modelPath string, cacheDir string) (*ONNXEmbedder, error) {
	embedder := &ONNXEmbedder{
		modelPath: modelPath,
	}

	// Initialize ONNX Runtime
	// Set the library path for macOS
	if runtime.GOOS == "darwin" {
		ort.SetSharedLibraryPath("/opt/homebrew/lib/libonnxruntime.dylib")
	}
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Download model if needed
	localModelPath, err := embedder.downloadModel(modelPath, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to download model: %w", err)
	}

	// Load the model
	session, err := ort.NewDynamicAdvancedSession(localModelPath, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONNX session: %w", err)
	}
	embedder.session = session

	// Load tokenizer
	tokenizerPath := filepath.Join(filepath.Dir(localModelPath), "tokenizer.json")
	tokenizer, err := NewBertTokenizer(tokenizerPath)
	if err != nil {
		// Try to download tokenizer if not found
		tokenizerURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/tokenizer.json", modelPath)
		if err := embedder.downloadFile(tokenizerURL, tokenizerPath); err != nil {
			return nil, fmt.Errorf("failed to download tokenizer: %w", err)
		}
		tokenizer, err = NewBertTokenizer(tokenizerPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load tokenizer: %w", err)
		}
	}
	embedder.tokenizer = tokenizer
	embedder.enabled = true

	return embedder, nil
}

// downloadModel downloads the ONNX model from HuggingFace if not cached
func (e *ONNXEmbedder) downloadModel(modelPath string, cacheDir string) (string, error) {
	// Expand cache directory
	if strings.HasPrefix(cacheDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cacheDir = filepath.Join(home, cacheDir[2:])
	}

	// Create cache directory
	modelDir := filepath.Join(cacheDir, strings.Replace(modelPath, "/", "_", -1))
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return "", err
	}

	// Check if model already exists
	onnxPath := filepath.Join(modelDir, "model.onnx")
	if _, err := os.Stat(onnxPath); err == nil {
		fmt.Printf("Using cached model: %s\n", onnxPath)
		return onnxPath, nil
	}

	// Download ONNX model from HuggingFace
	modelURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/onnx/model.onnx", modelPath)
	fmt.Printf("Downloading model from %s...\n", modelURL)

	if err := e.downloadFile(modelURL, onnxPath); err != nil {
		// Try alternative path
		modelURL = fmt.Sprintf("https://huggingface.co/%s/resolve/main/model.onnx", modelPath)
		if err := e.downloadFile(modelURL, onnxPath); err != nil {
			return "", fmt.Errorf("failed to download model: %w", err)
		}
	}

	return onnxPath, nil
}

// downloadFile downloads a file from URL to destination
func (e *ONNXEmbedder) downloadFile(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Embed generates an embedding for a single text
func (e *ONNXEmbedder) Embed(text string) ([]float32, error) {
	if !e.enabled {
		return nil, fmt.Errorf("embedder is not enabled")
	}

	// Tokenize the text
	tokens := e.tokenizer.Encode(text, 512) // Max sequence length

	// Prepare inputs
	inputIDs := make([]int64, len(tokens))
	attentionMask := make([]int64, len(tokens))
	for i, token := range tokens {
		inputIDs[i] = int64(token)
		attentionMask[i] = 1
	}

	// Create tensors
	inputShape := ort.NewShape(1, int64(len(tokens)))
	inputIDsTensor, err := ort.NewTensor[int64](inputShape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	maskTensor, err := ort.NewTensor[int64](inputShape, attentionMask)
	if err != nil {
		return nil, fmt.Errorf("failed to create mask tensor: %w", err)
	}
	defer maskTensor.Destroy()

	// Prepare outputs (will be allocated by Run)
	outputs := []ort.Value{nil} // Model has 1 output

	// Run inference
	err = e.session.Run(
		[]ort.Value{inputIDsTensor, maskTensor},
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

	// Mean pooling over sequence length
	seqLen := len(tokens)
	embDim := len(embeddings) / seqLen
	result := make([]float32, embDim)

	for i := 0; i < embDim; i++ {
		sum := float32(0)
		for j := 0; j < seqLen; j++ {
			sum += embeddings[j*embDim+i]
		}
		result[i] = sum / float32(seqLen)
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