package search

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/embeddings"
)

// Syncer handles synchronization between SQLite and Bleve index
type Syncer struct {
	config   *config.Config
	db       *database.DB
	backend  SearchBackend
	embedder *embeddings.Embedder
	stopChan chan struct{}
	running  bool
}

// NewSyncer creates a new index syncer
func NewSyncer(cfg *config.Config, db *database.DB, backend SearchBackend) (*Syncer, error) {
	s := &Syncer{
		config:   cfg,
		db:       db,
		backend:  backend,
		stopChan: make(chan struct{}),
	}

	// Initialize embedder if vectors are enabled
	if cfg.IsVectorEnabled() {
		embConfig := &embeddings.EmbedderConfig{
			ModelsDir:    cfg.Embeddings.ModelsDirectory,
			DefaultModel: cfg.Embeddings.DefaultModel,
			BatchSize:    cfg.Embeddings.BatchSize,
			MaxLength:    cfg.Embeddings.MaxTextLength,
			NumThreads:   cfg.Embeddings.NumThreads,
			CacheEnabled: cfg.Embeddings.CacheEmbeddings,
		}

		embedder, err := embeddings.NewEmbedder(embConfig)
		if err != nil {
			log.Printf("Warning: failed to initialize embedder: %v", err)
			// Continue without embeddings
		} else {
			// Try to load default model
			if err := embedder.LoadDefaultModel(); err != nil {
				log.Printf("Warning: failed to load default model: %v", err)
			} else {
				s.embedder = embedder
			}
		}
	}

	return s, nil
}

// Start begins the synchronization process
func (s *Syncer) Start(ctx context.Context) error {
	if s.running {
		return fmt.Errorf("syncer is already running")
	}

	s.running = true

	// Perform initial sync if needed
	if s.config.Search.RebuildOnStart {
		if err := s.FullSync(ctx); err != nil {
			return fmt.Errorf("initial sync failed: %w", err)
		}
	}

	// Start background sync if auto-sync is enabled
	if s.config.Search.AutoSync {
		go s.backgroundSync(ctx)
	}

	return nil
}

// Stop stops the synchronization process
func (s *Syncer) Stop() {
	if s.running {
		close(s.stopChan)
		s.running = false
	}
}

// backgroundSync runs periodic synchronization
func (s *Syncer) backgroundSync(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(s.config.Search.SyncInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := s.IncrementalSync(ctx); err != nil {
				log.Printf("Sync error: %v", err)
			}
		}
	}
}

// FullSync performs a complete rebuild of the search index from SQLite
func (s *Syncer) FullSync(ctx context.Context) error {
	log.Println("Starting full index sync...")

	// Rebuild the index
	if err := s.backend.Rebuild(ctx); err != nil {
		return fmt.Errorf("failed to rebuild index: %w", err)
	}

	// Index all studies
	if err := s.IndexStudies(ctx); err != nil {
		return fmt.Errorf("failed to index studies: %w", err)
	}

	// Index all experiments
	if err := s.IndexExperiments(ctx); err != nil {
		return fmt.Errorf("failed to index experiments: %w", err)
	}

	// Index all samples
	if err := s.IndexSamples(ctx); err != nil {
		return fmt.Errorf("failed to index samples: %w", err)
	}

	// Index all runs
	if err := s.IndexRuns(ctx); err != nil {
		return fmt.Errorf("failed to index runs: %w", err)
	}

	log.Println("Full index sync completed")
	return nil
}

// IncrementalSync performs incremental synchronization of changes
func (s *Syncer) IncrementalSync(ctx context.Context) error {
	// TODO: Implement change tracking
	// For now, we'll do a simple check for new records
	// In production, use timestamps or a change log table

	// Get last sync time (stored in index metadata or config)
	// For now, just sync recent records
	since := time.Now().Add(-time.Duration(s.config.Search.SyncInterval) * time.Second * 2)

	// Index recent studies
	if err := s.indexStudiesSince(ctx, since); err != nil {
		return fmt.Errorf("failed to sync studies: %w", err)
	}

	return nil
}

