package downloader

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DownloadSource represents the source for downloading files
type DownloadSource int

const (
	SourceAuto DownloadSource = iota
	SourceFTP
	SourceAWS
	SourceGCP
	SourceNCBI
)

// SRAFileType represents the type of file to download
type SRAFileType int

const (
	SRAFileTypeSRA SRAFileType = iota
	SRAFileTypeFASTQ
	SRAFileTypeFASTA
)

// Config holds configuration for the SRA downloader
type Config struct {
	Source        DownloadSource
	FileType      SRAFileType
	OutputDir     string
	Threads       int
	ParallelJobs  int
	UseAspera     bool
	RetryAttempts int
	Validate      bool
	DryRun        bool
	Verbose       bool
}

// DownloadResult contains information about a downloaded file
type DownloadResult struct {
	Accession string
	Path      string
	URL       string
	Source    string
	Size      int64
	MD5       string
	Duration  time.Duration
}

// SRADownloader handles downloading SRA files from various sources
type SRADownloader struct {
	config     Config
	httpClient *http.Client
	semaphore  chan struct{}
	mu         sync.Mutex
}

// NewSRADownloader creates a new SRA downloader
func NewSRADownloader(config Config) *SRADownloader {
	// Create semaphore for parallel downloads
	semaphore := make(chan struct{}, config.ParallelJobs)

	return &SRADownloader{
		config: config,
		httpClient: &http.Client{
			Timeout: 0, // No timeout for large downloads
		},
		semaphore: semaphore,
	}
}

