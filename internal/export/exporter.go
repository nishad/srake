package export

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nishad/srake/internal/database"
)

// Config holds the export configuration
type Config struct {
	SourceDB     string
	OutputPath   string
	FTSVersion   int
	BatchSize    int
	ShowProgress bool
	Compress     bool
	Verbose      bool
	Debug        bool
}

// Stats holds export statistics
type Stats struct {
	Studies     int
	Experiments int
	Samples     int
	Runs        int
	Submissions int
	SRARecords  int
	Duration    time.Duration
}

// Exporter handles the export process
type Exporter struct {
	cfg      *Config
	sourceDB *database.DB
	targetDB *sql.DB
	stats    *Stats
	writer   io.Writer
	file     *os.File
	gzWriter *gzip.Writer
}

// NewExporter creates a new exporter instance
func NewExporter(cfg *Config) (*Exporter, error) {
	// Open source database with minimal setup for read-only access
	// Using simple connection without heavy pragmas for large databases
	sourceConn, err := sql.Open("sqlite3", cfg.SourceDB+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open source database: %w", err)
	}

	// Wrap in database struct for compatibility
	sourceDB := &database.DB{
		DB: sourceConn,
	}

	// Create output directory if needed
	outputDir := filepath.Dir(cfg.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		sourceDB.Close()
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create temporary database file
	tempPath := cfg.OutputPath + ".tmp"

	// Remove any existing temp file
	os.Remove(tempPath)

	// Open target database
	targetDB, err := sql.Open("sqlite3", tempPath)
	if err != nil {
		sourceDB.Close()
		return nil, fmt.Errorf("failed to create target database: %w", err)
	}

	// Set pragmas for performance
	pragmas := []string{
		"PRAGMA journal_mode = OFF",
		"PRAGMA synchronous = OFF",
		"PRAGMA cache_size = 100000",
		"PRAGMA locking_mode = EXCLUSIVE",
		"PRAGMA temp_store = MEMORY",
	}

	for _, pragma := range pragmas {
		if _, err := targetDB.Exec(pragma); err != nil {
			targetDB.Close()
			sourceDB.Close()
			os.Remove(tempPath)
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	return &Exporter{
		cfg:      cfg,
		sourceDB: sourceDB,
		targetDB: targetDB,
		stats:    &Stats{},
	}, nil
}

// Close cleans up resources
func (e *Exporter) Close() {
	if e.targetDB != nil {
		e.targetDB.Close()
	}
	if e.sourceDB != nil {
		e.sourceDB.Close()
	}
	if e.gzWriter != nil {
		e.gzWriter.Close()
	}
	if e.file != nil {
		e.file.Close()
	}
}

// Export performs the export process
func (e *Exporter) Export() (*Stats, error) {
	startTime := time.Now()

	// Create schema
	if err := e.createSchema(); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Export tables in order
	if err := e.exportSubmissions(); err != nil {
		return nil, fmt.Errorf("failed to export submissions: %w", err)
	}

	if err := e.exportStudies(); err != nil {
		return nil, fmt.Errorf("failed to export studies: %w", err)
	}

	if err := e.exportSamples(); err != nil {
		return nil, fmt.Errorf("failed to export samples: %w", err)
	}

	if err := e.exportExperiments(); err != nil {
		return nil, fmt.Errorf("failed to export experiments: %w", err)
	}

	if err := e.exportRuns(); err != nil {
		return nil, fmt.Errorf("failed to export runs: %w", err)
	}

	// Create denormalized sra table
	if err := e.createSRATable(); err != nil {
		return nil, fmt.Errorf("failed to create sra table: %w", err)
	}

	// Create FTS index
	if err := e.createFTSIndex(); err != nil {
		return nil, fmt.Errorf("failed to create FTS index: %w", err)
	}

	// Create metaInfo table
	if err := e.createMetaInfo(); err != nil {
		return nil, fmt.Errorf("failed to create metaInfo: %w", err)
	}

	// Create col_desc table
	if err := e.createColDesc(); err != nil {
		return nil, fmt.Errorf("failed to create col_desc: %w", err)
	}

	// Close database before moving/compressing
	e.targetDB.Close()
	e.targetDB = nil

	// Move or compress the output file
	tempPath := e.cfg.OutputPath + ".tmp"
	if e.cfg.Compress {
		if err := e.compressDatabase(tempPath, e.cfg.OutputPath); err != nil {
			return nil, fmt.Errorf("failed to compress database: %w", err)
		}
		os.Remove(tempPath)
	} else {
		if err := os.Rename(tempPath, e.cfg.OutputPath); err != nil {
			return nil, fmt.Errorf("failed to move database: %w", err)
		}
	}

	e.stats.Duration = time.Since(startTime)
	return e.stats, nil
}

// compressDatabase compresses the database file
func (e *Exporter) compressDatabase(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	gzWriter.Name = filepath.Base(strings.TrimSuffix(dst, ".gz"))
	gzWriter.ModTime = time.Now()

	_, err = io.Copy(gzWriter, srcFile)
	return err
}

// Helper function to handle NULL strings
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// Helper function to handle NULL int64
func nullInt64(i int64) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: i, Valid: true}
}

// Helper function to handle NULL float64
func nullFloat64(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}

// Helper function to format time for SRAdb
func formatTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}