// IndexStudies indexes all studies from the database
func (s *Syncer) IndexStudies(ctx context.Context) error {
	query := `
		SELECT study_accession, study_title, study_abstract, study_type,
		       organism, submission_date
		FROM studies
		LIMIT ? OFFSET ?
	`

	batchSize := s.config.Search.BatchSize
	offset := 0
	totalIndexed := 0
	batchesSinceFlush := 0
	flushInterval := 10 // Flush every 10 batches

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rows, err := s.db.Query(query, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to query studies: %w", err)
		}

		docs := make([]interface{}, 0, batchSize)
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
				rows.Close()
				return fmt.Errorf("failed to scan study: %w", err)
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

			// Generate embedding if embedder is available
			if s.embedder != nil && s.embedder.IsModelLoaded() {
				text := embeddings.PrepareTextForEmbedding(
					study.Organism.String,
					"", // No library strategy for studies
					study.Title.String,
					study.Abstract.String,
				)

				if embedding, err := s.embedder.EmbedText(text); err == nil {
					doc["embedding"] = embedding
				}
			}

			docs = append(docs, doc)
			count++
		}
		rows.Close()

		if count == 0 {
			break // No more records
		}

		// Index batch
		if err := s.backend.IndexBatch(docs); err != nil {
			return fmt.Errorf("failed to index batch: %w", err)
		}

		totalIndexed += count
		offset += batchSize
		batchesSinceFlush++

		// Periodically flush the index for progressive saving
		if batchesSinceFlush >= flushInterval {
			log.Printf("Flushing index after %d batches (%d documents indexed so far)", batchesSinceFlush, totalIndexed)
			if err := s.backend.Flush(); err != nil {
				log.Printf("Warning: failed to flush index: %v", err)
			}
			batchesSinceFlush = 0
		}

		if count < batchSize {
			break // Last batch
		}
	}

	log.Printf("Indexed %d studies", totalIndexed)
	return nil
}

// IndexExperiments indexes all experiments from the database
func (s *Syncer) IndexExperiments(ctx context.Context) error {
	query := `
		SELECT experiment_accession, title, library_strategy,
		       platform, instrument_model
		FROM experiments
		LIMIT ? OFFSET ?
	`

	batchSize := s.config.Search.BatchSize
	offset := 0
	totalIndexed := 0
	batchesSinceFlush := 0
	flushInterval := 10 // Flush every 10 batches

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rows, err := s.db.Query(query, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to query experiments: %w", err)
		}

		docs := make([]interface{}, 0, batchSize)
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
				rows.Close()
				return fmt.Errorf("failed to scan experiment: %w", err)
			}

			doc := map[string]interface{}{
				"id":               exp.Accession,
				"type":             "experiment",
				"title":            exp.Title.String,
				"library_strategy": exp.LibraryStrategy.String,
				"platform":         exp.Platform.String,
				"instrument_model": exp.InstrumentModel.String,
			}

			// Generate embedding if embedder is available
			if s.embedder != nil && s.embedder.IsModelLoaded() {
				text := embeddings.PrepareTextForEmbedding(
					"", // No organism at experiment level
					exp.LibraryStrategy.String,
					exp.Title.String,
					"", // No abstract for experiments
				)

				if embedding, err := s.embedder.EmbedText(text); err == nil {
					doc["embedding"] = embedding
				}
			}

			docs = append(docs, doc)
			count++
		}
		rows.Close()

		if count == 0 {
			break // No more records
		}

		// Index batch
		if err := s.backend.IndexBatch(docs); err != nil {
			return fmt.Errorf("failed to index batch: %w", err)
		}

		totalIndexed += count
		offset += batchSize
		batchesSinceFlush++

		// Periodically flush the index for progressive saving
		if batchesSinceFlush >= flushInterval {
			log.Printf("Flushing index after %d batches (%d experiments indexed so far)", batchesSinceFlush, totalIndexed)
			if err := s.backend.Flush(); err != nil {
				log.Printf("Warning: failed to flush index: %v", err)
			}
			batchesSinceFlush = 0
		}

		if count < batchSize {
			break // Last batch
		}
	}

	log.Printf("Indexed %d experiments", totalIndexed)
	return nil
}

