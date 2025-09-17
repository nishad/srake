package progress

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDatabase(t *testing.T) (*sql.DB, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

func TestBasicTrackerOperations(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Create tracker
	tracker, err := NewTracker(db)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	// Test 1: Start new ingestion
	sourceURL := "https://example.com/test.tar.gz"
	progress, err := tracker.StartOrResume(sourceURL, false)
	if err != nil {
		t.Fatalf("Failed to start ingestion: %v", err)
	}
	if progress.SourceURL != sourceURL {
		t.Errorf("Expected URL %s, got %s", sourceURL, progress.SourceURL)
	}

	// Test 2: Update download progress
	err = tracker.UpdateDownloadProgress(1000, 10000)
	if err != nil {
		t.Fatalf("Failed to update download progress: %v", err)
	}

	// Test 3: Record processed file
	err = tracker.RecordFileProcessed("test.xml", 1000, 10, "checksum123")
	if err != nil {
		t.Fatalf("Failed to record processed file: %v", err)
	}

	// Test 4: Check if file is processed
	if !tracker.IsFileProcessed("test.xml") {
		t.Error("File should be marked as processed")
	}
	if tracker.IsFileProcessed("nonexistent.xml") {
		t.Error("Non-existent file should not be marked as processed")
	}

	// Test 5: Get statistics
	stats, err := tracker.GetStatistics()
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}
	if stats.TotalBytes != 10000 {
		t.Errorf("Expected total bytes 10000, got %d", stats.TotalBytes)
	}
	if stats.DownloadedBytes != 1000 {
		t.Errorf("Expected downloaded bytes 1000, got %d", stats.DownloadedBytes)
	}

	t.Log("✅ All basic tracker operations passed")
}

func TestResumeCapability(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	sourceURL := "https://example.com/resume-test.tar.gz"

	// Step 1: Start initial ingestion
	tracker1, err := NewTracker(db)
	if err != nil {
		t.Fatalf("Failed to create first tracker: %v", err)
	}

	progress1, err := tracker1.StartOrResume(sourceURL, false)
	if err != nil {
		t.Fatalf("Failed to start first ingestion: %v", err)
	}
	originalID := progress1.ID

	// Simulate some progress
	tracker1.UpdateDownloadProgress(5000, 20000)
	tracker1.UpdateProcessingProgress(2500, 5000, "file10.xml", 250)
	tracker1.RecordFileProcessed("file1.xml", 100, 5, "check1")
	tracker1.RecordFileProcessed("file2.xml", 200, 10, "check2")

	// Important: Set the progressID for second tracker
	tracker1.progressID = originalID

	// Step 2: Simulate restart - create new tracker
	tracker2, err := NewTracker(db)
	if err != nil {
		t.Fatalf("Failed to create second tracker: %v", err)
	}

	// Try to resume
	progress2, err := tracker2.StartOrResume(sourceURL, false)
	if err != nil {
		t.Fatalf("Failed to resume: %v", err)
	}

	// Verify it's resuming the same progress
	if progress2.ID != originalID {
		t.Errorf("Expected to resume ID %d, got %d", originalID, progress2.ID)
	}
	if progress2.DownloadedBytes != 5000 {
		t.Errorf("Expected downloaded bytes 5000, got %d", progress2.DownloadedBytes)
	}

	// Check that processed files are remembered
	if !tracker2.IsFileProcessed("file1.xml") {
		t.Error("Previously processed file1.xml should be marked as processed")
	}
	if !tracker2.IsFileProcessed("file2.xml") {
		t.Error("Previously processed file2.xml should be marked as processed")
	}

	// Get resume info
	resumeInfo, err := tracker2.GetResumeInfo()
	if err != nil {
		t.Fatalf("Failed to get resume info: %v", err)
	}
	if resumeInfo.RecordsProcessed != 250 {
		t.Errorf("Expected 250 records processed, got %d", resumeInfo.RecordsProcessed)
	}
	if resumeInfo.LastFile != "file10.xml" {
		t.Errorf("Expected last file to be file10.xml, got %s", resumeInfo.LastFile)
	}

	t.Log("✅ Resume capability test passed")
}

func TestForceRestart(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	tracker, err := NewTracker(db)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	sourceURL := "https://example.com/force-test.tar.gz"

	// Start first ingestion
	progress1, err := tracker.StartOrResume(sourceURL, false)
	if err != nil {
		t.Fatalf("Failed to start first ingestion: %v", err)
	}
	firstID := progress1.ID

	// Add some progress
	tracker.UpdateDownloadProgress(5000, 10000)

	// Force restart
	progress2, err := tracker.StartOrResume(sourceURL, true)
	if err != nil {
		t.Fatalf("Failed to force restart: %v", err)
	}

	// Should have different ID (new progress record)
	if progress2.ID == firstID {
		t.Error("Force restart should create new progress record")
	}
	if progress2.DownloadedBytes != 0 {
		t.Errorf("Force restart should have 0 downloaded bytes, got %d", progress2.DownloadedBytes)
	}

	t.Log("✅ Force restart test passed")
}

