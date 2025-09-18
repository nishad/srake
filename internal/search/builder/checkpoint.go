package builder

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CheckpointManager handles checkpoint creation and restoration
type CheckpointManager struct {
	builder       *IndexBuilder
	checkpointDir string
	maxCheckpoints int
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(builder *IndexBuilder) *CheckpointManager {
	return &CheckpointManager{
		builder:        builder,
		checkpointDir:  builder.options.CheckpointDir,
		maxCheckpoints: 3, // Keep last 3 checkpoints by default
	}
}

// CreateCheckpoint creates a new checkpoint of the current index state
func (cm *CheckpointManager) CreateCheckpoint(checkpointID string) (*Checkpoint, error) {
	// Ensure checkpoint directory exists
	if err := os.MkdirAll(cm.checkpointDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	// Create checkpoint metadata
	checkpoint := &Checkpoint{
		ID:          checkpointID,
		Timestamp:   time.Now(),
		BatchNumber: cm.builder.progress.CurrentBatch,
		DocOffset:   cm.builder.progress.ProcessedDocs,
		Resumable:   true,
	}

	// Create snapshot archive
	archivePath := filepath.Join(cm.checkpointDir, fmt.Sprintf("%s.tar.gz", checkpointID))
	size, hash, err := cm.createSnapshot(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	checkpoint.IndexSnapshot = archivePath
	checkpoint.Size = size
	checkpoint.Hash = hash

	// Clean up old checkpoints
	if err := cm.cleanupOldCheckpoints(); err != nil {
		// Log but don't fail on cleanup errors
		fmt.Printf("Warning: failed to cleanup old checkpoints: %v\n", err)
	}

	return checkpoint, nil
}

// createSnapshot creates a compressed archive of the index
func (cm *CheckpointManager) createSnapshot(archivePath string) (int64, string, error) {
	// Get index path from config
	indexPath := cm.builder.config.Search.IndexPath

	// Create archive file
	file, err := os.Create(archivePath)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create archive file: %w", err)
	}
	defer file.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Create hasher for integrity check
	hasher := sha256.New()
	multiWriter := io.MultiWriter(gzWriter, hasher)
	tarWriter = tar.NewWriter(multiWriter)

	// Walk index directory and add files to archive
	totalSize := int64(0)
	err = filepath.Walk(indexPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the archive file itself if it's in the index directory
		if path == archivePath {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Update header name to be relative to index path
		relPath, err := filepath.Rel(indexPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		header.Name = filepath.ToSlash(relPath)

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// Write file content if it's a regular file
		if info.Mode().IsRegular() {
			fileContent, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer fileContent.Close()

			written, err := io.Copy(tarWriter, fileContent)
			if err != nil {
				return fmt.Errorf("failed to write file content: %w", err)
			}
			totalSize += written
		}

		return nil
	})

	if err != nil {
		return 0, "", fmt.Errorf("failed to walk index directory: %w", err)
	}

	// Calculate hash
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Get final file size
	if err := tarWriter.Close(); err != nil {
		return 0, "", fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return 0, "", fmt.Errorf("failed to close gzip writer: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return 0, "", fmt.Errorf("failed to stat archive file: %w", err)
	}

	return fileInfo.Size(), hash, nil
}

// RestoreCheckpoint restores the index from a checkpoint
func (cm *CheckpointManager) RestoreCheckpoint(checkpoint *Checkpoint) error {
	if !checkpoint.Resumable {
		return fmt.Errorf("checkpoint %s is not resumable", checkpoint.ID)
	}

	// Verify checkpoint file exists
	if _, err := os.Stat(checkpoint.IndexSnapshot); err != nil {
		return fmt.Errorf("checkpoint file not found: %w", err)
	}

	// Verify hash if available
	if checkpoint.Hash != "" {
		calculatedHash, err := cm.calculateFileHash(checkpoint.IndexSnapshot)
		if err != nil {
			return fmt.Errorf("failed to verify checkpoint integrity: %w", err)
		}
		if calculatedHash != checkpoint.Hash {
			return fmt.Errorf("checkpoint integrity check failed: hash mismatch")
		}
	}

	// Get index path
	indexPath := cm.builder.config.Search.IndexPath

	// Backup existing index if it exists
	backupPath := ""
	if _, err := os.Stat(indexPath); err == nil {
		backupPath = fmt.Sprintf("%s.backup.%d", indexPath, time.Now().Unix())
		if err := os.Rename(indexPath, backupPath); err != nil {
			return fmt.Errorf("failed to backup existing index: %w", err)
		}
	}

	// Create index directory
	if err := os.MkdirAll(indexPath, 0755); err != nil {
		// Restore backup on error
		if backupPath != "" {
			os.Rename(backupPath, indexPath)
		}
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Extract checkpoint archive
	if err := cm.extractSnapshot(checkpoint.IndexSnapshot, indexPath); err != nil {
		// Restore backup on error
		os.RemoveAll(indexPath)
		if backupPath != "" {
			os.Rename(backupPath, indexPath)
		}
		return fmt.Errorf("failed to extract checkpoint: %w", err)
	}

	// Remove backup on success
	if backupPath != "" {
		os.RemoveAll(backupPath)
	}

	return nil
}

// extractSnapshot extracts a checkpoint archive to the target directory
func (cm *CheckpointManager) extractSnapshot(archivePath, targetPath string) error {
	// Open archive file
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct target path
		targetFile := filepath.Join(targetPath, header.Name)

		// Ensure directory exists
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetFile, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Extract regular file
		if header.Typeflag == tar.TypeReg {
			outFile, err := os.Create(targetFile)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file: %w", err)
			}

			outFile.Close()

			// Set file permissions
			if err := os.Chmod(targetFile, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set file permissions: %w", err)
			}
		}
	}

	return nil
}

// calculateFileHash calculates SHA256 hash of a file
func (cm *CheckpointManager) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// cleanupOldCheckpoints removes old checkpoints keeping only the most recent ones
func (cm *CheckpointManager) cleanupOldCheckpoints() error {
	// List all checkpoint files
	files, err := os.ReadDir(cm.checkpointDir)
	if err != nil {
		return fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	// Filter checkpoint files
	var checkpointFiles []os.DirEntry
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".tar.gz") {
			checkpointFiles = append(checkpointFiles, file)
		}
	}

	// If we have more than max checkpoints, remove the oldest
	if len(checkpointFiles) > cm.maxCheckpoints {
		// Sort by modification time (oldest first)
		sortedFiles := make([]string, 0, len(checkpointFiles))
		for _, file := range checkpointFiles {
			sortedFiles = append(sortedFiles, filepath.Join(cm.checkpointDir, file.Name()))
		}

		// Remove oldest files
		numToRemove := len(checkpointFiles) - cm.maxCheckpoints
		for i := 0; i < numToRemove; i++ {
			if err := os.Remove(sortedFiles[i]); err != nil {
				return fmt.Errorf("failed to remove old checkpoint: %w", err)
			}
		}
	}

	return nil
}

// GetAvailableCheckpoints returns a list of available checkpoint files
func (cm *CheckpointManager) GetAvailableCheckpoints() ([]string, error) {
	files, err := os.ReadDir(cm.checkpointDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	var checkpoints []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".tar.gz") {
			checkpoints = append(checkpoints, file.Name())
		}
	}

	return checkpoints, nil
}