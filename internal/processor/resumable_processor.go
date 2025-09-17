package processor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/parser"
	"github.com/nishad/srake/internal/progress"
)

// ResumableProcessor extends StreamProcessor with resume capabilities
type ResumableProcessor struct {
	*StreamProcessor
	tracker        *progress.Tracker
	resumeInfo     *progress.ResumeInfo
	skipUntilFile  string
	bytesSkipped   int64
	filesProcessed map[string]bool
}

// ResumeOptions configures resume behavior
type ResumeOptions struct {
	ForceRestart    bool          // Force fresh start even if progress exists
	Interactive     bool          // Ask user about resume
	CheckpointEvery time.Duration // How often to checkpoint
	MaxRetries      int           // Maximum retry attempts
	RetryDelay      time.Duration // Initial retry delay
}

// NewResumableProcessor creates a processor with resume capabilities
func NewResumableProcessor(db Database) (*ResumableProcessor, error) {
	// Create progress tracker with same database
	sqlDB := db.(*database.DB).GetSQLDB() // Assuming we can get underlying SQL DB
	tracker, err := progress.NewTracker(sqlDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create progress tracker: %w", err)
	}

	return &ResumableProcessor{
		StreamProcessor: NewStreamProcessor(db),
		tracker:         tracker,
		filesProcessed:  make(map[string]bool),
	}, nil
}

// ProcessFileWithResume processes a local file with resume support
func (rp *ResumableProcessor) ProcessFileWithResume(ctx context.Context, filePath string, opts ResumeOptions) error {
	// Start or resume progress tracking
	progressInfo, err := rp.tracker.StartOrResume(filePath, opts.ForceRestart)
	if err != nil {
		return fmt.Errorf("failed to start/resume progress: %w", err)
	}

	// Check if resuming
	if progressInfo.ProcessedBytes > 0 {
		if opts.Interactive {
			if !rp.confirmResume(progressInfo) {
				// User chose to restart
				progressInfo, err = rp.tracker.StartOrResume(filePath, true)
				if err != nil {
					return err
				}
			}
		}

		// Get resume information
		rp.resumeInfo, err = rp.tracker.GetResumeInfo()
		if err != nil {
			return fmt.Errorf("failed to get resume info: %w", err)
		}

		// Populate processed files map
		for _, file := range rp.resumeInfo.ProcessedFiles {
			rp.filesProcessed[file] = true
		}

		rp.skipUntilFile = rp.resumeInfo.LastFile
		rp.recordsInserted.Store(rp.resumeInfo.RecordsProcessed)

		fmt.Printf("Resuming from: %s (%.1f%% complete)\n",
			rp.skipUntilFile,
			float64(progressInfo.ProcessedBytes)*100/float64(progressInfo.TotalBytes))
	}

	// Set up checkpoint interval
	if opts.CheckpointEvery > 0 {
		rp.tracker.SetCheckpointInterval(opts.CheckpointEvery)
	}

	// Process file
	err = rp.StreamProcessor.ProcessFile(ctx, filePath)
	if err == nil {
		// Success - mark as completed
		return rp.tracker.MarkCompleted()
	}

	// Failed - mark as failed
	rp.tracker.MarkFailed(err.Error())
	return err
}

// ProcessURLWithResume processes a URL with resume support
func (rp *ResumableProcessor) ProcessURLWithResume(ctx context.Context, url string, opts ResumeOptions) error {
	// Start or resume progress tracking
	progressInfo, err := rp.tracker.StartOrResume(url, opts.ForceRestart)
	if err != nil {
		return fmt.Errorf("failed to start/resume progress: %w", err)
	}

	// Check if resuming
	if progressInfo.ProcessedBytes > 0 {
		if opts.Interactive {
			if !rp.confirmResume(progressInfo) {
				// User chose to restart
				progressInfo, err = rp.tracker.StartOrResume(url, true)
				if err != nil {
					return err
				}
			}
		}

		// Get resume information
		rp.resumeInfo, err = rp.tracker.GetResumeInfo()
		if err != nil {
			return fmt.Errorf("failed to get resume info: %w", err)
		}

		// Populate processed files map
		for _, file := range rp.resumeInfo.ProcessedFiles {
			rp.filesProcessed[file] = true
		}

		rp.skipUntilFile = rp.resumeInfo.LastFile
		rp.recordsInserted.Store(rp.resumeInfo.RecordsProcessed)

		fmt.Printf("Resuming from: %s (%.1f%% complete)\n",
			rp.skipUntilFile,
			float64(progressInfo.ProcessedBytes)*100/float64(progressInfo.TotalBytes))
	}

	// Set up checkpoint interval
	if opts.CheckpointEvery > 0 {
		rp.tracker.SetCheckpointInterval(opts.CheckpointEvery)
	}

	// Process with retries
	return rp.processWithRetry(ctx, url, progressInfo, opts)
}

