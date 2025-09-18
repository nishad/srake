package builder

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/embeddings"
	"github.com/nishad/srake/internal/search"
)

// BuildState represents the current state of the index building process
type BuildState string

const (
	StateIdle       BuildState = "idle"
	StateRunning    BuildState = "running"
	StatePaused     BuildState = "paused"
	StateCompleted  BuildState = "completed"
	StateFailed     BuildState = "failed"
	StateResuming   BuildState = "resuming"
)

// BuildOptions configures the index building process
type BuildOptions struct {
	BatchSize          int           // Number of documents per batch
	CheckpointInterval int           // Documents between checkpoints
	MaxMemoryMB        int           // Maximum memory usage in MB
	NumWorkers         int           // Number of parallel workers
	WithEmbeddings     bool          // Generate embeddings during indexing
	EmbeddingModel     string        // Model for embeddings
	ProgressFile       string        // Path to progress file
	CheckpointDir      string        // Directory for checkpoints
	Resume             bool          // Resume from checkpoint
	Verbose            bool          // Verbose logging
	DryRun             bool          // Simulate without actual indexing
}

// IndexBuilder manages the index building process with progress tracking
type IndexBuilder struct {
	config   *config.Config
	db       *database.DB
	backend  search.SearchBackend
	syncer   *search.Syncer
	progress *Progress
	options  BuildOptions
	embedder *embeddings.SearchEmbedder

	// Runtime state
	mu           sync.RWMutex
	state        BuildState
	stopChan     chan struct{}
	pauseChan    chan struct{}
	resumeChan   chan struct{}
	errorChan    chan error
	progressChan chan ProgressUpdate

	// Metrics
	startTime    time.Time
	lastUpdate   time.Time
	docsPerSec   float64
	avgBatchTime time.Duration
	memoryUsage  int64
}

// NewIndexBuilder creates a new index builder
func NewIndexBuilder(cfg *config.Config, db *database.DB, backend search.SearchBackend, options BuildOptions) (*IndexBuilder, error) {
	// Set defaults
	if options.BatchSize == 0 {
		options.BatchSize = 500
	}
	if options.CheckpointInterval == 0 {
		options.CheckpointInterval = 10000
	}
	if options.MaxMemoryMB == 0 {
		options.MaxMemoryMB = 2048
	}
	if options.ProgressFile == "" {
		options.ProgressFile = ".srake/index-progress.json"
	}
	if options.CheckpointDir == "" {
		options.CheckpointDir = ".srake/checkpoints"
	}

	builder := &IndexBuilder{
		config:       cfg,
		db:           db,
		backend:      backend,
		options:      options,
		state:        StateIdle,
		stopChan:     make(chan struct{}),
		pauseChan:    make(chan struct{}),
		resumeChan:   make(chan struct{}),
		errorChan:    make(chan error, 10),
		progressChan: make(chan ProgressUpdate, 100),
	}

	// Create syncer for indexing logic
	syncer, err := search.NewSyncer(cfg, db, backend)
	if err != nil {
		return nil, fmt.Errorf("failed to create syncer: %w", err)
	}
	builder.syncer = syncer

	// Initialize embedder if embeddings are enabled
	if options.WithEmbeddings {
		// Configure embeddings in config if not already set
		if !cfg.Embeddings.Enabled {
			cfg.Embeddings.Enabled = true
			if options.EmbeddingModel != "" {
				cfg.Embeddings.DefaultModel = options.EmbeddingModel
			} else {
				cfg.Embeddings.DefaultModel = "Xenova/SapBERT-from-PubMedBERT-fulltext"
			}
			if cfg.Embeddings.ModelsDirectory == "" {
				cfg.Embeddings.ModelsDirectory = "/Users/nishad/.srake/models"
			}
		}

		embedder, err := embeddings.NewSearchEmbedder(cfg)
		if err != nil {
			// Log warning but continue without embeddings
			fmt.Printf("Warning: Failed to initialize embedder: %v\n", err)
		} else {
			builder.embedder = embedder
		}
	}

	// Load or create progress
	if options.Resume {
		progress, err := LoadProgress(options.ProgressFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load progress: %w", err)
		}
		builder.progress = progress
	} else {
		builder.progress = NewProgress(options)
	}

	return builder, nil
}

