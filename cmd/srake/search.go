package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nishad/srake/internal/cli"
	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/paths"
	"github.com/nishad/srake/internal/search"
	"github.com/nishad/srake/internal/ui"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search SRA metadata using full-text or vector search",
	Long: `Search the SRA metadata database using powerful full-text search with Bleve.

The search supports:
  • Full-text search across all metadata fields
  • Organism names, accession numbers, and keywords
  • Advanced filtering by platform, library strategy, and other fields
  • Fuzzy search for typo tolerance
  • Multiple output formats (table, JSON, CSV, TSV)
  • Export results to file

Search modes:
  • text: Standard full-text search (default)
  • fuzzy: Typo-tolerant search
  • filter: Search with specific field filters
  • stats: Show search statistics only`,
	Example: `  # Basic search
  srake search "homo sapiens"
  srake search "RNA-seq human cancer"

  # Search with filters
  srake search --organism "homo sapiens" --platform ILLUMINA
  srake search --library-strategy "RNA-Seq" --limit 50

  # Fuzzy search for typo tolerance
  srake search "humna" --fuzzy

  # Export results
  srake search "mouse brain" --format json --output results.json
  srake search "COVID-19" --format csv > results.csv

  # Show all available data (no query)
  srake search --limit 100

  # Get search statistics
  srake search --stats`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSearch,
}

var (
	// Filter flags
	searchOrganism         string
	searchPlatform         string
	searchLibraryStrategy  string
	searchLibrarySource    string
	searchLibrarySelection string
	searchLibraryLayout    string
	searchStudyType        string
	searchInstrumentModel  string
	searchDateFrom         string
	searchDateTo           string
	searchSpotsMin         int64
	searchSpotsMax         int64
	searchBasesMin         int64
	searchBasesMax         int64

	// Output flags
	searchLimit    int
	searchOffset   int
	searchFormat   string
	searchOutput   string
	searchNoHeader bool
	searchFields   string

	// Search mode flags
	searchFuzzy       bool
	searchExact       bool
	searchStats       bool
	searchFacets      bool
	searchHighlight   bool
	searchAdvanced    bool
	searchBoolOp      string
	searchAggregateBy string
	searchCountOnly   bool
	searchGroupBy     string

	// Advanced flags
	searchIndexPath string
	searchNoCache   bool
	searchTimeout   int

	// Search mode flags
	searchMode         string
	searchNoFTS        bool
	searchNoVectors    bool
	searchVectorWeight float64
	searchKNN          int

	// Quality control flags
	searchSimilarityThreshold float32
	searchMinScore            float32
	searchTopPercentile       int
	searchShowConfidence      bool
	searchHybridWeight        float32
)