// processWithRetry handles retry logic for failed downloads/processing
func (rp *ResumableProcessor) processWithRetry(ctx context.Context, url string, progress *progress.Progress, opts ResumeOptions) error {
	maxRetries := opts.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	retryDelay := opts.RetryDelay
	if retryDelay == 0 {
		retryDelay = 5 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("Retry attempt %d/%d after %v...\n", attempt, maxRetries, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}

		err := rp.processURLInternal(ctx, url, progress)
		if err == nil {
			// Success - mark as completed
			return rp.tracker.MarkCompleted()
		}

		lastErr = err

		// Check if error is retryable
		if !rp.isRetryableError(err) {
			break
		}

		// Update progress state
		rp.tracker.MarkFailed(err.Error())
	}

	// Final failure
	rp.tracker.MarkFailed(fmt.Sprintf("Failed after %d attempts: %v", maxRetries, lastErr))
	return lastErr
}

// processURLInternal performs the actual processing with resume support
func (rp *ResumableProcessor) processURLInternal(ctx context.Context, url string, progressInfo *progress.Progress) error {
	rp.startTime = time.Now()
	rp.totalBytes = progressInfo.TotalBytes

	// Create HTTP request with range header if resuming
	req, err := rp.createHTTPRequest(ctx, url, progressInfo.DownloadedBytes)
	if err != nil {
		return err
	}

	resp, err := rp.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if server supports resume
	if progressInfo.DownloadedBytes > 0 && resp.StatusCode != http.StatusPartialContent {
		fmt.Println("Server doesn't support resume, starting from beginning...")
		// Reset progress
		progressInfo.DownloadedBytes = 0
		return rp.processURLInternal(ctx, url, progressInfo)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Update total bytes if not resuming
	if progressInfo.TotalBytes == 0 {
		rp.totalBytes = resp.ContentLength
		rp.tracker.UpdateDownloadProgress(0, rp.totalBytes)
	}

	// Create counting reader to track download progress
	countingReader := &countingReader{
		reader:   resp.Body,
		counter:  &rp.bytesProcessed,
		total:    rp.totalBytes,
		callback: rp.onBytesDownloaded,
	}

	// Process the tar.gz stream with resume support
	return rp.processTarGzStreamWithResume(ctx, countingReader)
}

// processTarGzStreamWithResume processes tar.gz with skip capability
func (rp *ResumableProcessor) processTarGzStreamWithResume(ctx context.Context, reader io.Reader) error {
	// Create gzip reader
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Process tar entries
	skipping := rp.skipUntilFile != ""
	var currentPosition int64

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar reader error: %w", err)
		}

		currentPosition += header.Size

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Check if we should skip this file
		if skipping {
			if header.Name == rp.skipUntilFile {
				skipping = false
				fmt.Printf("Found resume point: %s\n", header.Name)
			} else {
				rp.bytesSkipped += header.Size
				continue
			}
		}

		// Check if file was already processed
		if rp.tracker.IsFileProcessed(header.Name) {
			fmt.Printf("Skipping already processed: %s\n", header.Name)
			continue
		}

		// Only process XML files
		if !strings.HasSuffix(header.Name, ".xml") {
			continue
		}

		// Process the XML file
		recordCount, err := rp.processXMLFileWithTracking(tarReader, header)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", header.Name, err)
			// Continue with next file instead of failing completely
			continue
		}

		// Calculate checksum for the file
		checksum := rp.calculateChecksum(header.Name)

		// Record file as processed
		if err := rp.tracker.RecordFileProcessed(header.Name, header.Size, recordCount, checksum); err != nil {
			return fmt.Errorf("failed to record processed file: %w", err)
		}

		// Update processing progress
		processed := rp.bytesProcessed.Load() + rp.bytesSkipped
		records := rp.recordsInserted.Load()
		if err := rp.tracker.UpdateProcessingProgress(currentPosition, processed, header.Name, records); err != nil {
			return fmt.Errorf("failed to update progress: %w", err)
		}

		// Call progress callback if set
		if rp.progressFunc != nil {
			rp.reportProgress(header.Name)
		}
	}

	return nil
}

