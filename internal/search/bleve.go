package search

import (
	"fmt"
	"path/filepath"

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
func InitBleveIndex(dataDir string) (*BleveIndex, error) {
	indexPath := filepath.Join(dataDir, "search.blv")

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
	// Future: implement custom analyzers when Bleve supports them better
	indexMapping.DefaultAnalyzer = "standard"

	// Define document mappings
	studyMapping := bleve.NewDocumentMapping()
	studyMapping.AddFieldMappingsAt("study_accession", createKeywordFieldMapping())
	studyMapping.AddFieldMappingsAt("study_title", createTextFieldMapping())
	studyMapping.AddFieldMappingsAt("study_abstract", createTextFieldMapping())
	studyMapping.AddFieldMappingsAt("organism", createTextFieldMapping())
	studyMapping.AddFieldMappingsAt("study_type", createKeywordFieldMapping())

	experimentMapping := bleve.NewDocumentMapping()
	experimentMapping.AddFieldMappingsAt("experiment_accession", createKeywordFieldMapping())
	experimentMapping.AddFieldMappingsAt("title", createTextFieldMapping())
	experimentMapping.AddFieldMappingsAt("library_strategy", createKeywordFieldMapping())
	experimentMapping.AddFieldMappingsAt("platform", createKeywordFieldMapping())
	experimentMapping.AddFieldMappingsAt("instrument_model", createKeywordFieldMapping())

	sampleMapping := bleve.NewDocumentMapping()
	sampleMapping.AddFieldMappingsAt("sample_accession", createKeywordFieldMapping())
	sampleMapping.AddFieldMappingsAt("organism", createTextFieldMapping())
	sampleMapping.AddFieldMappingsAt("scientific_name", createTextFieldMapping())
	sampleMapping.AddFieldMappingsAt("tissue", createTextFieldMapping())
	sampleMapping.AddFieldMappingsAt("cell_type", createTextFieldMapping())
	sampleMapping.AddFieldMappingsAt("description", createTextFieldMapping())

	runMapping := bleve.NewDocumentMapping()
	runMapping.AddFieldMappingsAt("run_accession", createKeywordFieldMapping())
	runMapping.AddFieldMappingsAt("spots", createNumericFieldMapping())
	runMapping.AddFieldMappingsAt("bases", createNumericFieldMapping())

	// Add document mappings
	indexMapping.AddDocumentMapping("study", studyMapping)
	indexMapping.AddDocumentMapping("experiment", experimentMapping)
	indexMapping.AddDocumentMapping("sample", sampleMapping)
	indexMapping.AddDocumentMapping("run", runMapping)

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

// SearchWithFilters performs a search with additional filters
func (b *BleveIndex) SearchWithFilters(queryStr string, filters map[string]string, limit int) (*bleve.SearchResult, error) {
	// Build queries
	var queries []query.Query

	if queryStr != "" {
		queries = append(queries, bleve.NewQueryStringQuery(queryStr))
	}

	// Add filter queries
	// For keyword fields, we need exact matches
	for field, value := range filters {
		// Use MatchQuery which works with keyword analyzer fields
		matchQuery := bleve.NewMatchQuery(value)
		matchQuery.SetField(field)
		queries = append(queries, matchQuery)
	}

	// Create the final query
	var finalQuery query.Query
	if len(queries) == 0 {
		finalQuery = bleve.NewMatchAllQuery()
	} else if len(queries) == 1 {
		finalQuery = queries[0]
	} else {
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

		switch d := doc.(type) {
		case StudyDoc:
			id = d.StudyAccession
		case ExperimentDoc:
			id = d.ExperimentAccession
		case SampleDoc:
			id = d.SampleAccession
		case RunDoc:
			id = d.RunAccession
		default:
			continue
		}

		if err := batch.Index(id, doc); err != nil {
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
