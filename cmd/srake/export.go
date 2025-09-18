package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nishad/srake/internal/export"
	"github.com/nishad/srake/internal/paths"
	"github.com/spf13/cobra"
)

// Export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export database to SRAmetadb format",
	Long: `Export the srake database to SRAmetadb.sqlite format for compatibility with tools
expecting the original SRAmetadb schema.

This command creates a SQLite database with the classic SRAmetadb schema, including:
  • Standard tables: study, experiment, sample, run, submission
  • Denormalized sra table with all fields
  • Full-text search index (FTS3 or FTS5)
  • metaInfo table with version information

The export process handles:
  • Field mapping from srake's modern schema to legacy format
  • JSON to delimited string conversion
  • Date format standardization
  • Missing field generation with appropriate defaults`,
	Example: `  # Basic export with FTS5 (recommended)
  srake db export -o SRAmetadb.sqlite

  # Export with FTS3 for 100% compatibility
  srake db export -o SRAmetadb.sqlite --fts-version 3

  # Export with compression
  srake db export -o SRAmetadb.sqlite.gz --compress

  # Large dataset with custom batch size
  srake db export -o SRAmetadb.sqlite --batch-size 50000`,
	RunE: runExport,
}

// Export flags
var (
	exportOutput     string
	exportFTSVersion int
	exportBatchSize  int
	exportProgress   bool
	exportCompress   bool
	exportForce      bool
	exportDBPath     string
)

func init() {
	// Add export as subcommand to db
	dbCmd.AddCommand(exportCmd)

	// Define flags
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "SRAmetadb.sqlite", "Output database file path")
	exportCmd.Flags().StringVar(&exportDBPath, "db", "", "Source database path (defaults to ~/.local/share/srake/srake.db)")
	exportCmd.Flags().IntVar(&exportFTSVersion, "fts-version", 5, "FTS version (3 for compatibility, 5 for modern)")
	exportCmd.Flags().IntVar(&exportBatchSize, "batch-size", 10000, "Batch size for data transfer")
	exportCmd.Flags().BoolVar(&exportProgress, "progress", true, "Show progress bar")
	exportCmd.Flags().BoolVar(&exportCompress, "compress", false, "Compress output with gzip")
	exportCmd.Flags().BoolVarP(&exportForce, "force", "f", false, "Overwrite existing output file")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Validate FTS version
	if exportFTSVersion != 3 && exportFTSVersion != 5 {
		return fmt.Errorf("invalid FTS version: %d (must be 3 or 5)", exportFTSVersion)
	}

	// Get source database path
	srcDBPath := exportDBPath
	if srcDBPath == "" {
		srcDBPath = paths.GetDatabasePath()
	}

	// Check if source database exists
	if _, err := os.Stat(srcDBPath); os.IsNotExist(err) {
		printError("Database not found at %s", srcDBPath)
		fmt.Fprintf(os.Stderr, "\nIngest the database first:\n")
		fmt.Fprintf(os.Stderr, "  srake ingest --auto\n")
		return fmt.Errorf("database not found")
	}

	// Resolve output path
	outputPath := exportOutput
	if !filepath.IsAbs(outputPath) {
		pwd, _ := os.Getwd()
		outputPath = filepath.Join(pwd, outputPath)
	}

	// Check if output exists
	if _, err := os.Stat(outputPath); err == nil && !exportForce {
		printError("Output file already exists: %s", outputPath)
		fmt.Fprintf(os.Stderr, "\nUse --force to overwrite\n")
		return fmt.Errorf("output file exists")
	}

	// Handle compression
	if exportCompress && !strings.HasSuffix(outputPath, ".gz") {
		outputPath += ".gz"
	}

	// Create exporter
	cfg := &export.Config{
		SourceDB:    srcDBPath,
		OutputPath:  outputPath,
		FTSVersion:  exportFTSVersion,
		BatchSize:   exportBatchSize,
		ShowProgress: exportProgress && !quiet,
		Compress:    exportCompress,
		Verbose:     verbose,
		Debug:       debug,
	}

	exporter, err := export.NewExporter(cfg)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}
	defer exporter.Close()

	// Show export info
	if !quiet {
		printInfo("Exporting to SRAmetadb format")
		fmt.Printf("Source:      %s\n", srcDBPath)
		fmt.Printf("Output:      %s\n", outputPath)
		fmt.Printf("FTS Version: %d\n", exportFTSVersion)
		fmt.Printf("Batch Size:  %d\n", exportBatchSize)
		if exportCompress {
			fmt.Printf("Compression: enabled\n")
		}
		fmt.Println()
	}

	// Run export
	stats, err := exporter.Export()
	if err != nil {
		// Clean up partial output
		os.Remove(outputPath)
		return fmt.Errorf("export failed: %w", err)
	}

	// Show completion stats
	if !quiet {
		fmt.Println()
		printSuccess("Export completed successfully!")
		fmt.Printf("Studies:     %d\n", stats.Studies)
		fmt.Printf("Experiments: %d\n", stats.Experiments)
		fmt.Printf("Samples:     %d\n", stats.Samples)
		fmt.Printf("Runs:        %d\n", stats.Runs)
		fmt.Printf("SRA Records: %d\n", stats.SRARecords)
		fmt.Printf("Time:        %v\n", stats.Duration)

		// Show file size
		if info, err := os.Stat(outputPath); err == nil {
			sizeMB := float64(info.Size()) / (1024 * 1024)
			fmt.Printf("File Size:   %.2f MB\n", sizeMB)
		}
	}

	return nil
}