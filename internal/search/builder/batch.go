package builder

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// BatchProcessor handles batch processing of documents
type BatchProcessor struct {
	builder        *IndexBuilder
	batchSize      int
	progressChan   chan<- ProgressUpdate
	checkpointFunc func() bool
}

// ProcessDocumentType processes all documents of a given type in batches
func (b *IndexBuilder) ProcessDocumentType(ctx context.Context, docType string) error {
	// Initialize type progress if not exists
	if _, exists := b.progress.TypeProgress[docType]; !exists {
		b.progress.TypeProgress[docType] = &TypeProgress{
			Type:      docType,
			StartTime: time.Now(),
		}
	}

	typeProgress := b.progress.TypeProgress[docType]

	// We don't count total upfront - just process until no more rows
	typeProgress.TotalDocs = 0 // Will be updated as we process

	// Process in batches
	offset := typeProgress.LastOffset
	batchNum := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b.stopChan:
			return fmt.Errorf("stopped by user")
		case <-b.pauseChan:
			// Handle pause
			b.handlePause()
		default:
		}

		// Process batch
		processed, err := b.processBatch(ctx, docType, offset, b.options.BatchSize)
		if err != nil {
			return fmt.Errorf("failed to process batch %d: %w", batchNum, err)
		}

		// Update progress
		typeProgress.ProcessedDocs += int64(processed)
		typeProgress.IndexedDocs += int64(processed)
		typeProgress.LastOffset = offset + int64(processed)

		b.progress.ProcessedDocs += int64(processed)
		b.progress.IndexedDocs += int64(processed)
		b.progress.CurrentBatch++

		// Save progress periodically
		if b.progress.CurrentBatch%10 == 0 {
			if err := b.progress.Save(); err != nil {
				return fmt.Errorf("failed to save progress: %w", err)
			}
		}

		// Send progress update
		b.progressChan <- ProgressUpdate{
			Type:      UpdateTypeBatchComplete,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"type":      docType,
				"batch":     batchNum,
				"processed": processed,
				"offset":    offset,
				"total":     typeProgress.ProcessedDocs,
			},
		}

		// If we got fewer rows than batch size, we're done
		if processed < b.options.BatchSize {
			break
		}

		offset += int64(processed)
		batchNum++

		// Check if we should create a checkpoint
		if b.shouldCreateCheckpoint() {
			if err := b.createCheckpoint(); err != nil {
				b.errorChan <- fmt.Errorf("checkpoint failed: %w", err)
			}
		}
	}

	// Mark type as completed
	typeProgress.Completed = true
	now := time.Now()
	typeProgress.EndTime = &now

	return nil
}

// processBatch processes a single batch of documents
func (b *IndexBuilder) processBatch(ctx context.Context, docType string, offset int64, limit int) (int, error) {
	switch docType {
	case "studies":
		return b.processStudiesBatch(ctx, offset, limit)
	case "experiments":
		return b.processExperimentsBatch(ctx, offset, limit)
	case "samples":
		return b.processSamplesBatch(ctx, offset, limit)
	case "runs":
		return b.processRunsBatch(ctx, offset, limit)
	default:
		return 0, fmt.Errorf("unknown document type: %s", docType)
	}
}

