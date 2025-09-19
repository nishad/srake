package search

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

// createOptimizedIndexMapping creates an optimized index mapping based on GitHub issue #1007 insights
// Key optimizations:
// 1. Remove expensive numeric fields (spots, bases) - 16x more expensive than text
// 2. Use keyword analyzer for accession fields (exact match)
// 3. Disable term vectors (we don't use highlighting)
// 4. Only store fields needed in search results
func createOptimizedIndexMapping() mapping.IndexMapping {
	// Create a new index mapping
	indexMapping := bleve.NewIndexMapping()

	// Use standard analyzer for text fields
	indexMapping.DefaultAnalyzer = "standard"

	// Create optimized document mapping
	docMapping := bleve.NewDocumentMapping()

	// === TIER 1: Study fields (full indexing) ===

	// Type field (for filtering)
	docMapping.AddFieldMappingsAt("type", createKeywordField(true, true))

	// Study identification
	docMapping.AddFieldMappingsAt("study_accession", createKeywordField(true, true))
	docMapping.AddFieldMappingsAt("study_title", createTextField(true, true))
	docMapping.AddFieldMappingsAt("study_abstract", createTextField(true, false)) // Don't store, too large
	docMapping.AddFieldMappingsAt("study_type", createKeywordField(true, false))
	docMapping.AddFieldMappingsAt("center_name", createKeywordField(false, false))

	// === TIER 2: Experiment fields (selective indexing) ===

	// Experiment identification
	docMapping.AddFieldMappingsAt("experiment_accession", createKeywordField(true, false))
	docMapping.AddFieldMappingsAt("experiment_title", createTextField(false, false))

	// Library metadata (important for filtering)
	docMapping.AddFieldMappingsAt("library_strategy", createKeywordField(true, true))
	docMapping.AddFieldMappingsAt("library_source", createKeywordField(true, false))
	docMapping.AddFieldMappingsAt("library_selection", createKeywordField(false, false))
	docMapping.AddFieldMappingsAt("library_layout", createKeywordField(false, false))

	// Platform metadata
	docMapping.AddFieldMappingsAt("platform", createKeywordField(true, true))
	docMapping.AddFieldMappingsAt("instrument_model", createKeywordField(false, false))

	// === TIER 3: Sample fields (minimal indexing - will use FTS5) ===
	// Only index critical fields for cross-referencing

	docMapping.AddFieldMappingsAt("sample_accession", createKeywordField(false, false))
	docMapping.AddFieldMappingsAt("organism", createTextField(true, false))
	docMapping.AddFieldMappingsAt("scientific_name", createTextField(false, false))

	// === Fields to SKIP (expensive or rarely searched) ===
	// - spots (numeric, expensive)
	// - bases (numeric, expensive)
	// - submission_date (can query SQLite directly)
	// - tissue (use FTS5)
	// - cell_type (use FTS5)
	// - description (too verbose, use FTS5)
	// - run_accession (use FTS5)

	// Set the optimized mapping
	indexMapping.DefaultMapping = docMapping

	return indexMapping
}

