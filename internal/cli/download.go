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
	"github.com/nishad/srake/internal/processor"
	"github.com/spf13/cobra"
)

var (
	// Download flags
	downloadAuto       bool
	downloadDaily      bool
	downloadMonthly    bool
	downloadFile       string
	downloadList       bool
	downloadDBPath     string
	downloadForce      bool
	downloadNoProgress bool
)

// NewDownloadCmd creates the download command
func NewDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download and process SRA metadata from NCBI",
		Long: `Download and stream SRA metadata directly from NCBI without extracting to disk.

This command streams tar.gz files from NCBI FTP, processes them on-the-fly,
and inserts records into the database. It's optimized for low memory usage
even with large (14GB+) files.

Examples:
  # Auto-select and download the best file
  srake download --auto

  # Download the latest daily update
  srake download --daily

  # Download the latest monthly full dataset
  srake download --monthly

  # List available files
  srake download --list

  # Download a specific file
  srake download --file NCBI_SRA_Metadata_20250915.tar.gz`,
		RunE: runDownload,
	}

	// Add flags
	cmd.Flags().BoolVar(&downloadAuto, "auto", false, "Auto-select the best file to download")
	cmd.Flags().BoolVar(&downloadDaily, "daily", false, "Download the latest daily update")
	cmd.Flags().BoolVar(&downloadMonthly, "monthly", false, "Download the latest monthly full dataset")
	cmd.Flags().StringVar(&downloadFile, "file", "", "Download a specific file by name")
	cmd.Flags().BoolVar(&downloadList, "list", false, "List available files without downloading")
	cmd.Flags().StringVar(&downloadDBPath, "db", "./data/metadata.db", "Database path")
	cmd.Flags().BoolVar(&downloadForce, "force", false, "Force download even if data exists")
	cmd.Flags().BoolVar(&downloadNoProgress, "no-progress", false, "Disable progress bar")

	// Mark mutually exclusive flags
	cmd.MarkFlagsMutuallyExclusive("auto", "daily", "monthly", "file", "list")

	return cmd
}

func runDownload(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n📛 Download interrupted, cleaning up...")
		cancel()
	}()

	// Initialize metadata manager
	manager := downloader.NewMetadataManager()

	// List files if requested
	if downloadList {
		return listFiles(ctx, manager)
	}

	// Select file to download
	var targetFile *downloader.MetadataFile
	var err error

	switch {
	case downloadAuto:
		fmt.Println("🔍 Auto-selecting best file...")
		targetFile, err = manager.AutoSelectFile(ctx)
		if err != nil {
			return fmt.Errorf("failed to auto-select file: %w", err)
		}

	case downloadDaily:
		fmt.Println("🔍 Finding latest daily update...")
		targetFile, err = manager.GetLatestFile(ctx, downloader.FileTypeDaily)
		if err != nil {
			return fmt.Errorf("failed to find daily file: %w", err)
		}

	case downloadMonthly:
		fmt.Println("🔍 Finding latest monthly dataset...")
		targetFile, err = manager.GetLatestFile(ctx, downloader.FileTypeMonthly)
		if err != nil {
			return fmt.Errorf("failed to find monthly file: %w", err)
		}

	case downloadFile != "":
		fmt.Printf("🔍 Looking for file: %s\n", downloadFile)
		targetFile, err = manager.GetFileByName(ctx, downloadFile)
		if err != nil {
			return fmt.Errorf("file not found: %w", err)
		}

	default:
		// Default to auto-select
		fmt.Println("🔍 No option specified, auto-selecting...")
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
	fmt.Printf("\n🗄️  Initializing database at %s...\n", downloadDBPath)
	db, err := database.Initialize(downloadDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Check if database already has data (unless forced)
	if !downloadForce {
		stats, _ := db.GetStats()
		if stats.TotalExperiments > 0 || stats.TotalStudies > 0 {
			fmt.Printf("\n⚠️  Database already contains data:\n")
			fmt.Printf("   Studies:     %d\n", stats.TotalStudies)
			fmt.Printf("   Experiments: %d\n", stats.TotalExperiments)
			fmt.Printf("   Samples:     %d\n", stats.TotalSamples)
			fmt.Printf("   Runs:        %d\n", stats.TotalRuns)
			fmt.Printf("\nUse --force to overwrite existing data\n")

			// Ask for confirmation
			fmt.Print("\nContinue anyway? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println("Download cancelled")
				return nil
			}
		}
	}

	// Create stream processor
	streamProcessor := processor.NewStreamProcessor(db)

	// Set up progress reporting if not disabled
	if !downloadNoProgress {
		progressBar := newProgressBar(targetFile.Size)
		streamProcessor.SetProgressFunc(func(p processor.Progress) {
			progressBar.Update(p)
		})
		defer progressBar.Finish()
	}

	// Start processing
	fmt.Printf("\n🚀 Starting download and processing...\n")
	fmt.Println("   This may take a while for large files.")
	fmt.Println("   Press Ctrl+C to cancel.\n")

	startTime := time.Now()

	// Process the URL
	err = streamProcessor.ProcessURL(ctx, targetFile.URL)
	if err != nil {
		if err == context.Canceled {
			fmt.Println("\n❌ Download cancelled by user")
			return nil
		}
		return fmt.Errorf("processing failed: %w", err)
	}

	// Display final statistics
	elapsed := time.Since(startTime)
	stats := streamProcessor.GetStats()

	fmt.Printf("\n✅ Download completed successfully!\n\n")
	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("   Time elapsed:    %s\n", downloader.FormatDuration(elapsed))
	fmt.Printf("   Records processed: %v\n", stats["records_processed"])
	fmt.Printf("   Bytes processed:   %s\n", downloader.FormatSize(stats["bytes_processed"].(int64)))
	fmt.Printf("   Speed:            %.2f MB/s\n", stats["bytes_per_second"].(float64)/(1024*1024))
	fmt.Printf("   Records/second:   %.0f\n", stats["records_per_second"])

	// Get database statistics
	dbStats, _ := db.GetStats()
	fmt.Printf("\n📚 Database totals:\n")
	fmt.Printf("   Studies:     %d\n", dbStats.TotalStudies)
	fmt.Printf("   Experiments: %d\n", dbStats.TotalExperiments)
	fmt.Printf("   Samples:     %d\n", dbStats.TotalSamples)
	fmt.Printf("   Runs:        %d\n", dbStats.TotalRuns)

	return nil
}

// listFiles lists available files from NCBI
func listFiles(ctx context.Context, manager *downloader.MetadataManager) error {
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
	fmt.Printf("\nTo download a specific file:\n")
	fmt.Printf("   srake download --file %s\n", files[0].Name)

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
