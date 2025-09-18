package converter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nishad/srake/internal/database"
)

// ConversionResult represents the result of an accession conversion
type ConversionResult struct {
	Source     string   `json:"source" yaml:"source" xml:"source"`
	TargetType string   `json:"target_type" yaml:"target_type" xml:"target_type"`
	Targets    []string `json:"targets" yaml:"targets" xml:"targets"`
	Error      string   `json:"error,omitempty" yaml:"error,omitempty" xml:"error,omitempty"`
}

// Converter handles accession conversions between different databases
type Converter struct {
	db         *database.DB
	httpClient *http.Client
	cache      map[string]*ConversionResult
}

// NewConverter creates a new converter instance
func NewConverter(dbPath string) *Converter {
	db, _ := database.Initialize(dbPath)

	return &Converter{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: make(map[string]*ConversionResult),
	}
}

// Close closes the database connection
func (c *Converter) Close() {
	if c.db != nil {
		c.db.Close()
	}
}

// Convert performs accession conversion
func (c *Converter) Convert(accession, targetType string) (*ConversionResult, error) {
	// Normalize inputs
	accession = strings.TrimSpace(strings.ToUpper(accession))
	targetType = strings.ToUpper(targetType)

	// Check cache
	cacheKey := fmt.Sprintf("%s->%s", accession, targetType)
	if result, ok := c.cache[cacheKey]; ok {
		return result, nil
	}

	// Determine source type
	sourceType := c.detectAccessionType(accession)
	if sourceType == "" {
		return nil, fmt.Errorf("unknown accession type: %s", accession)
	}

	// Perform conversion based on source and target types
	var result *ConversionResult
	var err error

	switch sourceType {
	case "SRP":
		result, err = c.convertSRP(accession, targetType)
	case "SRX":
		result, err = c.convertSRX(accession, targetType)
	case "SRR":
		result, err = c.convertSRR(accession, targetType)
	case "SRS":
		result, err = c.convertSRS(accession, targetType)
	case "GSE":
		result, err = c.convertGSE(accession, targetType)
	case "GSM":
		result, err = c.convertGSM(accession, targetType)
	case "PRJNA", "PRJEB", "PRJDB":
		result, err = c.convertBioProject(accession, targetType)
	case "SAMN", "SAME", "SAMD":
		result, err = c.convertBioSample(accession, targetType)
	default:
		return nil, fmt.Errorf("conversion from %s not supported", sourceType)
	}

	if err != nil {
		return nil, err
	}

	// Cache the result
	c.cache[cacheKey] = result

	return result, nil
}

// detectAccessionType determines the type of accession
func (c *Converter) detectAccessionType(accession string) string {
	prefixes := map[string][]string{
		"SRP":   {"SRP"},
		"SRX":   {"SRX"},
		"SRR":   {"SRR"},
		"SRS":   {"SRS"},
		"GSE":   {"GSE"},
		"GSM":   {"GSM"},
		"PRJNA": {"PRJNA"},
		"PRJEB": {"PRJEB"},
		"PRJDB": {"PRJDB"},
		"SAMN":  {"SAMN"},
		"SAME":  {"SAME"},
		"SAMD":  {"SAMD"},
		"DRP":   {"DRP"},
		"DRX":   {"DRX"},
		"DRR":   {"DRR"},
		"DRS":   {"DRS"},
		"ERP":   {"ERP"},
		"ERX":   {"ERX"},
		"ERR":   {"ERR"},
		"ERS":   {"ERS"},
	}

	for accType, prefixList := range prefixes {
		for _, prefix := range prefixList {
			if strings.HasPrefix(accession, prefix) {
				return accType
			}
		}
	}

	return ""
}