// processXMLFileWithTracking processes an XML file and returns record count
func (rp *ResumableProcessor) processXMLFileWithTracking(reader io.Reader, header *tar.Header) (int, error) {
	decoder := xml.NewDecoder(reader)
	recordCount := 0

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return recordCount, err
		}

		if se, ok := token.(xml.StartElement); ok {
			switch se.Name.Local {
			case "EXPERIMENT_PACKAGE_SET", "EXPERIMENT_SET":
				var expSet parser.ExperimentSet
				if err := decoder.DecodeElement(&expSet, &se); err == nil {
					for _, exp := range expSet.Experiments {
						if err := rp.processExperiment(&exp); err == nil {
							recordCount++
						}
					}
				}

			case "SAMPLE_SET":
				var sampleSet parser.SampleSet
				if err := decoder.DecodeElement(&sampleSet, &se); err == nil {
					for _, sample := range sampleSet.Samples {
						if err := rp.processSample(&sample); err == nil {
							recordCount++
						}
					}
				}

			case "RUN_SET":
				var runSet parser.RunSet
				if err := decoder.DecodeElement(&runSet, &se); err == nil {
					for _, run := range runSet.Runs {
						if err := rp.processRun(&run); err == nil {
							recordCount++
						}
					}
				}

			case "STUDY_SET":
				var studySet parser.StudySet
				if err := decoder.DecodeElement(&studySet, &se); err == nil {
					for _, study := range studySet.Studies {
						if err := rp.processStudy(&study); err == nil {
							recordCount++
						}
					}
				}
			}
		}
	}

	rp.recordsInserted.Add(int64(recordCount))
	return recordCount, nil
}

// Helper methods for processing different record types
func (rp *ResumableProcessor) processExperiment(exp *parser.Experiment) error {
	dbExp := &database.Experiment{
		Accession:   exp.Accession,
		Title:       exp.Title,
		StudyRef:    exp.StudyRef.Accession,
		Platform:    rp.extractPlatform(exp),
		InstrumentModel: rp.extractInstrumentModel(exp),
	}

	if exp.Design != nil && exp.Design.LibraryDescriptor != nil {
		dbExp.LibraryName = exp.Design.LibraryDescriptor.LibraryName
		dbExp.LibraryStrategy = exp.Design.LibraryDescriptor.LibraryStrategy
		dbExp.LibrarySource = exp.Design.LibraryDescriptor.LibrarySource
		dbExp.LibrarySelection = exp.Design.LibraryDescriptor.LibrarySelection
	}

	return rp.db.InsertExperiment(dbExp)
}

func (rp *ResumableProcessor) processSample(sample *parser.Sample) error {
	dbSample := &database.Sample{
		Accession:    sample.Accession,
		Title:        sample.Title,
	}

	// Extract organism info
	if sample.SampleName != nil {
		if sample.SampleName.ScientificName != "" {
			dbSample.Organism = sample.SampleName.ScientificName
		}
		if sample.SampleName.TaxonID != "" {
			dbSample.TaxonID = sample.SampleName.TaxonID
		}
	}

	return rp.db.InsertSample(dbSample)
}

func (rp *ResumableProcessor) processRun(run *parser.Run) error {
	dbRun := &database.Run{
		Accession:    run.Accession,
		ExperimentRef: run.ExperimentRef.Accession,
		TotalSpots:   run.TotalSpots,
		TotalBases:   run.TotalBases,
	}
	return rp.db.InsertRun(dbRun)
}

