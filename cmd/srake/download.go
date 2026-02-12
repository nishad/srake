package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/downloader"
	"github.com/nishad/srake/internal/paths"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download [<accession> ...]",
	Short: "Download SRA data files from multiple sources",
	Long: `Download SRA/FASTQ files from various sources including FTP, AWS, GCP, and NCBI.

Supports downloading:
  - Individual runs (SRR)
  - All runs for an experiment (SRX)
  - All runs for a study (SRP)
  - All runs for a sample (SRS)`,
	Example: `  # Download a single run
  srake download SRR123456

  # Download from AWS with parallel transfers
  srake download SRR123456 --source aws --threads 4

  # Download all runs for a study as FASTQ
  srake download SRP123456 --type fastq --output ./data/

  # Download with Aspera high-speed transfer
  srake download SRR123456 --aspera

  # Download from a list file
  srake download --list runs.txt --parallel 4`,
	Args: cobra.MinimumNArgs(0),
	RunE: runDownload,
}

var (
	downloadSource   string
	downloadType     string
	downloadOutput   string
	downloadThreads  int
	downloadParallel int
	downloadAspera   bool
	downloadList     string
	downloadRetry    int
	downloadValidate bool
	downloadDryRun   bool
)

func init() {
	downloadCmd.Flags().StringVarP(&downloadSource, "source", "s", "auto", "Download source (auto|ftp|aws|gcp|ncbi)")
	downloadCmd.Flags().StringVarP(&downloadType, "type", "t", "sra", "File type (sra|fastq|fasta)")
	downloadCmd.Flags().StringVarP(&downloadOutput, "output", "o", "./", "Output directory")
	downloadCmd.Flags().IntVar(&downloadThreads, "threads", 1, "Number of download threads per file")
	downloadCmd.Flags().IntVarP(&downloadParallel, "parallel", "p", 1, "Number of parallel downloads")
	downloadCmd.Flags().BoolVar(&downloadAspera, "aspera", false, "Use Aspera for high-speed transfer")
	downloadCmd.Flags().StringVarP(&downloadList, "list", "l", "", "File containing accessions (one per line)")
	downloadCmd.Flags().IntVar(&downloadRetry, "retry", 3, "Number of retry attempts")
	downloadCmd.Flags().BoolVar(&downloadValidate, "validate", true, "Validate downloaded files")
	downloadCmd.Flags().BoolVar(&downloadDryRun, "dry-run", false, "Show what would be downloaded without downloading")
}

