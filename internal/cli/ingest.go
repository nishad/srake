package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/downloader"
	"github.com/nishad/srake/internal/paths"
	"github.com/nishad/srake/internal/processor"
	"github.com/spf13/cobra"
)

var (
	// Ingest flags
	ingestAuto       bool
	ingestDaily      bool
	ingestMonthly    bool
	ingestFile       string
	ingestList       bool
	ingestDBPath     string
	ingestForce      bool
	ingestNoProgress bool

	// Filter flags
	filterTaxonIDs      []int
	filterExcludeTaxIDs []int
	filterDateFrom      string
	filterDateTo        string
	filterOrganisms     []string
	filterPlatforms     []string
	filterStrategies    []string
	filterMinReads      int64
	filterMaxReads      int64
	filterMinBases      int64
	filterMaxBases      int64
	filterStatsOnly     bool
	filterVerbose       bool
	filterProfile       string
	skipStats           bool // Skip updating database statistics
)

// NewIngestCmd creates the ingest command
func NewIngestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest SRA metadata from NCBI or local archives",
		Long: `Ingest SRA metadata from NCBI FTP servers or local tar.gz archives.

This command streams tar.gz files directly without extracting to disk,
processes them on-the-fly, and inserts records into the database.
It's optimized for low memory usage even with large (14GB+) files.

Examples:
  # Auto-select and ingest the best file from NCBI
  srake ingest --auto

  # Ingest the latest daily update
  srake ingest --daily

  # Ingest the latest monthly full dataset
  srake ingest --monthly

  # List available files on NCBI
  srake ingest --list

  # Ingest a specific file from NCBI
  srake ingest --file NCBI_SRA_Metadata_20250915.tar.gz

  # Ingest a local archive file
  srake ingest --file /path/to/archive.tar.gz`,
		RunE: runIngest,
	}

	// Add basic flags
	cmd.Flags().BoolVar(&ingestAuto, "auto", false, "Auto-select the best file to ingest from NCBI")
	cmd.Flags().BoolVar(&ingestDaily, "daily", false, "Ingest the latest daily update from NCBI")
	cmd.Flags().BoolVar(&ingestMonthly, "monthly", false, "Ingest the latest monthly full dataset from NCBI")
	cmd.Flags().StringVar(&ingestFile, "file", "", "Ingest a specific file (local path or NCBI filename)")
	cmd.Flags().BoolVar(&ingestList, "list", false, "List available files on NCBI without ingesting")
	cmd.Flags().StringVar(&ingestDBPath, "db", "", "Database path (defaults to ~/.local/share/srake/srake.db)")
	cmd.Flags().BoolVar(&ingestForce, "force", false, "Force ingestion even if data exists")
	cmd.Flags().BoolVar(&ingestNoProgress, "no-progress", false, "Disable progress bar")

	// Add filter flags
	cmd.Flags().IntSliceVar(&filterTaxonIDs, "taxon-ids", nil, "Filter by taxonomy IDs (comma-separated, e.g., 9606,10090)")
	cmd.Flags().IntSliceVar(&filterExcludeTaxIDs, "exclude-taxon-ids", nil, "Exclude taxonomy IDs (comma-separated)")
	cmd.Flags().StringVar(&filterDateFrom, "date-from", "", "Start date for filtering (YYYY-MM-DD)")
	cmd.Flags().StringVar(&filterDateTo, "date-to", "", "End date for filtering (YYYY-MM-DD)")
	cmd.Flags().StringSliceVar(&filterOrganisms, "organisms", nil, "Filter by organism names (comma-separated)")
	cmd.Flags().StringSliceVar(&filterPlatforms, "platforms", nil, "Filter by platforms (ILLUMINA, OXFORD_NANOPORE, PACBIO_SMRT, etc.)")
	cmd.Flags().StringSliceVar(&filterStrategies, "strategies", nil, "Filter by library strategies (RNA-Seq, WGS, WES, etc.)")
	cmd.Flags().Int64Var(&filterMinReads, "min-reads", 0, "Minimum read count filter")
	cmd.Flags().Int64Var(&filterMaxReads, "max-reads", 0, "Maximum read count filter")
	cmd.Flags().Int64Var(&filterMinBases, "min-bases", 0, "Minimum base count filter")
	cmd.Flags().Int64Var(&filterMaxBases, "max-bases", 0, "Maximum base count filter")
	cmd.Flags().BoolVar(&filterStatsOnly, "stats-only", false, "Only show statistics without inserting data")
	cmd.Flags().BoolVar(&filterVerbose, "filter-verbose", false, "Show detailed filtering information")
	cmd.Flags().StringVar(&filterProfile, "filter-profile", "", "Load filter settings from YAML profile")
	cmd.Flags().BoolVar(&skipStats, "skip-stats", false, "Skip updating database statistics after ingestion")

	// Mark mutually exclusive flags
	cmd.MarkFlagsMutuallyExclusive("auto", "daily", "monthly", "file", "list")

	return cmd
}