func (rp *ResumableProcessor) processStudy(study *parser.Study) error {
	dbStudy := &database.Study{
		Accession: study.Accession,
		Title:     study.Descriptor.StudyTitle,
		Abstract:  study.Descriptor.StudyAbstract,
		StudyType: study.Descriptor.StudyType,
	}
	return rp.db.InsertStudy(dbStudy)
}

// Helper methods

func (rp *ResumableProcessor) extractPlatform(exp *parser.Experiment) string {
	if exp.Platform.Illumina != nil {
		return "ILLUMINA"
	}
	if exp.Platform.Nanopore != nil {
		return "OXFORD_NANOPORE"
	}
	if exp.Platform.PacBio != nil {
		return "PACBIO_SMRT"
	}
	if exp.Platform.IonTorrent != nil {
		return "ION_TORRENT"
	}
	if exp.Platform.LS454 != nil {
		return "LS454"
	}
	if exp.Platform.BGISEQ != nil {
		return "BGISEQ"
	}
	return ""
}

func (rp *ResumableProcessor) extractInstrumentModel(exp *parser.Experiment) string {
	if exp.Platform.Illumina != nil {
		return exp.Platform.Illumina.InstrumentModel
	}
	if exp.Platform.Nanopore != nil {
		return exp.Platform.Nanopore.InstrumentModel
	}
	if exp.Platform.PacBio != nil {
		return exp.Platform.PacBio.InstrumentModel
	}
	if exp.Platform.IonTorrent != nil {
		return exp.Platform.IonTorrent.InstrumentModel
	}
	return ""
}

func (rp *ResumableProcessor) createHTTPRequest(ctx context.Context, url string, startByte int64) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add range header if resuming
	if startByte > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
		fmt.Printf("Resuming download from byte %d\n", startByte)
	}

	// Add user agent
	req.Header.Set("User-Agent", "srake/0.0.1-alpha")

	return req, nil
}

func (rp *ResumableProcessor) confirmResume(progress *progress.Progress) bool {
	percentComplete := float64(progress.ProcessedBytes) * 100 / float64(progress.TotalBytes)
	fmt.Printf("Previous ingestion found:\n")
	fmt.Printf("  Source: %s\n", progress.SourceURL)
	fmt.Printf("  Progress: %.1f%% complete\n", percentComplete)
	fmt.Printf("  Records processed: %d\n", progress.RecordsProcessed)
	fmt.Printf("  Started: %s\n", progress.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("\nResume from last position? (y/n): ")

	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y"
}

func (rp *ResumableProcessor) onBytesDownloaded(bytes int64) {
	rp.tracker.UpdateDownloadProgress(bytes, rp.totalBytes)
}

func (rp *ResumableProcessor) calculateChecksum(data string) string {
	h := md5.Sum([]byte(data))
	return hex.EncodeToString(h[:])
}

func (rp *ResumableProcessor) isRetryableError(err error) bool {
	// Check for retryable errors
	errStr := err.Error()
	retryableErrors := []string{
		"connection reset",
		"broken pipe",
		"timeout",
		"temporary failure",
		"EOF",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(strings.ToLower(errStr), retryable) {
			return true
		}
	}

	return false
}

func (rp *ResumableProcessor) reportProgress(currentFile string) {
	stats, err := rp.tracker.GetStatistics()
	if err != nil {
		return
	}

	progress := Progress{
		BytesProcessed:         stats.ProcessedBytes,
		TotalBytes:             stats.TotalBytes,
		RecordsProcessed:       stats.RecordsProcessed,
		CurrentFile:            currentFile,
		PercentComplete:        stats.PercentComplete,
		BytesPerSecond:         stats.BytesPerSecond,
		TimeElapsed:            stats.Duration,
		EstimatedTimeRemaining: stats.EstimatedTimeRemaining,
	}

	rp.progressFunc(progress)
}

// countingReader tracks bytes read
type countingReader struct {
	reader   io.Reader
	counter  *atomic.Int64
	total    int64
	callback func(int64)
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.reader.Read(p)
	if n > 0 {
		newTotal := cr.counter.Add(int64(n))
		if cr.callback != nil {
			cr.callback(newTotal)
		}
	}
	return n, err
}