func init() {
	// Filter flags
	searchCmd.Flags().StringVarP(&searchOrganism, "organism", "o", "", "Filter by organism")
	searchCmd.Flags().StringVar(&searchPlatform, "platform", "", "Filter by platform")
	searchCmd.Flags().StringVar(&searchLibraryStrategy, "library-strategy", "", "Filter by library strategy")
	searchCmd.Flags().StringVar(&searchLibrarySource, "library-source", "", "Filter by library source")
	searchCmd.Flags().StringVar(&searchLibrarySelection, "library-selection", "", "Filter by library selection")
	searchCmd.Flags().StringVar(&searchLibraryLayout, "library-layout", "", "Filter by library layout")
	searchCmd.Flags().StringVar(&searchStudyType, "study-type", "", "Filter by study type")
	searchCmd.Flags().StringVar(&searchInstrumentModel, "instrument-model", "", "Filter by instrument model")
	searchCmd.Flags().StringVar(&searchDateFrom, "date-from", "", "Filter by submission date from (YYYY-MM-DD)")
	searchCmd.Flags().StringVar(&searchDateTo, "date-to", "", "Filter by submission date to (YYYY-MM-DD)")
	searchCmd.Flags().Int64Var(&searchSpotsMin, "spots-min", 0, "Filter by minimum number of spots")
	searchCmd.Flags().Int64Var(&searchSpotsMax, "spots-max", 0, "Filter by maximum number of spots")
	searchCmd.Flags().Int64Var(&searchBasesMin, "bases-min", 0, "Filter by minimum number of bases")
	searchCmd.Flags().Int64Var(&searchBasesMax, "bases-max", 0, "Filter by maximum number of bases")

	// Quality control flags with short aliases
	searchCmd.Flags().Float32VarP(&searchSimilarityThreshold, "similarity-threshold", "s", 0.5, "Minimum cosine similarity for vector search (0-1, where 1=exact match)")
	searchCmd.Flags().Float32VarP(&searchMinScore, "min-score", "m", 0.0, "Minimum BM25 score for text search results")
	searchCmd.Flags().IntVarP(&searchTopPercentile, "top-percentile", "t", 0, "Only show top N percentile of results (0=disabled)")
	searchCmd.Flags().BoolVarP(&searchShowConfidence, "show-confidence", "c", false, "Display confidence levels for results")
	searchCmd.Flags().Float32VarP(&searchHybridWeight, "hybrid-weight", "w", 0.7, "Weight for vector scores in hybrid search (0=text only, 1=vector only)")

	// Output flags
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 100, "Maximum results to return")
	searchCmd.Flags().IntVar(&searchOffset, "offset", 0, "Number of results to skip")
	searchCmd.Flags().StringVarP(&searchFormat, "format", "f", "table", "Output format (table|json|csv|tsv)")
	searchCmd.Flags().StringVar(&searchOutput, "output", "", "Save results to file")
	searchCmd.Flags().BoolVar(&searchNoHeader, "no-header", false, "Omit header in output")
	searchCmd.Flags().StringVar(&searchFields, "fields", "", "Comma-separated list of fields to display")

	// Search mode flags
	searchCmd.Flags().BoolVar(&searchFuzzy, "fuzzy", false, "Enable fuzzy search for typo tolerance")
	searchCmd.Flags().BoolVar(&searchExact, "exact", false, "Enable exact match search")
	searchCmd.Flags().BoolVar(&searchStats, "stats", false, "Show search statistics only")
	searchCmd.Flags().BoolVar(&searchFacets, "facets", false, "Show search facets")
	searchCmd.Flags().BoolVar(&searchHighlight, "highlight", false, "Highlight search terms in results")
	searchCmd.Flags().BoolVar(&searchAdvanced, "advanced", false, "Enable advanced search syntax")
	searchCmd.Flags().StringVar(&searchMode, "search-mode", "auto", "Search mode (auto|text|vector|hybrid|database)")
	searchCmd.Flags().BoolVar(&searchNoFTS, "no-fts", false, "Disable full-text search")
	searchCmd.Flags().BoolVar(&searchNoVectors, "no-vectors", false, "Disable vector search")

	// Advanced flags
	searchCmd.Flags().StringVar(&searchIndexPath, "index-path", "", "Path to search index")
	searchCmd.Flags().BoolVar(&searchNoCache, "no-cache", false, "Disable search cache")
	searchCmd.Flags().IntVar(&searchTimeout, "timeout", 30, "Search timeout in seconds")

	// Hide some advanced flags by default
	searchCmd.Flags().MarkHidden("no-cache")
	searchCmd.Flags().MarkHidden("timeout")
	searchCmd.Flags().MarkHidden("advanced")

	// Setup grouped help
	cli.SetupGroupedHelp(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	// If --stats flag is set, show statistics only
	if searchStats {
		return showSearchStats()
	}

	// Build filters from flags
	filters := make(map[string]string)
	if searchOrganism != "" {
		filters["organism"] = searchOrganism
	}
	if searchPlatform != "" {
		filters["platform"] = searchPlatform
	}
	if searchLibraryStrategy != "" {
		filters["library_strategy"] = searchLibraryStrategy
	}
	if searchLibrarySource != "" {
		filters["library_source"] = searchLibrarySource
	}
	if searchLibrarySelection != "" {
		filters["library_selection"] = searchLibrarySelection
	}
	if searchLibraryLayout != "" {
		filters["library_layout"] = searchLibraryLayout
	}
	if searchStudyType != "" {
		filters["study_type"] = searchStudyType
	}
	if searchInstrumentModel != "" {
		filters["instrument_model"] = searchInstrumentModel
	}
	if searchDateFrom != "" {
		filters["submission_date_from"] = searchDateFrom
	}
	if searchDateTo != "" {
		filters["submission_date_to"] = searchDateTo
	}
	if searchSpotsMin > 0 {
		filters["spots_min"] = fmt.Sprintf("%d", searchSpotsMin)
	}
	if searchSpotsMax > 0 {
		filters["spots_max"] = fmt.Sprintf("%d", searchSpotsMax)
	}
	if searchBasesMin > 0 {
		filters["bases_min"] = fmt.Sprintf("%d", searchBasesMin)
	}
	if searchBasesMax > 0 {
		filters["bases_max"] = fmt.Sprintf("%d", searchBasesMax)
	}

	// Start spinner for immediate feedback (within 100ms)
	var spinner *ui.Spinner
	if !quiet && isTerminal() {
		var message string
		if query != "" {
			message = fmt.Sprintf("Searching for \"%s\"", query)
		} else if len(filters) > 0 {
			message = "Searching with filters"
		} else {
			message = "Fetching all records"
		}
		spinner = ui.NewSpinner(message)
		spinner.Start()
		defer func() {
			if spinner != nil {
				spinner.Stop("")
			}
		}()
	}

	// Always use local search - CLI should work independently
	err := performSearch(query, filters)
	if spinner != nil {
		if err != nil {
			spinner.Stop(fmt.Sprintf("✗ Search failed: %v", err))
		} else {
			spinner.Stop("✓ Search completed")
		}
	}
	return err
}

