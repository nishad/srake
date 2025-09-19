package progress

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State represents the current state of an ingestion
type State string

const (
	StateDownloading State = "downloading"
	StateProcessing  State = "processing"
	StateCompleted   State = "completed"
	StateFailed      State = "failed"
	StatePaused      State = "paused"
)

// Progress represents the ingestion progress for a source
type Progress struct {
	ID               int64      `json:"id" db:"id"`
	SourceURL        string     `json:"source_url" db:"source_url"`
	SourceHash       string     `json:"source_hash" db:"source_hash"`
	TotalBytes       int64      `json:"total_bytes" db:"total_bytes"`
	DownloadedBytes  int64      `json:"downloaded_bytes" db:"downloaded_bytes"`
	ProcessedBytes   int64      `json:"processed_bytes" db:"processed_bytes"`
	LastTarPosition  int64      `json:"last_tar_position" db:"last_tar_position"`
	LastXMLFile      string     `json:"last_xml_file" db:"last_xml_file"`
	RecordsProcessed int64      `json:"records_processed" db:"records_processed"`
	State            State      `json:"state" db:"state"`
	StartedAt        time.Time  `json:"started_at" db:"started_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	ErrorMessage     string     `json:"error_message,omitempty" db:"error_message"`
}

// ProcessedFile represents a file that has been processed
type ProcessedFile struct {
	ID           int64     `json:"id" db:"id"`
	ProgressID   int64     `json:"progress_id" db:"progress_id"`
	FileName     string    `json:"file_name" db:"file_name"`
	FileSize     int64     `json:"file_size" db:"file_size"`
	RecordsCount int       `json:"records_count" db:"records_count"`
	ProcessedAt  time.Time `json:"processed_at" db:"processed_at"`
	Checksum     string    `json:"checksum" db:"checksum"`
}

// Checkpoint represents a checkpoint in the ingestion process
type Checkpoint struct {
	ID                int64     `json:"id" db:"id"`
	ProgressID        int64     `json:"progress_id" db:"progress_id"`
	CheckpointTime    time.Time `json:"checkpoint_time" db:"checkpoint_time"`
	TarPosition       int64     `json:"tar_position" db:"tar_position"`
	BytesProcessed    int64     `json:"bytes_processed" db:"bytes_processed"`
	RecordsProcessed  int64     `json:"records_processed" db:"records_processed"`
	LastTransactionID string    `json:"last_transaction_id" db:"last_transaction_id"`
}

// Tracker manages ingestion progress tracking
type Tracker struct {
	db              *sql.DB
	progressID      int64
	checkpointEvery time.Duration
	lastCheckpoint  time.Time
	processedFiles  map[string]bool // Cache of processed files
}

// NewTracker creates a new progress tracker
func NewTracker(db *sql.DB) (*Tracker, error) {
	t := &Tracker{
		db:              db,
		checkpointEvery: 30 * time.Second,
		processedFiles:  make(map[string]bool),
	}

	if err := t.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create progress tables: %w", err)
	}

	return t, nil
}

// SetCheckpointInterval sets the checkpoint interval
func (t *Tracker) SetCheckpointInterval(interval time.Duration) {
	t.checkpointEvery = interval
}

// createTables creates the necessary database tables
func (t *Tracker) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS ingest_progress (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_url TEXT NOT NULL,
			source_hash TEXT UNIQUE NOT NULL,
			total_bytes INTEGER DEFAULT 0,
			downloaded_bytes INTEGER DEFAULT 0,
			processed_bytes INTEGER DEFAULT 0,
			last_tar_position INTEGER DEFAULT 0,
			last_xml_file TEXT,
			records_processed INTEGER DEFAULT 0,
			state TEXT DEFAULT 'downloading',
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			error_message TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS processed_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			progress_id INTEGER NOT NULL,
			file_name TEXT NOT NULL,
			file_size INTEGER DEFAULT 0,
			records_count INTEGER DEFAULT 0,
			processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			checksum TEXT,
			UNIQUE(progress_id, file_name),
			FOREIGN KEY (progress_id) REFERENCES ingest_progress(id)
		)`,

		`CREATE TABLE IF NOT EXISTS ingest_checkpoints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			progress_id INTEGER NOT NULL,
			checkpoint_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			tar_position INTEGER DEFAULT 0,
			bytes_processed INTEGER DEFAULT 0,
			records_processed INTEGER DEFAULT 0,
			last_transaction_id TEXT,
			FOREIGN KEY (progress_id) REFERENCES ingest_progress(id)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_progress_hash ON ingest_progress(source_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_processed_files_progress ON processed_files(progress_id)`,
		`CREATE INDEX IF NOT EXISTS idx_checkpoints_progress ON ingest_checkpoints(progress_id)`,
	}

	for _, query := range queries {
		if _, err := t.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// StartOrResume starts a new ingestion or resumes an existing one
func (t *Tracker) StartOrResume(sourceURL string, forceRestart bool) (*Progress, error) {
	hash := t.hashSource(sourceURL)

	if !forceRestart {
		// Check for existing progress
		existing, err := t.GetProgress(hash)
		if err == nil && existing != nil && existing.State != StateCompleted {
			// Resume existing progress
			t.progressID = existing.ID

			// Load processed files cache
			if err := t.loadProcessedFiles(existing.ID); err != nil {
				return nil, fmt.Errorf("failed to load processed files: %w", err)
			}

			// Update state to processing
			existing.State = StateProcessing
			existing.UpdatedAt = time.Now()
			if err := t.updateProgress(existing); err != nil {
				return nil, fmt.Errorf("failed to update progress: %w", err)
			}

			return existing, nil
		}
	}

	// Start new ingestion or reuse existing completed/failed
	progress := &Progress{
		SourceURL:  sourceURL,
		SourceHash: hash,
		State:      StateDownloading,
		StartedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Check if there's an existing record first
	existing, _ := t.GetProgress(hash)
	if existing != nil && (existing.State == StateCompleted || existing.State == StateFailed) {
		// Reset the existing record for a fresh start
		progress = existing
		progress.State = StateDownloading
		progress.StartedAt = time.Now()
		progress.UpdatedAt = time.Now()
		progress.DownloadedBytes = 0
		progress.ProcessedBytes = 0
		progress.RecordsProcessed = 0
		progress.LastTarPosition = 0
		progress.LastXMLFile = ""
		progress.ErrorMessage = ""
		progress.CompletedAt = nil

		if err := t.resetProgress(progress); err != nil {
			return nil, fmt.Errorf("failed to reset progress: %w", err)
		}
		t.progressID = progress.ID
		return progress, nil
	}

	// Create brand new progress record
	id, err := t.createProgress(progress)
	if err != nil {
		return nil, fmt.Errorf("failed to create progress: %w", err)
	}

	progress.ID = id
	t.progressID = id

	return progress, nil
}

// GetProgress retrieves progress for a source hash
func (t *Tracker) GetProgress(sourceHash string) (*Progress, error) {
	var p Progress
	query := `SELECT id, source_url, source_hash, total_bytes, downloaded_bytes,
			  processed_bytes, last_tar_position, last_xml_file, records_processed,
			  state, started_at, updated_at, completed_at, error_message
			  FROM ingest_progress WHERE source_hash = ?`

	err := t.db.QueryRow(query, sourceHash).Scan(
		&p.ID, &p.SourceURL, &p.SourceHash, &p.TotalBytes, &p.DownloadedBytes,
		&p.ProcessedBytes, &p.LastTarPosition, &p.LastXMLFile, &p.RecordsProcessed,
		&p.State, &p.StartedAt, &p.UpdatedAt, &p.CompletedAt, &p.ErrorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// UpdateDownloadProgress updates download progress
func (t *Tracker) UpdateDownloadProgress(downloadedBytes, totalBytes int64) error {
	query := `UPDATE ingest_progress
			  SET downloaded_bytes = ?, total_bytes = ?, updated_at = CURRENT_TIMESTAMP
			  WHERE id = ?`

	_, err := t.db.Exec(query, downloadedBytes, totalBytes, t.progressID)
	return err
}

// UpdateProcessingProgress updates processing progress
func (t *Tracker) UpdateProcessingProgress(tarPosition int64, processedBytes int64, lastFile string, records int64) error {
	query := `UPDATE ingest_progress
			  SET last_tar_position = ?, processed_bytes = ?, last_xml_file = ?,
			      records_processed = ?, state = ?, updated_at = CURRENT_TIMESTAMP
			  WHERE id = ?`

	_, err := t.db.Exec(query, tarPosition, processedBytes, lastFile, records, StateProcessing, t.progressID)

	// Check if checkpoint is needed
	if time.Since(t.lastCheckpoint) > t.checkpointEvery {
		return t.createCheckpoint(tarPosition, processedBytes, records)
	}

	return err
}

// RecordFileProcessed records that a file has been processed
func (t *Tracker) RecordFileProcessed(fileName string, fileSize int64, recordsCount int, checksum string) error {
	query := `INSERT INTO processed_files (progress_id, file_name, file_size, records_count, checksum, processed_at)
			  VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			  ON CONFLICT(progress_id, file_name) DO UPDATE SET processed_at = CURRENT_TIMESTAMP`

	_, err := t.db.Exec(query, t.progressID, fileName, fileSize, recordsCount, checksum)

	if err == nil {
		t.processedFiles[fileName] = true
	}

	return err
}

// IsFileProcessed checks if a file has already been processed
func (t *Tracker) IsFileProcessed(fileName string) bool {
	// Check cache first
	if processed, exists := t.processedFiles[fileName]; exists {
		return processed
	}

	// Check database
	var count int
	query := `SELECT COUNT(*) FROM processed_files WHERE progress_id = ? AND file_name = ?`
	err := t.db.QueryRow(query, t.progressID, fileName).Scan(&count)

	processed := err == nil && count > 0
	t.processedFiles[fileName] = processed

	return processed
}

// MarkCompleted marks the ingestion as completed
func (t *Tracker) MarkCompleted() error {
	query := `UPDATE ingest_progress
			  SET state = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			  WHERE id = ?`

	_, err := t.db.Exec(query, StateCompleted, t.progressID)
	return err
}

// MarkFailed marks the ingestion as failed
func (t *Tracker) MarkFailed(errorMsg string) error {
	query := `UPDATE ingest_progress
			  SET state = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP
			  WHERE id = ?`

	_, err := t.db.Exec(query, StateFailed, errorMsg, t.progressID)
	return err
}

// GetResumeInfo returns information needed to resume an ingestion
func (t *Tracker) GetResumeInfo() (*ResumeInfo, error) {
	var info ResumeInfo

	query := `SELECT downloaded_bytes, last_tar_position, last_xml_file, records_processed
			  FROM ingest_progress WHERE id = ?`

	err := t.db.QueryRow(query, t.progressID).Scan(
		&info.DownloadedBytes, &info.TarPosition, &info.LastFile, &info.RecordsProcessed,
	)

	if err != nil {
		return nil, err
	}

	// Get list of processed files
	fileQuery := `SELECT file_name FROM processed_files WHERE progress_id = ? ORDER BY id`
	rows, err := t.db.Query(fileQuery, t.progressID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	info.ProcessedFiles = make([]string, 0)
	for rows.Next() {
		var fileName string
		if err := rows.Scan(&fileName); err != nil {
			return nil, err
		}
		info.ProcessedFiles = append(info.ProcessedFiles, fileName)
	}

	return &info, nil
}

// GetStatistics returns ingestion statistics
func (t *Tracker) GetStatistics() (*Statistics, error) {
	var stats Statistics

	query := `SELECT total_bytes, downloaded_bytes, processed_bytes, records_processed,
			  started_at, updated_at FROM ingest_progress WHERE id = ?`

	var startedAt, updatedAt time.Time
	err := t.db.QueryRow(query, t.progressID).Scan(
		&stats.TotalBytes, &stats.DownloadedBytes, &stats.ProcessedBytes,
		&stats.RecordsProcessed, &startedAt, &updatedAt,
	)

	if err != nil {
		return nil, err
	}

	stats.Duration = updatedAt.Sub(startedAt)
	if stats.Duration.Seconds() > 0 {
		stats.BytesPerSecond = float64(stats.ProcessedBytes) / stats.Duration.Seconds()
		stats.RecordsPerSecond = float64(stats.RecordsProcessed) / stats.Duration.Seconds()
	}

	if stats.TotalBytes > 0 {
		stats.PercentComplete = float64(stats.ProcessedBytes) * 100 / float64(stats.TotalBytes)

		if stats.BytesPerSecond > 0 {
			remainingBytes := stats.TotalBytes - stats.ProcessedBytes
			stats.EstimatedTimeRemaining = time.Duration(float64(remainingBytes)/stats.BytesPerSecond) * time.Second
		}
	}

	// Count processed files
	var fileCount int
	countQuery := `SELECT COUNT(*) FROM processed_files WHERE progress_id = ?`
	t.db.QueryRow(countQuery, t.progressID).Scan(&fileCount)
	stats.FilesProcessed = fileCount

	return &stats, nil
}

// CleanupOldProgress removes old completed or failed progress records
func (t *Tracker) CleanupOldProgress(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	// Delete old checkpoints first
	_, err := t.db.Exec(`DELETE FROM ingest_checkpoints
						  WHERE progress_id IN (
							  SELECT id FROM ingest_progress
							  WHERE (state = ? OR state = ?) AND updated_at < ?
						  )`, StateCompleted, StateFailed, cutoff)
	if err != nil {
		return err
	}

	// Delete processed files
	_, err = t.db.Exec(`DELETE FROM processed_files
						 WHERE progress_id IN (
							 SELECT id FROM ingest_progress
							 WHERE (state = ? OR state = ?) AND updated_at < ?
						 )`, StateCompleted, StateFailed, cutoff)
	if err != nil {
		return err
	}

	// Delete progress records
	_, err = t.db.Exec(`DELETE FROM ingest_progress
						 WHERE (state = ? OR state = ?) AND updated_at < ?`,
		StateCompleted, StateFailed, cutoff)

	return err
}

// Helper methods

func (t *Tracker) hashSource(source string) string {
	h := sha256.Sum256([]byte(source))
	return hex.EncodeToString(h[:])
}

func (t *Tracker) createProgress(p *Progress) (int64, error) {
	// Simple insert - let unique constraint handle duplicates
	query := `INSERT OR REPLACE INTO ingest_progress
			  (source_url, source_hash, state, started_at, updated_at,
			   downloaded_bytes, processed_bytes, last_tar_position,
			   last_xml_file, records_processed, completed_at, error_message)
			  VALUES (?, ?, ?, ?, ?, 0, 0, 0, '', 0, NULL, '')`

	result, err := t.db.Exec(query, p.SourceURL, p.SourceHash, p.State, p.StartedAt, p.UpdatedAt)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (t *Tracker) updateProgress(p *Progress) error {
	query := `UPDATE ingest_progress
			  SET state = ?, updated_at = ?
			  WHERE id = ?`

	_, err := t.db.Exec(query, p.State, p.UpdatedAt, p.ID)
	return err
}

func (t *Tracker) resetProgress(p *Progress) error {
	query := `UPDATE ingest_progress
			  SET state = ?, started_at = ?, updated_at = ?,
			      downloaded_bytes = 0, processed_bytes = 0,
			      last_tar_position = 0, last_xml_file = '',
			      records_processed = 0, completed_at = NULL, error_message = ''
			  WHERE id = ?`

	_, err := t.db.Exec(query, p.State, p.StartedAt, p.UpdatedAt, p.ID)

	// Also clear processed files for this progress
	if err == nil {
		_, err = t.db.Exec("DELETE FROM processed_files WHERE progress_id = ?", p.ID)
	}

	// Clear cache
	t.processedFiles = make(map[string]bool)

	return err
}

func (t *Tracker) createCheckpoint(tarPos, bytesProcessed, recordsProcessed int64) error {
	query := `INSERT INTO ingest_checkpoints (progress_id, tar_position, bytes_processed, records_processed)
			  VALUES (?, ?, ?, ?)`

	_, err := t.db.Exec(query, t.progressID, tarPos, bytesProcessed, recordsProcessed)
	t.lastCheckpoint = time.Now()
	return err
}

func (t *Tracker) loadProcessedFiles(progressID int64) error {
	query := `SELECT file_name FROM processed_files WHERE progress_id = ?`
	rows, err := t.db.Query(query, progressID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var fileName string
		if err := rows.Scan(&fileName); err != nil {
			return err
		}
		t.processedFiles[fileName] = true
	}

	return rows.Err()
}

// ResumeInfo contains information needed to resume processing
type ResumeInfo struct {
	DownloadedBytes  int64    `json:"downloaded_bytes"`
	TarPosition      int64    `json:"tar_position"`
	LastFile         string   `json:"last_file"`
	RecordsProcessed int64    `json:"records_processed"`
	ProcessedFiles   []string `json:"processed_files"`
}

// Statistics contains ingestion statistics
type Statistics struct {
	TotalBytes             int64         `json:"total_bytes"`
	DownloadedBytes        int64         `json:"downloaded_bytes"`
	ProcessedBytes         int64         `json:"processed_bytes"`
	RecordsProcessed       int64         `json:"records_processed"`
	FilesProcessed         int           `json:"files_processed"`
	Duration               time.Duration `json:"duration"`
	BytesPerSecond         float64       `json:"bytes_per_second"`
	RecordsPerSecond       float64       `json:"records_per_second"`
	PercentComplete        float64       `json:"percent_complete"`
	EstimatedTimeRemaining time.Duration `json:"estimated_time_remaining"`
}

// SaveCheckpointFile saves a checkpoint to a JSON file for backup
func (t *Tracker) SaveCheckpointFile(dir string) error {
	info, err := t.GetResumeInfo()
	if err != nil {
		return err
	}

	progress, err := t.GetProgress(t.hashSource(""))
	if err != nil {
		return err
	}

	checkpoint := struct {
		Progress   *Progress   `json:"progress"`
		ResumeInfo *ResumeInfo `json:"resume_info"`
		Timestamp  time.Time   `json:"timestamp"`
	}{
		Progress:   progress,
		ResumeInfo: info,
		Timestamp:  time.Now(),
	}

	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return err
	}

	checkpointPath := filepath.Join(dir, fmt.Sprintf("checkpoint_%s.json", progress.SourceHash))
	return os.WriteFile(checkpointPath, data, 0644)
}