func runIngest(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get global flags
	yes, _ := cmd.Flags().GetBool("yes")

	// Resolve database path
	if ingestDBPath == "" {
		ingestDBPath = paths.GetDatabasePath()
	}

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n📛 Ingestion interrupted, cleaning up...")
		cancel()
	}()

	// Initialize metadata manager for NCBI operations
	manager := downloader.NewMetadataManager()

	// List files if requested
	if ingestList {
		return listAvailableFiles(ctx, manager)
	}

	// Select file to ingest
	var targetFile *downloader.MetadataFile
	var err error

	switch {
	case ingestAuto:
		fmt.Println("🔍 Auto-selecting best file from NCBI...")
		targetFile, err = manager.AutoSelectFile(ctx)
		if err != nil {
			return fmt.Errorf("failed to auto-select file: %w", err)
		}

	case ingestDaily:
		fmt.Println("🔍 Finding latest daily update...")
		targetFile, err = manager.GetLatestFile(ctx, downloader.FileTypeDaily)
		if err != nil {
			return fmt.Errorf("failed to find daily file: %w", err)
		}

	case ingestMonthly:
		fmt.Println("🔍 Finding latest monthly dataset...")
		targetFile, err = manager.GetLatestFile(ctx, downloader.FileTypeMonthly)
		if err != nil {
			return fmt.Errorf("failed to find monthly file: %w", err)
		}

	case ingestFile != "":
		// Check if it's a local file first
		if _, err := os.Stat(ingestFile); err == nil {
			// Local file exists, ingest it directly
			return ingestLocalFile(ctx, ingestFile, ingestDBPath, ingestForce, ingestNoProgress, yes)
		}

		// Not a local file, try to find it on NCBI
		fmt.Printf("🔍 Looking for file on NCBI: %s\n", ingestFile)
		targetFile, err = manager.GetFileByName(ctx, ingestFile)
		if err != nil {
			return fmt.Errorf("file not found on NCBI: %w", err)
		}

	default:
		// Default to auto-select
		fmt.Println("🔍 No option specified, auto-selecting from NCBI...")
		targetFile, err = manager.AutoSelectFile(ctx)
		if err != nil {
			return fmt.Errorf("failed to auto-select file: %w", err)
		}
	}

	// Display file information
	fmt.Printf("\n📦 Selected file:\n")
	fmt.Printf("   Name: %s\n", colorBold(targetFile.Name))
	fmt.Printf("   Type: %s\n", colorize(string(targetFile.Type)))
	fmt.Printf("   Size: %s\n", colorize(downloader.FormatSize(targetFile.Size)))
	fmt.Printf("   Date: %s\n", targetFile.Date.Format("2006-01-02"))
	fmt.Printf("   URL:  %s\n", targetFile.URL)

	// Initialize database
	fmt.Printf("\n🗄️  Initializing database at %s...\n", ingestDBPath)
	db, err := database.Initialize(ingestDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Check if database already has data (unless forced)
	if !ingestForce {
		stats, _ := db.GetStats()
		if stats.TotalExperiments > 0 || stats.TotalStudies > 0 {
			fmt.Printf("\n⚠️  Database already contains data:\n")
			fmt.Printf("   Studies:     %d\n", stats.TotalStudies)
			fmt.Printf("   Experiments: %d\n", stats.TotalExperiments)
			fmt.Printf("   Samples:     %d\n", stats.TotalSamples)
			fmt.Printf("   Runs:        %d\n", stats.TotalRuns)
			fmt.Printf("\nUse --force to overwrite existing data\n")

			// Ask for confirmation (unless --yes flag is set)
			if !yes {
				fmt.Print("\nContinue anyway? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" {
					fmt.Println("Ingestion cancelled")
					return nil
				}
			} else {
				fmt.Println("\n--yes flag set, continuing without confirmation")
			}
		}
	}

	// Check if filters are specified and create appropriate processor
	if hasFilters() {
		filterOpts, err := buildFilterOptions()
		if err != nil {
			return fmt.Errorf("invalid filter options: %w", err)
		}

		// Display filter summary
		fmt.Printf("\n🔍 Applying filters:\n")
		fmt.Printf("   %s\n", filterOpts.String())

		// Create filtered processor
		filteredProcessor, err := processor.NewFilteredProcessor(db, *filterOpts)
		if err != nil {
			return fmt.Errorf("failed to create filtered processor: %w", err)
		}

		// Set up progress reporting if not disabled
		if !ingestNoProgress {
			progressBar := newProgressBar(targetFile.Size)
			filteredProcessor.SetProgressFunc(func(p processor.Progress) {
				progressBar.Update(p)
			})
			defer progressBar.Finish()
		}

		// Start ingestion
		fmt.Printf("\n🚀 Starting filtered ingestion...\n")
		fmt.Println("   This may take a while for large files.")
		fmt.Println("   Press Ctrl+C to cancel.")

		startTime := time.Now()

		// Process the URL with filters
		err = filteredProcessor.ProcessWithFilters(ctx, targetFile.URL)

		if err != nil {
			if err == context.Canceled {
				fmt.Println("\n❌ Ingestion cancelled by user")
				return nil
			}
			return fmt.Errorf("ingestion failed: %w", err)
		}

		// Display final statistics
		elapsed := time.Since(startTime)
		stats := filteredProcessor.StreamProcessor.GetStats()

		fmt.Printf("\n✅ Ingestion completed successfully!\n\n")
		fmt.Printf("📊 Statistics:\n")
		fmt.Printf("   Time elapsed:      %s\n", downloader.FormatDuration(elapsed))
		fmt.Printf("   Records processed: %v\n", stats["records_processed"])
		fmt.Printf("   Bytes processed:   %s\n", downloader.FormatSize(stats["bytes_processed"].(int64)))
		fmt.Printf("   Speed:             %.2f MB/s\n", stats["bytes_per_second"].(float64)/(1024*1024))
		fmt.Printf("   Records/second:    %.0f\n", stats["records_per_second"])

		// Display filter statistics if available
		filterStats := filteredProcessor.GetStats()
		if filterStats != nil {
			fmt.Print("\n")
			fmt.Print(filterStats.GetSummary())
		}

		// Update database statistics after successful ingestion
		if !skipStats {
			fmt.Printf("\n📈 Updating database statistics...")
			if err := db.UpdateStatistics(); err != nil {
				fmt.Printf(" ⚠️ Warning: Failed to update statistics: %v\n", err)
			} else {
				fmt.Printf(" ✓\n")
			}
		}
	} else {
		// No filters, use standard processor
		streamProcessor := processor.NewStreamProcessor(db)

		// Set up progress reporting if not disabled
		if !ingestNoProgress {
			progressBar := newProgressBar(targetFile.Size)
			streamProcessor.SetProgressFunc(func(p processor.Progress) {
				progressBar.Update(p)
			})
			defer progressBar.Finish()
		}

		// Start ingestion
		fmt.Printf("\n🚀 Starting ingestion...\n")
		fmt.Println("   This may take a while for large files.")
		fmt.Println("   Press Ctrl+C to cancel.")

		startTime := time.Now()

		// Process the URL
		err = streamProcessor.ProcessURL(ctx, targetFile.URL)

		if err != nil {
			if err == context.Canceled {
				fmt.Println("\n❌ Ingestion cancelled by user")
				return nil
			}
			return fmt.Errorf("ingestion failed: %w", err)
		}

		// Display final statistics
		elapsed := time.Since(startTime)
		stats := streamProcessor.GetStats()

		fmt.Printf("\n✅ Ingestion completed successfully!\n\n")
		fmt.Printf("📊 Statistics:\n")
		fmt.Printf("   Time elapsed:      %s\n", downloader.FormatDuration(elapsed))
		fmt.Printf("   Records processed: %v\n", stats["records_processed"])
		fmt.Printf("   Bytes processed:   %s\n", downloader.FormatSize(stats["bytes_processed"].(int64)))
		fmt.Printf("   Speed:             %.2f MB/s\n", stats["bytes_per_second"].(float64)/(1024*1024))
		fmt.Printf("   Records/second:    %.0f\n", stats["records_per_second"])
	}

	// Get database statistics
	dbStats, _ := db.GetStats()
	fmt.Printf("\n📚 Database totals:\n")
	fmt.Printf("   Studies:     %d\n", dbStats.TotalStudies)
	fmt.Printf("   Experiments: %d\n", dbStats.TotalExperiments)
	fmt.Printf("   Samples:     %d\n", dbStats.TotalSamples)
	fmt.Printf("   Runs:        %d\n", dbStats.TotalRuns)

	return nil
}

// listAvailableFiles lists available files from NCBI
func listAvailableFiles(ctx context.Context, manager *downloader.MetadataManager) error {
	fmt.Println("🔍 Fetching available files from NCBI...")

	files, err := manager.ListAvailableFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No files found")
		return nil
	}

	fmt.Printf("\n📋 Found %d files:\n\n", len(files))

	// Create table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		colorBold("Name"),
		colorBold("Type"),
		colorBold("Size"),
		colorBold("Date"),
		colorBold("Age"))

	for _, f := range files {
		age := time.Since(f.Date)
		ageStr := downloader.FormatDuration(age)

		typeColor := ""
		switch f.Type {
		case downloader.FileTypeMonthly:
			typeColor = colorGreen(string(f.Type))
		case downloader.FileTypeDaily:
			typeColor = colorBlue(string(f.Type))
		default:
			typeColor = string(f.Type)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			f.Name,
			typeColor,
			downloader.FormatSize(f.Size),
			f.Date.Format("2006-01-02"),
			ageStr+" ago")
	}

	w.Flush()

	fmt.Printf("\n💡 Tips:\n")
	fmt.Println("   • Monthly files contain the complete dataset")
	fmt.Println("   • Daily files contain incremental updates")
	fmt.Println("   • Use --auto to automatically select the best file")
	fmt.Printf("\nTo ingest a specific file:\n")
	fmt.Printf("   srake ingest --file %s\n", files[0].Name)

	return nil
}