// Download downloads a single accession
func (d *SRADownloader) Download(ctx context.Context, accession string) (*DownloadResult, error) {
	// Acquire semaphore slot
	d.semaphore <- struct{}{}
	defer func() { <-d.semaphore }()

	startTime := time.Now()

	// Determine download URL based on source
	url, source, err := d.getDownloadURL(accession)
	if err != nil {
		return nil, err
	}

	// Determine output filename
	filename := d.getOutputFilename(accession)
	outputPath := filepath.Join(d.config.OutputDir, filename)

	result := &DownloadResult{
		Accession: accession,
		Path:      outputPath,
		URL:       url,
		Source:    source,
	}

	// Dry run - just return the information
	if d.config.DryRun {
		size, _ := d.getFileSize(url)
		result.Size = size
		return result, nil
	}

	// Check if file already exists
	if !d.config.DryRun {
		if stat, err := os.Stat(outputPath); err == nil {
			if d.config.Verbose {
				fmt.Printf("File already exists: %s (%.2f MB)\n",
					outputPath, float64(stat.Size())/(1024*1024))
			}
			result.Size = stat.Size()
			result.Duration = time.Since(startTime)
			return result, nil
		}
	}

	// Download based on method
	var downloadErr error
	if d.config.UseAspera && d.canUseAspera() {
		downloadErr = d.downloadWithAspera(ctx, url, outputPath)
	} else {
		downloadErr = d.downloadWithHTTP(ctx, url, outputPath)
	}

	if downloadErr != nil {
		return nil, downloadErr
	}

	// Get file info
	stat, err := os.Stat(outputPath)
	if err != nil {
		return nil, err
	}
	result.Size = stat.Size()

	// Validate if requested
	if d.config.Validate {
		md5sum, err := d.calculateMD5(outputPath)
		if err == nil {
			result.MD5 = md5sum
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// getDownloadURL determines the download URL based on source
func (d *SRADownloader) getDownloadURL(accession string) (string, string, error) {
	source := d.config.Source

	// Auto-detect best source
	if source == SourceAuto {
		source = d.detectBestSource(accession)
	}

	var url string
	var sourceName string

	switch source {
	case SourceAWS:
		url = d.getAWSURL(accession)
		sourceName = "AWS"

	case SourceGCP:
		url = d.getGCPURL(accession)
		sourceName = "GCP"

	case SourceNCBI:
		url = d.getNCBIURL(accession)
		sourceName = "NCBI"

	case SourceFTP:
		fallthrough
	default:
		url = d.getFTPURL(accession)
		sourceName = "FTP"
	}

	if url == "" {
		return "", "", fmt.Errorf("could not determine download URL for %s", accession)
	}

	return url, sourceName, nil
}

// URL generation methods for different sources
func (d *SRADownloader) getFTPURL(accession string) string {
	// NCBI FTP structure
	prefix := accession[:6]
	prefix3 := accession[:3]

	switch d.config.FileType {
	case SRAFileTypeFASTQ:
		// EBI ENA FASTQ files
		if strings.HasPrefix(accession, "SRR") {
			// ftp://ftp.sra.ebi.ac.uk/vol1/fastq/SRR123/SRR123456/SRR123456.fastq.gz
			subdir := ""
			if len(accession) > 9 {
				subdir = "00" + accession[len(accession)-1:]
			}
			return fmt.Sprintf("ftp://ftp.sra.ebi.ac.uk/vol1/fastq/%s/%s/%s/%s.fastq.gz",
				prefix, subdir, accession, accession)
		}

	case SRAFileTypeSRA:
		fallthrough
	default:
		// ftp://ftp-trace.ncbi.nlm.nih.gov/sra/sra-instant/reads/ByRun/sra/SRR/SRR123/SRR123456/SRR123456.sra
		return fmt.Sprintf("ftp://ftp-trace.ncbi.nlm.nih.gov/sra/sra-instant/reads/ByRun/sra/%s/%s/%s/%s.sra",
			prefix3, prefix, accession, accession)
	}

	return ""
}

func (d *SRADownloader) getAWSURL(accession string) string {
	// AWS Open Data Registry
	// https://sra-pub-run-odp.s3.amazonaws.com/sra/SRR123456/SRR123456
	switch d.config.FileType {
	case SRAFileTypeSRA:
		return fmt.Sprintf("https://sra-pub-run-odp.s3.amazonaws.com/sra/%s/%s", accession, accession)
	case SRAFileTypeFASTQ:
		return fmt.Sprintf("https://sra-pub-run-odp.s3.amazonaws.com/sra/%s/%s.fastq.gz", accession, accession)
	}
	return ""
}

func (d *SRADownloader) getGCPURL(accession string) string {
	// Google Cloud Public Datasets
	// gs://sra-pub-run-1/SRR123456/SRR123456.1
	switch d.config.FileType {
	case SRAFileTypeSRA:
		return fmt.Sprintf("https://storage.googleapis.com/sra-pub-run-1/%s/%s.1", accession, accession)
	}
	return ""
}

func (d *SRADownloader) getNCBIURL(accession string) string {
	// NCBI direct download
	return fmt.Sprintf("https://sra-downloadb.be-md.ncbi.nlm.nih.gov/sos2/sra-pub-run-11/%s/%s.1",
		accession, accession)
}

// getOutputFilename determines the output filename
func (d *SRADownloader) getOutputFilename(accession string) string {
	switch d.config.FileType {
	case SRAFileTypeFASTQ:
		return accession + ".fastq.gz"
	case SRAFileTypeFASTA:
		return accession + ".fasta.gz"
	case SRAFileTypeSRA:
		fallthrough
	default:
		return accession + ".sra"
	}
}

// detectBestSource tries to determine the best download source
func (d *SRADownloader) detectBestSource(accession string) DownloadSource {
	// Simple heuristic - could be improved with actual testing
	// Check if running in cloud environment
	if os.Getenv("AWS_REGION") != "" {
		return SourceAWS
	}
	if os.Getenv("GCP_PROJECT") != "" {
		return SourceGCP
	}

	// Default to FTP
	return SourceFTP
}

// downloadWithHTTP downloads a file using HTTP/HTTPS
func (d *SRADownloader) downloadWithHTTP(ctx context.Context, url, outputPath string) error {
	// Create temporary file
	tmpPath := outputPath + ".tmp"

	// Create the file
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Copy with progress
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// Move temp file to final location
	return os.Rename(tmpPath, outputPath)
}

// downloadWithAspera downloads using Aspera if available
func (d *SRADownloader) downloadWithAspera(ctx context.Context, url, outputPath string) error {
	// Convert URL to Aspera format
	asperaURL := d.convertToAsperaURL(url)
	if asperaURL == "" {
		// Fall back to HTTP
		return d.downloadWithHTTP(ctx, url, outputPath)
	}

	// Run ascp command
	cmd := exec.CommandContext(ctx, "ascp",
		"-i", d.getAsperaKeyPath(),
		"-k", "1", // Resume partial transfers
		"-T",         // No encryption
		"-l", "300m", // Target rate
		asperaURL,
		outputPath,
	)

	if d.config.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// canUseAspera checks if Aspera is available
func (d *SRADownloader) canUseAspera() bool {
	_, err := exec.LookPath("ascp")
	return err == nil
}

// convertToAsperaURL converts an FTP/HTTP URL to Aspera format
func (d *SRADownloader) convertToAsperaURL(url string) string {
	// Convert NCBI FTP to Aspera
	if strings.Contains(url, "ftp-trace.ncbi.nlm.nih.gov") {
		return strings.Replace(url, "ftp://ftp-trace.ncbi.nlm.nih.gov",
			"anonftp@ftp.ncbi.nlm.nih.gov:", 1)
	}

	// Convert EBI FTP to Aspera
	if strings.Contains(url, "ftp.sra.ebi.ac.uk") {
		return strings.Replace(url, "ftp://ftp.sra.ebi.ac.uk",
			"era-fasp@fasp.sra.ebi.ac.uk:", 1)
	}

	return ""
}

// getAsperaKeyPath returns the path to Aspera SSH key
func (d *SRADownloader) getAsperaKeyPath() string {
	// Common locations for Aspera keys
	paths := []string{
		"~/.aspera/connect/etc/asperaweb_id_dsa.openssh",
		"/opt/aspera/etc/asperaweb_id_dsa.openssh",
	}

	for _, p := range paths {
		expanded := os.ExpandEnv(p)
		if _, err := os.Stat(expanded); err == nil {
			return expanded
		}
	}

	return ""
}

// getFileSize attempts to get the size of a remote file
func (d *SRADownloader) getFileSize(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.ContentLength, nil
}

// calculateMD5 calculates the MD5 checksum of a file
func (d *SRADownloader) calculateMD5(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Helper functions for parsing configuration

// ParseSource parses a source string
func ParseSource(s string) DownloadSource {
	switch strings.ToLower(s) {
	case "aws":
		return SourceAWS
	case "gcp":
		return SourceGCP
	case "ncbi":
		return SourceNCBI
	case "ftp":
		return SourceFTP
	default:
		return SourceAuto
	}
}

// ParseFileType parses a file type string
func ParseFileType(s string) SRAFileType {
	switch strings.ToLower(s) {
	case "fastq":
		return SRAFileTypeFASTQ
	case "fasta":
		return SRAFileTypeFASTA
	case "sra":
		fallthrough
	default:
		return SRAFileTypeSRA
	}
}