// Build starts the index building process
func (b *IndexBuilder) Build(ctx context.Context) error {
	b.mu.Lock()
	if b.state == StateRunning {
		b.mu.Unlock()
		return fmt.Errorf("builder is already running")
	}
	b.state = StateRunning
	b.startTime = time.Now()
	b.mu.Unlock()

	// Start progress monitor
	go b.monitorProgress()

	// Start checkpoint manager
	go b.manageCheckpoints()

	// Execute build
	err := b.executeBuild(ctx)

	// Update final state
	b.mu.Lock()
	if err != nil {
		b.state = StateFailed
		b.progress.SetError(err)
	} else {
		b.state = StateCompleted
	}
	b.mu.Unlock()

	// Save final progress
	if err := b.progress.Save(); err != nil {
		return fmt.Errorf("failed to save final progress: %w", err)
	}

	return err
}

// Resume resumes index building from a checkpoint
func (b *IndexBuilder) Resume(ctx context.Context) error {
	b.mu.Lock()
	if b.state == StateRunning {
		b.mu.Unlock()
		return fmt.Errorf("builder is already running")
	}
	b.state = StateResuming
	b.mu.Unlock()

	// Load checkpoint
	checkpoint, err := b.progress.GetLatestCheckpoint()
	if err != nil {
		return fmt.Errorf("failed to get checkpoint: %w", err)
	}

	if checkpoint == nil {
		return fmt.Errorf("no checkpoint found to resume from")
	}

	// Restore index from checkpoint
	if err := b.restoreFromCheckpoint(checkpoint); err != nil {
		return fmt.Errorf("failed to restore from checkpoint: %w", err)
	}

	// Continue building from checkpoint
	b.mu.Lock()
	b.state = StateRunning
	b.mu.Unlock()

	return b.Build(ctx)
}

// Pause pauses the index building process
func (b *IndexBuilder) Pause() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != StateRunning {
		return fmt.Errorf("builder is not running")
	}

	b.state = StatePaused
	close(b.pauseChan)
	return nil
}

// Stop stops the index building process
func (b *IndexBuilder) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != StateRunning && b.state != StatePaused {
		return fmt.Errorf("builder is not running")
	}

	// Clean up embedder if initialized
	if b.embedder != nil {
		b.embedder.Close()
		b.embedder = nil
	}

	close(b.stopChan)
	b.state = StateIdle
	return nil
}

// GetProgress returns the current progress
func (b *IndexBuilder) GetProgress() *Progress {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.progress
}

// GetState returns the current state
func (b *IndexBuilder) GetState() BuildState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// executeBuild performs the actual index building
func (b *IndexBuilder) executeBuild(ctx context.Context) error {
	// We don't count total documents upfront to avoid slow COUNT queries
	// Progress will show documents processed so far
	b.progress.TotalDocuments = 0

	// Index each document type
	docTypes := []string{"studies", "experiments", "samples", "runs"}

	for _, docType := range docTypes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b.stopChan:
			return fmt.Errorf("build stopped by user")
		default:
		}

		// Check if we should skip this type (already completed)
		if b.progress.IsTypeCompleted(docType) {
			continue
		}

		// Index documents of this type
		if err := b.indexDocumentType(ctx, docType); err != nil {
			return fmt.Errorf("failed to index %s: %w", docType, err)
		}

		// Mark type as completed
		b.progress.MarkTypeCompleted(docType)

		// Save progress
		if err := b.progress.Save(); err != nil {
			return fmt.Errorf("failed to save progress: %w", err)
		}
	}

	return nil
}

// indexDocumentType indexes all documents of a specific type
func (b *IndexBuilder) indexDocumentType(ctx context.Context, docType string) error {
	// Use the new batch processing with progress tracking
	return b.ProcessDocumentType(ctx, docType)
}

// Note: We removed getTotalDocumentCount to avoid slow COUNT queries on large tables
// Progress tracking now just shows documents processed so far

