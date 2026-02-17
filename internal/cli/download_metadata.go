package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/nishad/srake/internal/downloader"
	"github.com/spf13/cobra"
)

// NewDownloadMetadataCmd creates the download metadata subcommand
func NewDownloadMetadataCmd() *cobra.Command {
	var (
		list          bool
		daily         bool
		monthly       bool
		auto          bool
		accessionsTab bool
		runMembersTab bool
		file          string
		output        string
		resume        bool
		noProgress    bool
	)

	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "Download NCBI SRA metadata dump files",
		Long: `Download bulk metadata files from NCBI SRA FTP servers.

Available file types:
  - Daily metadata dumps (~6GB tar.gz) — daily XML metadata snapshots
  - Monthly full dumps (~15GB tar.gz) — comprehensive monthly archives
  - SRA_Accessions.tab (~29GB) — master accession table
  - SRA_Run_Members.tab (~3.6GB) — run-to-experiment/sample/study mapping

Downloads support resume: if interrupted (Ctrl+C), a .part file is left in place.
Re-run with --resume to continue from where you left off.`,
		Example: `  # List all available metadata files
  srake download metadata --list

  # Download latest daily metadata dump
  srake download metadata --daily

  # Download SRA_Run_Members.tab to /tmp
  srake download metadata --run-members-tab --output /tmp

  # Resume a partially downloaded file
  srake download metadata --accessions-tab --output /data --resume

  # Download a specific file by name
  srake download metadata --file NCBI_SRA_Metadata_20250915.tar.gz`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDownloadMetadata(cmd, list, daily, monthly, auto, accessionsTab, runMembersTab, file, output, resume, noProgress)
		},
	}

	cmd.Flags().BoolVar(&list, "list", false, "List all available metadata files")
	cmd.Flags().BoolVar(&daily, "daily", false, "Download the latest daily metadata dump")
	cmd.Flags().BoolVar(&monthly, "monthly", false, "Download the latest monthly full dump")
	cmd.Flags().BoolVar(&auto, "auto", false, "Auto-select the best tar.gz file (default behavior)")
	cmd.Flags().BoolVar(&accessionsTab, "accessions-tab", false, "Download SRA_Accessions.tab (~29GB)")
	cmd.Flags().BoolVar(&runMembersTab, "run-members-tab", false, "Download SRA_Run_Members.tab (~3.6GB)")
	cmd.Flags().StringVar(&file, "file", "", "Download a specific file by name")
	cmd.Flags().StringVarP(&output, "output", "o", ".", "Destination directory")
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume a partial download")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress bar")

	cmd.MarkFlagsMutuallyExclusive("list", "daily", "monthly", "auto", "accessions-tab", "run-members-tab", "file")

	return cmd
}

func runDownloadMetadata(cmd *cobra.Command, list, daily, monthly, auto, accessionsTab, runMembersTab bool, file, output string, resume, noProgress bool) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals — leave .part file for resume
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintf(os.Stderr, "\nDownload interrupted. Partial .part file preserved for resume.\n")
		fmt.Fprintf(os.Stderr, "Re-run with --resume to continue.\n")
		cancel()
	}()

	manager := downloader.NewMetadataManager()

	// List mode
	if list {
		return listMetadataFiles(ctx, manager)
	}

	// Select target file
	var targetFile *downloader.MetadataFile
	var err error

	switch {
	case daily:
		fmt.Fprintf(os.Stderr, "Finding latest daily metadata dump...\n")
		targetFile, err = manager.GetLatestFile(ctx, downloader.FileTypeDaily)
		if err != nil {
			return fmt.Errorf("failed to find daily file: %w", err)
		}

	case monthly:
		fmt.Fprintf(os.Stderr, "Finding latest monthly full dump...\n")
		targetFile, err = manager.GetLatestFile(ctx, downloader.FileTypeMonthly)
		if err != nil {
			return fmt.Errorf("failed to find monthly file: %w", err)
		}

	case accessionsTab:
		fmt.Fprintf(os.Stderr, "Finding SRA_Accessions.tab...\n")
		targetFile, err = manager.GetFileByName(ctx, "SRA_Accessions.tab")
		if err != nil {
			return fmt.Errorf("failed to find SRA_Accessions.tab: %w", err)
		}

	case runMembersTab:
		fmt.Fprintf(os.Stderr, "Finding SRA_Run_Members.tab...\n")
		targetFile, err = manager.GetFileByName(ctx, "SRA_Run_Members.tab")
		if err != nil {
			return fmt.Errorf("failed to find SRA_Run_Members.tab: %w", err)
		}

	case file != "":
		fmt.Fprintf(os.Stderr, "Finding file: %s\n", file)
		targetFile, err = manager.GetFileByName(ctx, file)
		if err != nil {
			return fmt.Errorf("file not found: %w", err)
		}

	default:
		// Default to --auto for tar.gz selection
		fmt.Fprintf(os.Stderr, "Auto-selecting best metadata file...\n")
		targetFile, err = manager.AutoSelectFile(ctx)
		if err != nil {
			return fmt.Errorf("failed to auto-select file: %w", err)
		}
	}

	// Display file info
	fmt.Fprintf(os.Stderr, "\nSelected file:\n")
	fmt.Fprintf(os.Stderr, "  Name: %s\n", targetFile.Name)
	fmt.Fprintf(os.Stderr, "  Type: %s\n", targetFile.Type)
	fmt.Fprintf(os.Stderr, "  Size: %s\n", downloader.FormatSize(targetFile.Size))
	if !targetFile.Date.IsZero() {
		fmt.Fprintf(os.Stderr, "  Date: %s\n", targetFile.Date.Format("2006-01-02"))
	}
	fmt.Fprintf(os.Stderr, "  URL:  %s\n", targetFile.URL)
	fmt.Fprintf(os.Stderr, "\n")

	// Create output directory
	if err := os.MkdirAll(output, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(output, targetFile.Name)

	// Create downloader
	fd := downloader.NewFileDownloader()

	// Set up progress bar
	if !noProgress {
		pb := newDownloadProgressBar(targetFile.Size)
		fd.SetProgressFunc(func(p downloader.DownloadProgress) {
			pb.Update(p)
		})
		defer pb.Finish()
	}

	// Download
	fmt.Fprintf(os.Stderr, "Downloading to %s\n", outputPath)
	if resume {
		fmt.Fprintf(os.Stderr, "Resume mode enabled\n")
	}
	fmt.Fprintf(os.Stderr, "\n")

	if err := fd.Download(ctx, targetFile.URL, outputPath, resume); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("download interrupted (partial file saved as %s.part)", outputPath)
		}
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nDownload complete: %s\n", outputPath)
	return nil
}

