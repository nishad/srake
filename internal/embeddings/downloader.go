package embeddings

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DownloadProgress represents download progress information
type DownloadProgress struct {
	File            string
	BytesDownloaded int64
	TotalBytes      int64
	Percentage      float64
	Speed           float64 // MB/s
	ETA             time.Duration
}

// Downloader handles model downloading from HuggingFace
type Downloader struct {
	client   *http.Client
	manager  *Manager
	progress chan<- DownloadProgress
}

// NewDownloader creates a new model downloader
func NewDownloader(manager *Manager, progress chan<- DownloadProgress) *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 0, // No timeout for large file downloads
		},
		manager:  manager,
		progress: progress,
	}
}

// DownloadModel downloads a complete model with specified variant
func (d *Downloader) DownloadModel(modelID string, variantName string) error {
	config, err := GetModelConfig(modelID)
	if err != nil {
		return fmt.Errorf("model %s not found in registry", modelID)
	}

	// Get the specific variant to download
	var variant *ModelVariant
	if variantName == "" {
		variant, err = GetDefaultVariant(modelID)
		if err != nil {
			return err
		}
	} else {
		variant, err = GetVariant(modelID, variantName)
		if err != nil {
			return err
		}
	}

	// Create model directory structure
	modelPath := d.manager.GetModelPath(modelID)
	if err := os.MkdirAll(modelPath, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Download config files first
	for _, configFile := range config.ConfigFiles {
		url := fmt.Sprintf("%s/%s", config.BaseURL, configFile)
		destPath := filepath.Join(modelPath, configFile)
		if err := d.downloadFile(url, destPath, configFile); err != nil {
			return fmt.Errorf("failed to download %s: %w", configFile, err)
		}
	}

	// Download tokenizer files
	for _, tokenizerFile := range config.TokenizerFiles {
		url := fmt.Sprintf("%s/%s", config.BaseURL, tokenizerFile)
		destPath := filepath.Join(modelPath, tokenizerFile)
		if err := d.downloadFile(url, destPath, tokenizerFile); err != nil {
			// Some tokenizer files might be optional, so just log the error
			fmt.Printf("Warning: could not download %s: %v\n", tokenizerFile, err)
		}
	}

	// Download the ONNX model variant
	onnxDir := filepath.Join(modelPath, "onnx")
	if err := os.MkdirAll(onnxDir, 0755); err != nil {
		return fmt.Errorf("failed to create onnx directory: %w", err)
	}

	destPath := filepath.Join(modelPath, variant.Filename)
	if err := d.downloadFileWithProgress(variant.URL, destPath, variant.Size, variant.Name); err != nil {
		return fmt.Errorf("failed to download model variant %s: %w", variant.Name, err)
	}

	// Register the model with the manager
	if err := d.manager.RegisterModel(modelID); err != nil {
		return fmt.Errorf("failed to register model: %w", err)
	}

	// Set the downloaded variant as active
	if err := d.manager.SetActiveVariant(modelID, variant.Name); err != nil {
		return fmt.Errorf("failed to set active variant: %w", err)
	}

	return nil
}

// DownloadVariant downloads only a specific variant of an already installed model
func (d *Downloader) DownloadVariant(modelID string, variantName string) error {
	variant, err := GetVariant(modelID, variantName)
	if err != nil {
		return err
	}

	modelPath := d.manager.GetModelPath(modelID)
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model %s is not installed, use DownloadModel first", modelID)
	}

	destPath := filepath.Join(modelPath, variant.Filename)

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := d.downloadFileWithProgress(variant.URL, destPath, variant.Size, variant.Name); err != nil {
		return fmt.Errorf("failed to download variant %s: %w", variant.Name, err)
	}

	// Refresh model info
	return d.manager.RefreshModels()
}

// downloadFile downloads a file without progress tracking (for small files)
func (d *Downloader) downloadFile(url, destPath, displayName string) error {
	// Check if file already exists
	if _, err := os.Stat(destPath); err == nil {
		return nil // File already exists
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	resp, err := d.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create temporary file
	tempFile := destPath + ".tmp"
	out, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy content
	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(tempFile)
		return err
	}

	// Atomic move
	return os.Rename(tempFile, destPath)
}

// downloadFileWithProgress downloads a file with progress tracking
func (d *Downloader) downloadFileWithProgress(url, destPath string, expectedSize int64, displayName string) error {
	// Check if file already exists and has correct size
	if info, err := os.Stat(destPath); err == nil {
		if expectedSize == 0 || info.Size() == expectedSize {
			return nil // File already exists with correct size
		}
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	resp, err := d.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create temporary file
	tempFile := destPath + ".tmp"
	out, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get total size
	totalSize := resp.ContentLength
	if totalSize < 0 && expectedSize > 0 {
		totalSize = expectedSize
	}

	// Create progress reader
	reader := &progressReader{
		reader:      resp.Body,
		total:       totalSize,
		progress:    d.progress,
		displayName: displayName,
		startTime:   time.Now(),
	}

	// Copy with progress
	if _, err := io.Copy(out, reader); err != nil {
		os.Remove(tempFile)
		return err
	}

	// Close file before renaming (important on Windows)
	out.Close()

	// Atomic move
	return os.Rename(tempFile, destPath)
}

// progressReader wraps a reader to track progress
type progressReader struct {
	reader      io.Reader
	total       int64
	current     int64
	progress    chan<- DownloadProgress
	displayName string
	startTime   time.Time
	lastReport  time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)

	// Report progress every 100ms
	if pr.progress != nil && time.Since(pr.lastReport) > 100*time.Millisecond {
		elapsed := time.Since(pr.startTime).Seconds()
		speed := float64(pr.current) / elapsed / (1024 * 1024) // MB/s

		var eta time.Duration
		if pr.total > 0 && speed > 0 {
			remaining := pr.total - pr.current
			eta = time.Duration(float64(remaining)/speed/(1024*1024)) * time.Second
		}

		percentage := 0.0
		if pr.total > 0 {
			percentage = float64(pr.current) / float64(pr.total) * 100
		}

		select {
		case pr.progress <- DownloadProgress{
			File:            pr.displayName,
			BytesDownloaded: pr.current,
			TotalBytes:      pr.total,
			Percentage:      percentage,
			Speed:           speed,
			ETA:             eta,
		}:
		default:
			// Don't block if channel is full
		}

		pr.lastReport = time.Now()
	}

	return n, err
}

// VerifyFile verifies a downloaded file against its expected hash
func VerifyFile(filePath string, expectedSHA256 string) error {
	if expectedSHA256 == "" {
		return nil // No hash to verify
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedSHA256 {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedSHA256, actualHash)
	}

	return nil
}

// CleanupTempFiles removes any .tmp files in the models directory
func (d *Downloader) CleanupTempFiles() error {
	return filepath.Walk(d.manager.GetModelsDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if filepath.Ext(path) == ".tmp" {
			os.Remove(path)
		}
		return nil
	})
}