// performSearch performs search using local Bleve index and database
func performSearch(query string, filters map[string]string) error {
	// Load config
	cfg := config.DefaultConfig()

	// Find data directory
	dataDir := paths.GetPaths().DataDir
	cfg.DataDirectory = dataDir

	// Apply search mode overrides from CLI flags
	if searchNoFTS {
		cfg.Search.Enabled = false
	}
	if searchNoVectors {
		cfg.Vectors.Enabled = false
		cfg.Embeddings.Enabled = false
	}

	// Determine effective search mode
	effectiveMode := determineSearchMode(cfg)

	// Override index path if specified
	if searchIndexPath != "" {
		cfg.Search.IndexPath = searchIndexPath
	} else {
		cfg.Search.IndexPath = paths.GetIndexPath()
	}

	// For database-only mode, skip index check
	if effectiveMode == "database" {
		return performDatabaseSearch(query, filters)
	}

	// Check if index exists for FTS/vector modes
	if _, err := os.Stat(cfg.Search.IndexPath); os.IsNotExist(err) {
		if searchMode != "database" {
			printError("Search index not found at %s", cfg.Search.IndexPath)
			fmt.Fprintf(os.Stderr, "\nPlease build the search index first:\n")
			fmt.Fprintf(os.Stderr, "  srake search index --build\n")
			fmt.Fprintf(os.Stderr, "\nOr use database-only mode:\n")
			fmt.Fprintf(os.Stderr, "  srake search --search-mode database \"your query\"\n")
			return fmt.Errorf("index not found")
		}
	}

	// Initialize Bleve index
	idx, err := search.InitBleveIndex(cfg.Search.IndexPath)
	if err != nil {
		return fmt.Errorf("failed to open search index: %v", err)
	}
	defer idx.Close()

	// Perform search based on mode
	var results interface{}
	startTime := time.Now()

	if searchAdvanced && query != "" {
		// Advanced query parsing
		parser := search.NewQueryParser()
		advancedQuery, err := parser.ParseAdvancedQuery(query)
		if err != nil {
			return fmt.Errorf("failed to parse advanced query: %v", err)
		}

		// Add filters to advanced query
		if len(filters) > 0 {
			filterQueries := parser.ParseFilters(filters)
			allQueries := []interface{}{advancedQuery}
			for _, fq := range filterQueries {
				allQueries = append(allQueries, fq)
			}
			// Use bleve directly for conjunction
			var finalQuery interface{}
			if len(allQueries) > 1 {
				// Import needed at top
				finalQuery = idx.BuildConjunctionQuery(allQueries)
			} else {
				finalQuery = advancedQuery
			}
			bleveResult, err := idx.SearchWithQuery(finalQuery, searchLimit)
			if err != nil {
				return fmt.Errorf("advanced search failed: %v", err)
			}
			results = bleveResult
		} else {
			bleveResult, err := idx.SearchWithQuery(advancedQuery, searchLimit)
			if err != nil {
				return fmt.Errorf("advanced search failed: %v", err)
			}
			results = bleveResult
		}
	} else if searchFuzzy && query != "" {
		// Fuzzy search
		bleveResult, err := idx.FuzzySearch(query, 2, searchLimit)
		if err != nil {
			return fmt.Errorf("fuzzy search failed: %v", err)
		}
		results = bleveResult
	} else if len(filters) > 0 {
		// Filtered search
		bleveResult, err := idx.SearchWithFilters(query, filters, searchLimit)
		if err != nil {
			return fmt.Errorf("filtered search failed: %v", err)
		}
		results = bleveResult
	} else {
		// Regular search
		bleveResult, err := idx.Search(query, searchLimit)
		if err != nil {
			return fmt.Errorf("search failed: %v", err)
		}
		results = bleveResult
	}

	elapsed := time.Since(startTime)

	// Handle aggregation if requested
	if searchAggregateBy != "" || searchCountOnly {
		return formatAggregatedResults(results, query, elapsed)
	}

	// Format and output results
	return formatSearchResults(results, query, elapsed)
}

