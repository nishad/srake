package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nishad/srake/internal/paths"
	"gopkg.in/yaml.v3"
)

// Config represents the SRAKE configuration
type Config struct {
	DataDirectory string          `yaml:"data_directory"`
	Database      DatabaseConfig  `yaml:"database"`  // SQLite settings
	Search        SearchConfig    `yaml:"search"`    // Optional search
	Vectors       VectorConfig    `yaml:"vectors"`   // Optional vectors
	Embeddings    EmbeddingConfig `yaml:"embeddings"`
}

// DatabaseConfig contains SQLite database settings
type DatabaseConfig struct {
	Path        string `yaml:"path"`
	CacheSize   int    `yaml:"cache_size"`   // in KB
	MMapSize    int64  `yaml:"mmap_size"`    // in bytes
	JournalMode string `yaml:"journal_mode"` // WAL
}

// SearchConfig contains search-related settings
type SearchConfig struct {
	Enabled         bool   `yaml:"enabled"`          // Enable Bleve search
	Backend         string `yaml:"backend"`          // "bleve" or "sqlite"
	IndexPath       string `yaml:"index_path"`       // Path to Bleve index
	RebuildOnStart  bool   `yaml:"rebuild_on_start"` // Rebuild index on startup
	AutoSync        bool   `yaml:"auto_sync"`        // Auto-sync with SQLite
	SyncInterval    int    `yaml:"sync_interval"`    // Sync interval in seconds
	DefaultLimit    int    `yaml:"default_limit"`    // Default result limit
	BatchSize       int    `yaml:"batch_size"`       // Indexing batch size
	UseCache        bool   `yaml:"use_cache"`        // Enable search cache
	CacheTTL        int    `yaml:"cache_ttl"`        // Cache TTL in seconds
}

// VectorConfig contains vector search settings
type VectorConfig struct {
	Enabled          bool   `yaml:"enabled"`           // Enable vector search
	RequiresSearch   bool   `yaml:"requires_search"`   // Requires search to be enabled
	SimilarityMetric string `yaml:"similarity_metric"` // cosine, dot_product, l2_norm
	UseQuantized     bool   `yaml:"use_quantized"`     // Use INT8 for speed
	Dimensions       int    `yaml:"dimensions"`        // Vector dimensions (768 for SapBERT)
	Optimization     string `yaml:"optimization"`      // memory_efficient, latency, recall
}

// EmbeddingConfig contains embedding settings
type EmbeddingConfig struct {
	Enabled         bool     `yaml:"enabled"`
	ModelsDirectory string   `yaml:"models_directory"`
	DefaultModel    string   `yaml:"default_model"`     // HuggingFace model path
	DefaultVariant  string   `yaml:"default_variant"`   // quantized, fp16, or default
	BatchSize       int      `yaml:"batch_size"`        // Batch size for embedding
	NumThreads      int      `yaml:"num_threads"`       // ONNX runtime threads
	MaxTextLength   int      `yaml:"max_text_length"`   // Max tokens
	CombineFields   []string `yaml:"combine_fields"`    // Fields to combine for embedding
	CacheEmbeddings bool     `yaml:"cache_embeddings"`  // Cache computed embeddings
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	p := paths.GetPaths()

	return &Config{
		DataDirectory: p.DataDir,
		Database: DatabaseConfig{
			Path:        paths.GetDatabasePath(),
			CacheSize:   10000,     // 40MB
			MMapSize:    268435456, // 256MB
			JournalMode: "WAL",
		},
		Search: SearchConfig{
			Enabled:        true,
			Backend:        "bleve",
			IndexPath:      paths.GetIndexPath(),
			RebuildOnStart: false,
			AutoSync:       true,
			SyncInterval:   300, // 5 minutes
			DefaultLimit:   100,
			BatchSize:      1000,
			UseCache:       true,
			CacheTTL:       3600,
		},
		Vectors: VectorConfig{
			Enabled:          true,
			RequiresSearch:   true,
			SimilarityMetric: "cosine",
			UseQuantized:     true,
			Dimensions:       768,
			Optimization:     "memory_efficient",
		},
		Embeddings: EmbeddingConfig{
			Enabled:         true,
			ModelsDirectory: paths.GetModelsPath(),
			DefaultModel:    "Xenova/SapBERT-from-PubMedBERT-fulltext",
			DefaultVariant:  "quantized",
			BatchSize:       32,
			NumThreads:      4,
			MaxTextLength:   512,
			CacheEmbeddings: true,
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
	config.Database.Path = expandPath(config.Database.Path)
	config.Search.IndexPath = expandPath(config.Search.IndexPath)
	config.Embeddings.ModelsDirectory = expandPath(config.Embeddings.ModelsDirectory)

	// Validate vector config
	if config.Vectors.Enabled && config.Vectors.RequiresSearch && !config.Search.Enabled {
		// Disable vectors if search is disabled
		config.Vectors.Enabled = false
	}

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
	p := paths.GetPaths()
	return filepath.Join(p.ConfigDir, "config.yaml")
}

// EnsureDirectories creates necessary directories
func (c *Config) EnsureDirectories() error {
	// First ensure base directories using paths package
	if err := paths.EnsureDirectories(); err != nil {
		return err
	}

	// Then ensure any custom directories from config
	dirs := []string{
		c.DataDirectory,
		filepath.Dir(c.Database.Path),
		filepath.Dir(c.Search.IndexPath),
		c.Embeddings.ModelsDirectory,
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

// IsSearchEnabled returns true if search is enabled
func (c *Config) IsSearchEnabled() bool {
	return c.Search.Enabled
}

// IsVectorEnabled returns true if vectors are enabled
func (c *Config) IsVectorEnabled() bool {
	return c.Vectors.Enabled && (!c.Vectors.RequiresSearch || c.Search.Enabled)
}

// GetOperationalMode returns the operational mode based on config
func (c *Config) GetOperationalMode() string {
	if c.IsVectorEnabled() {
		return "full" // SQLite + Bleve + Vectors
	}
	if c.IsSearchEnabled() {
		return "search" // SQLite + Bleve
	}
	return "minimal" // SQLite only
}