// listMetadataFiles lists all available files from NCBI
func listMetadataFiles(ctx context.Context, manager *downloader.MetadataManager) error {
	fmt.Fprintf(os.Stderr, "Fetching file list from NCBI...\n\n")

	files, err := manager.ListAvailableFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "No files found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d files:\n\n", len(files))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "NAME\tTYPE\tSIZE\tDATE\n")
	fmt.Fprintf(w, "----\t----\t----\t----\n")

	for _, f := range files {
		dateStr := ""
		if !f.Date.IsZero() {
			dateStr = f.Date.Format("2006-01-02")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			f.Name,
			f.Type,
			downloader.FormatSize(f.Size),
			dateStr)
	}

	w.Flush()

	fmt.Fprintf(os.Stderr, "\nTo download a file:\n")
	fmt.Fprintf(os.Stderr, "  srake download metadata --daily\n")
	fmt.Fprintf(os.Stderr, "  srake download metadata --accessions-tab --output /data\n")
	fmt.Fprintf(os.Stderr, "  srake download metadata --file %s\n", files[0].Name)

	return nil
}

// downloadProgressBar renders a progress bar for file downloads
type downloadProgressBar struct {
	totalBytes int64
	lastUpdate time.Time
}

func newDownloadProgressBar(total int64) *downloadProgressBar {
	return &downloadProgressBar{
		totalBytes: total,
		lastUpdate: time.Now(),
	}
}

func (pb *downloadProgressBar) Update(p downloader.DownloadProgress) {
	// Rate-limit updates to at most once per second
	if time.Since(pb.lastUpdate) < time.Second {
		return
	}
	pb.lastUpdate = time.Now()

	total := p.TotalBytes
	if total <= 0 {
		total = pb.totalBytes
	}

	barWidth := 30
	var filled int
	if total > 0 {
		filled = int(p.PercentComplete * float64(barWidth) / 100)
		if filled > barWidth {
			filled = barWidth
		}
	}

	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
	speedMB := p.BytesPerSecond / (1024 * 1024)

	etaStr := "calculating..."
	if p.EstimatedRemaining > 0 {
		etaStr = formatETA(p.EstimatedRemaining)
	}

	downloaded := formatGB(p.BytesDownloaded)
	totalStr := formatGB(total)

	fmt.Fprintf(os.Stderr, "\r[%s] %5.1f%% | %s/%s | %.1f MB/s | ETA: %s",
		bar,
		p.PercentComplete,
		downloaded,
		totalStr,
		speedMB,
		etaStr)
}

func (pb *downloadProgressBar) Finish() {
	fmt.Fprintln(os.Stderr)
}

func formatGB(bytes int64) string {
	gb := float64(bytes) / (1024 * 1024 * 1024)
	if gb >= 1.0 {
		return fmt.Sprintf("%.1f GB", gb)
	}
	mb := float64(bytes) / (1024 * 1024)
	return fmt.Sprintf("%.0f MB", mb)
}

func formatETA(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