// formatSearchResults formats search results based on output format
func formatSearchResults(results interface{}, query string, elapsed time.Duration) error {
	// Type assertion for Bleve results
	bleveResult, ok := results.(*search.BleveSearchResult)
	if !ok {
		return fmt.Errorf("unexpected result type")
	}

	// Handle different output formats
	switch searchFormat {
	case "json":
		return outputJSON(bleveResult)
	case "csv":
		return outputCSV(bleveResult, ",")
	case "tsv":
		return outputCSV(bleveResult, "\t")
	case "accession":
		return outputAccessions(bleveResult)
	default:
		return outputTable(bleveResult, query, elapsed)
	}
}

// outputTable outputs results in table format
func outputTable(result *search.BleveSearchResult, query string, elapsed time.Duration) error {
	if result.Total == 0 {
		if query != "" {
			printInfo("No results found for \"%s\"", query)
		} else {
			printInfo("No results found")
		}
		return nil
	}

	// Create output writer
	var w *tabwriter.Writer
	if searchOutput != "" {
		file, err := os.Create(searchOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		w = tabwriter.NewWriter(file, 0, 0, 2, ' ', 0)
	} else {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	}

	// Header
	if !searchNoHeader {
		headers := []string{"ACCESSION", "TYPE", "TITLE", "ORGANISM", "PLATFORM"}
		if searchFields != "" {
			headers = strings.Split(strings.ToUpper(searchFields), ",")
		}

		for i, h := range headers {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, colorize(colorBold, h))
		}
		fmt.Fprintln(w)

		// Separator line
		if isTerminal() && !noColor {
			fmt.Fprintln(w, colorize(colorGray, strings.Repeat("─", 100)))
		}
	}

	// Results
	for _, hit := range result.Hits {
		fields := hit.Fields

		// Extract common fields
		accession := hit.ID
		docType := getField(fields, "type")
		title := truncate(getField(fields, "title", "study_title"), 40)
		organism := getField(fields, "organism")
		platform := getField(fields, "platform")

		if searchFields != "" {
			// Custom fields
			requestedFields := strings.Split(searchFields, ",")
			for i, f := range requestedFields {
				if i > 0 {
					fmt.Fprint(w, "\t")
				}
				value := getField(fields, strings.TrimSpace(f))
				if i == 0 && isTerminal() && !noColor {
					fmt.Fprint(w, colorize(colorCyan, value))
				} else {
					fmt.Fprint(w, value)
				}
			}
		} else {
			// Default fields
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s",
				colorize(colorCyan, accession),
				docType,
				title,
				organism,
				platform)
		}

		// Add highlights if requested
		if searchHighlight && len(hit.Fragments) > 0 {
			fmt.Fprint(w, "\t")
			for field, fragments := range hit.Fragments {
				for _, fragment := range fragments {
					fmt.Fprintf(w, "[%s: %s] ", field, fragment)
				}
			}
		}

		fmt.Fprintln(w)
	}

	w.Flush()

	// Summary
	if !quiet {
		fmt.Printf("\n%s\n", colorize(colorGray,
			fmt.Sprintf("Found %d results in %v (showing %d)",
				result.Total, elapsed, len(result.Hits))))

		// Show facets if requested
		if searchFacets && len(result.Facets) > 0 {
			fmt.Println("\n" + colorize(colorBold, "Facets:"))
			for facetName, facet := range result.Facets {
				fmt.Printf("\n  %s:\n", colorize(colorBold, facetName))
				for _, term := range facet.Terms.Terms() {
					fmt.Printf("    %s: %d\n", term.Term, term.Count)
				}
			}
		}
	}

	return nil
}

