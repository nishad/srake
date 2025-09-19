package builder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ProgressUpdate represents a progress update event
type ProgressUpdate struct {
	Type      UpdateType
	Message   string
	Timestamp time.Time
	Data      interface{}
}

// UpdateType represents the type of progress update
type UpdateType string

const (
	UpdateTypeDocProcessed  UpdateType = "doc_processed"
	UpdateTypeDocIndexed    UpdateType = "doc_indexed"
	UpdateTypeDocFailed     UpdateType = "doc_failed"
	UpdateTypeBatchComplete UpdateType = "batch_complete"
	UpdateTypeCheckpoint    UpdateType = "checkpoint"
	UpdateTypeError         UpdateType = "error"
	UpdateTypeStateChange   UpdateType = "state_change"
)

// Progress tracks the progress of index building
type Progress struct {
	mu sync.RWMutex

	// Version and metadata
	Version    string    `json:"version"`
	BuilderID  string    `json:"builder_id"`
	StartTime  time.Time `json:"started_at"`
	LastUpdate time.Time `json:"updated_at"`

	// Paths
	DatabasePath string `json:"database_path"`
	IndexPath    string `json:"index_path"`
	ProgressFile string `json:"progress_file"`

	// Configuration
	Config BuildConfig `json:"config"`

	// State
	State               BuildState `json:"state"`
	CurrentBatch        int64      `json:"current_batch"`
	TotalBatches        int64      `json:"total_batches"`
	TotalDocuments      int64      `json:"total_documents"`
	ProcessedDocs       int64      `json:"documents_processed"`
	IndexedDocs         int64      `json:"documents_indexed"`
	FailedDocs          int64      `json:"documents_failed"`
	EmbeddingsGenerated int64      `json:"embeddings_generated"`
	EmbeddingsFailed    int64      `json:"embeddings_failed"`
	LastDocumentID      string     `json:"last_document_id"`
	LastCheckpointDocs  int64      `json:"last_checkpoint_docs"`
	Error               *string    `json:"error,omitempty"`

	// Document type progress
	TypeProgress map[string]*TypeProgress `json:"type_progress"`

	// Checkpoints
	Checkpoints []*Checkpoint `json:"checkpoints"`

	// Metrics
	Metrics BuildMetrics `json:"metrics"`
}

// BuildConfig stores the build configuration
type BuildConfig struct {
	BatchSize          int    `json:"batch_size"`
	CheckpointInterval int    `json:"checkpoint_interval"`
	WithEmbeddings     bool   `json:"with_embeddings"`
	EmbeddingModel     string `json:"embedding_model,omitempty"`
	MaxMemoryMB        int    `json:"max_memory_mb"`
	NumWorkers         int    `json:"num_workers"`
}