// processStudiesBatch processes a batch of studies
func (b *IndexBuilder) processStudiesBatch(ctx context.Context, offset int64, limit int) (int, error) {
	query := `
		SELECT study_accession, study_title, study_abstract, study_type,
		       organism, submission_date
		FROM studies
		LIMIT ? OFFSET ?
	`

	rows, err := b.db.Query(query, limit, offset)
	if err != nil {
		return 0, fmt.Errorf("failed to query studies: %w", err)
	}
	defer rows.Close()

	docs := make([]interface{}, 0, limit)
	texts := make([]string, 0, limit) // For batch embedding
	count := 0

	for rows.Next() {
		var study struct {
			Accession      string
			Title          sql.NullString
			Abstract       sql.NullString
			Type           sql.NullString
			Organism       sql.NullString
			SubmissionDate sql.NullTime
		}

		if err := rows.Scan(&study.Accession, &study.Title, &study.Abstract,
			&study.Type, &study.Organism, &study.SubmissionDate); err != nil {
			return count, fmt.Errorf("failed to scan study: %w", err)
		}

		doc := map[string]interface{}{
			"id":       study.Accession,
			"type":     "study",
			"title":    study.Title.String,
			"abstract": study.Abstract.String,
			"organism": study.Organism.String,
		}

		if study.Type.Valid {
			doc["study_type"] = study.Type.String
		}

		if study.SubmissionDate.Valid {
			doc["submission_date"] = study.SubmissionDate.Time
		}

		// Prepare text for embedding if enabled
		if b.isEmbeddingEnabled() {
			text := prepareStudyText(study.Title.String, study.Abstract.String)
			texts = append(texts, text)
		}

		docs = append(docs, doc)
		count++

		// Update last document ID for progress tracking
		b.progress.LastDocumentID = study.Accession
	}

	// Generate embeddings if enabled
	if b.isEmbeddingEnabled() && len(texts) > 0 {
		embeddings, err := b.generateBatchEmbeddings(texts)
		if err != nil {
			// Log error but continue without embeddings
			fmt.Printf("Warning: Failed to generate embeddings for studies batch: %v\n", err)
		} else {
			// Add embeddings to documents
			for i, doc := range docs {
				if i < len(embeddings) && embeddings[i] != nil {
					docMap := doc.(map[string]interface{})
					docMap["embedding"] = embeddings[i]
				}
			}
		}
	}

	// Index the batch
	if count > 0 {
		if err := b.backend.IndexBatch(docs); err != nil {
			return count, fmt.Errorf("failed to index batch: %w", err)
		}
	}

	return count, nil
}

// processExperimentsBatch processes a batch of experiments
func (b *IndexBuilder) processExperimentsBatch(ctx context.Context, offset int64, limit int) (int, error) {
	query := `
		SELECT experiment_accession, title, library_strategy,
		       platform, instrument_model
		FROM experiments
		LIMIT ? OFFSET ?
	`

	rows, err := b.db.Query(query, limit, offset)
	if err != nil {
		return 0, fmt.Errorf("failed to query experiments: %w", err)
	}
	defer rows.Close()

	docs := make([]interface{}, 0, limit)
	texts := make([]string, 0, limit) // For batch embedding
	count := 0

	for rows.Next() {
		var exp struct {
			Accession       string
			Title           sql.NullString
			LibraryStrategy sql.NullString
			Platform        sql.NullString
			InstrumentModel sql.NullString
		}

		if err := rows.Scan(&exp.Accession, &exp.Title, &exp.LibraryStrategy,
			&exp.Platform, &exp.InstrumentModel); err != nil {
			return count, fmt.Errorf("failed to scan experiment: %w", err)
		}

		doc := map[string]interface{}{
			"id":               exp.Accession,
			"type":             "experiment",
			"title":            exp.Title.String,
			"library_strategy": exp.LibraryStrategy.String,
			"platform":         exp.Platform.String,
			"instrument_model": exp.InstrumentModel.String,
		}

		// Prepare text for embedding if enabled
		if b.isEmbeddingEnabled() {
			text := prepareExperimentText(exp.Title.String, exp.LibraryStrategy.String, exp.Platform.String)
			texts = append(texts, text)
		}

		docs = append(docs, doc)
		count++

		// Update last document ID
		b.progress.LastDocumentID = exp.Accession
	}

	// Generate embeddings if enabled
	if b.isEmbeddingEnabled() && len(texts) > 0 {
		embeddings, err := b.generateBatchEmbeddings(texts)
		if err != nil {
			// Log error but continue without embeddings
			fmt.Printf("Warning: Failed to generate embeddings for experiments batch: %v\n", err)
		} else {
			// Add embeddings to documents
			for i, doc := range docs {
				if i < len(embeddings) && embeddings[i] != nil {
					docMap := doc.(map[string]interface{})
					docMap["embedding"] = embeddings[i]
				}
			}
		}
	}

	// Index the batch
	if count > 0 {
		if err := b.backend.IndexBatch(docs); err != nil {
			return count, fmt.Errorf("failed to index batch: %w", err)
		}
	}

	return count, nil
}