// createOptimizedTieredMapping creates separate mappings for each tier
func createOptimizedTieredMapping() map[string]mapping.IndexMapping {
	mappings := make(map[string]mapping.IndexMapping)

	// === TIER 1: Studies mapping ===
	studyMapping := bleve.NewIndexMapping()
	studyMapping.DefaultAnalyzer = "standard"

	studyDoc := bleve.NewDocumentMapping()
	studyDoc.AddFieldMappingsAt("type", createKeywordField(true, true))
	studyDoc.AddFieldMappingsAt("study_accession", createKeywordField(true, true))
	studyDoc.AddFieldMappingsAt("study_title", createTextField(true, true))
	studyDoc.AddFieldMappingsAt("study_abstract", createTextField(true, false))
	studyDoc.AddFieldMappingsAt("study_type", createKeywordField(true, false))
	studyDoc.AddFieldMappingsAt("organism", createTextField(true, false))

	// Aggregated fields from child records
	studyDoc.AddFieldMappingsAt("library_strategies", createTextField(true, false))
	studyDoc.AddFieldMappingsAt("platforms", createKeywordField(true, false))
	studyDoc.AddFieldMappingsAt("experiment_count", createDisabledField()) // Don't index counts
	studyDoc.AddFieldMappingsAt("sample_count", createDisabledField())
	studyDoc.AddFieldMappingsAt("run_count", createDisabledField())

	studyMapping.DefaultMapping = studyDoc
	mappings["studies"] = studyMapping

	// === TIER 2: Experiments mapping ===
	expMapping := bleve.NewIndexMapping()
	expMapping.DefaultAnalyzer = "standard"

	expDoc := bleve.NewDocumentMapping()
	expDoc.AddFieldMappingsAt("type", createKeywordField(true, true))
	expDoc.AddFieldMappingsAt("experiment_accession", createKeywordField(true, true))
	expDoc.AddFieldMappingsAt("study_accession", createKeywordField(true, false))
	expDoc.AddFieldMappingsAt("title", createTextField(false, false))
	expDoc.AddFieldMappingsAt("library_strategy", createKeywordField(true, true))
	expDoc.AddFieldMappingsAt("platform", createKeywordField(true, true))
	expDoc.AddFieldMappingsAt("instrument_model", createKeywordField(false, false))

	expMapping.DefaultMapping = expDoc
	mappings["experiments"] = expMapping

	return mappings
}

// Helper functions for creating optimized field mappings

func createKeywordField(store bool, includeInAll bool) *mapping.FieldMapping {
	fieldMapping := bleve.NewTextFieldMapping()
	fieldMapping.Analyzer = "keyword" // Exact match
	fieldMapping.Store = store
	fieldMapping.IncludeInAll = includeInAll
	fieldMapping.IncludeTermVectors = false // We don't use highlighting
	fieldMapping.DocValues = true           // Enable for faceting/sorting
	return fieldMapping
}

func createTextField(store bool, includeInAll bool) *mapping.FieldMapping {
	fieldMapping := bleve.NewTextFieldMapping()
	fieldMapping.Analyzer = "standard"
	fieldMapping.Store = store
	fieldMapping.IncludeInAll = includeInAll
	fieldMapping.IncludeTermVectors = false // We don't use highlighting
	fieldMapping.DocValues = false          // Text fields don't need DocValues
	return fieldMapping
}

func createDisabledField() *mapping.FieldMapping {
	fieldMapping := bleve.NewTextFieldMapping()
	fieldMapping.Index = false
	fieldMapping.Store = false
	fieldMapping.IncludeInAll = false
	fieldMapping.DocValues = false
	return fieldMapping
}

// Memory estimation helpers based on GitHub issue #1007

// EstimateIndexMemory estimates memory usage for a given document count
func EstimateIndexMemory(docCount int, avgFieldsPerDoc int) int64 {
	// Based on GitHub issue #1007:
	// - Text fields: ~100 bytes per field per doc
	// - Numeric fields: ~1600 bytes per field per doc (16x more)
	// - Keyword fields: ~50 bytes per field per doc

	// With optimized mapping (no numeric fields):
	textFields := 5    // title, abstract, organism, etc.
	keywordFields := 8 // accessions, platform, etc.

	bytesPerDoc := int64(textFields*100 + keywordFields*50)
	totalBytes := bytesPerDoc * int64(docCount)

	// Add 20% overhead for index structures
	return totalBytes * 120 / 100
}

// EstimateQueryMemory estimates memory needed for a query
func EstimateQueryMemory(queryComplexity int, resultSize int) int64 {
	// Simple estimation based on query complexity
	baseMemory := int64(1024 * 1024)                   // 1MB base
	queryMemory := int64(queryComplexity * 100 * 1024) // 100KB per query term
	resultMemory := int64(resultSize * 10 * 1024)      // 10KB per result

	return baseMemory + queryMemory + resultMemory
}