// TypeProgress tracks progress for a specific document type
type TypeProgress struct {
	Type          string     `json:"type"`
	TotalDocs     int64      `json:"total_docs"`
	ProcessedDocs int64      `json:"processed_docs"`
	IndexedDocs   int64      `json:"indexed_docs"`
	FailedDocs    int64      `json:"failed_docs"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	Completed     bool       `json:"completed"`
	LastOffset    int64      `json:"last_offset"`
}

// Checkpoint represents a resumable checkpoint
type Checkpoint struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	BatchNumber   int64     `json:"batch_number"`
	DocOffset     int64     `json:"doc_offset"`
	IndexSnapshot string    `json:"index_snapshot"`
	Resumable     bool      `json:"resumable"`
	Size          int64     `json:"size,omitempty"`
	Hash          string    `json:"hash,omitempty"`
}

// BuildMetrics contains performance metrics
type BuildMetrics struct {
	AvgDocsPerSecond    float64 `json:"avg_docs_per_second"`
	AvgBatchTimeMs      int64   `json:"avg_batch_time_ms"`
	TotalIndexSizeBytes int64   `json:"total_index_size_bytes"`
	MemoryPeakMB        int64   `json:"memory_peak_mb"`
	EmbeddingRate       float64 `json:"embedding_rate"`
	AvgEmbeddingTimeMs  int64   `json:"avg_embedding_time_ms"`
	TotalDuration       string  `json:"total_duration,omitempty"`
}

// NewProgress creates a new progress tracker
func NewProgress(options BuildOptions) *Progress {
	return &Progress{
		Version:    "1.0",
		BuilderID:  generateBuilderID(),
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
		State:      StateIdle,
		Config: BuildConfig{
			BatchSize:          options.BatchSize,
			CheckpointInterval: options.CheckpointInterval,
			WithEmbeddings:     options.WithEmbeddings,
			EmbeddingModel:     options.EmbeddingModel,
			MaxMemoryMB:        options.MaxMemoryMB,
			NumWorkers:         options.NumWorkers,
		},
		TypeProgress: make(map[string]*TypeProgress),
		Checkpoints:  make([]*Checkpoint, 0),
		ProgressFile: options.ProgressFile,
	}
}

// LoadProgress loads progress from a file
func LoadProgress(progressFile string) (*Progress, error) {
	data, err := ioutil.ReadFile(progressFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("progress file not found: %s", progressFile)
		}
		return nil, fmt.Errorf("failed to read progress file: %w", err)
	}

	var progress Progress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, fmt.Errorf("failed to parse progress file: %w", err)
	}

	progress.ProgressFile = progressFile
	return &progress, nil
}

// Save saves the progress to a file
func (p *Progress) Save() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.LastUpdate = time.Now()

	// Calculate total duration if completed
	if p.State == StateCompleted {
		duration := p.LastUpdate.Sub(p.StartTime)
		p.Metrics.TotalDuration = duration.String()
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(p.ProgressFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create progress directory: %w", err)
	}

	// Write atomically
	tempFile := p.ProgressFile + ".tmp"
	if err := ioutil.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write progress file: %w", err)
	}

	if err := os.Rename(tempFile, p.ProgressFile); err != nil {
		return fmt.Errorf("failed to rename progress file: %w", err)
	}

	return nil
}

// SetError sets an error in the progress
func (p *Progress) SetError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	errStr := err.Error()
	p.Error = &errStr
	p.State = StateFailed
}

// GetLatestCheckpoint returns the most recent checkpoint
func (p *Progress) GetLatestCheckpoint() (*Checkpoint, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.Checkpoints) == 0 {
		return nil, nil
	}

	// Find the latest resumable checkpoint
	for i := len(p.Checkpoints) - 1; i >= 0; i-- {
		if p.Checkpoints[i].Resumable {
			return p.Checkpoints[i], nil
		}
	}

	return nil, nil
}

// AddCheckpoint adds a new checkpoint
func (p *Progress) AddCheckpoint(checkpoint *Checkpoint) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Checkpoints = append(p.Checkpoints, checkpoint)

	// Keep only the last N checkpoints to save space
	maxCheckpoints := 5
	if len(p.Checkpoints) > maxCheckpoints {
		// Mark old checkpoints as non-resumable
		for i := 0; i < len(p.Checkpoints)-maxCheckpoints; i++ {
			p.Checkpoints[i].Resumable = false
		}
	}
}

// IsTypeCompleted checks if a document type has been completed
func (p *Progress) IsTypeCompleted(docType string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tp, exists := p.TypeProgress[docType]
	if !exists {
		return false
	}

	return tp.Completed
}

// MarkTypeCompleted marks a document type as completed
func (p *Progress) MarkTypeCompleted(docType string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if tp, exists := p.TypeProgress[docType]; exists {
		tp.Completed = true
		now := time.Now()
		tp.EndTime = &now
	}
}

// GetTypeProgress returns progress for a specific document type
func (p *Progress) GetTypeProgress(docType string) *TypeProgress {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.TypeProgress[docType]
}

// SetTypeProgress sets progress for a specific document type
func (p *Progress) SetTypeProgress(docType string, progress *TypeProgress) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.TypeProgress[docType] = progress
}

// UpdateDocumentCounts updates document counts
func (p *Progress) UpdateDocumentCounts(processed, indexed, failed int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ProcessedDocs += processed
	p.IndexedDocs += indexed
	p.FailedDocs += failed
	p.LastUpdate = time.Now()
}

// GetCompletionPercentage returns the completion percentage
func (p *Progress) GetCompletionPercentage() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.TotalDocuments == 0 {
		return 0
	}

	return float64(p.ProcessedDocs) / float64(p.TotalDocuments) * 100
}

// GetEstimatedTimeRemaining returns the estimated time remaining
func (p *Progress) GetEstimatedTimeRemaining() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.Metrics.AvgDocsPerSecond == 0 {
		return 0
	}

	remainingDocs := p.TotalDocuments - p.ProcessedDocs
	secondsRemaining := float64(remainingDocs) / p.Metrics.AvgDocsPerSecond
	return time.Duration(secondsRemaining) * time.Second
}

// GetElapsedTime returns the elapsed time since start
func (p *Progress) GetElapsedTime() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.State == StateCompleted && p.Metrics.TotalDuration != "" {
		// Parse the stored duration
		duration, _ := time.ParseDuration(p.Metrics.TotalDuration)
		return duration
	}

	return time.Since(p.StartTime)
}

// Clone creates a deep copy of the progress
func (p *Progress) Clone() *Progress {
	p.mu.RLock()
	defer p.mu.RUnlock()

	data, _ := json.Marshal(p)
	var clone Progress
	json.Unmarshal(data, &clone)
	return &clone
}

// generateBuilderID generates a unique builder ID
func generateBuilderID() string {
	return fmt.Sprintf("build-%d", time.Now().Unix())
}
