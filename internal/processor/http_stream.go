package processor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/parser"
)

// Database interface for testing
type Database interface {
	InsertStudy(study *database.Study) error
	InsertExperiment(exp *database.Experiment) error
	InsertSample(sample *database.Sample) error
	InsertRun(run *database.Run) error
	InsertSubmission(submission *database.Submission) error
	InsertAnalysis(analysis *database.Analysis) error
	BatchInsertExperiments(experiments []database.Experiment) error

	// Pool/multiplex support
	InsertSamplePool(pool *database.SamplePool) error
	GetSamplePools(parentSample string) ([]database.SamplePool, error)
	CountSamplePools() (int, error)
	GetAveragePoolSize() (float64, error)
	GetMaxPoolSize() (int, error)

	// Identifier and link support
	InsertIdentifier(identifier *database.Identifier) error
	GetIdentifiers(recordType, recordAccession string) ([]database.Identifier, error)
	FindRecordsByIdentifier(idValue string) ([]database.Identifier, error)
	InsertLink(link *database.Link) error
	GetLinks(recordType, recordAccession string) ([]database.Link, error)
}

// StreamProcessor handles streaming processing of tar.gz files from HTTP
type StreamProcessor struct {
	db              Database
	client          *http.Client
	progressFunc    ProgressFunc
	bytesProcessed  atomic.Int64
	totalBytes      int64
	recordsInserted atomic.Int64
	startTime       time.Time
}

// ProgressFunc is called periodically with progress updates
type ProgressFunc func(progress Progress)

// Progress contains information about the current processing progress
type Progress struct {
	BytesProcessed         int64
	TotalBytes             int64
	RecordsProcessed       int64
	CurrentFile            string
	PercentComplete        float64
	BytesPerSecond         float64
	TimeElapsed            time.Duration
	EstimatedTimeRemaining time.Duration
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(db Database) *StreamProcessor {
	return &StreamProcessor{
		db: db,
		client: &http.Client{
			Timeout: 0, // No timeout for large files
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  true, // We handle gzip ourselves
				MaxIdleConnsPerHost: 10,
			},
		},
	}
}

// SetProgressFunc sets the progress callback function
func (sp *StreamProcessor) SetProgressFunc(f ProgressFunc) {
	sp.progressFunc = f
}

// ProcessURL streams and processes a tar.gz file from the given URL
func (sp *StreamProcessor) ProcessURL(ctx context.Context, url string) error {
	sp.startTime = time.Now()
	sp.bytesProcessed.Store(0)
	sp.recordsInserted.Store(0)

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := sp.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	// Get total size if available
	sp.totalBytes = resp.ContentLength

	// Create a counting reader to track progress
	countingReader := &countingReader{
		reader:   resp.Body,
		counter:  &sp.bytesProcessed,
		callback: sp.updateProgress,
	}

	// Chain: HTTP Body → Gzip Reader → Tar Reader
	gzipReader, err := gzip.NewReader(countingReader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	// Process each file in the tar archive
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
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Skip non-files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Process XML files
		if strings.HasSuffix(header.Name, ".xml") {
			sp.updateProgress(header.Name)

			if err := sp.processXMLStream(ctx, tarReader, header.Name); err != nil {
				// Log error but continue processing
				fmt.Printf("Warning: failed to process %s: %v\n", header.Name, err)
				continue
			}
		}
	}

	return nil
}

// processXMLStream processes a single XML file from the tar stream
func (sp *StreamProcessor) processXMLStream(ctx context.Context, reader io.Reader, filename string) error {
	decoder := xml.NewDecoder(reader)

	// Use a smaller buffer for streaming
	decoder.CharsetReader = nil // Use default UTF-8

	// Determine file type from name
	switch {
	case strings.Contains(filename, "experiment"):
		return sp.processExperiments(ctx, decoder)
	case strings.Contains(filename, "study"):
		return sp.processStudies(ctx, decoder)
	case strings.Contains(filename, "sample"):
		return sp.processSamples(ctx, decoder)
	case strings.Contains(filename, "run"):
		return sp.processRuns(ctx, decoder)
	default:
		// Skip unknown file types
		return nil
	}
}

