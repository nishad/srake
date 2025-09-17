package processor

import (
	"sync"
)

// OptimizedStreamProcessor extends StreamProcessor with performance optimizations
type OptimizedStreamProcessor struct {
	*StreamProcessor
	bufferPool *sync.Pool
}

// ProcessorOptions contains configuration for optimized processing
type ProcessorOptions struct {
	BatchSize         int  // Records per batch (default: 5000)
	UseTransactions   bool // Use DB transactions (default: true)
	EnableCompression bool // Enable response compression (default: true)
	WorkerCount       int  // Parallel workers (default: 4)
	BufferSize        int  // Read buffer size (default: 64KB)
}

// DefaultProcessorOptions returns optimized default options
func DefaultProcessorOptions() ProcessorOptions {
	return ProcessorOptions{
		BatchSize:         5000, // Increased from 1000
		UseTransactions:   true,
		EnableCompression: true,
		WorkerCount:       4,
		BufferSize:        65536, // 64KB
	}
}

// NewOptimizedStreamProcessor creates an optimized processor
func NewOptimizedStreamProcessor(db Database, opts ProcessorOptions) *OptimizedStreamProcessor {
	sp := NewStreamProcessor(db)

	// Configure optimizations
	if opts.BatchSize == 0 {
		opts.BatchSize = 5000
	}

	// Create buffer pool to reuse memory
	bufferPool := &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, opts.BufferSize)
			return &buf
		},
	}

	return &OptimizedStreamProcessor{
		StreamProcessor: sp,
		bufferPool:      bufferPool,
	}
}

// Optimization recommendations based on testing
var OptimizationRecommendations = []string{
	"1. Increase batch size from 1000 to 5000 for better database throughput",
	"2. Use database transactions to wrap batch inserts",
	"3. Implement connection pooling with max connections = CPU cores",
	"4. Use sync.Pool for buffer reuse to reduce GC pressure",
	"5. Enable HTTP/2 for better network efficiency",
	"6. Consider parallel processing for independent record types",
	"7. Use prepared statements for repeated queries",
	"8. Implement write-ahead logging (WAL) for SQLite",
	"9. Add indexes on frequently queried fields",
	"10. Use PRAGMA optimizations for SQLite performance",
}

// Database optimizations for SQLite
var SQLiteOptimizations = []string{
	"PRAGMA journal_mode = WAL",         // Write-ahead logging
	"PRAGMA synchronous = NORMAL",       // Balanced safety/speed
	"PRAGMA cache_size = 100000",        // ~400MB cache
	"PRAGMA temp_store = MEMORY",        // Use memory for temp tables
	"PRAGMA mmap_size = 1073741824",     // 1GB memory mapping
	"PRAGMA page_size = 32768",          // Larger page size
	"PRAGMA wal_checkpoint = PASSIVE",   // Background checkpointing
	"PRAGMA wal_autocheckpoint = 10000", // Checkpoint every 10k pages
	"PRAGMA busy_timeout = 10000",       // 10 second timeout
	"PRAGMA foreign_keys = OFF",         // Disable FK checks during import
}

// PerformanceMetrics tracks detailed performance metrics
type PerformanceMetrics struct {
	RecordsPerSecond  float64
	BytesPerSecond    float64
	AvgBatchTime      float64
	DatabaseWriteTime float64
	XMLParseTime      float64
	NetworkTime       float64
	TotalMemoryUsed   int64
	GCPauses          int
	CPUUtilization    float64
}

// GetOptimizationSuggestions returns suggestions based on current performance
func GetOptimizationSuggestions(metrics PerformanceMetrics) []string {
	var suggestions []string

	// Check records per second
	if metrics.RecordsPerSecond < 10000 {
		suggestions = append(suggestions, "Consider increasing batch size - current throughput is below 10k records/sec")
	}

	// Check memory usage
	if metrics.TotalMemoryUsed > 1024*1024*1024 { // > 1GB
		suggestions = append(suggestions, "High memory usage detected - enable buffer pooling")
	}

	// Check GC pressure
	if metrics.GCPauses > 100 {
		suggestions = append(suggestions, "High GC activity - consider using sync.Pool for object reuse")
	}

	// Check database performance
	if metrics.DatabaseWriteTime > metrics.XMLParseTime*2 {
		suggestions = append(suggestions, "Database writes are bottleneck - enable transactions and increase batch size")
	}

	// Check network performance
	if metrics.BytesPerSecond < 10*1024*1024 { // < 10MB/s
		suggestions = append(suggestions, "Network throughput is low - check connection and enable HTTP/2")
	}

	return suggestions
}