// IndexSamples indexes all samples from the database
func (s *Syncer) IndexSamples(ctx context.Context) error {
	query := `
		SELECT sample_accession, organism, scientific_name,
		       tissue, cell_type, description
		FROM samples
		LIMIT ? OFFSET ?
	`

	batchSize := s.config.Search.BatchSize
	offset := 0
	totalIndexed := 0
	batchesSinceFlush := 0
	flushInterval := 10 // Flush every 10 batches

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rows, err := s.db.Query(query, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to query samples: %w", err)
		}

		docs := make([]interface{}, 0, batchSize)
		count := 0

		for rows.Next() {
			var sample struct {
				Accession      string
				Organism       sql.NullString
				ScientificName sql.NullString
				Tissue         sql.NullString
				CellType       sql.NullString
				Description    sql.NullString
			}

			if err := rows.Scan(&sample.Accession, &sample.Organism, &sample.ScientificName,
				&sample.Tissue, &sample.CellType, &sample.Description); err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan sample: %w", err)
			}

			doc := map[string]interface{}{
				"id":              sample.Accession,
				"type":            "sample",
				"organism":        sample.Organism.String,
				"scientific_name": sample.ScientificName.String,
				"tissue":          sample.Tissue.String,
				"cell_type":       sample.CellType.String,
				"description":     sample.Description.String,
			}

			// Generate embedding if embedder is available
			if s.embedder != nil && s.embedder.IsModelLoaded() {
				text := embeddings.PrepareTextForEmbedding(
					sample.Organism.String,
					"", // No library strategy for samples
					sample.Tissue.String+" "+sample.CellType.String,
					sample.Description.String,
				)

				if embedding, err := s.embedder.EmbedText(text); err == nil {
					doc["embedding"] = embedding
				}
			}

			docs = append(docs, doc)
			count++
		}
		rows.Close()

		if count == 0 {
			break // No more records
		}

		// Index batch
		if err := s.backend.IndexBatch(docs); err != nil {
			return fmt.Errorf("failed to index batch: %w", err)
		}

		totalIndexed += count
		offset += batchSize
		batchesSinceFlush++

		// Periodically flush the index for progressive saving
		if batchesSinceFlush >= flushInterval {
			log.Printf("Flushing index after %d batches (%d samples indexed so far)", batchesSinceFlush, totalIndexed)
			if err := s.backend.Flush(); err != nil {
				log.Printf("Warning: failed to flush index: %v", err)
			}
			batchesSinceFlush = 0
		}

		if count < batchSize {
			break // Last batch
		}
	}

	log.Printf("Indexed %d samples", totalIndexed)
	return nil
}

// IndexRuns indexes all runs from the database
func (s *Syncer) IndexRuns(ctx context.Context) error {
	query := `
		SELECT run_accession, total_spots, total_bases
		FROM runs
		LIMIT ? OFFSET ?
	`

	batchSize := s.config.Search.BatchSize
	offset := 0
	totalIndexed := 0
	batchesSinceFlush := 0
	flushInterval := 10 // Flush every 10 batches

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rows, err := s.db.Query(query, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to query runs: %w", err)
		}

		docs := make([]interface{}, 0, batchSize)
		count := 0

		for rows.Next() {
			var run struct {
				Accession string
				Spots     sql.NullInt64
				Bases     sql.NullInt64
			}

			if err := rows.Scan(&run.Accession, &run.Spots, &run.Bases); err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan run: %w", err)
			}

			doc := map[string]interface{}{
				"id":   run.Accession,
				"type": "run",
			}

			if run.Spots.Valid {
				doc["spots"] = run.Spots.Int64
			}
			if run.Bases.Valid {
				doc["bases"] = run.Bases.Int64
			}

			docs = append(docs, doc)
			count++
		}
		rows.Close()

		if count == 0 {
			break // No more records
		}

		// Index batch
		if err := s.backend.IndexBatch(docs); err != nil {
			return fmt.Errorf("failed to index batch: %w", err)
		}

		totalIndexed += count
		offset += batchSize
		batchesSinceFlush++

		// Periodically flush the index for progressive saving
		if batchesSinceFlush >= flushInterval {
			log.Printf("Flushing index after %d batches (%d runs indexed so far)", batchesSinceFlush, totalIndexed)
			if err := s.backend.Flush(); err != nil {
				log.Printf("Warning: failed to flush index: %v", err)
			}
			batchesSinceFlush = 0
		}

		if count < batchSize {
			break // Last batch
		}
	}

	log.Printf("Indexed %d runs", totalIndexed)
	return nil
}

// indexStudiesSince indexes studies modified since a given time
func (s *Syncer) indexStudiesSince(ctx context.Context, since time.Time) error {
	// TODO: Implement incremental sync based on modification time
	// For now, this is a placeholder
	return nil
}

// GetEmbedder returns the embedder instance
func (s *Syncer) GetEmbedder() *embeddings.Embedder {
	return s.embedder
}

// SetEmbedder sets a custom embedder
func (s *Syncer) SetEmbedder(embedder *embeddings.Embedder) {
	s.embedder = embedder
}