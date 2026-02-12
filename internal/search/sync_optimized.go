package search

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// allowedSyncTables is the whitelist of tables that can be synced.
// This prevents SQL injection by ensuring only known tables are queried.
var allowedSyncTables = map[string]bool{
	"studies":     true,
	"experiments": true,
	"samples":     true,
	"runs":        true,
	"study":       true, // Alternate naming convention
	"experiment":  true,
	"sample":      true,
	"run":         true,
}

// validateSyncTable checks if a table name is allowed for syncing.
func validateSyncTable(table string) error {
	if !allowedSyncTables[table] {
		return fmt.Errorf("invalid sync table name: %q", table)
	}
	return nil
}

// OptimizedSyncer provides high-performance synchronization
type OptimizedSyncer struct {
	*Syncer
	workers    int
	checkpoint map[string]string // Track checksums for incremental sync
	mu         sync.RWMutex
	progress   atomic.Int64
}

// NewOptimizedSyncer creates an optimized syncer with parallel processing
func NewOptimizedSyncer(s *Syncer) *OptimizedSyncer {
	return &OptimizedSyncer{
		Syncer:     s,
		workers:    runtime.NumCPU(),
		checkpoint: make(map[string]string),
	}
}

// workItem represents a unit of work for parallel processing
type workItem struct {
	table string
	batch []interface{}
}

// ParallelFullSync performs a full sync with parallel processing
func (o *OptimizedSyncer) ParallelFullSync(ctx context.Context) error {
	start := time.Now()
	log.Printf("Starting parallel full sync with %d workers...", o.workers)

	// Create work channels
	workChan := make(chan workItem, o.workers*2)
	errChan := make(chan error, o.workers)

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < o.workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for item := range workChan {
				if err := o.processBatch(ctx, item.table, item.batch); err != nil {
					errChan <- fmt.Errorf("worker %d failed: %w", id, err)
					return
				}
				o.progress.Add(int64(len(item.batch)))
			}
		}(i)
	}

	// Process tables in parallel
	tables := []string{"study", "experiment", "sample", "run"}
	var totalErr error

	for _, table := range tables {
		if err := o.streamTable(ctx, table, workChan); err != nil {
			totalErr = err
			break
		}
	}

	// Close work channel and wait for workers
	close(workChan)
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if totalErr == nil {
			totalErr = err
		}
	}

	elapsed := time.Since(start)
	processed := o.progress.Load()
	log.Printf("Parallel sync completed: %d documents in %v (%.0f docs/sec)",
		processed, elapsed, float64(processed)/elapsed.Seconds())

	return totalErr
}