// processSamplesBatch processes a batch of samples
func (b *IndexBuilder) processSamplesBatch(ctx context.Context, offset int64, limit int) (int, error) {
	query := `
		SELECT sample_accession, description, organism, scientific_name
		FROM samples
		LIMIT ? OFFSET ?
	`

	rows, err := b.db.Query(query, limit, offset)
	if err != nil {
		return 0, fmt.Errorf("failed to query samples: %w", err)
	}
	defer rows.Close()

	docs := make([]interface{}, 0, limit)
	texts := make([]string, 0, limit) // For batch embedding
	count := 0

	for rows.Next() {
		var sample struct {
			Accession      string
			Description    sql.NullString
			Organism       sql.NullString
			ScientificName sql.NullString
		}

		if err := rows.Scan(&sample.Accession, &sample.Description,
			&sample.Organism, &sample.ScientificName); err != nil {
			return count, fmt.Errorf("failed to scan sample: %w", err)
		}

		doc := map[string]interface{}{
			"id":          sample.Accession,
			"type":        "sample",
			"description": sample.Description.String,
			"organism":    sample.Organism.String,
		}

		if sample.ScientificName.Valid {
			doc["scientific_name"] = sample.ScientificName.String
		}

		// Prepare text for embedding if enabled
		if b.isEmbeddingEnabled() {
			text := prepareSampleText(sample.Description.String, sample.Organism.String, sample.ScientificName.String)
			texts = append(texts, text)
		}

		docs = append(docs, doc)
		count++

		// Update last document ID
		b.progress.LastDocumentID = sample.Accession
	}

	// Generate embeddings if enabled
	if b.isEmbeddingEnabled() && len(texts) > 0 {
		embeddings, err := b.generateBatchEmbeddings(texts)
		if err != nil {
			// Log error but continue without embeddings
			fmt.Printf("Warning: Failed to generate embeddings for samples batch: %v\n", err)
		} else {
			// Add embeddings to documents
			for i, doc := range docs {
				if i < len(embeddings) && embeddings[i] != nil {
					docMap := doc.(map[string]interface{})
					docMap["embedding"] = embeddings[i]
				}
			}
		}
	}

	// Index the batch
	if count > 0 {
		if err := b.backend.IndexBatch(docs); err != nil {
			return count, fmt.Errorf("failed to index batch: %w", err)
		}
	}

	return count, nil
}

// processRunsBatch processes a batch of runs
func (b *IndexBuilder) processRunsBatch(ctx context.Context, offset int64, limit int) (int, error) {
	query := `
		SELECT run_accession, published, total_spots, total_bases
		FROM runs
		LIMIT ? OFFSET ?
	`

	rows, err := b.db.Query(query, limit, offset)
	if err != nil {
		return 0, fmt.Errorf("failed to query runs: %w", err)
	}
	defer rows.Close()

	docs := make([]interface{}, 0, limit)
	count := 0

	for rows.Next() {
		var run struct {
			Accession  string
			Published  sql.NullString
			TotalSpots sql.NullInt64
			TotalBases sql.NullInt64
		}

		if err := rows.Scan(&run.Accession, &run.Published,
			&run.TotalSpots, &run.TotalBases); err != nil {
			return count, fmt.Errorf("failed to scan run: %w", err)
		}

		doc := map[string]interface{}{
			"id":   run.Accession,
			"type": "run",
		}

		if run.Published.Valid && run.Published.String != "" {
			doc["published"] = run.Published.String
		}

		if run.TotalSpots.Valid {
			doc["total_spots"] = run.TotalSpots.Int64
		}

		if run.TotalBases.Valid {
			doc["total_bases"] = run.TotalBases.Int64
		}

		docs = append(docs, doc)
		count++

		// Update last document ID
		b.progress.LastDocumentID = run.Accession
	}

	// Index the batch
	if count > 0 {
		if err := b.backend.IndexBatch(docs); err != nil {
			return count, fmt.Errorf("failed to index batch: %w", err)
		}
	}

	return count, nil
}

// Note: We removed getDocumentTypeCount to avoid slow COUNT queries
// The loop now continues until it gets fewer rows than batch size

// handlePause handles the pause state
func (b *IndexBuilder) handlePause() {
	b.mu.Lock()
	b.state = StatePaused
	b.mu.Unlock()

	// Wait for resume signal
	<-b.resumeChan

	b.mu.Lock()
	b.state = StateRunning
	b.mu.Unlock()
}
