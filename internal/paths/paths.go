package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

type Paths struct {
	ConfigDir string
	DataDir   string
	CacheDir  string
	StateDir  string
}

// GetPaths returns all base paths respecting environment variables
func GetPaths() Paths {
	return Paths{
		ConfigDir: getDir("SRAKE_CONFIG_HOME", "XDG_CONFIG_HOME", ".config", "srake"),
		DataDir:   getDir("SRAKE_DATA_HOME", "XDG_DATA_HOME", ".local/share", "srake"),
		CacheDir:  getDir("SRAKE_CACHE_HOME", "XDG_CACHE_HOME", ".cache", "srake"),
		StateDir:  getDir("SRAKE_STATE_HOME", "XDG_STATE_HOME", ".local/state", "srake"),
	}
}

func getDir(srakeEnv, xdgEnv, defaultBase, appName string) string {
	// 1. Check SRAKE-specific env
	if dir := os.Getenv(srakeEnv); dir != "" {
		return dir
	}

	// 2. Check XDG env
	if xdgBase := os.Getenv(xdgEnv); xdgBase != "" {
		return filepath.Join(xdgBase, appName)
	}

	// 3. Use default
	home, _ := os.UserHomeDir()
	return filepath.Join(home, defaultBase, appName)
}

// GetDatabasePath returns the path to the database
func GetDatabasePath() string {
	if path := os.Getenv("SRAKE_DB_PATH"); path != "" {
		return path
	}
	return filepath.Join(GetPaths().DataDir, "srake.db")
}

// GetIndexPath returns the path to the search index
// Default: adjacent to database for easy backup/migration
func GetIndexPath() string {
	if path := os.Getenv("SRAKE_INDEX_PATH"); path != "" {
		return path
	}

	// Get database path and place index adjacent to it
	dbPath := GetDatabasePath()
	dir := filepath.Dir(dbPath)
	dbName := filepath.Base(dbPath)
	dbNameNoExt := dbName[:len(dbName)-len(filepath.Ext(dbName))]

	// Return path like: /data/myproject/srake.bleve (next to srake.db)
	return filepath.Join(dir, dbNameNoExt+".bleve")
}

// GetEmbeddingsPath returns the path to the embeddings directory
// Default: adjacent to database for easy backup/migration
func GetEmbeddingsPath() string {
	if path := os.Getenv("SRAKE_EMBEDDINGS_PATH"); path != "" {
		return path
	}

	// Get database path and place embeddings adjacent to it
	dbPath := GetDatabasePath()
	dir := filepath.Dir(dbPath)
	dbName := filepath.Base(dbPath)
	dbNameNoExt := dbName[:len(dbName)-len(filepath.Ext(dbName))]

	// Return path like: /data/myproject/srake.embeddings (next to srake.db)
	return filepath.Join(dir, dbNameNoExt+".embeddings")
}

// GetModelsPath returns the path to the models directory
func GetModelsPath() string {
	if path := os.Getenv("SRAKE_MODELS_PATH"); path != "" {
		return path
	}
	return filepath.Join(GetPaths().DataDir, "models")
}

// GetDownloadsPath returns the path to the downloads directory
func GetDownloadsPath() string {
	return filepath.Join(GetPaths().CacheDir, "downloads")
}

// GetResumePath returns the path to the resume/checkpoint directory
func GetResumePath() string {
	return filepath.Join(GetPaths().StateDir, "resume")
}

// EnsureDirectories creates all necessary directories
func EnsureDirectories() error {
	paths := GetPaths()
	dirs := []string{
		paths.ConfigDir,
		paths.DataDir,
		filepath.Join(paths.DataDir, "models"),
		paths.CacheDir,
		filepath.Join(paths.CacheDir, "downloads"),
		filepath.Join(paths.CacheDir, "index"),
		filepath.Join(paths.CacheDir, "embeddings"),
		filepath.Join(paths.CacheDir, "search"),
		paths.StateDir,
		filepath.Join(paths.StateDir, "resume"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}