func runDownload(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Collect all accessions
	accessions := args

	// Read from stdin if no arguments provided and stdin is available
	if len(args) == 0 && downloadList == "" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Data is being piped to stdin
			stdinAccessions, err := readAccessionsFromReader(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			accessions = stdinAccessions
		}
	}

	// Read from list file if specified
	if downloadList != "" {
		listAccessions, err := readAccessionFile(downloadList)
		if err != nil {
			return fmt.Errorf("failed to read list file: %w", err)
		}
		accessions = append(accessions, listAccessions...)
	}

	if len(accessions) == 0 {
		return fmt.Errorf("no accessions provided")
	}

	// Create output directory if needed
	if !downloadDryRun {
		if err := os.MkdirAll(downloadOutput, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Initialize downloader
	dlConfig := downloader.Config{
		Source:        downloader.ParseSource(downloadSource),
		FileType:      downloader.ParseFileType(downloadType),
		OutputDir:     downloadOutput,
		Threads:       downloadThreads,
		ParallelJobs:  downloadParallel,
		UseAspera:     downloadAspera,
		RetryAttempts: downloadRetry,
		Validate:      downloadValidate,
		DryRun:        downloadDryRun,
		Verbose:       verbose,
	}

	dl := downloader.NewSRADownloader(dlConfig)

	// Expand accessions (e.g., SRP to multiple SRRs)
	expandedAccessions, err := expandAccessions(ctx, accessions)
	if err != nil {
		return fmt.Errorf("failed to expand accessions: %w", err)
	}

	if verbose || downloadDryRun {
		printInfo("Will download %d files", len(expandedAccessions))
	}

	// Download each accession
	successCount := 0
	failedAccessions := []string{}
	printDebug("Starting download loop for %d accessions", len(expandedAccessions))

	for i, acc := range expandedAccessions {
		if !quiet {
			fmt.Printf("\n[%d/%d] Downloading %s\n", i+1, len(expandedAccessions), acc)
		}
		printDebug("Attempting download: %s (source: %s, type: %s)", acc, downloadSource, downloadType)

		result, err := dl.Download(ctx, acc)
		if err != nil {
			printError("Failed to download %s: %v", acc, err)
			failedAccessions = append(failedAccessions, acc)
			continue
		}

		if !downloadDryRun {
			printSuccess("Downloaded %s to %s (%.2f MB)",
				acc, result.Path, float64(result.Size)/(1024*1024))

			if downloadValidate && result.MD5 != "" {
				printInfo("  MD5: %s", result.MD5)
			}
		} else {
			fmt.Printf("Would download: %s\n", result.URL)
			fmt.Printf("  Source: %s\n", result.Source)
			fmt.Printf("  Size: %.2f MB\n", float64(result.Size)/(1024*1024))
		}

		successCount++
	}

	// Summary
	fmt.Print("\n" + strings.Repeat("─", 60) + "\n")
	printSuccess("Successfully downloaded: %d/%d files", successCount, len(expandedAccessions))

	if len(failedAccessions) > 0 {
		printError("Failed downloads: %s", strings.Join(failedAccessions, ", "))
	}

	return nil
}

// expandAccessions expands project/experiment/sample accessions to run accessions
func expandAccessions(ctx context.Context, accessions []string) ([]string, error) {
	expanded := []string{}

	for _, acc := range accessions {
		acc = strings.ToUpper(strings.TrimSpace(acc))

		switch {
		case strings.HasPrefix(acc, "SRR"), strings.HasPrefix(acc, "ERR"), strings.HasPrefix(acc, "DRR"):
			// Already a run accession
			expanded = append(expanded, acc)

		case strings.HasPrefix(acc, "SRP"), strings.HasPrefix(acc, "ERP"), strings.HasPrefix(acc, "DRP"):
			// Expand study to runs
			runs, err := getRunsForStudy(acc)
			if err != nil {
				return nil, fmt.Errorf("failed to expand study %s: %w", acc, err)
			}
			expanded = append(expanded, runs...)

		case strings.HasPrefix(acc, "SRX"), strings.HasPrefix(acc, "ERX"), strings.HasPrefix(acc, "DRX"):
			// Expand experiment to runs
			runs, err := getRunsForExperiment(acc)
			if err != nil {
				return nil, fmt.Errorf("failed to expand experiment %s: %w", acc, err)
			}
			expanded = append(expanded, runs...)

		case strings.HasPrefix(acc, "SRS"), strings.HasPrefix(acc, "ERS"), strings.HasPrefix(acc, "DRS"):
			// Expand sample to runs
			runs, err := getRunsForSample(acc)
			if err != nil {
				return nil, fmt.Errorf("failed to expand sample %s: %w", acc, err)
			}
			expanded = append(expanded, runs...)

		default:
			return nil, fmt.Errorf("unsupported accession type: %s", acc)
		}
	}

	return expanded, nil
}

// Helper functions to get runs from database
func getRunsForStudy(studyAccession string) ([]string, error) {
	db, err := database.Initialize(paths.GetDatabasePath())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	query := `
		SELECT r.run_accession
		FROM runs r
		JOIN experiments e ON r.experiment_accession = e.experiment_accession
		WHERE e.study_accession = ?
		ORDER BY r.run_accession
	`
	rows, err := db.Query(query, studyAccession)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var runs []string
	for rows.Next() {
		var run string
		if err := rows.Scan(&run); err != nil {
			continue
		}
		runs = append(runs, run)
	}

	if len(runs) == 0 {
		return nil, fmt.Errorf("no runs found for study %s", studyAccession)
	}

	return runs, nil
}

func getRunsForExperiment(expAccession string) ([]string, error) {
	db, err := database.Initialize(paths.GetDatabasePath())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	query := `
		SELECT run_accession
		FROM runs
		WHERE experiment_accession = ?
		ORDER BY run_accession
	`
	rows, err := db.Query(query, expAccession)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var runs []string
	for rows.Next() {
		var run string
		if err := rows.Scan(&run); err != nil {
			continue
		}
		runs = append(runs, run)
	}

	if len(runs) == 0 {
		return nil, fmt.Errorf("no runs found for experiment %s", expAccession)
	}

	return runs, nil
}

func getRunsForSample(sampleAccession string) ([]string, error) {
	db, err := database.Initialize(paths.GetDatabasePath())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Samples are linked via experiments
	query := `
		SELECT r.run_accession
		FROM runs r
		JOIN experiments e ON r.experiment_accession = e.experiment_accession
		WHERE e.sample_accession = ?
		ORDER BY r.run_accession
	`
	rows, err := db.Query(query, sampleAccession)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var runs []string
	for rows.Next() {
		var run string
		if err := rows.Scan(&run); err != nil {
			continue
		}
		runs = append(runs, run)
	}

	if len(runs) == 0 {
		return nil, fmt.Errorf("no runs found for sample %s", sampleAccession)
	}

	return runs, nil
}