// outputJSON outputs results as JSON
func outputJSON(result *search.BleveSearchResult) error {
	output := map[string]interface{}{
		"total":     result.Total,
		"hits":      result.Hits,
		"facets":    result.Facets,
		"max_score": result.MaxScore,
	}

	var encoder *json.Encoder
	if searchOutput != "" {
		file, err := os.Create(searchOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		encoder = json.NewEncoder(file)
	} else {
		encoder = json.NewEncoder(os.Stdout)
	}

	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// outputCSV outputs results as CSV or TSV
func outputCSV(result *search.BleveSearchResult, separator string) error {
	var writer *csv.Writer

	if searchOutput != "" {
		file, err := os.Create(searchOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		writer = csv.NewWriter(file)
	} else {
		writer = csv.NewWriter(os.Stdout)
	}

	if separator == "\t" {
		writer.Comma = '\t'
	}

	// Write header
	if !searchNoHeader {
		headers := []string{"accession", "type", "title", "organism", "platform", "library_strategy"}
		if searchFields != "" {
			headers = strings.Split(searchFields, ",")
		}
		if err := writer.Write(headers); err != nil {
			return err
		}
	}

	// Write data
	for _, hit := range result.Hits {
		fields := hit.Fields

		row := []string{
			hit.ID,
			getField(fields, "type"),
			getField(fields, "title", "study_title"),
			getField(fields, "organism"),
			getField(fields, "platform"),
			getField(fields, "library_strategy"),
		}

		if searchFields != "" {
			row = nil
			for _, f := range strings.Split(searchFields, ",") {
				row = append(row, getField(fields, strings.TrimSpace(f)))
			}
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

// outputAccessions outputs only accession numbers
func outputAccessions(result *search.BleveSearchResult) error {
	var output *os.File
	if searchOutput != "" {
		file, err := os.Create(searchOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		output = file
	} else {
		output = os.Stdout
	}

	for _, hit := range result.Hits {
		fmt.Fprintln(output, hit.ID)
	}

	return nil
}

// showSearchStats displays search index statistics
func showSearchStats() error {
	cfg := config.DefaultConfig()
	dataDir := paths.GetPaths().DataDir

	// Open database for stats
	dbPath := paths.GetDatabasePath()
	sqlDB, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer sqlDB.Close()

	db := &database.DB{DB: sqlDB}

	// Create search manager
	cfg.DataDirectory = dataDir
	cfg.Search.Enabled = true

	manager, err := search.NewManager(cfg, db)
	if err != nil {
		return fmt.Errorf("failed to create search manager: %v", err)
	}
	defer manager.Close()

	// Get statistics
	stats, err := manager.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get search statistics: %v", err)
	}

	// Display statistics
	fmt.Println(colorize(colorBold, "Search Index Statistics"))
	fmt.Println(strings.Repeat("─", 50))

	fmt.Printf("Backend:         %s\n", stats.Backend)
	fmt.Printf("Document Count:  %d\n", stats.DocumentCount)
	fmt.Printf("Index Size:      %.2f MB\n", float64(stats.IndexSize)/(1024*1024))
	fmt.Printf("Last Modified:   %s\n", stats.LastModified.Format(time.RFC3339))
	fmt.Printf("Vectors Enabled: %v\n", stats.VectorsEnabled)
	fmt.Printf("Index Healthy:   %v\n", stats.IsHealthy)

	return nil
}

// Helper functions

// determineSearchMode determines the effective search mode based on config and flags
func determineSearchMode(cfg *config.Config) string {
	// Explicit mode from CLI
	if searchMode != "auto" && searchMode != "" {
		return searchMode
	}

	// Auto-detect based on what's enabled
	if !cfg.Search.Enabled || searchNoFTS {
		return "database"
	}

	if cfg.Vectors.Enabled && !searchNoVectors {
		if searchVectorWeight >= 1.0 {
			return "vector"
		}
		return "hybrid"
	}

	return "fts"
}

// performDatabaseSearch performs search using only SQLite database
func performDatabaseSearch(query string, filters map[string]string) error {
	db, err := database.Initialize(paths.GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Build SQL query with filters
	sqlQuery := buildSQLQuery(query, filters)

	// Execute query
	rows, err := db.GetSQLDB().Query(sqlQuery)
	if err != nil {
		return fmt.Errorf("database query failed: %v", err)
	}
	defer rows.Close()

	// Process and display results
	return displayDatabaseResults(rows)
}

// buildSQLQuery builds a SQL query for database-only search
func buildSQLQuery(query string, filters map[string]string) string {
	// Basic implementation - will be expanded
	whereClause := []string{}

	if query != "" {
		// Simple text search across key fields
		whereClause = append(whereClause, fmt.Sprintf(
			"(study_title LIKE '%%%s%%' OR study_abstract LIKE '%%%s%%' OR organism LIKE '%%%s%%')",
			query, query, query))
	}

	for field, value := range filters {
		// Map filter fields to database columns
		dbField := field
		switch field {
		case "library_strategy", "library_source", "library_selection", "library_layout":
			// These are in metadata JSON
			whereClause = append(whereClause, fmt.Sprintf("json_extract(metadata, '$.%s') = '%s'", field, value))
		case "platform", "instrument_model":
			// Also in metadata
			whereClause = append(whereClause, fmt.Sprintf("json_extract(metadata, '$.%s') = '%s'", field, value))
		default:
			whereClause = append(whereClause, fmt.Sprintf("%s = '%s'", dbField, value))
		}
	}

	sql := "SELECT * FROM studies"
	if len(whereClause) > 0 {
		sql += " WHERE " + strings.Join(whereClause, " AND ")
	}
	sql += fmt.Sprintf(" LIMIT %d OFFSET %d", searchLimit, searchOffset)

	return sql
}

// displayDatabaseResults displays results from database-only search
func displayDatabaseResults(rows *sql.Rows) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}

	// Read all results
	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice of interface{} to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		// Convert to map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		if !quiet {
			fmt.Println("No results found")
		}
		return nil
	}

	// Format based on output format
	switch searchFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(results)

	case "csv", "tsv":
		delimiter := ','
		if searchFormat == "tsv" {
			delimiter = '\t'
		}
		writer := csv.NewWriter(os.Stdout)
		writer.Comma = delimiter

		// Write header
		if !searchNoHeader {
			displayCols := []string{"study_accession", "study_title", "organism", "study_type"}
			writer.Write(displayCols)
		}

		// Write data
		for _, row := range results {
			record := []string{
				fmt.Sprintf("%v", row["study_accession"]),
				truncate(fmt.Sprintf("%v", row["study_title"]), 60),
				fmt.Sprintf("%v", row["organism"]),
				fmt.Sprintf("%v", row["study_type"]),
			}
			writer.Write(record)
		}
		writer.Flush()
		return writer.Error()

	default: // table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		if !searchNoHeader {
			fmt.Fprintln(w, "ACCESSION\tTITLE\tORGANISM\tTYPE")
			fmt.Fprintln(w, strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 50)+"\t"+strings.Repeat("-", 20)+"\t"+strings.Repeat("-", 15))
		}

		for _, row := range results {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n",
				row["study_accession"],
				truncate(fmt.Sprintf("%v", row["study_title"]), 50),
				truncate(fmt.Sprintf("%v", row["organism"]), 20),
				row["study_type"],
			)
		}
		w.Flush()
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "\nFound %d results\n", len(results))
	}

	return nil
}

