package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/nishad/srake/internal/database"
)

// ExportService handles data export in various formats
type ExportService struct {
	db           *database.DB
	searchSvc    *SearchService
}

// NewExportService creates a new export service instance
func NewExportService(db *database.DB, searchSvc *SearchService) *ExportService {
	return &ExportService{
		db:        db,
		searchSvc: searchSvc,
	}
}

// Export exports data based on the request parameters
func (e *ExportService) Export(ctx context.Context, req *ExportRequest, writer io.Writer) error {
	// First perform search to get results
	searchReq := &SearchRequest{
		Query:   req.Query,
		Filters: req.Filters,
		Limit:   req.Limit,
		Fields:  req.Fields,
	}

	searchResp, err := e.searchSvc.Search(ctx, searchReq)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Export based on format
	switch strings.ToLower(req.Format) {
	case "json":
		return e.exportJSON(searchResp.Results, writer, req.Fields)
	case "csv":
		return e.exportCSV(searchResp.Results, writer, req.Fields)
	case "tsv":
		return e.exportTSV(searchResp.Results, writer, req.Fields)
	case "xml":
		return e.exportXML(searchResp.Results, writer, req.Fields)
	case "jsonl", "ndjson":
		return e.exportJSONLines(searchResp.Results, writer, req.Fields)
	default:
		return fmt.Errorf("unsupported export format: %s", req.Format)
	}
}

// ExportToFile exports data to a file
func (e *ExportService) ExportToFile(ctx context.Context, req *ExportRequest, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return e.Export(ctx, req, file)
}

// exportJSON exports results as JSON
func (e *ExportService) exportJSON(results []SearchHit, writer io.Writer, fields []string) error {
	// Convert results to exportable format
	exportData := e.prepareExportData(results, fields)

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(exportData)
}

// exportJSONLines exports results as newline-delimited JSON
func (e *ExportService) exportJSONLines(results []SearchHit, writer io.Writer, fields []string) error {
	encoder := json.NewEncoder(writer)

	for _, hit := range results {
		data := e.prepareHitData(hit, fields)
		if err := encoder.Encode(data); err != nil {
			return err
		}
	}

	return nil
}