// monitorProgress monitors and reports progress
func (b *IndexBuilder) monitorProgress() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopChan:
			return
		case <-ticker.C:
			b.updateMetrics()
			b.broadcastProgress()
		case update := <-b.progressChan:
			b.handleProgressUpdate(update)
		}
	}
}

// manageCheckpoints manages checkpoint creation
func (b *IndexBuilder) manageCheckpoints() {
	for {
		select {
		case <-b.stopChan:
			return
		default:
			// Check if checkpoint is needed
			if b.shouldCreateCheckpoint() {
				if err := b.createCheckpoint(); err != nil {
					b.errorChan <- fmt.Errorf("checkpoint failed: %w", err)
				}
			}
			time.Sleep(10 * time.Second)
		}
	}
}

// shouldCreateCheckpoint determines if a checkpoint is needed
func (b *IndexBuilder) shouldCreateCheckpoint() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.state != StateRunning {
		return false
	}

	docsSinceCheckpoint := b.progress.ProcessedDocs - b.progress.LastCheckpointDocs
	return docsSinceCheckpoint >= int64(b.options.CheckpointInterval)
}

// createCheckpoint creates a new checkpoint
func (b *IndexBuilder) createCheckpoint() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	checkpoint := &Checkpoint{
		ID:          fmt.Sprintf("ckpt-%d", time.Now().Unix()),
		Timestamp:   time.Now(),
		BatchNumber: b.progress.CurrentBatch,
		DocOffset:   b.progress.ProcessedDocs,
		Resumable:   true,
	}

	// Create index snapshot
	snapshotPath, err := b.createIndexSnapshot(checkpoint.ID)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}
	checkpoint.IndexSnapshot = snapshotPath

	// Add checkpoint to progress
	b.progress.AddCheckpoint(checkpoint)
	b.progress.LastCheckpointDocs = b.progress.ProcessedDocs

	// Save progress
	return b.progress.Save()
}

// createIndexSnapshot creates a snapshot of the current index
func (b *IndexBuilder) createIndexSnapshot(checkpointID string) (string, error) {
	manager := NewCheckpointManager(b)
	checkpoint, err := manager.CreateCheckpoint(checkpointID)
	if err != nil {
		return "", err
	}
	return checkpoint.IndexSnapshot, nil
}

// restoreFromCheckpoint restores the index from a checkpoint
func (b *IndexBuilder) restoreFromCheckpoint(checkpoint *Checkpoint) error {
	manager := NewCheckpointManager(b)
	if err := manager.RestoreCheckpoint(checkpoint); err != nil {
		return err
	}

	// Update progress to match checkpoint state
	b.progress.ProcessedDocs = checkpoint.DocOffset
	b.progress.CurrentBatch = checkpoint.BatchNumber
	return nil
}

// updateMetrics updates runtime metrics
func (b *IndexBuilder) updateMetrics() {
	b.mu.Lock()
	defer b.mu.Unlock()

	elapsed := time.Since(b.startTime)
	if elapsed.Seconds() > 0 {
		b.docsPerSec = float64(b.progress.ProcessedDocs) / elapsed.Seconds()
	}

	b.lastUpdate = time.Now()
}

// broadcastProgress sends progress updates
func (b *IndexBuilder) broadcastProgress() {
	// This would send progress to any listeners
	// For now, just update the progress metrics
	b.progress.Metrics.AvgDocsPerSecond = b.docsPerSec
	b.progress.Metrics.AvgBatchTimeMs = int64(b.avgBatchTime.Milliseconds())
}

// handleProgressUpdate handles incoming progress updates
func (b *IndexBuilder) handleProgressUpdate(update ProgressUpdate) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch update.Type {
	case UpdateTypeDocProcessed:
		b.progress.ProcessedDocs++
	case UpdateTypeDocIndexed:
		b.progress.IndexedDocs++
	case UpdateTypeDocFailed:
		b.progress.FailedDocs++
	case UpdateTypeBatchComplete:
		b.progress.CurrentBatch++
	}
}