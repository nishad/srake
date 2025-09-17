package embeddings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ModelInfo represents metadata about an installed model
type ModelInfo struct {
	ID            string                 `json:"id"`             // e.g., "Xenova/SapBERT-from-PubMedBERT-fulltext"
	Organization  string                 `json:"organization"`   // e.g., "Xenova"
	Name          string                 `json:"name"`           // e.g., "SapBERT-from-PubMedBERT-fulltext"
	Path          string                 `json:"path"`           // Full path to model directory
	Variants      []VariantInfo          `json:"variants"`       // Available variants
	ActiveVariant string                 `json:"active_variant"` // Currently active variant
	Config        map[string]interface{} `json:"config"`         // Model configuration from config.json
	InstalledAt   time.Time              `json:"installed_at"`
	LastUsed      time.Time              `json:"last_used"`
}

// VariantInfo describes a specific model variant
type VariantInfo struct {
	Name       string    `json:"name"`       // e.g., "quantized", "fp16", "full"
	Filename   string    `json:"filename"`   // e.g., "onnx/model_quantized.onnx"
	Size       int64     `json:"size"`       // File size in bytes
	Downloaded bool      `json:"downloaded"` // Whether this variant is downloaded
	Path       string    `json:"path"`       // Full path to the ONNX file
	CreatedAt  time.Time `json:"created_at"`
}

// Manager handles model storage and management following HuggingFace conventions
type Manager struct {
	modelsDir string
	models    map[string]*ModelInfo
	mu        sync.RWMutex
}

// NewManager creates a new model manager
func NewManager(modelsDir string) (*Manager, error) {
	// Expand home directory if needed
	if strings.HasPrefix(modelsDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		modelsDir = filepath.Join(home, modelsDir[2:])
	}

	// Create models directory if it doesn't exist
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create models directory: %w", err)
	}

	m := &Manager{
		modelsDir: modelsDir,
		models:    make(map[string]*ModelInfo),
	}

	// Scan for existing models
	if err := m.scanModels(); err != nil {
		return nil, fmt.Errorf("failed to scan models: %w", err)
	}

	return m, nil
}

// GetModelsDir returns the models directory path
func (m *Manager) GetModelsDir() string {
	return m.modelsDir
}

// scanModels discovers installed models in the models directory
func (m *Manager) scanModels() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Walk through the models directory looking for model directories
	err := filepath.Walk(m.modelsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}

		// Look for config.json files which indicate a model directory
		if info.Name() == "config.json" {
			modelPath := filepath.Dir(path)
			if modelInfo := m.loadModelInfo(modelPath); modelInfo != nil {
				m.models[modelInfo.ID] = modelInfo
			}
		}
		return nil
	})

	return err
}

// loadModelInfo loads model information from a directory
func (m *Manager) loadModelInfo(modelPath string) *ModelInfo {
	// Read model_info.json if it exists (our metadata file)
	infoPath := filepath.Join(modelPath, "model_info.json")
	var modelInfo ModelInfo

	if data, err := ioutil.ReadFile(infoPath); err == nil {
		if err := json.Unmarshal(data, &modelInfo); err == nil {
			return &modelInfo
		}
	}

	// Otherwise, construct from directory structure
	relPath, _ := filepath.Rel(m.modelsDir, modelPath)
	parts := strings.Split(relPath, string(filepath.Separator))

	if len(parts) >= 2 {
		org := parts[0]
		name := strings.Join(parts[1:], "/")
		modelID := org + "/" + name

		modelInfo = ModelInfo{
			ID:           modelID,
			Organization: org,
			Name:         name,
			Path:         modelPath,
			InstalledAt:  time.Now(),
		}

		// Load config.json if it exists
		configPath := filepath.Join(modelPath, "config.json")
		if data, err := ioutil.ReadFile(configPath); err == nil {
			var config map[string]interface{}
			if err := json.Unmarshal(data, &config); err == nil {
				modelInfo.Config = config
			}
		}

		// Scan for ONNX variants
		modelInfo.Variants = m.scanVariants(modelPath)

		// Set default active variant
		for _, variant := range modelInfo.Variants {
			if variant.Downloaded {
				if modelInfo.ActiveVariant == "" || variant.Name == "quantized" {
					modelInfo.ActiveVariant = variant.Name
				}
			}
		}

		return &modelInfo
	}

	return nil
}