// exportCSV exports results as CSV
func (e *ExportService) exportCSV(results []SearchHit, writer io.Writer, fields []string) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write headers
	headers := e.getHeaders(results, fields)
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	// Write data rows
	for _, hit := range results {
		row := e.hitToRow(hit, headers)
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// exportTSV exports results as TSV
func (e *ExportService) exportTSV(results []SearchHit, writer io.Writer, fields []string) error {
	csvWriter := csv.NewWriter(writer)
	csvWriter.Comma = '\t'
	defer csvWriter.Flush()

	// Write headers
	headers := e.getHeaders(results, fields)
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	// Write data rows
	for _, hit := range results {
		row := e.hitToRow(hit, headers)
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// exportXML exports results as XML
func (e *ExportService) exportXML(results []SearchHit, writer io.Writer, fields []string) error {
	type XMLExport struct {
		XMLName xml.Name     `xml:"export"`
		Results []XMLResult  `xml:"result"`
	}

	type XMLResult struct {
		Type   string      `xml:"type,attr"`
		Data   interface{} `xml:"data"`
		Score  float64     `xml:"score,omitempty"`
	}

	export := XMLExport{
		Results: make([]XMLResult, 0, len(results)),
	}

	for _, hit := range results {
		result := XMLResult{
			Score: hit.Score,
		}

		// Determine type and data
		if hit.Study != nil {
			result.Type = "study"
			result.Data = e.filterFields(hit.Study, fields)
		} else if hit.Experiment != nil {
			result.Type = "experiment"
			result.Data = e.filterFields(hit.Experiment, fields)
		} else if hit.Sample != nil {
			result.Type = "sample"
			result.Data = e.filterFields(hit.Sample, fields)
		} else if hit.Run != nil {
			result.Type = "run"
			result.Data = e.filterFields(hit.Run, fields)
		}

		export.Results = append(export.Results, result)
	}

	// Write XML header
	writer.Write([]byte(xml.Header))

	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")
	return encoder.Encode(export)
}

// prepareExportData prepares data for export
func (e *ExportService) prepareExportData(results []SearchHit, fields []string) []map[string]interface{} {
	exportData := make([]map[string]interface{}, 0, len(results))

	for _, hit := range results {
		data := e.prepareHitData(hit, fields)
		exportData = append(exportData, data)
	}

	return exportData
}

// prepareHitData converts a SearchHit to a map for export
func (e *ExportService) prepareHitData(hit SearchHit, fields []string) map[string]interface{} {
	data := make(map[string]interface{})

	// Add metadata fields
	data["score"] = hit.Score
	if hit.Confidence != "" {
		data["confidence"] = hit.Confidence
	}

	// Add entity data
	if hit.Study != nil {
		data["type"] = "study"
		e.addEntityFields(data, hit.Study, fields)
	} else if hit.Experiment != nil {
		data["type"] = "experiment"
		e.addEntityFields(data, hit.Experiment, fields)
	} else if hit.Sample != nil {
		data["type"] = "sample"
		e.addEntityFields(data, hit.Sample, fields)
	} else if hit.Run != nil {
		data["type"] = "run"
		e.addEntityFields(data, hit.Run, fields)
	}

	return data
}

// addEntityFields adds entity fields to the data map
func (e *ExportService) addEntityFields(data map[string]interface{}, entity interface{}, fields []string) {
	// Convert entity to map using JSON marshaling/unmarshaling
	jsonData, err := json.Marshal(entity)
	if err != nil {
		return
	}

	var entityMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &entityMap); err != nil {
		return
	}

	// If no fields specified, include all
	if len(fields) == 0 {
		for k, v := range entityMap {
			data[k] = v
		}
		return
	}

	// Include only specified fields
	for _, field := range fields {
		if value, exists := entityMap[field]; exists {
			data[field] = value
		}
	}
}

// filterFields filters entity fields based on the field list
func (e *ExportService) filterFields(entity interface{}, fields []string) interface{} {
	if len(fields) == 0 {
		return entity
	}

	// Convert to map, filter, and return
	jsonData, err := json.Marshal(entity)
	if err != nil {
		return entity
	}

	var entityMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &entityMap); err != nil {
		return entity
	}

	filtered := make(map[string]interface{})
	for _, field := range fields {
		if value, exists := entityMap[field]; exists {
			filtered[field] = value
		}
	}

	return filtered
}

// getHeaders generates CSV/TSV headers based on results
func (e *ExportService) getHeaders(results []SearchHit, fields []string) []string {
	if len(fields) > 0 {
		// Add type and score to specified fields
		headers := []string{"type", "score"}
		headers = append(headers, fields...)
		return headers
	}

	// Auto-detect headers from first result
	headers := []string{"type", "score", "confidence"}

	if len(results) > 0 {
		hit := results[0]
		if hit.Study != nil {
			headers = append(headers, getStudyHeaders()...)
		} else if hit.Experiment != nil {
			headers = append(headers, getExperimentHeaders()...)
		} else if hit.Sample != nil {
			headers = append(headers, getSampleHeaders()...)
		} else if hit.Run != nil {
			headers = append(headers, getRunHeaders()...)
		}
	}

	return headers
}

// hitToRow converts a SearchHit to a CSV/TSV row
func (e *ExportService) hitToRow(hit SearchHit, headers []string) []string {
	row := make([]string, len(headers))
	data := e.prepareHitData(hit, nil)

	for i, header := range headers {
		if value, exists := data[header]; exists {
			row[i] = fmt.Sprint(value)
		} else {
			row[i] = ""
		}
	}

	return row
}

// Helper functions to get entity headers
func getStudyHeaders() []string {
	return []string{
		"accession", "title", "study_type", "abstract", "center_name",
		"center_project_name", "description", "organism", "submission_accession",
	}
}

func getExperimentHeaders() []string {
	return []string{
		"accession", "title", "study_accession", "design_description",
		"library_name", "library_strategy", "library_source", "library_selection",
		"library_layout", "platform", "instrument_model", "submission_accession",
	}
}

func getSampleHeaders() []string {
	return []string{
		"accession", "title", "sample_accession", "organism", "description",
		"submission_accession",
	}
}

func getRunHeaders() []string {
	return []string{
		"accession", "experiment_accession", "total_spots", "total_bases",
		"total_size", "load_done", "published", "submission_accession",
	}
}

// ExportDatabase exports the entire database in a specified format
func (e *ExportService) ExportDatabase(ctx context.Context, outputPath string, format string) error {
	switch format {
	case "sqlite":
		return e.exportSQLite(ctx, outputPath)
	case "json":
		return e.exportDatabaseJSON(ctx, outputPath)
	default:
		return fmt.Errorf("unsupported database export format: %s", format)
	}
}

// exportSQLite creates a copy of the database
func (e *ExportService) exportSQLite(ctx context.Context, outputPath string) error {
	// Use SQLite backup API or copy file
	// This is a simplified version - in production you'd use proper backup
	return fmt.Errorf("SQLite export not yet implemented")
}

// exportDatabaseJSON exports entire database as JSON
func (e *ExportService) exportDatabaseJSON(ctx context.Context, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Export all tables
	data := make(map[string]interface{})

	// Export studies
	studies, err := e.getAllStudies(ctx)
	if err != nil {
		return err
	}
	data["studies"] = studies

	// Export metadata
	data["export_date"] = time.Now().Format(time.RFC3339)
	data["total_studies"] = len(studies)

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// getAllStudies retrieves all studies from database
func (e *ExportService) getAllStudies(ctx context.Context) ([]*database.Study, error) {
	query := `SELECT * FROM studies`

	rows, err := e.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var studies []*database.Study
	for rows.Next() {
		var study database.Study
		if err := e.db.ScanStudy(rows, &study); err != nil {
			continue
		}
		studies = append(studies, &study)
	}

	return studies, nil
}

// Health checks if the service is operational
func (e *ExportService) Health(ctx context.Context) error {
	// Check database connection
	if err := e.db.Ping(); err != nil {
		return fmt.Errorf("database unhealthy: %w", err)
	}

	return nil
}

// Close releases resources
func (e *ExportService) Close() error {
	// ExportService doesn't hold any resources that need explicit cleanup
	return nil
}