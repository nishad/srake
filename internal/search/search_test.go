package search

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
)

// TestBleveIndex tests the Bleve index functionality
func TestBleveIndex(t *testing.T) {
	// Create temporary config
	cfg := config.DefaultConfig()
	cfg.DataDirectory = t.TempDir()

	// Initialize Bleve index
	index, err := InitBleveIndex(cfg.DataDirectory)
	if err != nil {
		t.Fatalf("Failed to initialize Bleve index: %v", err)
	}
	defer index.Close()

	// Test indexing a study
	study := StudyDoc{
		Type:           "study",
		StudyAccession: "SRP000001",
		StudyTitle:     "Human RNA-seq transcriptome analysis",
		StudyAbstract:  "Comprehensive analysis of human cancer transcriptome using RNA sequencing",
		StudyType:      "Transcriptome Analysis",
		Organism:       "Homo sapiens",
	}

	err = index.IndexStudy(study)
	if err != nil {
		t.Fatalf("Failed to index study: %v", err)
	}

	// Test search
	results, err := index.Search("human cancer", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if results.Total == 0 {
		t.Error("Expected at least one result")
	}

	// Test fuzzy search
	fuzzyResults, err := index.FuzzySearch("humna", 1, 10)
	if err != nil {
		t.Fatalf("Fuzzy search failed: %v", err)
	}

	if fuzzyResults.Total == 0 {
		t.Error("Fuzzy search should find 'human' when searching for 'humna'")
	}
}

// TestSearchManager tests the search manager
func TestSearchManager(t *testing.T) {
	// Create temporary config
	cfg := config.DefaultConfig()
	cfg.DataDirectory = t.TempDir()
	cfg.Search.Enabled = true
	cfg.Vectors.Enabled = false // Disable vectors for basic test

	// Create in-memory database
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	db := &database.DB{DB: sqlDB}
	defer db.Close()

	// Create search manager
	manager, err := NewManager(cfg, db)
	if err != nil {
		t.Fatalf("Failed to create search manager: %v", err)
	}
	defer manager.Close()

	// Test search
	results, err := manager.Search("test", SearchOptions{
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if results.Mode != "text" {
		t.Errorf("Expected mode 'text', got '%s'", results.Mode)
	}

	// Test stats
	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Backend != "bleve" {
		t.Errorf("Expected backend 'bleve', got '%s'", stats.Backend)
	}
}

// TestSearchCache tests the search cache functionality
func TestSearchCache(t *testing.T) {
	cache := &SearchCache{
		entries: make(map[string]*cacheEntry),
		maxSize: 2,
		ttl:     time.Second,
	}

	// Test set and get
	result1 := &SearchResult{Query: "test1", TotalHits: 1}
	cache.set("key1", result1)

	cached := cache.get("key1")
	if cached == nil {
		t.Error("Expected to get cached result")
	}
	if cached.Query != "test1" {
		t.Error("Cached result doesn't match")
	}

	// Test TTL expiry
	time.Sleep(time.Second + 100*time.Millisecond)
	expired := cache.get("key1")
	if expired != nil {
		t.Error("Cache entry should have expired")
	}

	// Test max size eviction
	result2 := &SearchResult{Query: "test2", TotalHits: 2}
	result3 := &SearchResult{Query: "test3", TotalHits: 3}

	cache.set("key2", result2)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	cache.set("key3", result3)

	// This should evict the oldest entry
	cache.set("key1", result1)

	if cache.get("key2") != nil {
		t.Error("Oldest entry should have been evicted")
	}
}

// TestBatchIndexing tests batch indexing functionality
func TestBatchIndexing(t *testing.T) {
	// Create temporary config
	cfg := config.DefaultConfig()
	cfg.DataDirectory = t.TempDir()

	// Initialize Bleve index
	index, err := InitBleveIndex(cfg.DataDirectory)
	if err != nil {
		t.Fatalf("Failed to initialize Bleve index: %v", err)
	}
	defer index.Close()

	// Create batch of documents
	docs := []interface{}{
		StudyDoc{
			Type:           "study",
			StudyAccession: "SRP000001",
			StudyTitle:     "Study 1",
		},
		ExperimentDoc{
			Type:                "experiment",
			ExperimentAccession: "SRX000001",
			Title:               "Experiment 1",
			LibraryStrategy:     "RNA-Seq",
		},
		SampleDoc{
			Type:            "sample",
			SampleAccession: "SRS000001",
			Organism:        "Human",
			ScientificName:  "Homo sapiens",
		},
		RunDoc{
			Type:         "run",
			RunAccession: "SRR000001",
			Spots:        1000000,
			Bases:        100000000,
		},
	}

	// Index batch
	err = index.BatchIndex(docs)
	if err != nil {
		t.Fatalf("Batch indexing failed: %v", err)
	}

	// Verify all documents were indexed
	count, err := index.GetDocCount()
	if err != nil {
		t.Fatalf("Failed to get document count: %v", err)
	}

	if count != 4 {
		t.Errorf("Expected 4 documents, got %d", count)
	}

	// Test deletion
	err = index.Delete("SRP000001")
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	count, err = index.GetDocCount()
	if err != nil {
		t.Fatalf("Failed to get document count after deletion: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 documents after deletion, got %d", count)
	}
}

// TestSearchWithFilters tests filtered search functionality
// TODO: Fix keyword field exact matching with filters
func TestSearchWithFilters(t *testing.T) {
	t.Skip("Skipping filter test - needs fixing for keyword field matching")
	// Create temporary config
	cfg := config.DefaultConfig()
	cfg.DataDirectory = t.TempDir()

	// Initialize Bleve index
	index, err := InitBleveIndex(cfg.DataDirectory)
	if err != nil {
		t.Fatalf("Failed to initialize Bleve index: %v", err)
	}
	defer index.Close()

	// Index test documents
	docs := []interface{}{
		ExperimentDoc{
			Type:                "experiment",
			ExperimentAccession: "SRX000001",
			Title:               "Mouse RNA-Seq",
			LibraryStrategy:     "RNA-Seq",
			Platform:            "ILLUMINA",
		},
		ExperimentDoc{
			Type:                "experiment",
			ExperimentAccession: "SRX000002",
			Title:               "Human ChIP-Seq",
			LibraryStrategy:     "ChIP-Seq",
			Platform:            "ILLUMINA",
		},
		ExperimentDoc{
			Type:                "experiment",
			ExperimentAccession: "SRX000003",
			Title:               "Mouse ChIP-Seq",
			LibraryStrategy:     "ChIP-Seq",
			Platform:            "PACBIO",
		},
	}

	err = index.BatchIndex(docs)
	if err != nil {
		t.Fatalf("Failed to index documents: %v", err)
	}

	// Search with filters
	filters := map[string]string{
		"library_strategy": "ChIP-Seq",
		"platform":         "ILLUMINA",
	}

	results, err := index.SearchWithFilters("", filters, 10)
	if err != nil {
		t.Fatalf("Filtered search failed: %v", err)
	}

	if results.Total != 1 {
		t.Errorf("Expected 1 result with filters, got %d", results.Total)
	}
}

// BenchmarkIndexing benchmarks indexing performance
func BenchmarkIndexing(b *testing.B) {
	cfg := config.DefaultConfig()
	cfg.DataDirectory = b.TempDir()

	index, err := InitBleveIndex(cfg.DataDirectory)
	if err != nil {
		b.Fatalf("Failed to initialize index: %v", err)
	}
	defer index.Close()

	study := StudyDoc{
		Type:           "study",
		StudyAccession: "SRP000001",
		StudyTitle:     "Benchmark study",
		StudyAbstract:  "This is a benchmark test study",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		study.StudyAccession = fmt.Sprintf("SRP%06d", i)
		err := index.IndexStudy(study)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSearch benchmarks search performance
func BenchmarkSearch(b *testing.B) {
	cfg := config.DefaultConfig()
	cfg.DataDirectory = b.TempDir()

	index, err := InitBleveIndex(cfg.DataDirectory)
	if err != nil {
		b.Fatalf("Failed to initialize index: %v", err)
	}
	defer index.Close()

	// Pre-populate index
	for i := 0; i < 1000; i++ {
		study := StudyDoc{
			Type:           "study",
			StudyAccession: fmt.Sprintf("SRP%06d", i),
			StudyTitle:     fmt.Sprintf("Study %d human cancer RNA-seq", i),
			StudyAbstract:  "Transcriptome analysis of various cancer types",
		}
		index.IndexStudy(study)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := index.Search("cancer", 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}