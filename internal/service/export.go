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

	"github.com/nishad/srake/internal/database"
)

// ExportService handles data export in various formats
type ExportService struct {
	db        *database.DB
	searchSvc *SearchService
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
func (e *ExportService) exportJSON(results []*SearchResult, writer io.Writer, fields []string) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	// If fields are specified, filter the results
	if len(fields) > 0 {
		filtered := make([]map[string]interface{}, 0, len(results))
		for _, res := range results {
			filtered = append(filtered, e.filterFields(res.Fields, fields))
		}
		return encoder.Encode(filtered)
	}

	return encoder.Encode(results)
}

// exportJSONLines exports results as newline-delimited JSON
func (e *ExportService) exportJSONLines(results []*SearchResult, writer io.Writer, fields []string) error {
	encoder := json.NewEncoder(writer)

	for _, res := range results {
		data := res.Fields
		if len(fields) > 0 {
			data = e.filterFields(res.Fields, fields)
		}
		if err := encoder.Encode(data); err != nil {
			return err
		}
	}

	return nil
}

// exportCSV exports results as CSV
func (e *ExportService) exportCSV(results []*SearchResult, writer io.Writer, fields []string) error {
	w := csv.NewWriter(writer)
	defer w.Flush()

	// Write header
	headers := fields
	if len(headers) == 0 {
		// Use default headers
		headers = []string{"id", "type", "title", "description", "organism", "platform", "library_strategy"}
	}
	if err := w.Write(headers); err != nil {
		return err
	}

	// Write data rows
	for _, res := range results {
		row := make([]string, len(headers))
		for i, header := range headers {
			if header == "id" {
				row[i] = res.ID
			} else if header == "type" {
				row[i] = res.Type
			} else if val, ok := res.Fields[header]; ok {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// exportTSV exports results as TSV
func (e *ExportService) exportTSV(results []*SearchResult, writer io.Writer, fields []string) error {
	w := csv.NewWriter(writer)
	w.Comma = '\t'
	defer w.Flush()

	// Write header
	headers := fields
	if len(headers) == 0 {
		// Use default headers
		headers = []string{"id", "type", "title", "description", "organism", "platform", "library_strategy"}
	}
	if err := w.Write(headers); err != nil {
		return err
	}

	// Write data rows
	for _, res := range results {
		row := make([]string, len(headers))
		for i, header := range headers {
			if header == "id" {
				row[i] = res.ID
			} else if header == "type" {
				row[i] = res.Type
			} else if val, ok := res.Fields[header]; ok {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// exportXML exports results as XML
func (e *ExportService) exportXML(results []*SearchResult, writer io.Writer, fields []string) error {
	type XMLResult struct {
		Type  string      `xml:"type,attr"`
		ID    string      `xml:"id,attr"`
		Score float32     `xml:"score,omitempty"`
		Data  interface{} `xml:"data"`
	}

	type XMLExport struct {
		XMLName xml.Name    `xml:"export"`
		Results []XMLResult `xml:"result"`
	}

	export := XMLExport{
		Results: make([]XMLResult, 0, len(results)),
	}

	for _, res := range results {
		result := XMLResult{
			Type:  res.Type,
			ID:    res.ID,
			Score: res.Score,
		}

		if len(fields) > 0 {
			result.Data = e.filterFields(res.Fields, fields)
		} else {
			result.Data = res.Fields
		}

		export.Results = append(export.Results, result)
	}

	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	// Write XML header
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return err
	}

	return encoder.Encode(export)
}

// filterFields filters the fields to export
func (e *ExportService) filterFields(data map[string]interface{}, fields []string) map[string]interface{} {
	if len(fields) == 0 || data == nil {
		return data
	}

	filtered := make(map[string]interface{})
	for _, field := range fields {
		if val, ok := data[field]; ok {
			filtered[field] = val
		}
	}

	return filtered
}

// Close cleans up the export service
func (e *ExportService) Close() error {
	// Nothing to clean up for now
	return nil
}

// Health checks if the export service is operational
func (e *ExportService) Health(ctx context.Context) error {
	// Export service is healthy if search service is healthy
	if e.searchSvc != nil {
		return e.searchSvc.Health(ctx)
	}
	return nil
}
