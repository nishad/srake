package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the SRAKE configuration
type Config struct {
	DataDirectory string          `yaml:"data_directory"`
	Search        SearchConfig    `yaml:"search"`
	Bleve         BleveConfig     `yaml:"bleve"`
	SQLite        SQLiteConfig    `yaml:"sqlite"`
	Vectors       VectorConfig    `yaml:"vectors"`
	Embeddings    EmbeddingConfig `yaml:"embeddings"`
}

// SearchConfig contains search-related settings
type SearchConfig struct {
	DefaultLimit int  `yaml:"default_limit"`
	UseCache     bool `yaml:"use_cache"`
	CacheTTL     int  `yaml:"cache_ttl"` // seconds
}

// BleveConfig contains Bleve settings
type BleveConfig struct {
	Backend   string `yaml:"backend"` // scorch (default)
	BatchSize int    `yaml:"batch_size"`
}

// SQLiteConfig contains SQLite settings
type SQLiteConfig struct {
	CacheSize   int    `yaml:"cache_size"`   // in KB
	MMapSize    int64  `yaml:"mmap_size"`    // in bytes
	JournalMode string `yaml:"journal_mode"` // WAL
}

// VectorConfig contains vector search settings
type VectorConfig struct {
	Enabled          bool   `yaml:"enabled"`
	SimilarityMetric string `yaml:"similarity_metric"` // cosine, euclidean
	UseQuantized     bool   `yaml:"use_quantized"`     // Use INT8 for speed
}

// EmbeddingConfig contains embedding settings
type EmbeddingConfig struct {
	Enabled         bool     `yaml:"enabled"`
	ModelsDirectory string   `yaml:"models_directory"`
	DefaultModel    string   `yaml:"default_model"`
	DefaultVariant  string   `yaml:"default_variant"`
	BatchSize       int      `yaml:"batch_size"`
	NumThreads      int      `yaml:"num_threads"`
	MaxTextLength   int      `yaml:"max_text_length"`
	CombineFields   []string `yaml:"combine_fields"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".srake", "data")
	modelsDir := filepath.Join(homeDir, ".srake", "models")

	return &Config{
		DataDirectory: dataDir,
		Search: SearchConfig{
			DefaultLimit: 100,
			UseCache:     true,
			CacheTTL:     3600,
		},
		Bleve: BleveConfig{
			Backend:   "scorch",
			BatchSize: 1000,
		},
		SQLite: SQLiteConfig{
			CacheSize:   10000,     // 40MB
			MMapSize:    268435456, // 256MB
			JournalMode: "WAL",
		},
		Vectors: VectorConfig{
			Enabled:          true,
			SimilarityMetric: "cosine",
			UseQuantized:     true,
		},
		Embeddings: EmbeddingConfig{
			Enabled:         false, // Off by default
			ModelsDirectory: modelsDir,
			DefaultModel:    "Xenova/SapBERT-from-PubMedBERT-fulltext",
			DefaultVariant:  "quantized",
			BatchSize:       32,
			NumThreads:      4,
			MaxTextLength:   512,
			CombineFields: []string{
				"organism",
				"library_strategy",
				"title",
				"abstract",
			},
		},
	}
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	// Start with defaults
	config := DefaultConfig()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return defaults if file doesn't exist
		return config, nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate and expand paths
	config.DataDirectory = expandPath(config.DataDirectory)
	config.Embeddings.ModelsDirectory = expandPath(config.Embeddings.ModelsDirectory)

	return config, nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	// Check environment variable first
	if path := os.Getenv("SRAKE_CONFIG"); path != "" {
		return path
	}

	// Check current directory
	if _, err := os.Stat("srake.yaml"); err == nil {
		return "srake.yaml"
	}

	// Use default location
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".srake", "config.yaml")
}

// EnsureDirectories creates necessary directories
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.DataDirectory,
		c.Embeddings.ModelsDirectory,
		filepath.Join(c.DataDirectory, "bleve"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if len(path) == 0 {
		return path
	}

	if path[0] == '~' {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[1:])
	}

	return path
}