// processExperiments streams and processes experiment records
func (sp *StreamProcessor) processExperiments(ctx context.Context, decoder *xml.Decoder) error {
	batch := make([]database.Experiment, 0, 5000) // Optimized batch size

	// Decode the entire ExperimentSet
	var expSet parser.ExperimentSet
	if err := decoder.Decode(&expSet); err != nil {
		if err != io.EOF {
			return fmt.Errorf("failed to decode experiment set: %w", err)
		}
		return nil
	}

	// Process each experiment in the set
	for _, exp := range expSet.Experiments {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Extract platform and instrument
		platform := ""
		instrument := ""
		if exp.Platform.Illumina != nil {
			platform = "ILLUMINA"
			instrument = exp.Platform.Illumina.InstrumentModel
		} else if exp.Platform.IonTorrent != nil {
			platform = "ION_TORRENT"
			instrument = exp.Platform.IonTorrent.InstrumentModel
		} else if exp.Platform.PacBio != nil {
			platform = "PACBIO_SMRT"
			instrument = exp.Platform.PacBio.InstrumentModel
		}

		// Convert to database model
		dbExp := database.Experiment{
			ExperimentAccession: exp.Accession,
			StudyAccession:      exp.StudyRef.Accession,
			Title:               exp.Title,
			LibraryStrategy:     exp.Design.LibraryDescriptor.LibraryStrategy,
			LibrarySource:       exp.Design.LibraryDescriptor.LibrarySource,
			Platform:            platform,
			InstrumentModel:     instrument,
			Metadata:            "{}",
		}

		batch = append(batch, dbExp)

		// Insert batch when full
		if len(batch) >= 5000 { // Optimized batch size
			if err := sp.db.BatchInsertExperiments(batch); err != nil {
				return fmt.Errorf("failed to insert experiments: %w", err)
			}
			sp.recordsInserted.Add(int64(len(batch)))
			batch = batch[:0]
		}
	}

	// Insert remaining batch
	if len(batch) > 0 {
		if err := sp.db.BatchInsertExperiments(batch); err != nil {
			return fmt.Errorf("failed to insert final experiments batch: %w", err)
		}
		sp.recordsInserted.Add(int64(len(batch)))
	}

	return nil
}

// processStudies streams and processes study records
func (sp *StreamProcessor) processStudies(ctx context.Context, decoder *xml.Decoder) error {
	// Decode the entire StudySet
	var studySet parser.StudySet
	if err := decoder.Decode(&studySet); err != nil {
		if err != io.EOF {
			return fmt.Errorf("failed to decode study set: %w", err)
		}
		return nil
	}

	// Process each study in the set
	for _, study := range studySet.Studies {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Extract study type
		studyType := ""
		if study.Descriptor.StudyType != nil {
			studyType = study.Descriptor.StudyType.ExistingStudyType
			if studyType == "" && study.Descriptor.StudyType.NewStudyType != "" {
				studyType = study.Descriptor.StudyType.NewStudyType
			}
		}

		// Convert to database model
		dbStudy := database.Study{
			StudyAccession: study.Accession,
			StudyTitle:     study.Descriptor.StudyTitle,
			StudyAbstract:  study.Descriptor.StudyAbstract,
			StudyType:      studyType,
			Metadata:       "{}",
		}

		if err := sp.db.InsertStudy(&dbStudy); err != nil {
			// Log but continue
			fmt.Printf("Warning: failed to insert study %s: %v\n", study.Accession, err)
			continue
		}

		sp.recordsInserted.Add(1)
	}

	return nil
}

// processSamples streams and processes sample records
func (sp *StreamProcessor) processSamples(ctx context.Context, decoder *xml.Decoder) error {
	// Decode the entire SampleSet
	var sampleSet parser.SampleSet
	if err := decoder.Decode(&sampleSet); err != nil {
		if err != io.EOF {
			return fmt.Errorf("failed to decode sample set: %w", err)
		}
		return nil
	}

	// Process each sample in the set
	for _, sample := range sampleSet.Samples {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Convert to database model
		dbSample := database.Sample{
			SampleAccession: sample.Accession,
			ScientificName:  sample.SampleName.ScientificName,
			TaxonID:         sample.SampleName.TaxonID,
			Description:     sample.Description,
			Metadata:        "{}",
		}

		// Extract organism from attributes
		if sample.SampleAttributes != nil {
			for _, attr := range sample.SampleAttributes.Attributes {
				switch attr.Tag {
				case "organism":
					dbSample.Organism = attr.Value
				case "tissue":
					dbSample.Tissue = attr.Value
				case "cell_type":
					dbSample.CellType = attr.Value
				}
			}
		}

		if err := sp.db.InsertSample(&dbSample); err != nil {
			fmt.Printf("Warning: failed to insert sample %s: %v\n", sample.Accession, err)
			continue
		}

		sp.recordsInserted.Add(1)
	}

	return nil
}

