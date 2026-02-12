package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Check database defaults
	if cfg.Database.JournalMode != "WAL" {
		t.Errorf("expected journal_mode WAL, got %q", cfg.Database.JournalMode)
	}
	if cfg.Database.CacheSize != 10000 {
		t.Errorf("expected cache_size 10000, got %d", cfg.Database.CacheSize)
	}

	// Check search defaults
	if !cfg.Search.Enabled {
		t.Error("expected search to be enabled by default")
	}
	if cfg.Search.DefaultLimit != 100 {
		t.Errorf("expected default_limit 100, got %d", cfg.Search.DefaultLimit)
	}
	if cfg.Search.BatchSize != 1000 {
		t.Errorf("expected batch_size 1000, got %d", cfg.Search.BatchSize)
	}

	// Check vector defaults
	if !cfg.Vectors.Enabled {
		t.Error("expected vectors to be enabled by default")
	}
	if cfg.Vectors.Dimensions != 768 {
		t.Errorf("expected dimensions 768, got %d", cfg.Vectors.Dimensions)
	}
	if cfg.Vectors.SimilarityMetric != "cosine" {
		t.Errorf("expected similarity_metric cosine, got %q", cfg.Vectors.SimilarityMetric)
	}

	// Check embedding defaults
	if !cfg.Embeddings.Enabled {
		t.Error("expected embeddings to be enabled by default")
	}
	if cfg.Embeddings.BatchSize != 32 {
		t.Errorf("expected batch_size 32, got %d", cfg.Embeddings.BatchSize)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("Load should return defaults for non-existent file, got error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config for non-existent file")
	}
}

func TestLoadValidFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	yamlContent := `
data_directory: /tmp/srake-test
database:
  path: /tmp/srake-test/test.db
  cache_size: 5000
  journal_mode: WAL
search:
  enabled: false
  backend: sqlite
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.DataDirectory != "/tmp/srake-test" {
		t.Errorf("expected data_directory /tmp/srake-test, got %q", cfg.DataDirectory)
	}
	if cfg.Database.CacheSize != 5000 {
		t.Errorf("expected cache_size 5000, got %d", cfg.Database.CacheSize)
	}
	if cfg.Search.Enabled {
		t.Error("expected search to be disabled")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(configPath, []byte("invalid: yaml: [broken"), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Database.CacheSize = 999
	cfg.Search.Enabled = false

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Database.CacheSize != 999 {
		t.Errorf("expected cache_size 999, got %d", loaded.Database.CacheSize)
	}
	if loaded.Search.Enabled {
		t.Error("expected search to be disabled after save/load")
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(string) bool
		desc  string
	}{
		{
			name:  "empty string",
			input: "",
			check: func(s string) bool { return s == "" },
			desc:  "should return empty string",
		},
		{
			name:  "absolute path",
			input: "/usr/local/bin",
			check: func(s string) bool { return s == "/usr/local/bin" },
			desc:  "should return unchanged",
		},
		{
			name:  "tilde expansion",
			input: "~/Documents",
			check: func(s string) bool { return s != "~/Documents" && len(s) > 0 },
			desc:  "should expand tilde",
		},
		{
			name:  "relative path",
			input: "relative/path",
			check: func(s string) bool { return s == "relative/path" },
			desc:  "should return unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if !tt.check(result) {
				t.Errorf("expandPath(%q) = %q, %s", tt.input, result, tt.desc)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	// Test with environment variable
	t.Setenv("SRAKE_CONFIG", "/custom/config.yaml")
	path := GetConfigPath()
	if path != "/custom/config.yaml" {
		t.Errorf("expected /custom/config.yaml, got %q", path)
	}
}

func TestIsSearchEnabled(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.IsSearchEnabled() {
		t.Error("expected search to be enabled by default")
	}

	cfg.Search.Enabled = false
	if cfg.IsSearchEnabled() {
		t.Error("expected search to be disabled")
	}
}

func TestIsVectorEnabled(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.IsVectorEnabled() {
		t.Error("expected vectors to be enabled by default")
	}

	// Vectors disabled when search is disabled and RequiresSearch is true
	cfg.Search.Enabled = false
	cfg.Vectors.RequiresSearch = true
	if cfg.IsVectorEnabled() {
		t.Error("expected vectors to be disabled when search is disabled and vectors require search")
	}

	// Vectors enabled even without search if RequiresSearch is false
	cfg.Vectors.RequiresSearch = false
	if !cfg.IsVectorEnabled() {
		t.Error("expected vectors to be enabled when RequiresSearch is false")
	}
}

func TestGetOperationalMode(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(*Config)
		expected string
	}{
		{
			name:     "full mode with vectors",
			modify:   func(c *Config) {},
			expected: "full",
		},
		{
			name: "search mode without vectors",
			modify: func(c *Config) {
				c.Vectors.Enabled = false
			},
			expected: "search",
		},
		{
			name: "minimal mode",
			modify: func(c *Config) {
				c.Vectors.Enabled = false
				c.Search.Enabled = false
			},
			expected: "minimal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			mode := cfg.GetOperationalMode()
			if mode != tt.expected {
				t.Errorf("expected mode %q, got %q", tt.expected, mode)
			}
		})
	}
}

func TestVectorDisabledWhenSearchDisabled(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	yamlContent := `
search:
  enabled: false
vectors:
  enabled: true
  requires_search: true
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Vectors.Enabled {
		t.Error("vectors should be disabled when search is disabled and requires_search is true")
	}
}

func TestEnsureDirectories(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.DataDirectory = filepath.Join(dir, "data")
	cfg.Database.Path = filepath.Join(dir, "data", "test.db")
	cfg.Search.IndexPath = filepath.Join(dir, "data", "test.bleve")
	cfg.Embeddings.ModelsDirectory = filepath.Join(dir, "data", "models")

	err := cfg.EnsureDirectories()
	if err != nil {
		t.Fatalf("EnsureDirectories failed: %v", err)
	}

	// Verify directories were created
	if _, err := os.Stat(cfg.DataDirectory); os.IsNotExist(err) {
		t.Error("data directory was not created")
	}
}

func TestGetSearchBackend(t *testing.T) {
	// Default should be "tiered"
	result := getSearchBackend()
	if result != "tiered" {
		t.Errorf("expected default backend 'tiered', got %q", result)
	}

	// Test environment variable override
	t.Setenv("SRAKE_SEARCH_BACKEND", "bleve")
	result = getSearchBackend()
	if result != "bleve" {
		t.Errorf("expected backend 'bleve' from env, got %q", result)
	}
}
