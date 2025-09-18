package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/paths"
	"github.com/nishad/srake/internal/search"
	"github.com/spf13/cobra"
)

var searchIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Manage search index",
	Long: `Build, rebuild, or manage the Bleve search index for fast full-text search.

The search index enables powerful search capabilities including:
  • Full-text search across all metadata
  • Fuzzy search for typo tolerance
  • Faceted search and filtering
  • Fast response times even for large datasets`,
	Example: `  # Build or rebuild the search index
  srake search index --build

  # Build index with custom batch size
  srake search index --build --batch-size 1000

  # Show index statistics
  srake search index --stats

  # Verify index integrity
  srake search index --verify`,
}

var (
	indexBuild     bool
	indexRebuild   bool
	indexVerify    bool
	indexStats     bool
	indexBatchSize int
	indexWorkers   int
	indexPath      string
)

func init() {
	searchIndexCmd.Flags().BoolVar(&indexBuild, "build", false, "Build search index from database")
	searchIndexCmd.Flags().BoolVar(&indexRebuild, "rebuild", false, "Rebuild index from scratch")
	searchIndexCmd.Flags().BoolVar(&indexVerify, "verify", false, "Verify index integrity")
	searchIndexCmd.Flags().BoolVar(&indexStats, "stats", false, "Show index statistics")
	searchIndexCmd.Flags().IntVar(&indexBatchSize, "batch-size", 500, "Batch size for indexing")
	searchIndexCmd.Flags().IntVar(&indexWorkers, "workers", 0, "Number of workers (0 = auto)")
	searchIndexCmd.Flags().StringVar(&indexPath, "path", "", "Custom index path")

	searchIndexCmd.RunE = runSearchIndex

	// Add as subcommand to search
	searchCmd.AddCommand(searchIndexCmd)
}

func runSearchIndex(cmd *cobra.Command, args []string) error {
	// Determine action
	if !indexBuild && !indexRebuild && !indexVerify && !indexStats {
		indexStats = true // Default to showing stats
	}

	// Setup configuration
	cfg := config.DefaultConfig()
	cfg.DataDirectory = paths.GetPaths().DataDir

	if indexPath != "" {
		cfg.Search.IndexPath = indexPath
	} else {
		cfg.Search.IndexPath = paths.GetIndexPath()
	}

	cfg.Search.Enabled = true
	cfg.Search.BatchSize = indexBatchSize

	// Open database
	dbPath := paths.GetDatabasePath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database not found at %s\nPlease run 'srake ingest' first", dbPath)
	}

	sqlDB, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer sqlDB.Close()

	db := &database.DB{DB: sqlDB}

	// Handle different actions
	if indexStats {
		return showIndexStats(cfg, db)
	}

	if indexVerify {
		return verifyIndex(cfg, db)
	}

	if indexBuild || indexRebuild {
		return buildIndex(cfg, db, indexRebuild)
	}

	return nil
}

func buildIndex(cfg *config.Config, db *database.DB, rebuild bool) error {
	// Check if index exists
	indexExists := false
	if _, err := os.Stat(cfg.Search.IndexPath); err == nil {
		indexExists = true
	}

	if indexExists && !rebuild {
		printInfo("Index already exists at %s", cfg.Search.IndexPath)
		fmt.Println("\nUse --rebuild to rebuild from scratch")
		return nil
	}

	// Create search manager
	manager, err := search.NewManager(cfg, db)
	if err != nil {
		return fmt.Errorf("failed to create search manager: %v", err)
	}
	defer manager.Close()

	// Create syncer
	syncer, err := search.NewSyncer(cfg, db, manager.GetBackend())
	if err != nil {
		return fmt.Errorf("failed to create syncer: %v", err)
	}

	// Show progress
	if !quiet {
		if rebuild {
			printInfo("Rebuilding search index...")
		} else {
			printInfo("Building search index...")
		}
		fmt.Printf("Index path: %s\n", cfg.Search.IndexPath)
		fmt.Printf("Batch size: %d\n", cfg.Search.BatchSize)
	}

	// Perform full sync
	ctx := context.Background()
	startTime := time.Now()

	// Use optimized syncer if workers specified
	if indexWorkers > 0 {
		optimizedSyncer := search.NewOptimizedSyncer(syncer)
		err = optimizedSyncer.ParallelFullSync(ctx)
	} else {
		err = syncer.FullSync(ctx)
	}

	if err != nil {
		return fmt.Errorf("indexing failed: %v", err)
	}

	elapsed := time.Since(startTime)

	// Show completion message
	stats, _ := manager.GetStats()
	if stats != nil {
		printSuccess("Index built successfully!")
		fmt.Printf("Indexed %d documents in %v\n", stats.DocumentCount, elapsed)
		fmt.Printf("Index size: %.2f MB\n", float64(stats.IndexSize)/(1024*1024))
	}

	return nil
}

