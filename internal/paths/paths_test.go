package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetPaths(t *testing.T) {
	p := GetPaths()

	if p.ConfigDir == "" {
		t.Error("ConfigDir should not be empty")
	}
	if p.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
	if p.CacheDir == "" {
		t.Error("CacheDir should not be empty")
	}
	if p.StateDir == "" {
		t.Error("StateDir should not be empty")
	}

	// All paths should contain "srake"
	if !strings.Contains(p.ConfigDir, "srake") {
		t.Errorf("ConfigDir should contain 'srake', got %q", p.ConfigDir)
	}
	if !strings.Contains(p.DataDir, "srake") {
		t.Errorf("DataDir should contain 'srake', got %q", p.DataDir)
	}
}

func TestGetPathsWithSRAKEEnv(t *testing.T) {
	t.Setenv("SRAKE_CONFIG_HOME", "/custom/config")
	t.Setenv("SRAKE_DATA_HOME", "/custom/data")
	t.Setenv("SRAKE_CACHE_HOME", "/custom/cache")
	t.Setenv("SRAKE_STATE_HOME", "/custom/state")

	p := GetPaths()

	if p.ConfigDir != "/custom/config" {
		t.Errorf("expected ConfigDir '/custom/config', got %q", p.ConfigDir)
	}
	if p.DataDir != "/custom/data" {
		t.Errorf("expected DataDir '/custom/data', got %q", p.DataDir)
	}
	if p.CacheDir != "/custom/cache" {
		t.Errorf("expected CacheDir '/custom/cache', got %q", p.CacheDir)
	}
	if p.StateDir != "/custom/state" {
		t.Errorf("expected StateDir '/custom/state', got %q", p.StateDir)
	}
}

func TestGetPathsWithXDGEnv(t *testing.T) {
	// Clear SRAKE-specific vars to test XDG fallback
	t.Setenv("SRAKE_CONFIG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")

	p := GetPaths()
	if p.ConfigDir != "/xdg/config/srake" {
		t.Errorf("expected ConfigDir '/xdg/config/srake', got %q", p.ConfigDir)
	}
}

func TestGetDatabasePath(t *testing.T) {
	path := GetDatabasePath()
	if path == "" {
		t.Error("GetDatabasePath should not return empty string")
	}
	if !strings.HasSuffix(path, "srake.db") {
		t.Errorf("expected path to end with 'srake.db', got %q", path)
	}
}

func TestGetDatabasePathWithEnv(t *testing.T) {
	t.Setenv("SRAKE_DB_PATH", "/custom/path/custom.db")
	path := GetDatabasePath()
	if path != "/custom/path/custom.db" {
		t.Errorf("expected '/custom/path/custom.db', got %q", path)
	}
}

func TestGetIndexPath(t *testing.T) {
	path := GetIndexPath()
	if path == "" {
		t.Error("GetIndexPath should not return empty string")
	}
	if !strings.HasSuffix(path, ".bleve") {
		t.Errorf("expected path to end with '.bleve', got %q", path)
	}
}

func TestGetIndexPathWithEnv(t *testing.T) {
	t.Setenv("SRAKE_INDEX_PATH", "/custom/path/custom.bleve")
	path := GetIndexPath()
	if path != "/custom/path/custom.bleve" {
		t.Errorf("expected '/custom/path/custom.bleve', got %q", path)
	}
}

func TestGetEmbeddingsPath(t *testing.T) {
	path := GetEmbeddingsPath()
	if path == "" {
		t.Error("GetEmbeddingsPath should not return empty string")
	}
	if !strings.HasSuffix(path, ".embeddings") {
		t.Errorf("expected path to end with '.embeddings', got %q", path)
	}
}

func TestGetEmbeddingsPathWithEnv(t *testing.T) {
	t.Setenv("SRAKE_EMBEDDINGS_PATH", "/custom/embeddings")
	path := GetEmbeddingsPath()
	if path != "/custom/embeddings" {
		t.Errorf("expected '/custom/embeddings', got %q", path)
	}
}

func TestGetModelsPath(t *testing.T) {
	path := GetModelsPath()
	if path == "" {
		t.Error("GetModelsPath should not return empty string")
	}
	if !strings.HasSuffix(path, "models") {
		t.Errorf("expected path to end with 'models', got %q", path)
	}
}

func TestGetModelsPathWithEnv(t *testing.T) {
	t.Setenv("SRAKE_MODELS_PATH", "/custom/models")
	path := GetModelsPath()
	if path != "/custom/models" {
		t.Errorf("expected '/custom/models', got %q", path)
	}
}

func TestGetDownloadsPath(t *testing.T) {
	path := GetDownloadsPath()
	if path == "" {
		t.Error("GetDownloadsPath should not return empty string")
	}
	if !strings.HasSuffix(path, "downloads") {
		t.Errorf("expected path to end with 'downloads', got %q", path)
	}
}

func TestGetResumePath(t *testing.T) {
	path := GetResumePath()
	if path == "" {
		t.Error("GetResumePath should not return empty string")
	}
	if !strings.HasSuffix(path, "resume") {
		t.Errorf("expected path to end with 'resume', got %q", path)
	}
}

func TestEnsureDirectories(t *testing.T) {
	// Use temp directory to avoid polluting the filesystem
	dir := t.TempDir()

	t.Setenv("SRAKE_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("SRAKE_DATA_HOME", filepath.Join(dir, "data"))
	t.Setenv("SRAKE_CACHE_HOME", filepath.Join(dir, "cache"))
	t.Setenv("SRAKE_STATE_HOME", filepath.Join(dir, "state"))

	err := EnsureDirectories()
	if err != nil {
		t.Fatalf("EnsureDirectories failed: %v", err)
	}

	// Verify key directories were created
	expectedDirs := []string{
		filepath.Join(dir, "config"),
		filepath.Join(dir, "data"),
		filepath.Join(dir, "data", "models"),
		filepath.Join(dir, "cache"),
		filepath.Join(dir, "cache", "downloads"),
		filepath.Join(dir, "state"),
		filepath.Join(dir, "state", "resume"),
	}

	for _, d := range expectedDirs {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Errorf("expected directory %q to be created", d)
		}
	}
}

func TestIndexPathAdjacentToDatabase(t *testing.T) {
	t.Setenv("SRAKE_INDEX_PATH", "")
	t.Setenv("SRAKE_DB_PATH", "/data/myproject/custom.db")

	path := GetIndexPath()
	expected := "/data/myproject/custom.bleve"
	if path != expected {
		t.Errorf("expected index path %q, got %q", expected, path)
	}
}

func TestEmbeddingsPathAdjacentToDatabase(t *testing.T) {
	t.Setenv("SRAKE_EMBEDDINGS_PATH", "")
	t.Setenv("SRAKE_DB_PATH", "/data/myproject/custom.db")

	path := GetEmbeddingsPath()
	expected := "/data/myproject/custom.embeddings"
	if path != expected {
		t.Errorf("expected embeddings path %q, got %q", expected, path)
	}
}
