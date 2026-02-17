package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DownloadProgress reports the current download progress
type DownloadProgress struct {
	BytesDownloaded    int64
	TotalBytes         int64
	BytesPerSecond     float64
	PercentComplete    float64
	EstimatedRemaining time.Duration
}

// ProgressFunc is called periodically with download progress
type ProgressFunc func(DownloadProgress)

// FileDownloader handles downloading large files with resume support
type FileDownloader struct {
	client     *http.Client
	progressFn ProgressFunc
}

// NewFileDownloader creates a new file downloader
func NewFileDownloader() *FileDownloader {
	return &FileDownloader{
		client: &http.Client{
			Timeout: 0, // No timeout for large files
		},
	}
}

// SetProgressFunc sets the callback for progress updates
func (fd *FileDownloader) SetProgressFunc(fn ProgressFunc) {
	fd.progressFn = fn
}

// Download downloads a file from url to the given output path.
// If resume is true and a .part file exists, it will attempt to resume.
func (fd *FileDownloader) Download(ctx context.Context, url, outputPath string, resume bool) error {
	partPath := outputPath + ".part"

	// Check if final file already exists
	if stat, err := os.Stat(outputPath); err == nil {
		// Do a HEAD request to check if sizes match
		totalSize, _ := fd.getRemoteSize(ctx, url)
		if totalSize > 0 && stat.Size() == totalSize {
			// File already downloaded
			if fd.progressFn != nil {
				fd.progressFn(DownloadProgress{
					BytesDownloaded: totalSize,
					TotalBytes:      totalSize,
					PercentComplete: 100,
				})
			}
			return nil
		}
	}

	// Determine starting offset for resume
	var offset int64
	if resume {
		if stat, err := os.Stat(partPath); err == nil {
			offset = stat.Size()
		}
	}

	// HEAD request to get total size and check range support
	totalSize, supportsRange, err := fd.probeRemote(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to probe remote file: %w", err)
	}

	// If we can't resume, start from scratch
	if offset > 0 && !supportsRange {
		fmt.Fprintf(os.Stderr, "Server does not support range requests, starting from scratch\n")
		offset = 0
	}

	// If offset equals total size, we're already done
	if totalSize > 0 && offset == totalSize {
		return os.Rename(partPath, outputPath)
	}

	// If offset exceeds total size, start over
	if totalSize > 0 && offset > totalSize {
		offset = 0
	}

	// Create GET request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	resp, err := fd.client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	switch resp.StatusCode {
	case http.StatusOK:
		// Full response — reset offset since server sent everything
		offset = 0
	case http.StatusPartialContent:
		// Resumed — offset stays as is
	default:
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Use Content-Length from response if we didn't get it from HEAD
	if totalSize <= 0 && resp.ContentLength > 0 {
		totalSize = resp.ContentLength + offset
	}

	// Open part file for writing
	var out *os.File
	if offset > 0 {
		out, err = os.OpenFile(partPath, os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		out, err = os.Create(partPath)
	}
	if err != nil {
		return fmt.Errorf("failed to open part file: %w", err)
	}
	defer out.Close()

	// Set up progress tracking writer
	pw := &progressWriter{
		writer:         out,
		bytesWritten:   offset,
		totalBytes:     totalSize,
		progressFn:     fd.progressFn,
		lastReport:     time.Now(),
		startTime:      time.Now(),
		startBytes:     offset,
		reportInterval: 500 * time.Millisecond,
	}

	// Stream the download
	_, err = io.Copy(pw, resp.Body)
	if err != nil {
		// On context cancellation, leave .part file in place for resume
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("download interrupted: %w", err)
	}

	// Close the file before renaming
	out.Close()

	// Rename .part to final filename
	if err := os.Rename(partPath, outputPath); err != nil {
		return fmt.Errorf("failed to rename part file: %w", err)
	}

	return nil
}

// getRemoteSize returns the Content-Length of the remote file
func (fd *FileDownloader) getRemoteSize(ctx context.Context, url string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := fd.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.ContentLength, nil
}

// probeRemote performs a HEAD request to get file size and range support
func (fd *FileDownloader) probeRemote(ctx context.Context, url string) (size int64, supportsRange bool, err error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return 0, false, err
	}

	resp, err := fd.client.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("HEAD request failed: HTTP %d", resp.StatusCode)
	}

	size = resp.ContentLength
	acceptRanges := resp.Header.Get("Accept-Ranges")
	supportsRange = acceptRanges == "bytes"

	return size, supportsRange, nil
}

// progressWriter wraps an io.Writer and reports progress
type progressWriter struct {
	writer         io.Writer
	bytesWritten   int64
	totalBytes     int64
	progressFn     ProgressFunc
	lastReport     time.Time
	startTime      time.Time
	startBytes     int64
	reportInterval time.Duration
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.bytesWritten += int64(n)

	if pw.progressFn != nil && time.Since(pw.lastReport) >= pw.reportInterval {
		pw.lastReport = time.Now()

		elapsed := time.Since(pw.startTime).Seconds()
		bytesTransferred := pw.bytesWritten - pw.startBytes

		var speed float64
		if elapsed > 0 {
			speed = float64(bytesTransferred) / elapsed
		}

		var pct float64
		var eta time.Duration
		if pw.totalBytes > 0 {
			pct = float64(pw.bytesWritten) / float64(pw.totalBytes) * 100
			remaining := pw.totalBytes - pw.bytesWritten
			if speed > 0 {
				eta = time.Duration(float64(remaining)/speed) * time.Second
			}
		}

		pw.progressFn(DownloadProgress{
			BytesDownloaded:    pw.bytesWritten,
			TotalBytes:         pw.totalBytes,
			BytesPerSecond:     speed,
			PercentComplete:    pct,
			EstimatedRemaining: eta,
		})
	}

	return n, err
}