func TestCheckpointCreation(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	tracker, err := NewTracker(db)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	// Set very short checkpoint interval for testing
	tracker.checkpointEvery = 50 * time.Millisecond

	sourceURL := "https://example.com/checkpoint-test.tar.gz"
	_, err = tracker.StartOrResume(sourceURL, false)
	if err != nil {
		t.Fatalf("Failed to start ingestion: %v", err)
	}

	// First update - no checkpoint yet
	err = tracker.UpdateProcessingProgress(1000, 5000, "file1.xml", 100)
	if err != nil {
		t.Fatalf("Failed to update progress: %v", err)
	}

	// Wait for checkpoint interval to pass
	time.Sleep(100 * time.Millisecond)

	// Second update - should trigger checkpoint
	err = tracker.UpdateProcessingProgress(2000, 10000, "file2.xml", 200)
	if err != nil {
		t.Fatalf("Failed to update progress: %v", err)
	}

	// Verify checkpoint was created
	var count int
	query := "SELECT COUNT(*) FROM ingest_checkpoints WHERE progress_id = ?"
	err = db.QueryRow(query, tracker.progressID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query checkpoints: %v", err)
	}
	if count < 1 {
		t.Error("Expected at least one checkpoint to be created")
	}

	t.Log("✅ Checkpoint creation test passed")
}

func TestErrorHandling(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	tracker, err := NewTracker(db)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	sourceURL := "https://example.com/error-test.tar.gz"
	_, err = tracker.StartOrResume(sourceURL, false)
	if err != nil {
		t.Fatalf("Failed to start ingestion: %v", err)
	}

	// Test marking as failed
	errorMsg := "Network timeout"
	err = tracker.MarkFailed(errorMsg)
	if err != nil {
		t.Fatalf("Failed to mark as failed: %v", err)
	}

	// Verify error was recorded
	hash := tracker.hashSource(sourceURL)
	progress, err := tracker.GetProgress(hash)
	if err != nil {
		t.Fatalf("Failed to get progress: %v", err)
	}
	if progress.State != StateFailed {
		t.Errorf("Expected state %s, got %s", StateFailed, progress.State)
	}
	if progress.ErrorMessage != errorMsg {
		t.Errorf("Expected error message '%s', got '%s'", errorMsg, progress.ErrorMessage)
	}

	t.Log("✅ Error handling test passed")
}

func TestCleanup(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	tracker, err := NewTracker(db)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	// Create old completed progress
	oldSource := "https://example.com/old.tar.gz"
	progress, err := tracker.StartOrResume(oldSource, false)
	if err != nil {
		t.Fatalf("Failed to start old ingestion: %v", err)
	}
	tracker.MarkCompleted()

	// Manually make it old
	_, err = db.Exec("UPDATE ingest_progress SET updated_at = datetime('now', '-10 days') WHERE id = ?", progress.ID)
	if err != nil {
		t.Fatalf("Failed to update timestamp: %v", err)
	}

	// Create recent progress
	recentSource := "https://example.com/recent.tar.gz"
	_, err = tracker.StartOrResume(recentSource, false)
	if err != nil {
		t.Fatalf("Failed to start recent ingestion: %v", err)
	}

	// Clean up old progress
	err = tracker.CleanupOldProgress(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}

	// Verify old was deleted
	oldHash := tracker.hashSource(oldSource)
	oldProgress, _ := tracker.GetProgress(oldHash)
	if oldProgress != nil {
		t.Error("Old progress should have been deleted")
	}

	// Verify recent still exists
	recentHash := tracker.hashSource(recentSource)
	recentProgress, _ := tracker.GetProgress(recentHash)
	if recentProgress == nil {
		t.Error("Recent progress should still exist")
	}

	t.Log("✅ Cleanup test passed")
}

// Run all tests
func TestAll(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"BasicOperations", TestBasicTrackerOperations},
		{"ResumeCapability", TestResumeCapability},
		{"ForceRestart", TestForceRestart},
		{"CheckpointCreation", TestCheckpointCreation},
		{"ErrorHandling", TestErrorHandling},
		{"Cleanup", TestCleanup},
	}

	for _, test := range tests {
		t.Run(test.name, test.fn)
	}
}

// Benchmarks
func BenchmarkFileProcessedCheck(b *testing.B) {
	db, cleanup := setupTestDatabase(&testing.T{})
	defer cleanup()

	tracker, _ := NewTracker(db)
	tracker.StartOrResume("https://example.com/bench.tar.gz", false)

	// Pre-populate with 1000 files
	for i := 0; i < 1000; i++ {
		tracker.RecordFileProcessed(fmt.Sprintf("file%d.xml", i), 1000, 10, "check")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.IsFileProcessed(fmt.Sprintf("file%d.xml", i%1000))
	}
}