// scanVariants looks for ONNX model files in the model directory
func (m *Manager) scanVariants(modelPath string) []VariantInfo {
	var variants []VariantInfo

	// Check standard ONNX directory
	onnxDir := filepath.Join(modelPath, "onnx")

	// Define expected variants for SapBERT model
	expectedVariants := []struct {
		name     string
		filename string
	}{
		{"quantized", "model_quantized.onnx"},
		{"fp16", "model_fp16.onnx"},
		{"full", "model.onnx"},
	}

	for _, expected := range expectedVariants {
		variantPath := filepath.Join(onnxDir, expected.filename)
		variant := VariantInfo{
			Name:     expected.name,
			Filename: filepath.Join("onnx", expected.filename),
			Path:     variantPath,
		}

		if info, err := os.Stat(variantPath); err == nil {
			variant.Downloaded = true
			variant.Size = info.Size()
			variant.CreatedAt = info.ModTime()
		}

		variants = append(variants, variant)
	}

	// Also check for any other .onnx files
	filepath.Walk(modelPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(path, ".onnx") {
			relPath, _ := filepath.Rel(modelPath, path)

			// Check if we already have this variant
			found := false
			for _, v := range variants {
				if v.Filename == relPath {
					found = true
					break
				}
			}

			if !found {
				name := strings.TrimSuffix(filepath.Base(path), ".onnx")
				variants = append(variants, VariantInfo{
					Name:       name,
					Filename:   relPath,
					Path:       path,
					Downloaded: true,
					Size:       info.Size(),
					CreatedAt:  info.ModTime(),
				})
			}
		}
		return nil
	})

	return variants
}

// ListModels returns all installed models
func (m *Manager) ListModels() ([]*ModelInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var models []*ModelInfo
	for _, model := range m.models {
		models = append(models, model)
	}
	return models, nil
}

// GetModel returns information about a specific model
func (m *Manager) GetModel(modelID string) (*ModelInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	model, exists := m.models[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}
	return model, nil
}

// GetModelPath returns the path to a model's directory
func (m *Manager) GetModelPath(modelID string) string {
	parts := strings.Split(modelID, "/")
	return filepath.Join(append([]string{m.modelsDir}, parts...)...)
}

// SetActiveVariant sets the active variant for a model
func (m *Manager) SetActiveVariant(modelID, variantName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	model, exists := m.models[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	// Check if variant exists and is downloaded
	variantFound := false
	for _, variant := range model.Variants {
		if variant.Name == variantName {
			if !variant.Downloaded {
				return fmt.Errorf("variant %s is not downloaded", variantName)
			}
			variantFound = true
			break
		}
	}

	if !variantFound {
		return fmt.Errorf("variant %s not found for model %s", variantName, modelID)
	}

	model.ActiveVariant = variantName
	model.LastUsed = time.Now()

	// Save updated model info
	return m.saveModelInfo(model)
}

// saveModelInfo saves model metadata to disk
func (m *Manager) saveModelInfo(model *ModelInfo) error {
	infoPath := filepath.Join(model.Path, "model_info.json")
	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal model info: %w", err)
	}

	if err := ioutil.WriteFile(infoPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write model info: %w", err)
	}

	return nil
}

// RegisterModel registers a newly downloaded model
func (m *Manager) RegisterModel(modelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	modelPath := m.GetModelPath(modelID)
	modelInfo := m.loadModelInfo(modelPath)
	if modelInfo == nil {
		return fmt.Errorf("failed to load model info from %s", modelPath)
	}

	m.models[modelID] = modelInfo
	return m.saveModelInfo(modelInfo)
}

// RemoveModel removes a model from the manager (but doesn't delete files)
func (m *Manager) RemoveModel(modelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.models[modelID]; !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	delete(m.models, modelID)
	return nil
}

// GetActiveVariantPath returns the path to the active variant's ONNX file
func (m *Manager) GetActiveVariantPath(modelID string) (string, error) {
	model, err := m.GetModel(modelID)
	if err != nil {
		return "", err
	}

	if model.ActiveVariant == "" {
		return "", fmt.Errorf("no active variant set for model %s", modelID)
	}

	for _, variant := range model.Variants {
		if variant.Name == model.ActiveVariant && variant.Downloaded {
			return variant.Path, nil
		}
	}

	return "", fmt.Errorf("active variant %s not found or not downloaded", model.ActiveVariant)
}

// RefreshModels rescans the models directory
func (m *Manager) RefreshModels() error {
	m.mu.Lock()
	m.models = make(map[string]*ModelInfo)
	m.mu.Unlock()

	return m.scanModels()
}