func showIndexStats(cfg *config.Config, db *database.DB) error {
	// Check if index exists
	if _, err := os.Stat(cfg.Search.IndexPath); os.IsNotExist(err) {
		printError("Search index not found at %s", cfg.Search.IndexPath)
		fmt.Println("\nBuild the index first with:")
		fmt.Println("  srake search index --build")
		return nil
	}

	// Create search manager
	manager, err := search.NewManager(cfg, db)
	if err != nil {
		return fmt.Errorf("failed to create search manager: %v", err)
	}
	defer manager.Close()

	// Get statistics
	stats, err := manager.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get statistics: %v", err)
	}

	// Display statistics
	fmt.Println(colorize(colorBold, "Search Index Statistics"))
	fmt.Println(strings.Repeat("─", 50))

	fmt.Printf("Index Path:      %s\n", cfg.Search.IndexPath)
	fmt.Printf("Backend:         %s\n", stats.Backend)
	fmt.Printf("Document Count:  %d\n", stats.DocumentCount)
	fmt.Printf("Index Size:      %.2f MB\n", float64(stats.IndexSize)/(1024*1024))
	fmt.Printf("Last Modified:   %s\n", stats.LastModified.Format(time.RFC3339))
	fmt.Printf("Vectors Enabled: %v\n", stats.VectorsEnabled)
	fmt.Printf("Index Healthy:   %v\n", stats.IsHealthy)

	// Get database counts for comparison
	var studyCount, experimentCount, sampleCount, runCount int
	db.DB.QueryRow("SELECT COUNT(*) FROM studies").Scan(&studyCount)
	db.DB.QueryRow("SELECT COUNT(*) FROM experiments").Scan(&experimentCount)
	db.DB.QueryRow("SELECT COUNT(*) FROM samples").Scan(&sampleCount)
	db.DB.QueryRow("SELECT COUNT(*) FROM runs").Scan(&runCount)

	totalDBDocs := studyCount + experimentCount + sampleCount + runCount

	fmt.Println("\n" + colorize(colorBold, "Database Comparison"))
	fmt.Printf("Studies:     %d\n", studyCount)
	fmt.Printf("Experiments: %d\n", experimentCount)
	fmt.Printf("Samples:     %d\n", sampleCount)
	fmt.Printf("Runs:        %d\n", runCount)
	fmt.Printf("Total:       %d\n", totalDBDocs)

	if totalDBDocs > 0 {
		coverage := float64(stats.DocumentCount) / float64(totalDBDocs) * 100
		fmt.Printf("\nIndex Coverage: %.1f%%\n", coverage)

		if coverage < 90 {
			printInfo("Index may be incomplete. Consider rebuilding with:")
			fmt.Println("  srake search index --rebuild")
		}
	}

	return nil
}

func verifyIndex(cfg *config.Config, db *database.DB) error {
	// Check if index exists
	if _, err := os.Stat(cfg.Search.IndexPath); os.IsNotExist(err) {
		printError("Search index not found at %s", cfg.Search.IndexPath)
		return fmt.Errorf("index not found")
	}

	printInfo("Verifying search index...")

	// Open index
	idx, err := search.InitBleveIndex(cfg.Search.IndexPath)
	if err != nil {
		return fmt.Errorf("failed to open index: %v", err)
	}
	defer idx.Close()

	// Test search functionality
	testQueries := []string{
		"human",
		"RNA-seq",
		"ILLUMINA",
		"cancer",
	}

	errors := 0
	for _, query := range testQueries {
		results, err := idx.Search(query, 10)
		if err != nil {
			printError("Search failed for '%s': %v", query, err)
			errors++
			continue
		}
		if verbose {
			fmt.Printf("Query '%s': %d results\n", query, results.Total)
		}
	}

	// Test document retrieval
	docCount, err := idx.GetDocCount()
	if err != nil {
		printError("Failed to get document count: %v", err)
		errors++
	} else {
		fmt.Printf("Document count: %d\n", docCount)
	}

	if errors > 0 {
		printError("Index verification failed with %d errors", errors)
		return fmt.Errorf("verification failed")
	}

	printSuccess("Index verification passed!")
	return nil
}