// streamTable streams table data to workers.
// The table name is validated against allowedSyncTables to prevent SQL injection.
func (o *OptimizedSyncer) streamTable(ctx context.Context, table string, workChan chan<- workItem) error {
	// Validate table name to prevent SQL injection
	if err := validateSyncTable(table); err != nil {
		return fmt.Errorf("streamTable: %w", err)
	}

	query := fmt.Sprintf("SELECT * FROM %s", table)
	rows, err := o.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	batch := make([]interface{}, 0, o.config.Search.BatchSize)
	var scanSkipped int

	for rows.Next() {
		doc, err := o.scanTableRow(table, rows)
		if err != nil {
			scanSkipped++
			continue
		}

		batch = append(batch, doc)

		if len(batch) >= o.config.Search.BatchSize {
			select {
			case workChan <- workItem{table: table, batch: batch}:
				batch = make([]interface{}, 0, o.config.Search.BatchSize)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// Send remaining batch
	if len(batch) > 0 {
		workChan <- workItem{table: table, batch: batch}
	}

	if scanSkipped > 0 {
		log.Printf("Warning: skipped %d rows during %s table sync", scanSkipped, table)
	}

	return nil
}

// IncrementalSync performs an incremental sync using checksums
func (o *OptimizedSyncer) IncrementalSync(ctx context.Context) error {
	start := time.Now()
	log.Println("Starting incremental sync...")

	tables := []string{"study", "experiment", "sample", "run"}
	var totalUpdated int

	for _, table := range tables {
		updated, err := o.syncTableIncremental(ctx, table)
		if err != nil {
			return fmt.Errorf("incremental sync failed for %s: %w", table, err)
		}
		totalUpdated += updated
	}

	elapsed := time.Since(start)
	log.Printf("Incremental sync completed: %d updates in %v", totalUpdated, elapsed)

	return nil
}

// syncTableIncremental syncs only changed records.
// The table name is validated against allowedSyncTables to prevent SQL injection.
func (o *OptimizedSyncer) syncTableIncremental(ctx context.Context, table string) (int, error) {
	// Validate table name to prevent SQL injection
	if err := validateSyncTable(table); err != nil {
		return 0, fmt.Errorf("syncTableIncremental: %w", err)
	}

	// Get records with checksums
	query := fmt.Sprintf(`
		SELECT *,
		       MD5(CAST(COALESCE(%s_accession, '') ||
		                COALESCE(CAST(updated_at AS TEXT), '') AS TEXT)) as checksum
		FROM %s
		WHERE updated_at > datetime('now', '-1 hour')
	`, table, table)

	rows, err := o.db.QueryContext(ctx, query)
	if err != nil {
		// Fallback for tables without updated_at
		query = fmt.Sprintf("SELECT * FROM %s", table)
		rows, err = o.db.QueryContext(ctx, query)
		if err != nil {
			return 0, err
		}
	}
	defer rows.Close()

	var updated int
	var incrementalSkipped int
	batch := make([]interface{}, 0, 100)

	for rows.Next() {
		doc, err := o.scanTableRow(table, rows)
		if err != nil {
			incrementalSkipped++
			continue
		}

		// Calculate checksum
		checksum := o.calculateChecksum(doc)
		key := fmt.Sprintf("%s:%s", table, o.getDocID(doc))

		o.mu.RLock()
		oldChecksum, exists := o.checkpoint[key]
		o.mu.RUnlock()

		// Only index if new or changed
		if !exists || oldChecksum != checksum {
			batch = append(batch, doc)

			o.mu.Lock()
			o.checkpoint[key] = checksum
			o.mu.Unlock()

			updated++
		}

		// Process batch
		if len(batch) >= 100 {
			if err := o.processBatch(ctx, table, batch); err != nil {
				return updated, err
			}
			batch = make([]interface{}, 0, 100)
		}
	}

	// Process remaining
	if len(batch) > 0 {
		if err := o.processBatch(ctx, table, batch); err != nil {
			return updated, err
		}
	}

	if incrementalSkipped > 0 {
		log.Printf("Warning: skipped %d rows during %s incremental sync", incrementalSkipped, table)
	}

	return updated, nil
}

// processBatch processes a batch with embeddings if enabled
func (o *OptimizedSyncer) processBatch(ctx context.Context, table string, batch []interface{}) error {
	// Note: Embedding generation would be handled by the search manager's embedder
	// This is just the indexing part

	// Index batch
	return o.backend.IndexBatch(batch)
}

// calculateChecksum calculates a checksum for a document
func (o *OptimizedSyncer) calculateChecksum(doc interface{}) string {
	h := md5.New()
	fmt.Fprintf(h, "%+v", doc)
	return hex.EncodeToString(h.Sum(nil))
}

// getDocID extracts the document ID
func (o *OptimizedSyncer) getDocID(doc interface{}) string {
	switch d := doc.(type) {
	case StudyDoc:
		return d.StudyAccession
	case ExperimentDoc:
		return d.ExperimentAccession
	case SampleDoc:
		return d.SampleAccession
	case RunDoc:
		return d.RunAccession
	default:
		return ""
	}
}

// extractTextForEmbedding extracts text for embedding generation
func (o *OptimizedSyncer) extractTextForEmbedding(doc interface{}) string {
	switch d := doc.(type) {
	case StudyDoc:
		return fmt.Sprintf("%s %s %s", d.StudyTitle, d.StudyAbstract, d.Organism)
	case ExperimentDoc:
		return fmt.Sprintf("%s %s %s", d.Title, d.LibraryStrategy, d.Platform)
	case SampleDoc:
		return fmt.Sprintf("%s %s %s %s", d.Organism, d.ScientificName, d.Tissue, d.CellType)
	case RunDoc:
		return d.RunAccession
	default:
		return ""
	}
}

// addEmbeddingToDoc adds embedding to document
func (o *OptimizedSyncer) addEmbeddingToDoc(doc interface{}, embedding []float32) {
	// This would require modifying the document structs to include embedding field
	// For now, it's a placeholder
}

// scanTableRow scans a row based on table type
func (o *OptimizedSyncer) scanTableRow(table string, rows *sql.Rows) (interface{}, error) {
	switch table {
	case "study":
		var s StudyDoc
		s.Type = "study"
		// Scan logic here
		return s, rows.Scan(&s.StudyAccession, &s.StudyTitle, &s.StudyAbstract, &s.StudyType, &s.Organism)
	case "experiment":
		var e ExperimentDoc
		e.Type = "experiment"
		// Scan logic here
		return e, rows.Scan(&e.ExperimentAccession, &e.Title, &e.LibraryStrategy, &e.Platform, &e.InstrumentModel)
	case "sample":
		var s SampleDoc
		s.Type = "sample"
		// Scan logic here
		return s, rows.Scan(&s.SampleAccession, &s.Organism, &s.ScientificName, &s.Tissue, &s.CellType, &s.Description)
	case "run":
		var r RunDoc
		r.Type = "run"
		// Scan logic here
		return r, rows.Scan(&r.RunAccession, &r.Spots, &r.Bases)
	default:
		return nil, fmt.Errorf("unknown table: %s", table)
	}
}

// GetProgress returns sync progress information
func (o *OptimizedSyncer) GetProgress() SyncProgress {
	return SyncProgress{
		DocumentsProcessed: o.progress.Load(),
		TablesCompleted:    4, // Simplified
		StartTime:          time.Now(),
		EstimatedTime:      time.Duration(0),
	}
}

// SyncProgress tracks sync progress
type SyncProgress struct {
	DocumentsProcessed int64
	TablesCompleted    int
	StartTime          time.Time
	EstimatedTime      time.Duration
}