// ingestLocalFile processes a local tar.gz file
func ingestLocalFile(ctx context.Context, filePath string, dbPath string, force bool, noProgress bool, yes bool) error {
	// Get file info
	stat, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Display file information
	fmt.Printf("\n📦 Ingesting local archive:\n")
	fmt.Printf("   Path: %s\n", colorBold(filePath))
	fmt.Printf("   Size: %s\n", colorize(downloader.FormatSize(stat.Size())))
	fmt.Printf("   Modified: %s\n", stat.ModTime().Format("2006-01-02 15:04:05"))

	// Initialize database
	fmt.Printf("\n🗄️  Initializing database at %s...\n", dbPath)
	db, err := database.Initialize(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Check if database already has data (unless forced)
	if !force {
		stats, _ := db.GetStats()
		if stats.TotalExperiments > 0 || stats.TotalStudies > 0 {
			fmt.Printf("\n⚠️  Database already contains data:\n")
			fmt.Printf("   Studies:     %d\n", stats.TotalStudies)
			fmt.Printf("   Experiments: %d\n", stats.TotalExperiments)
			fmt.Printf("   Samples:     %d\n", stats.TotalSamples)
			fmt.Printf("   Runs:        %d\n", stats.TotalRuns)
			fmt.Printf("\nUse --force to overwrite existing data\n")

			// Ask for confirmation (unless --yes flag is set)
			if !yes {
				fmt.Print("\nContinue anyway? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" {
					fmt.Println("Ingestion cancelled")
					return nil
				}
			} else {
				fmt.Println("\n--yes flag set, continuing without confirmation")
			}
		}
	}

	// Check if filters are specified and create appropriate processor
	if hasFilters() {
		filterOpts, err := buildFilterOptions()
		if err != nil {
			return fmt.Errorf("invalid filter options: %w", err)
		}

		// Display filter summary
		fmt.Printf("\n🔍 Applying filters:\n")
		fmt.Printf("   %s\n", filterOpts.String())

		// Create filtered processor
		filteredProcessor, err := processor.NewFilteredProcessor(db, *filterOpts)
		if err != nil {
			return fmt.Errorf("failed to create filtered processor: %w", err)
		}

		// Set up progress reporting if not disabled
		if !noProgress {
			progressBar := newProgressBar(stat.Size())
			filteredProcessor.SetProgressFunc(func(p processor.Progress) {
				progressBar.Update(p)
			})
			defer progressBar.Finish()
		}

		// Start ingestion
		fmt.Printf("\n🚀 Starting filtered ingestion...\n")
		fmt.Println("   This may take a while for large files.")
		fmt.Println("   Press Ctrl+C to cancel.")

		startTime := time.Now()

		// Process the local file with filters
		err = filteredProcessor.ProcessWithFilters(ctx, filePath)

		if err != nil {
			if err == context.Canceled {
				fmt.Println("\n\n❌ Ingestion cancelled")
			}
			return err
		}

		// Display completion stats
		duration := time.Since(startTime)
		stats := filteredProcessor.StreamProcessor.GetStats()

		fmt.Printf("\n\n✅ Ingestion completed successfully!\n")
		fmt.Printf("\n📊 Statistics:\n")
		fmt.Printf("   Duration:    %s\n", downloader.FormatDuration(duration))

		// Safely get statistics with nil checks
		if bytesProcessed, ok := stats["bytes_processed"].(int64); ok {
			fmt.Printf("   Processed:   %s\n", downloader.FormatSize(bytesProcessed))
			if duration.Seconds() > 0 {
				fmt.Printf("   Speed:       %.1f MB/s\n", float64(bytesProcessed)/duration.Seconds()/(1024*1024))
			}
		}
		if recordsInserted, ok := stats["records_inserted"].(int64); ok {
			fmt.Printf("   Records:     %d\n", recordsInserted)
		}
		fmt.Printf("   Database:    %s\n", dbPath)

		// Display filter statistics
		filterStats := filteredProcessor.GetStats()
		if filterStats != nil {
			fmt.Print("\n")
			fmt.Print(filterStats.GetSummary())
		}
	} else {
		// No filters, use standard processor
		streamProcessor := processor.NewStreamProcessor(db)

		// Set up progress reporting if not disabled
		if !noProgress {
			progressBar := newProgressBar(stat.Size())
			streamProcessor.SetProgressFunc(func(p processor.Progress) {
				progressBar.Update(p)
			})
			defer progressBar.Finish()
		}

		// Start ingestion
		fmt.Printf("\n🚀 Starting ingestion...\n")
		fmt.Println("   This may take a while for large files.")
		fmt.Println("   Press Ctrl+C to cancel.")

		startTime := time.Now()

		// Process the local file
		err = streamProcessor.ProcessFile(ctx, filePath)

		if err != nil {
			if err == context.Canceled {
				fmt.Println("\n\n❌ Ingestion cancelled")
			}
			return err
		}

		// Display completion stats
		duration := time.Since(startTime)
		stats := streamProcessor.GetStats()

		fmt.Printf("\n\n✅ Ingestion completed successfully!\n")
		fmt.Printf("\n📊 Statistics:\n")
		fmt.Printf("   Duration:    %s\n", downloader.FormatDuration(duration))

		// Safely get statistics with nil checks
		if bytesProcessed, ok := stats["bytes_processed"].(int64); ok {
			fmt.Printf("   Processed:   %s\n", downloader.FormatSize(bytesProcessed))
			if duration.Seconds() > 0 {
				fmt.Printf("   Speed:       %.1f MB/s\n", float64(bytesProcessed)/duration.Seconds()/(1024*1024))
			}
		}
		if recordsInserted, ok := stats["records_inserted"].(int64); ok {
			fmt.Printf("   Records:     %d\n", recordsInserted)
		}
		fmt.Printf("   Database:    %s\n", dbPath)
	}

	// Update database statistics after successful ingestion
	if !skipStats {
		fmt.Printf("\n📈 Updating database statistics...")
		if err := db.UpdateStatistics(); err != nil {
			fmt.Printf(" ⚠️ Warning: Failed to update statistics: %v\n", err)
		} else {
			fmt.Printf(" ✓\n")
		}
	}

	// Get database stats
	dbStats, _ := db.GetStats()
	fmt.Printf("\n📈 Database Contents:\n")
	fmt.Printf("   Studies:     %d\n", dbStats.TotalStudies)
	fmt.Printf("   Experiments: %d\n", dbStats.TotalExperiments)
	fmt.Printf("   Samples:     %d\n", dbStats.TotalSamples)
	fmt.Printf("   Runs:        %d\n", dbStats.TotalRuns)

	fmt.Printf("\n💡 Next steps:\n")
	fmt.Printf("   • Search records: srake search 'your query'\n")
	fmt.Printf("   • Start API server: srake server\n")
	fmt.Printf("   • View database info: srake db info\n")

	return nil
}

// progressBar handles progress display
type progressBar struct {
	totalBytes int64
	lastUpdate time.Time
	startTime  time.Time
}

func newProgressBar(total int64) *progressBar {
	return &progressBar{
		totalBytes: total,
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}
}

func (pb *progressBar) Update(p processor.Progress) {
	// Update at most once per second
	if time.Since(pb.lastUpdate) < time.Second {
		return
	}
	pb.lastUpdate = time.Now()

	// Calculate progress bar
	barWidth := 40
	filled := int(p.PercentComplete * float64(barWidth) / 100)
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Format speed
	speedMB := p.BytesPerSecond / (1024 * 1024)

	// Format remaining time
	remainingStr := "calculating..."
	if p.EstimatedTimeRemaining > 0 {
		remainingStr = downloader.FormatDuration(p.EstimatedTimeRemaining)
	}

	// Clear line and print progress
	fmt.Printf("\r[%s] %.1f%% | %s / %s | %.1f MB/s | ETA: %s | Records: %d",
		bar,
		p.PercentComplete,
		downloader.FormatSize(p.BytesProcessed),
		downloader.FormatSize(pb.totalBytes),
		speedMB,
		remainingStr,
		p.RecordsProcessed)
}

func (pb *progressBar) Finish() {
	fmt.Println() // New line after progress bar
}

// Color functions for terminal output
func colorBold(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return fmt.Sprintf("\033[1m%s\033[0m", s)
}

func colorize(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return fmt.Sprintf("\033[36m%s\033[0m", s) // Cyan
}

func colorGreen(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return fmt.Sprintf("\033[32m%s\033[0m", s)
}

func colorBlue(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return fmt.Sprintf("\033[34m%s\033[0m", s)
}

// hasFilters checks if any filters are specified
func hasFilters() bool {
	return len(filterTaxonIDs) > 0 ||
		len(filterExcludeTaxIDs) > 0 ||
		filterDateFrom != "" ||
		filterDateTo != "" ||
		len(filterOrganisms) > 0 ||
		len(filterPlatforms) > 0 ||
		len(filterStrategies) > 0 ||
		filterMinReads > 0 ||
		filterMaxReads > 0 ||
		filterMinBases > 0 ||
		filterMaxBases > 0 ||
		filterProfile != ""
}

// buildFilterOptions creates a FilterOptions struct from command-line flags
func buildFilterOptions() (*processor.FilterOptions, error) {
	opts := &processor.FilterOptions{
		TaxonomyIDs:   filterTaxonIDs,
		ExcludeTaxIDs: filterExcludeTaxIDs,
		Organisms:     filterOrganisms,
		Platforms:     filterPlatforms,
		Strategies:    filterStrategies,
		MinReads:      filterMinReads,
		MaxReads:      filterMaxReads,
		MinBases:      filterMinBases,
		MaxBases:      filterMaxBases,
		StatsOnly:     filterStatsOnly,
		Verbose:       filterVerbose,
	}

	// Parse date filters
	if filterDateFrom != "" {
		t, err := time.Parse("2006-01-02", filterDateFrom)
		if err != nil {
			return nil, fmt.Errorf("invalid date-from format: %w", err)
		}
		opts.DateFrom = t
	}

	if filterDateTo != "" {
		t, err := time.Parse("2006-01-02", filterDateTo)
		if err != nil {
			return nil, fmt.Errorf("invalid date-to format: %w", err)
		}
		opts.DateTo = t
	}

	// Load profile if specified
	if filterProfile != "" {
		// TODO: Implement YAML profile loading
		fmt.Printf("⚠️  Filter profiles not yet implemented, using command-line flags only\n")
	}

	// Validate the options
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	return opts, nil
}