// convertSRP converts SRP (SRA Project) accessions
func (c *Converter) convertSRP(accession, targetType string) (*ConversionResult, error) {
	result := &ConversionResult{
		Source:     accession,
		TargetType: targetType,
		Targets:    []string{},
	}

	switch targetType {
	case "GSE":
		// Query local database first
		targets, err := c.queryLocalDB("GSE", "study_accession", accession)
		if err == nil && len(targets) > 0 {
			result.Targets = targets
			return result, nil
		}

		// Fallback to NCBI API
		targets, err = c.queryNCBILink(accession, "gds")
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "SRX":
		// Get all experiments for this study
		targets, err := c.queryLocalDB("experiment_accession", "study_accession", accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "SRR":
		// Get all runs for this study
		query := `
			SELECT r.run_accession
			FROM runs r
			JOIN experiments e ON r.experiment_accession = e.experiment_accession
			WHERE e.study_accession = ?`
		targets, err := c.queryCustom(query, accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "SRS":
		// Get all samples for this study
		query := `
			SELECT DISTINCT s.sample_accession
			FROM samples s
			JOIN experiment_samples es ON s.sample_accession = es.sample_accession
			JOIN experiments e ON es.experiment_accession = e.experiment_accession
			WHERE e.study_accession = ?`
		targets, err := c.queryCustom(query, accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "PRJNA", "BIOPROJECT":
		// Convert to BioProject
		targets, err := c.queryNCBILink(accession, "bioproject")
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	default:
		return nil, fmt.Errorf("cannot convert SRP to %s", targetType)
	}

	return result, nil
}

// convertSRX converts SRX (SRA Experiment) accessions
func (c *Converter) convertSRX(accession, targetType string) (*ConversionResult, error) {
	result := &ConversionResult{
		Source:     accession,
		TargetType: targetType,
		Targets:    []string{},
	}

	switch targetType {
	case "GSM":
		// Query NCBI for GEO link
		targets, err := c.queryNCBILink(accession, "gds")
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "SRP":
		// Get parent study
		targets, err := c.queryLocalDB("study_accession", "experiment_accession", accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "SRR":
		// Get all runs for this experiment
		targets, err := c.queryLocalDB("run_accession", "experiment_accession", accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "SRS":
		// Get associated samples
		query := `
			SELECT sample_accession
			FROM experiment_samples
			WHERE experiment_accession = ?`
		targets, err := c.queryCustom(query, accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	default:
		return nil, fmt.Errorf("cannot convert SRX to %s", targetType)
	}

	return result, nil
}

// convertGSE converts GSE (GEO Series) accessions
func (c *Converter) convertGSE(accession, targetType string) (*ConversionResult, error) {
	result := &ConversionResult{
		Source:     accession,
		TargetType: targetType,
		Targets:    []string{},
	}

	switch targetType {
	case "SRP":
		// Query NCBI for SRA link
		targets, err := c.queryNCBILink(accession, "sra")
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "GSM":
		// Get all samples in this series
		targets, err := c.queryGEOSamples(accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	default:
		return nil, fmt.Errorf("cannot convert GSE to %s", targetType)
	}

	return result, nil
}

// convertGSM converts GSM (GEO Sample) accessions
func (c *Converter) convertGSM(accession, targetType string) (*ConversionResult, error) {
	result := &ConversionResult{
		Source:     accession,
		TargetType: targetType,
		Targets:    []string{},
	}

	switch targetType {
	case "SRX":
		// Query NCBI for SRA experiment link
		targets, err := c.queryNCBILink(accession, "sra")
		if err != nil {
			return nil, err
		}
		// Filter for SRX accessions
		srxTargets := []string{}
		for _, t := range targets {
			if strings.HasPrefix(t, "SRX") {
				srxTargets = append(srxTargets, t)
			}
		}
		result.Targets = srxTargets

	case "SRR":
		// Query NCBI for SRA run links
		targets, err := c.queryNCBILink(accession, "sra")
		if err != nil {
			return nil, err
		}
		// Filter for SRR accessions
		srrTargets := []string{}
		for _, t := range targets {
			if strings.HasPrefix(t, "SRR") {
				srrTargets = append(srrTargets, t)
			}
		}
		result.Targets = srrTargets

	case "GSE":
		// Get parent series
		targets, err := c.queryGEOParentSeries(accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	default:
		return nil, fmt.Errorf("cannot convert GSM to %s", targetType)
	}

	return result, nil
}

// convertSRR converts SRR (SRA Run) accessions
func (c *Converter) convertSRR(accession, targetType string) (*ConversionResult, error) {
	result := &ConversionResult{
		Source:     accession,
		TargetType: targetType,
		Targets:    []string{},
	}

	switch targetType {
	case "SRX":
		// Get parent experiment
		targets, err := c.queryLocalDB("experiment_accession", "run_accession", accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "SRP":
		// Get parent study through experiment
		query := `
			SELECT e.study_accession
			FROM experiments e
			JOIN runs r ON e.experiment_accession = r.experiment_accession
			WHERE r.run_accession = ?`
		targets, err := c.queryCustom(query, accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "GSM":
		// Query NCBI for GEO link
		targets, err := c.queryNCBILink(accession, "gds")
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	default:
		return nil, fmt.Errorf("cannot convert SRR to %s", targetType)
	}

	return result, nil
}

// convertSRS converts SRS (SRA Sample) accessions
func (c *Converter) convertSRS(accession, targetType string) (*ConversionResult, error) {
	result := &ConversionResult{
		Source:     accession,
		TargetType: targetType,
		Targets:    []string{},
	}

	switch targetType {
	case "SRX":
		// Get associated experiments
		query := `
			SELECT experiment_accession
			FROM experiment_samples
			WHERE sample_accession = ?`
		targets, err := c.queryCustom(query, accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "GSM":
		// Query NCBI for GEO link
		targets, err := c.queryNCBILink(accession, "gds")
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	case "BIOSAMPLE":
		// Get BioSample ID
		targets, err := c.queryLocalDB("id_value", "record_accession", accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	default:
		return nil, fmt.Errorf("cannot convert SRS to %s", targetType)
	}

	return result, nil
}

// convertBioProject converts BioProject accessions
func (c *Converter) convertBioProject(accession, targetType string) (*ConversionResult, error) {
	result := &ConversionResult{
		Source:     accession,
		TargetType: targetType,
		Targets:    []string{},
	}

	switch targetType {
	case "SRP":
		// Query NCBI for SRA link
		targets, err := c.queryNCBILink(accession, "sra")
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	default:
		return nil, fmt.Errorf("cannot convert BioProject to %s", targetType)
	}

	return result, nil
}

// convertBioSample converts BioSample accessions
func (c *Converter) convertBioSample(accession, targetType string) (*ConversionResult, error) {
	result := &ConversionResult{
		Source:     accession,
		TargetType: targetType,
		Targets:    []string{},
	}

	switch targetType {
	case "SRS":
		// Query local database for SRA sample
		query := `
			SELECT record_accession
			FROM identifiers
			WHERE id_value = ? AND record_type = 'sample'`
		targets, err := c.queryCustom(query, accession)
		if err != nil {
			return nil, err
		}
		result.Targets = targets

	default:
		return nil, fmt.Errorf("cannot convert BioSample to %s", targetType)
	}

	return result, nil
}

// Helper methods for database queries

func (c *Converter) queryLocalDB(selectField, whereField, value string) ([]string, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Determine table based on field names
	table := c.determineTable(selectField, whereField)
	if table == "" {
		return nil, fmt.Errorf("cannot determine table for fields")
	}

	query := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s = ?", selectField, table, whereField)
	return c.queryCustom(query, value)
}

func (c *Converter) queryCustom(query string, args ...interface{}) ([]string, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := c.db.GetSQLDB().Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []string{}
	for rows.Next() {
		var value sql.NullString
		if err := rows.Scan(&value); err != nil {
			continue
		}
		if value.Valid && value.String != "" {
			results = append(results, value.String)
		}
	}

	return results, nil
}

func (c *Converter) determineTable(fields ...string) string {
	// Simple heuristic to determine table
	for _, field := range fields {
		if strings.Contains(field, "study") {
			return "studies"
		}
		if strings.Contains(field, "experiment") {
			return "experiments"
		}
		if strings.Contains(field, "sample") {
			return "samples"
		}
		if strings.Contains(field, "run") {
			return "runs"
		}
	}
	return ""
}

// NCBI API query methods

func (c *Converter) queryNCBILink(accession, targetDB string) ([]string, error) {
	// Use NCBI eutils to find linked records
	url := fmt.Sprintf("https://eutils.ncbi.nlm.nih.gov/entrez/eutils/elink.fcgi?dbfrom=sra&db=%s&id=%s&retmode=json",
		targetDB, accession)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Extract linked IDs
	targets := []string{}
	// Note: This is simplified - actual NCBI response parsing would be more complex
	// You would need to navigate the JSON structure to find linked accessions

	return targets, nil
}

func (c *Converter) queryGEOSamples(gseAccession string) ([]string, error) {
	// Query GEO for samples in a series
	// This would typically use GEO API or web scraping
	// For now, returning empty as a placeholder
	return []string{}, nil
}

func (c *Converter) queryGEOParentSeries(gsmAccession string) ([]string, error) {
	// Query GEO for parent series of a sample
	// This would typically use GEO API or web scraping
	// For now, returning empty as a placeholder
	return []string{}, nil
}