// processRuns streams and processes run records
func (sp *StreamProcessor) processRuns(ctx context.Context, decoder *xml.Decoder) error {
	// Decode the entire RunSet
	var runSet parser.RunSet
	if err := decoder.Decode(&runSet); err != nil {
		if err != io.EOF {
			return fmt.Errorf("failed to decode run set: %w", err)
		}
		return nil
	}

	// Process each run in the set
	for _, r := range runSet.Runs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Extract statistics safely
		totalSpots := int64(0)
		totalBases := int64(0)
		if r.Statistics != nil {
			totalSpots = r.Statistics.TotalSpots
			totalBases = r.Statistics.TotalBases
		}

		// Convert to database model
		dbRun := database.Run{
			RunAccession:        r.Accession,
			ExperimentAccession: r.ExperimentRef.Accession,
			TotalSpots:          totalSpots,
			TotalBases:          totalBases,
			Published:           "", // Empty string as we changed this to string type
			Metadata:            "{}",
		}

		if err := sp.db.InsertRun(&dbRun); err != nil {
			fmt.Printf("Warning: failed to insert run %s: %v\n", r.Accession, err)
			continue
		}

		sp.recordsInserted.Add(1)
	}

	return nil
}

// updateProgress updates and reports progress
func (sp *StreamProcessor) updateProgress(currentFile string) {
	if sp.progressFunc == nil {
		return
	}

	bytesProcessed := sp.bytesProcessed.Load()
	recordsProcessed := sp.recordsInserted.Load()
	elapsed := time.Since(sp.startTime)

	var percentComplete float64
	var estimatedRemaining time.Duration

	if sp.totalBytes > 0 {
		percentComplete = float64(bytesProcessed) / float64(sp.totalBytes) * 100
		if bytesProcessed > 0 {
			totalTime := elapsed.Seconds() * float64(sp.totalBytes) / float64(bytesProcessed)
			estimatedRemaining = time.Duration(totalTime-elapsed.Seconds()) * time.Second
		}
	}

	bytesPerSecond := float64(bytesProcessed) / elapsed.Seconds()

	sp.progressFunc(Progress{
		BytesProcessed:         bytesProcessed,
		TotalBytes:             sp.totalBytes,
		RecordsProcessed:       recordsProcessed,
		CurrentFile:            currentFile,
		PercentComplete:        percentComplete,
		BytesPerSecond:         bytesPerSecond,
		TimeElapsed:            elapsed,
		EstimatedTimeRemaining: estimatedRemaining,
	})
}

// countingReader wraps an io.Reader and counts bytes read
type countingReader struct {
	reader   io.Reader
	counter  *atomic.Int64
	callback func(string)
}

func (cr *countingReader) Read(p []byte) (n int, err error) {
	n, err = cr.reader.Read(p)
	if n > 0 {
		cr.counter.Add(int64(n))
		if cr.callback != nil {
			cr.callback("")
		}
	}
	return n, err
}

// GetStats returns processing statistics
func (sp *StreamProcessor) GetStats() map[string]interface{} {
	elapsed := time.Since(sp.startTime)
	bytesProcessed := sp.bytesProcessed.Load()
	recordsProcessed := sp.recordsInserted.Load()

	return map[string]interface{}{
		"bytes_processed":    bytesProcessed,
		"records_processed":  recordsProcessed,
		"elapsed_time":       elapsed.String(),
		"bytes_per_second":   float64(bytesProcessed) / elapsed.Seconds(),
		"records_per_second": float64(recordsProcessed) / elapsed.Seconds(),
	}
}
