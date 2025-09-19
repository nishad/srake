package search

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
)

// LazyIndex wraps a Bleve index with lazy loading and auto-close
type LazyIndex struct {
	path       string
	index      *BleveIndex
	lastAccess time.Time
	idleTimer  *time.Timer
	idleTime   time.Duration
	mu         sync.RWMutex

	// Stats
	loadCount   int
	closeCount  int
	searchCount int
}

// NewLazyIndex creates a new lazy-loading index wrapper
func NewLazyIndex(path string, idleTime time.Duration) *LazyIndex {
	if idleTime == 0 {
		idleTime = 5 * time.Minute // Default idle timeout
	}

	return &LazyIndex{
		path:     path,
		idleTime: idleTime,
	}
}

// ensureOpen loads the index if not already loaded
func (l *LazyIndex) ensureOpen() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.index != nil {
		// Index already loaded, update access time
		l.lastAccess = time.Now()
		l.resetTimer()
		return nil
	}

	// Load the index
	log.Printf("[LAZY] Loading index from %s (load #%d)", l.path, l.loadCount+1)
	start := time.Now()

	index, err := InitBleveIndex(l.path)
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	l.index = index
	l.lastAccess = time.Now()
	l.loadCount++

	// Get document count for logging
	if docCount, err := index.GetDocCount(); err == nil {
		log.Printf("[LAZY] Index loaded with %d documents in %v", docCount, time.Since(start))
	} else {
		log.Printf("[LAZY] Index loaded in %v", time.Since(start))
	}

	// Start idle timer
	l.resetTimer()

	return nil
}

// resetTimer resets the idle timer
func (l *LazyIndex) resetTimer() {
	// Cancel existing timer
	if l.idleTimer != nil {
		l.idleTimer.Stop()
	}

	// Start new timer
	l.idleTimer = time.AfterFunc(l.idleTime, func() {
		l.closeIfIdle()
	})
}

// closeIfIdle closes the index if it's been idle
func (l *LazyIndex) closeIfIdle() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.index == nil {
		return
	}

	// Check if still idle
	if time.Since(l.lastAccess) >= l.idleTime {
		log.Printf("[LAZY] Closing idle index (was open for %v)", time.Since(l.lastAccess))

		if err := l.index.Close(); err != nil {
			log.Printf("[LAZY] Error closing index: %v", err)
		}

		l.index = nil
		l.closeCount++

		// Cancel timer
		if l.idleTimer != nil {
			l.idleTimer.Stop()
			l.idleTimer = nil
		}
	}
}

// Search performs a search, loading the index if needed
func (l *LazyIndex) Search(queryStr string, limit int) (*bleve.SearchResult, error) {
	if err := l.ensureOpen(); err != nil {
		return nil, err
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	l.searchCount++
	return l.index.Search(queryStr, limit)
}

// SearchWithFilters performs a filtered search
func (l *LazyIndex) SearchWithFilters(queryStr string, filters map[string]string, limit int) (*bleve.SearchResult, error) {
	if err := l.ensureOpen(); err != nil {
		return nil, err
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	l.searchCount++
	return l.index.SearchWithFilters(queryStr, filters, limit)
}

// BatchIndex indexes multiple documents
func (l *LazyIndex) BatchIndex(docs []interface{}) error {
	if err := l.ensureOpen(); err != nil {
		return err
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.index.BatchIndex(docs)
}

// IndexStudy indexes a study document
func (l *LazyIndex) IndexStudy(study StudyDoc) error {
	if err := l.ensureOpen(); err != nil {
		return err
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.index.IndexStudy(study)
}

// IndexExperiment indexes an experiment document
func (l *LazyIndex) IndexExperiment(exp ExperimentDoc) error {
	if err := l.ensureOpen(); err != nil {
		return err
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.index.IndexExperiment(exp)
}

// GetDocCount returns the document count
func (l *LazyIndex) GetDocCount() (uint64, error) {
	if err := l.ensureOpen(); err != nil {
		return 0, err
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.index.GetDocCount()
}

// Delete removes a document from the index
func (l *LazyIndex) Delete(id string) error {
	if err := l.ensureOpen(); err != nil {
		return err
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.index.Delete(id)
}

// Close closes the index if open
func (l *LazyIndex) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Cancel timer
	if l.idleTimer != nil {
		l.idleTimer.Stop()
		l.idleTimer = nil
	}

	if l.index != nil {
		log.Printf("[LAZY] Closing index (stats: loaded=%d, closed=%d, searches=%d)",
			l.loadCount, l.closeCount, l.searchCount)
		err := l.index.Close()
		l.index = nil
		return err
	}

	return nil
}

// IsLoaded returns true if the index is currently loaded
func (l *LazyIndex) IsLoaded() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.index != nil
}

// GetStats returns lazy index statistics
func (l *LazyIndex) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := map[string]interface{}{
		"loaded":       l.index != nil,
		"load_count":   l.loadCount,
		"close_count":  l.closeCount,
		"search_count": l.searchCount,
		"idle_time":    l.idleTime.String(),
	}

	if l.index != nil {
		stats["last_access"] = l.lastAccess.Format(time.RFC3339)
		stats["idle_for"] = time.Since(l.lastAccess).String()
		if docCount, err := l.index.GetDocCount(); err == nil {
			stats["doc_count"] = docCount
		}
	}

	return stats
}

// ForceClose immediately closes the index without waiting for idle
func (l *LazyIndex) ForceClose() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Cancel timer
	if l.idleTimer != nil {
		l.idleTimer.Stop()
		l.idleTimer = nil
	}

	if l.index != nil {
		log.Printf("[LAZY] Force closing index")
		err := l.index.Close()
		l.index = nil
		l.closeCount++
		return err
	}

	return nil
}