func getField(fields map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := fields[key]; ok {
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatAggregatedResults formats aggregated search results
func formatAggregatedResults(results interface{}, query string, elapsed time.Duration) error {
	bleveResult, ok := results.(*search.BleveSearchResult)
	if !ok {
		return fmt.Errorf("unexpected result type")
	}

	// If count-only, just show the count
	if searchCountOnly {
		if searchFormat == "json" {
			output := map[string]interface{}{
				"query":   query,
				"count":   bleveResult.Total,
				"time_ms": elapsed.Milliseconds(),
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Printf("%d\n", bleveResult.Total)
		if verbose {
			fmt.Printf("# Query: %s, Time: %v\n", query, elapsed)
		}
		return nil
	}

	// Aggregate by field
	if searchAggregateBy != "" && len(bleveResult.Facets) > 0 {
		// Find the requested facet
		facet, exists := bleveResult.Facets[searchAggregateBy]
		if !exists {
			// Try to find it in the results
			aggregation := make(map[string]int)
			for _, hit := range bleveResult.Hits {
				if val, ok := hit.Fields[searchAggregateBy]; ok {
					key := fmt.Sprintf("%v", val)
					aggregation[key]++
				}
			}

			// Output aggregation
			if searchFormat == "json" {
				output := map[string]interface{}{
					"query":             query,
					"aggregation_field": searchAggregateBy,
					"values":            aggregation,
					"total":             bleveResult.Total,
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(output)
			}

			// Table format
			fmt.Println(colorize(colorBold, fmt.Sprintf("Aggregation by %s", searchAggregateBy)))
			fmt.Println(strings.Repeat("─", 50))

			for key, count := range aggregation {
				fmt.Printf("%-40s %d\n", key, count)
			}

			if !quiet {
				fmt.Printf("\n%s\n", colorize(colorGray,
					fmt.Sprintf("Total: %d results in %v", bleveResult.Total, elapsed)))
			}
		} else {
			// Use facet data
			if searchFormat == "json" {
				output := map[string]interface{}{
					"query":             query,
					"aggregation_field": searchAggregateBy,
					"facet":             facet,
					"total":             bleveResult.Total,
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(output)
			}

			// Table format
			fmt.Println(colorize(colorBold, fmt.Sprintf("Aggregation by %s", searchAggregateBy)))
			fmt.Println(strings.Repeat("─", 50))

			for _, term := range facet.Terms.Terms() {
				fmt.Printf("%-40s %d\n", term.Term, term.Count)
			}

			if !quiet {
				fmt.Printf("\n%s\n", colorize(colorGray,
					fmt.Sprintf("Total: %d results in %v", bleveResult.Total, elapsed)))
			}
		}
	}

	return nil
}
