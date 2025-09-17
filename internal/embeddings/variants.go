package embeddings

import "fmt"

const (
	MB = 1024 * 1024
	GB = 1024 * MB
)

// ModelVariant defines a specific variant of a model
type ModelVariant struct {
	Name        string `json:"name"`        // Variant name (e.g., "quantized")
	Filename    string `json:"filename"`    // Relative path to ONNX file
	Size        int64  `json:"size"`        // Expected file size in bytes
	URL         string `json:"url"`         // Download URL
	SHA256      string `json:"sha256"`      // Expected SHA256 hash
	Default     bool   `json:"default"`     // Is this the default variant?
	Description string `json:"description"` // Human-readable description
}

// ModelRegistry contains information about available models and their variants
var ModelRegistry = map[string]ModelConfig{
	"Xenova/SapBERT-from-PubMedBERT-fulltext": {
		ID:           "Xenova/SapBERT-from-PubMedBERT-fulltext",
		Organization: "Xenova",
		Name:         "SapBERT-from-PubMedBERT-fulltext",
		Description:  "Biomedical entity embedding model trained on PubMed",
		ModelType:    "bert",
		HiddenSize:   768,
		MaxLength:    512,
		BaseURL:      "https://huggingface.co/Xenova/SapBERT-from-PubMedBERT-fulltext/resolve/main",
		Variants: []ModelVariant{
			{
				Name:        "quantized",
				Filename:    "onnx/model_quantized.onnx",
				Size:        111 * MB,
				URL:         "https://huggingface.co/Xenova/SapBERT-from-PubMedBERT-fulltext/resolve/main/onnx/model_quantized.onnx",
				Default:     true,
				Description: "Quantized model (INT8) - fastest, smallest, good accuracy",
			},
			{
				Name:        "fp16",
				Filename:    "onnx/model_fp16.onnx",
				Size:        219 * MB,
				URL:         "https://huggingface.co/Xenova/SapBERT-from-PubMedBERT-fulltext/resolve/main/onnx/model_fp16.onnx",
				Description: "Half precision (FP16) - balanced speed and accuracy",
			},
			{
				Name:        "full",
				Filename:    "onnx/model.onnx",
				Size:        438 * MB,
				URL:         "https://huggingface.co/Xenova/SapBERT-from-PubMedBERT-fulltext/resolve/main/onnx/model.onnx",
				Description: "Full precision (FP32) - highest accuracy, slowest",
			},
		},
		TokenizerFiles: []string{
			"tokenizer.json",
			"tokenizer_config.json",
			"special_tokens_map.json",
			"vocab.txt",
		},
		ConfigFiles: []string{
			"config.json",
		},
	},
	"sentence-transformers/all-MiniLM-L6-v2": {
		ID:           "sentence-transformers/all-MiniLM-L6-v2",
		Organization: "sentence-transformers",
		Name:         "all-MiniLM-L6-v2",
		Description:  "General-purpose sentence embedding model",
		ModelType:    "bert",
		HiddenSize:   384,
		MaxLength:    256,
		BaseURL:      "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main",
		Variants: []ModelVariant{
			{
				Name:        "quantized",
				Filename:    "onnx/model_quantized.onnx",
				Size:        23 * MB,
				URL:         "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model_quantized.onnx",
				Default:     true,
				Description: "Quantized model - very fast and small",
			},
			{
				Name:        "full",
				Filename:    "onnx/model.onnx",
				Size:        90 * MB,
				URL:         "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx",
				Description: "Full precision model",
			},
		},
		TokenizerFiles: []string{
			"tokenizer.json",
			"tokenizer_config.json",
			"special_tokens_map.json",
			"vocab.txt",
		},
		ConfigFiles: []string{
			"config.json",
			"sentence_bert_config.json",
		},
	},
}

// ModelConfig contains complete configuration for a model
type ModelConfig struct {
	ID             string         `json:"id"`
	Organization   string         `json:"organization"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	ModelType      string         `json:"model_type"`      // "bert", "roberta", etc.
	HiddenSize     int            `json:"hidden_size"`     // Embedding dimension
	MaxLength      int            `json:"max_length"`      // Maximum sequence length
	BaseURL        string         `json:"base_url"`        // Base URL for downloading files
	Variants       []ModelVariant `json:"variants"`        // Available model variants
	TokenizerFiles []string       `json:"tokenizer_files"` // Required tokenizer files
	ConfigFiles    []string       `json:"config_files"`    // Required config files
}

// GetModelConfig returns the configuration for a specific model
func GetModelConfig(modelID string) (*ModelConfig, error) {
	config, exists := ModelRegistry[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found in registry", modelID)
	}
	return &config, nil
}

// GetDefaultVariant returns the default variant for a model
func GetDefaultVariant(modelID string) (*ModelVariant, error) {
	config, err := GetModelConfig(modelID)
	if err != nil {
		return nil, err
	}

	for _, variant := range config.Variants {
		if variant.Default {
			return &variant, nil
		}
	}

	// If no default is set, return the first variant
	if len(config.Variants) > 0 {
		return &config.Variants[0], nil
	}

	return nil, fmt.Errorf("no variants available for model %s", modelID)
}

// GetVariant returns a specific variant for a model
func GetVariant(modelID, variantName string) (*ModelVariant, error) {
	config, err := GetModelConfig(modelID)
	if err != nil {
		return nil, err
	}

	for _, variant := range config.Variants {
		if variant.Name == variantName {
			return &variant, nil
		}
	}

	return nil, fmt.Errorf("variant %s not found for model %s", variantName, modelID)
}

// ListAvailableModels returns a list of all available models in the registry
func ListAvailableModels() []string {
	var models []string
	for modelID := range ModelRegistry {
		models = append(models, modelID)
	}
	return models
}

// FormatSize formats a size in bytes to a human-readable string
func FormatSize(bytes int64) string {
	if bytes >= GB {
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	}
	if bytes >= MB {
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	}
	if bytes >= 1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d bytes", bytes)
}
