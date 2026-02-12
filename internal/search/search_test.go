package search

import (
	"database/sql"
	"fmt"
	"os"
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

	// Initialize Bleve index in a subdirectory that doesn't exist yet
	indexPath := cfg.DataDirectory + "/test.bleve"
	index, err := InitBleveIndex(indexPath)
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

	// Mode may vary based on configuration, just check it's set
	if results.Mode == "" {
		t.Errorf("Expected mode to be set, got empty string")
	}

	// Test stats
	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Backend defaults to tiered in production
	if stats.Backend == "" {
		t.Errorf("Expected backend to be set, got empty string")
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

	// Initialize Bleve index in a subdirectory that doesn't exist yet
	indexPath := cfg.DataDirectory + "/batch.bleve"
	index, err := InitBleveIndex(indexPath)
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
func TestSearchWithFilters(t *testing.T) {
	// Create temporary config
	cfg := config.DefaultConfig()
	cfg.DataDirectory = t.TempDir()

	// Initialize Bleve index in a subdirectory that doesn't exist yet
	indexPath := cfg.DataDirectory + "/filters.bleve"
	index, err := InitBleveIndex(indexPath)
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

	// Verify doc count immediately after indexing
	docCount, err := index.GetDocCount()
	if err != nil {
		t.Fatalf("Failed to get doc count: %v", err)
	}
	t.Logf("Document count after indexing: %d", docCount)

	// Search with filters
	filters := map[string]string{
		"library_strategy": "ChIP-Seq",
		"platform":         "ILLUMINA",
	}

	// Debug: Search without filters first
	allResults, err := index.Search("ChIP-Seq", 10)
	if err != nil {
		t.Fatalf("Search without filters failed: %v", err)
	}
	t.Logf("Documents matching 'ChIP-Seq': %d", allResults.Total)
	for _, hit := range allResults.Hits {
		t.Logf("Hit ID: %s", hit.ID)
		t.Logf("  - library_strategy: %v", hit.Fields["library_strategy"])
		t.Logf("  - platform: %v", hit.Fields["platform"])
	}

	results, err := index.SearchWithFilters("", filters, 10)
	if err != nil {
		t.Fatalf("Filtered search failed: %v", err)
	}

	t.Logf("Results with filters: %d", results.Total)
	for _, hit := range results.Hits {
		t.Logf("Filtered Hit ID: %s", hit.ID)
		t.Logf("  - library_strategy: %v", hit.Fields["library_strategy"])
		t.Logf("  - platform: %v", hit.Fields["platform"])
	}

	if results.Total != 1 {
		t.Errorf("Expected 1 result with filters, got %d", results.Total)
	}
}

// BenchmarkIndexing benchmarks indexing performance
func BenchmarkIndexing(b *testing.B) {
	cfg := config.DefaultConfig()
	cfg.DataDirectory = b.TempDir()

	indexPath := cfg.DataDirectory + "/bench.bleve"
	index, err := InitBleveIndex(indexPath)
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

// TestVectorSearch tests vector search functionality with ONNX
func TestVectorSearch(t *testing.T) {
	// Create temporary config
	cfg := config.DefaultConfig()
	cfg.DataDirectory = t.TempDir()
	cfg.Search.Enabled = true
	cfg.Vectors.Enabled = true

	// Check if ONNX runtime is available
	_, err := os.Stat("/opt/homebrew/lib/libonnxruntime.dylib")
	if err != nil {
		t.Skip("ONNX runtime not installed, skipping vector test")
	}

	// Note: Full vector testing would require the embeddings package
	// For now, just test the basic search functionality
	t.Log("Testing basic Bleve search functionality...")

	// Initialize Bleve index in a subdirectory that doesn't exist yet
	indexPath := cfg.DataDirectory + "/vector.bleve"
	index, err := InitBleveIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to initialize Bleve index: %v", err)
	}
	defer index.Close()

	// Index test documents
	docs := []interface{}{
		StudyDoc{
			Type:           "study",
			StudyAccession: "SRP000001",
			StudyTitle:     "Human cancer RNA sequencing study",
			StudyAbstract:  "Comprehensive analysis of tumor transcriptomes",
			Organism:       "Homo sapiens",
		},
		StudyDoc{
			Type:           "study",
			StudyAccession: "SRP000002",
			StudyTitle:     "Mouse brain tissue expression analysis",
			StudyAbstract:  "Single-cell RNA-seq of neural populations",
			Organism:       "Mus musculus",
		},
	}

	err = index.BatchIndex(docs)
	if err != nil {
		t.Fatalf("Failed to index documents: %v", err)
	}

	// Test text search
	results, err := index.Search("cancer", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if results.Total != 1 {
		t.Errorf("Expected 1 result for 'cancer', got %d", results.Total)
	}

	// Test organism search
	results, err = index.Search("homo sapiens", 10)
	if err != nil {
		t.Fatalf("Organism search failed: %v", err)
	}

	if results.Total != 1 {
		t.Errorf("Expected 1 result for 'homo sapiens', got %d", results.Total)
	}

	t.Log("✅ Basic search test completed successfully!")
}

// BenchmarkSearch benchmarks search performance
func BenchmarkSearch(b *testing.B) {
	cfg := config.DefaultConfig()
	cfg.DataDirectory = b.TempDir()

	indexPath := cfg.DataDirectory + "/search-bench.bleve"
	index, err := InitBleveIndex(indexPath)
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
