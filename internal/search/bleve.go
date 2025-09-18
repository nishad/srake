package search

import (
	"fmt"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
)

// BleveIndex wraps the Bleve search index
type BleveIndex struct {
	index bleve.Index
	path  string
}

// InitBleveIndex initializes or opens a Bleve index
func InitBleveIndex(indexPath string) (*BleveIndex, error) {
	// Use the provided index path directly

	// Try to open existing index
	index, err := bleve.Open(indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		// Create new index with biological analyzer
		indexMapping := createBiologicalIndexMapping()
		index, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to open index: %w", err)
	}

	return &BleveIndex{
		index: index,
		path:  indexPath,
	}, nil
}

// createBiologicalIndexMapping creates an index mapping optimized for biological terms
func createBiologicalIndexMapping() mapping.IndexMapping {
	// Create a new index mapping
	indexMapping := bleve.NewIndexMapping()

	// Use standard analyzer for now
	indexMapping.DefaultAnalyzer = "standard"

	// Use a single default mapping for all documents
	// This avoids issues with type detection
	docMapping := bleve.NewDocumentMapping()

	// Common fields
	docMapping.AddFieldMappingsAt("type", createKeywordFieldMapping())

	// Study fields
	docMapping.AddFieldMappingsAt("study_accession", createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("study_title", createTextFieldMapping())
	docMapping.AddFieldMappingsAt("study_abstract", createTextFieldMapping())
	docMapping.AddFieldMappingsAt("study_type", createKeywordFieldMapping())

	// Experiment fields
	docMapping.AddFieldMappingsAt("experiment_accession", createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("title", createTextFieldMapping())
	docMapping.AddFieldMappingsAt("library_strategy", createTextFieldMapping())
	docMapping.AddFieldMappingsAt("library_source", createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("library_selection", createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("library_layout", createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("platform", createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("instrument_model", createKeywordFieldMapping())

	// Sample fields
	docMapping.AddFieldMappingsAt("sample_accession", createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("organism", createTextFieldMapping())
	docMapping.AddFieldMappingsAt("scientific_name", createTextFieldMapping())
	docMapping.AddFieldMappingsAt("tissue", createTextFieldMapping())
	docMapping.AddFieldMappingsAt("cell_type", createTextFieldMapping())
	docMapping.AddFieldMappingsAt("description", createTextFieldMapping())

	// Run fields
	docMapping.AddFieldMappingsAt("run_accession", createKeywordFieldMapping())
	docMapping.AddFieldMappingsAt("spots", createNumericFieldMapping())
	docMapping.AddFieldMappingsAt("bases", createNumericFieldMapping())

	// Date fields
	docMapping.AddFieldMappingsAt("submission_date", createDateFieldMapping())

	// Set the default mapping (applies to all documents)
	indexMapping.DefaultMapping = docMapping

	return indexMapping
}

// Helper functions to create field mappings
func createKeywordFieldMapping() *mapping.FieldMapping {
	fieldMapping := bleve.NewTextFieldMapping()
	fieldMapping.Analyzer = "keyword"
	fieldMapping.Store = true
	fieldMapping.IncludeInAll = true
	return fieldMapping
}

func createTextFieldMapping() *mapping.FieldMapping {
	fieldMapping := bleve.NewTextFieldMapping()
	fieldMapping.Analyzer = "standard"
	fieldMapping.Store = true
	fieldMapping.IncludeInAll = true
	return fieldMapping
}

func createSimpleFieldMapping() *mapping.FieldMapping {
	// Use keyword analyzer for exact matches (case-sensitive)
	// Will handle case normalization in the query
	fieldMapping := bleve.NewTextFieldMapping()
	fieldMapping.Analyzer = "keyword"
	fieldMapping.Store = true
	fieldMapping.IncludeInAll = true
	return fieldMapping
}

func createNumericFieldMapping() *mapping.FieldMapping {
	fieldMapping := bleve.NewNumericFieldMapping()
	fieldMapping.Store = true
	fieldMapping.IncludeInAll = false
	return fieldMapping
}

func createDateFieldMapping() *mapping.FieldMapping {
	fieldMapping := bleve.NewDateTimeFieldMapping()
	fieldMapping.Store = true
	fieldMapping.IncludeInAll = false
	return fieldMapping
}

// Document types for indexing
type StudyDoc struct {
	Type           string `json:"type"`
	StudyAccession string `json:"study_accession"`
	StudyTitle     string `json:"study_title"`
	StudyAbstract  string `json:"study_abstract"`
	StudyType      string `json:"study_type"`
	Organism       string `json:"organism"`
}

type ExperimentDoc struct {
	Type                string `json:"type"`
	ExperimentAccession string `json:"experiment_accession"`
	Title               string `json:"title"`
	LibraryStrategy     string `json:"library_strategy"`
	Platform            string `json:"platform"`
	InstrumentModel     string `json:"instrument_model"`
}

type SampleDoc struct {
	Type            string `json:"type"`
	SampleAccession string `json:"sample_accession"`
	Organism        string `json:"organism"`
	ScientificName  string `json:"scientific_name"`
	Tissue          string `json:"tissue"`
	CellType        string `json:"cell_type"`
	Description     string `json:"description"`
}

type RunDoc struct {
	Type         string `json:"type"`
	RunAccession string `json:"run_accession"`
	Spots        int64  `json:"spots"`
	Bases        int64  `json:"bases"`
}

// Index operations
func (b *BleveIndex) IndexStudy(study StudyDoc) error {
	study.Type = "study"
	return b.index.Index(study.StudyAccession, study)
}

func (b *BleveIndex) IndexExperiment(exp ExperimentDoc) error {
	exp.Type = "experiment"
	return b.index.Index(exp.ExperimentAccession, exp)
}

func (b *BleveIndex) IndexSample(sample SampleDoc) error {
	sample.Type = "sample"
	return b.index.Index(sample.SampleAccession, sample)
}

func (b *BleveIndex) IndexRun(run RunDoc) error {
	run.Type = "run"
	return b.index.Index(run.RunAccession, run)
}

// BleveSearchResult is an alias for bleve.SearchResult for easier access
type BleveSearchResult = bleve.SearchResult

// Search performs a full-text search
func (b *BleveIndex) Search(queryStr string, limit int) (*bleve.SearchResult, error) {
	query := bleve.NewQueryStringQuery(queryStr)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"*"}

	// Add facets for filtering
	searchRequest.AddFacet("organism", bleve.NewFacetRequest("organism", 10))
	searchRequest.AddFacet("library_strategy", bleve.NewFacetRequest("library_strategy", 10))
	searchRequest.AddFacet("platform", bleve.NewFacetRequest("platform", 10))
	searchRequest.AddFacet("type", bleve.NewFacetRequest("type", 5))

	return b.index.Search(searchRequest)
}

// SearchWithQuery performs a search with a pre-built query
func (b *BleveIndex) SearchWithQuery(q interface{}, limit int) (*bleve.SearchResult, error) {
	var searchQuery query.Query

	switch qt := q.(type) {
	case query.Query:
		searchQuery = qt
	default:
		return nil, fmt.Errorf("invalid query type")
	}

	searchRequest := bleve.NewSearchRequest(searchQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"*"}

	// Add facets for filtering
	searchRequest.AddFacet("organism", bleve.NewFacetRequest("organism", 10))
	searchRequest.AddFacet("library_strategy", bleve.NewFacetRequest("library_strategy", 10))
	searchRequest.AddFacet("platform", bleve.NewFacetRequest("platform", 10))
	searchRequest.AddFacet("type", bleve.NewFacetRequest("type", 5))
	searchRequest.AddFacet("library_source", bleve.NewFacetRequest("library_source", 10))
	searchRequest.AddFacet("library_layout", bleve.NewFacetRequest("library_layout", 5))

	return b.index.Search(searchRequest)
}

// BuildConjunctionQuery creates a conjunction query from multiple queries
func (b *BleveIndex) BuildConjunctionQuery(queries []interface{}) query.Query {
	var queryList []query.Query
	for _, q := range queries {
		if qq, ok := q.(query.Query); ok {
			queryList = append(queryList, qq)
		}
	}
	if len(queryList) == 1 {
		return queryList[0]
	}
	return bleve.NewConjunctionQuery(queryList...)
}

// SearchWithFilters performs a search with additional filters
func (b *BleveIndex) SearchWithFilters(queryStr string, filters map[string]string, limit int) (*bleve.SearchResult, error) {
	// Build queries
	var queries []query.Query

	if queryStr != "" {
		queries = append(queries, bleve.NewQueryStringQuery(queryStr))
	}

	// Add filter queries
	// Use appropriate query types based on field mapping
	for field, value := range filters {
		var fieldQuery query.Query

		// Platform uses keyword analyzer (exact match)
		if field == "platform" {
			termQuery := bleve.NewTermQuery(value)
			termQuery.SetField(field)
			fieldQuery = termQuery
		} else {
			// For text fields, use phrase match for exact matching
			phraseQuery := bleve.NewMatchPhraseQuery(value)
			phraseQuery.SetField(field)
			fieldQuery = phraseQuery
		}

		queries = append(queries, fieldQuery)
	}

	// Create the final query
	var finalQuery query.Query
	if len(queries) == 0 {
		finalQuery = bleve.NewMatchAllQuery()
	} else if len(queries) == 1 {
		finalQuery = queries[0]
	} else {
		// Use ConjunctionQuery to AND all conditions
		finalQuery = bleve.NewConjunctionQuery(queries...)
	}

	searchRequest := bleve.NewSearchRequest(finalQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"*"}

	return b.index.Search(searchRequest)
}

// FuzzySearch performs a fuzzy search for typo tolerance
func (b *BleveIndex) FuzzySearch(queryStr string, fuzziness int, limit int) (*bleve.SearchResult, error) {
	fuzzyQuery := bleve.NewFuzzyQuery(queryStr)
	fuzzyQuery.Fuzziness = fuzziness

	searchRequest := bleve.NewSearchRequest(fuzzyQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"*"}

	return b.index.Search(searchRequest)
}

// BatchIndex indexes multiple documents in a batch
func (b *BleveIndex) BatchIndex(docs []interface{}) error {
	batch := b.index.NewBatch()

	for _, doc := range docs {
		var id string
		var typedDoc interface{}

		switch d := doc.(type) {
		case StudyDoc:
			id = d.StudyAccession
			d.Type = "study"
			typedDoc = d
		case ExperimentDoc:
			id = d.ExperimentAccession
			d.Type = "experiment"
			typedDoc = d
		case SampleDoc:
			id = d.SampleAccession
			d.Type = "sample"
			typedDoc = d
		case RunDoc:
			id = d.RunAccession
			d.Type = "run"
			typedDoc = d
		case map[string]interface{}:
			// Handle generic documents from sync
			if docID, ok := d["id"].(string); ok {
				id = docID
				typedDoc = d
			} else {
				continue
			}
		default:
			continue
		}

		if err := batch.Index(id, typedDoc); err != nil {
			return fmt.Errorf("failed to add document %s to batch: %w", id, err)
		}
	}

	return b.index.Batch(batch)
}

// Close closes the Bleve index
func (b *BleveIndex) Close() error {
	return b.index.Close()
}

// GetDocCount returns the number of documents in the index
func (b *BleveIndex) GetDocCount() (uint64, error) {
	return b.index.DocCount()
}

// Delete removes a document from the index
func (b *BleveIndex) Delete(id string) error {
	return b.index.Delete(id)
}
