package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/gorilla/mux"
	"github.com/nishad/srake/internal/cli"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/embeddings"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	noColor bool
	verbose bool
	quiet   bool

	// Version information
	version = "0.0.1-alpha"
	commit  = "dev"
	date    = "unknown"
)

// Color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// Check if output is to terminal
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Apply color if terminal output and color enabled
func colorize(color, text string) string {
	if !noColor && isTerminal() && os.Getenv("NO_COLOR") == "" {
		return color + text + colorReset
	}
	return text
}

// Print error message in user-friendly format
func printError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", colorize(colorRed, "✗"), msg)
}

// Print success message
func printSuccess(format string, args ...interface{}) {
	if !quiet {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", colorize(colorGreen, "✓"), msg)
	}
}

// Print info message
func printInfo(format string, args ...interface{}) {
	if !quiet {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s\n", colorize(colorCyan, msg))
	}
}

// Root command
var rootCmd = &cobra.Command{
	Use:   "srake",
	Short: "Search and download SRA metadata",
	Long: colorize(colorBold, "srake") + ` - A fast, user-friendly tool for SRA metadata

Srake provides a unified interface for searching, downloading, and serving
SRA (Sequence Read Archive) metadata from NCBI.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	Example: `  srake search "homo sapiens"
  srake server --port 8080
  srake db info`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Check NO_COLOR environment variable
		if os.Getenv("NO_COLOR") != "" {
			noColor = true
		}
	},
}

// Server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server",
	Long: `Start the srake API server for programmatic access to SRA metadata.

The server provides RESTful endpoints for searching and retrieving
SRA metadata from the local database.`,
	Example: `  srake server
  srake server --port 3000
  srake server --dev --log-level debug`,
	RunE: runServer,
}

var (
	serverPort     int
	serverHost     string
	serverDBPath   string
	serverLogLevel string
	serverDev      bool
)

func runServer(cmd *cobra.Command, args []string) error {
	// Initialize database
	db, err := database.Initialize(serverDBPath)
	if err != nil {
		printError("Failed to initialize database: %v", err)
		return err
	}
	defer db.Close()

	// Setup interrupt handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check database connection
		dbStatus := "healthy"
		if err := db.Ping(); err != nil {
			dbStatus = "unhealthy"
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status":   "healthy",
			"version":  version,
			"database": dbStatus,
		})
	})

	// Search endpoint
	router.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		query := r.URL.Query().Get("q")
		organism := r.URL.Query().Get("organism")
		strategy := r.URL.Query().Get("library_strategy")
		limitStr := r.URL.Query().Get("limit")

		limit := 100
		if limitStr != "" {
			fmt.Sscanf(limitStr, "%d", &limit)
		}

		var results interface{}
		var err error

		if query != "" {
			// Full-text search
			results, err = db.FullTextSearch(query)
		} else if organism != "" {
			// Search by organism
			results, err = db.SearchByOrganism(organism, limit)
		} else if strategy != "" {
			// Search by library strategy
			results, err = db.SearchByLibraryStrategy(strategy, limit)
		} else {
			http.Error(w, "No search parameters provided", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"query":   query + organism + strategy,
			"results": results,
			"limit":   limit,
		}
		json.NewEncoder(w).Encode(response)
	})

	// Get specific accession endpoints
	router.HandleFunc("/api/experiment/{accession}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		exp, err := db.GetExperiment(vars["accession"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(exp)
	})

	router.HandleFunc("/api/run/{accession}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		run, err := db.GetRun(vars["accession"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(run)
	})

	router.HandleFunc("/api/sample/{accession}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		sample, err := db.GetSample(vars["accession"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(sample)
	})

	router.HandleFunc("/api/study/{accession}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		study, err := db.GetStudy(vars["accession"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(study)
	})

	// Statistics endpoint
	router.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats, err := db.GetStats()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(stats)
	})

	// Setup server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", serverHost, serverPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start server in goroutine
	go func() {
		printInfo("Starting server on %s:%d", serverHost, serverPort)
		if serverDev {
			printInfo("Development mode enabled")
		}
		printSuccess("Server ready at http://%s:%d", serverHost, serverPort)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			printError("Server failed: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	printInfo("\nShutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %v", err)
	}

	printSuccess("Server stopped gracefully")
	return nil
}

// Search command
var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search SRA metadata",
	Long: `Search the SRA metadata database for experiments matching your query.

The search supports organism names, accession numbers, and keywords.
Results can be filtered by platform, strategy, and other criteria.`,
	Example: `  srake search "homo sapiens"
  srake search mouse --limit 10
  srake search human --platform ILLUMINA --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

var (
	searchOrganism string
	searchPlatform string
	searchStrategy string
	searchLimit    int
	searchFormat   string
	searchOutput   string
	searchNoHeader bool
)

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Show search progress
	if !quiet && isTerminal() {
		printInfo("Searching for \"%s\"...", query)
	}

	// Make request to server (or query database directly)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/search?q=%s&limit=%d",
		serverPort, query, searchLimit))
	if err != nil {
		printError("Search failed: Cannot connect to server")
		fmt.Fprintf(os.Stderr, "\nMake sure the server is running:\n")
		fmt.Fprintf(os.Stderr, "  srake server\n")
		return fmt.Errorf("connection failed")
	}
	defer resp.Body.Close()

	// Parse response
	var result struct {
		Query   string                   `json:"query"`
		Results []map[string]interface{} `json:"results"`
		Total   int                      `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	// Format output based on requested format
	switch searchFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)

	case "csv", "tsv":
		sep := ","
		if searchFormat == "tsv" {
			sep = "\t"
		}
		if !searchNoHeader {
			fmt.Println(strings.Join([]string{"accession", "title", "platform", "strategy"}, sep))
		}
		for _, r := range result.Results {
			fmt.Printf("%s%s%s%s%s%s%s\n",
				r["accession"], sep,
				r["title"], sep,
				r["platform"], sep,
				r["strategy"])
		}

	default: // table format
		if len(result.Results) == 0 {
			printInfo("No results found for \"%s\"", query)
			return nil
		}

		// Create table writer
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		// Header
		if !searchNoHeader {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				colorize(colorBold, "ACCESSION"),
				colorize(colorBold, "TITLE"),
				colorize(colorBold, "PLATFORM"),
				colorize(colorBold, "STRATEGY"))

			// Separator line
			if isTerminal() && !noColor {
				fmt.Fprintf(w, "%s\n", colorize(colorGray, strings.Repeat("─", 80)))
			}
		}

		// Results
		for _, r := range result.Results {
			title := fmt.Sprintf("%v", r["title"])
			if len(title) > 40 {
				title = title[:37] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				colorize(colorCyan, fmt.Sprintf("%v", r["accession"])),
				title,
				fmt.Sprintf("%v", r["platform"]),
				fmt.Sprintf("%v", r["strategy"]))
		}

		w.Flush()

		// Summary
		if !quiet {
			fmt.Printf("\n%s\n", colorize(colorGray,
				fmt.Sprintf("Found %d results (showing %d)", result.Total, len(result.Results))))
		}
	}

	return nil
}

// Database command
var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management",
	Long:  `Manage the local SRA metadata database.`,
	Example: `  srake db info
  srake db update
  srake db vacuum`,
}

// Database info subcommand
var dbInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show database statistics",
	Long:  `Display information about the local SRA metadata database.`,
	RunE:  runDBInfo,
}

func runDBInfo(cmd *cobra.Command, args []string) error {
	dbPath := serverDBPath
	if dbPath == "" {
		dbPath = "./data/SRAmetadb.sqlite"
	}

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		printError("Database not found at %s", dbPath)
		fmt.Fprintf(os.Stderr, "\nDownload the database first:\n")
		fmt.Fprintf(os.Stderr, "  srake download\n")
		return fmt.Errorf("database not found")
	}

	// Open database using our database package
	db, err := database.Initialize(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Get database statistics
	stats, err := db.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get statistics: %v", err)
	}

	printInfo("Database Information")
	fmt.Println(colorize(colorGray, strings.Repeat("─", 40)))

	// File info
	fileInfo, _ := os.Stat(dbPath)
	fmt.Printf("%s %s\n", colorize(colorBold, "Path:"), dbPath)
	fmt.Printf("%s %.2f MB\n", colorize(colorBold, "Size:"),
		float64(fileInfo.Size())/(1024*1024))
	fmt.Printf("%s %s\n", colorize(colorBold, "Modified:"),
		fileInfo.ModTime().Format("2006-01-02 15:04:05"))

	// Table statistics
	fmt.Println()
	fmt.Printf("%s\n", colorize(colorBold, "Tables:"))
	fmt.Printf("  studies:     %s\n", colorize(colorCyan, fmt.Sprintf("%d", stats.TotalStudies)))
	fmt.Printf("  experiments: %s\n", colorize(colorCyan, fmt.Sprintf("%d", stats.TotalExperiments)))
	fmt.Printf("  runs:        %s\n", colorize(colorCyan, fmt.Sprintf("%d", stats.TotalRuns)))
	fmt.Printf("  samples:     %s\n", colorize(colorCyan, fmt.Sprintf("%d", stats.TotalSamples)))

	if !stats.LastUpdate.IsZero() {
		fmt.Println()
		fmt.Printf("%s %s\n", colorize(colorBold, "Last Update:"),
			stats.LastUpdate.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// Import the CLI package for download command
// (download command is now in cmd/cli/download.go)

// Metadata command
var metadataCmd = &cobra.Command{
	Use:   "metadata <accession> [accessions...]",
	Short: "Get metadata for specific accessions",
	Long: `Retrieve detailed metadata for one or more SRA accessions.

Supports SRX (experiment), SRR (run), SRP (project), and SRS (sample) accessions.`,
	Example: `  srake metadata SRX123456
  srake metadata SRX123456 SRX123457 --format json
  srake metadata SRR999999 --fields title,platform,strategy`,
	Args: cobra.MinimumNArgs(1),
	RunE: runMetadata,
}

var (
	metadataFormat string
	metadataFields string
	metadataExpand bool
)

func runMetadata(cmd *cobra.Command, args []string) error {
	accessions := args

	// Mock response for now
	for _, acc := range accessions {
		if metadataFormat == "json" {
			data := map[string]interface{}{
				"accession": acc,
				"title":     "Sample experiment for " + acc,
				"platform":  "ILLUMINA",
				"strategy":  "RNA-Seq",
				"organism":  "Homo sapiens",
				"spots":     1000000,
				"bases":     150000000,
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			encoder.Encode(data)
		} else {
			printInfo("Metadata for %s:", colorize(colorCyan, acc))
			fmt.Printf("  Title:     Sample experiment\n")
			fmt.Printf("  Platform:  ILLUMINA\n")
			fmt.Printf("  Strategy:  RNA-Seq\n")
			fmt.Printf("  Organism:  Homo sapiens\n")
			fmt.Println()
		}
	}

	return nil
}

// Models command
var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Manage embedding models",
	Long:  `Download and manage ONNX models for generating embeddings.`,
	Example: `  srake models list
  srake models download Xenova/SapBERT-from-PubMedBERT-fulltext
  srake models test Xenova/SapBERT-from-PubMedBERT-fulltext "test text"`,
}

// Models list subcommand
var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed models",
	RunE:  runModelsList,
}

func runModelsList(cmd *cobra.Command, args []string) error {
	config := embeddings.DefaultEmbedderConfig()
	manager, err := embeddings.NewManager(config.ModelsDir)
	if err != nil {
		return fmt.Errorf("failed to create model manager: %v", err)
	}

	models, err := manager.ListModels()
	if err != nil {
		return fmt.Errorf("failed to list models: %v", err)
	}

	if len(models) == 0 {
		printInfo("No models installed")
		fmt.Println("\nAvailable models to download:")
		for _, modelID := range embeddings.ListAvailableModels() {
			fmt.Printf("  %s\n", modelID)
		}
		fmt.Printf("\nUse 'srake models download <model-id>' to download a model\n")
		return nil
	}

	printInfo("Installed Models")
	fmt.Println(colorize(colorGray, strings.Repeat("─", 80)))

	for _, model := range models {
		fmt.Printf("%s %s\n", colorize(colorBold, "Model:"), model.ID)
		fmt.Printf("  Path: %s\n", model.Path)
		fmt.Printf("  Active variant: %s\n", colorize(colorCyan, model.ActiveVariant))

		fmt.Printf("  Variants:\n")
		for _, variant := range model.Variants {
			status := "not downloaded"
			if variant.Downloaded {
				status = colorize(colorGreen, fmt.Sprintf("downloaded (%s)", embeddings.FormatSize(variant.Size)))
			}
			fmt.Printf("    - %s: %s\n", variant.Name, status)
		}
		fmt.Println()
	}

	return nil
}

// Models download subcommand
var modelsDownloadCmd = &cobra.Command{
	Use:   "download <model-id> [--variant <variant>]",
	Short: "Download a model",
	Args:  cobra.ExactArgs(1),
	RunE:  runModelsDownload,
}

var downloadVariant string

func runModelsDownload(cmd *cobra.Command, args []string) error {
	modelID := args[0]

	config := embeddings.DefaultEmbedderConfig()
	manager, err := embeddings.NewManager(config.ModelsDir)
	if err != nil {
		return fmt.Errorf("failed to create model manager: %v", err)
	}

	// Create progress channel
	progress := make(chan embeddings.DownloadProgress, 100)
	done := make(chan bool)

	// Display progress
	go func() {
		for p := range progress {
			fmt.Printf("\r%s: %.1f%% (%.1f MB/s, ETA: %s)",
				p.File,
				p.Percentage,
				p.Speed,
				p.ETA.Round(time.Second))
		}
		done <- true
	}()

	printInfo("Downloading model %s...", modelID)

	downloader := embeddings.NewDownloader(manager, progress)
	err = downloader.DownloadModel(modelID, downloadVariant)

	close(progress)
	<-done
	fmt.Println() // New line after progress

	if err != nil {
		return fmt.Errorf("failed to download model: %v", err)
	}

	printSuccess("Model %s downloaded successfully", modelID)
	return nil
}

// Models test subcommand
var modelsTestCmd = &cobra.Command{
	Use:   "test <model-id> <text>",
	Short: "Test a model by generating an embedding",
	Args:  cobra.ExactArgs(2),
	RunE:  runModelsTest,
}

func runModelsTest(cmd *cobra.Command, args []string) error {
	modelID := args[0]
	text := args[1]

	config := embeddings.DefaultEmbedderConfig()
	embedder, err := embeddings.NewEmbedder(config)
	if err != nil {
		return fmt.Errorf("failed to create embedder: %v", err)
	}
	defer embedder.Close()

	printInfo("Loading model %s...", modelID)
	if err := embedder.LoadModel(modelID); err != nil {
		return fmt.Errorf("failed to load model: %v", err)
	}

	printInfo("Generating embedding for: \"%s\"", text)
	embedding, err := embedder.EmbedText(text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %v", err)
	}

	printSuccess("Embedding generated successfully")
	fmt.Printf("Dimension: %d\n", len(embedding))
	fmt.Printf("First 10 values: [")
	for i := 0; i < 10 && i < len(embedding); i++ {
		fmt.Printf("%.4f", embedding[i])
		if i < 9 && i < len(embedding)-1 {
			fmt.Printf(", ")
		}
	}
	fmt.Printf("...]\n")

	return nil
}

// Embed command
var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Manage embeddings",
	Long:  `Generate and manage embeddings for SRA metadata.`,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")

	// Server command flags
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "Port to listen on")
	serverCmd.Flags().StringVar(&serverHost, "host", "localhost", "Host to bind to")
	serverCmd.Flags().StringVar(&serverDBPath, "db", "./data/SRAmetadb.sqlite", "Database path")
	serverCmd.Flags().StringVar(&serverLogLevel, "log-level", "info", "Log level (debug|info|warn|error)")
	serverCmd.Flags().BoolVar(&serverDev, "dev", false, "Enable development mode")

	// Search command flags
	searchCmd.Flags().StringVarP(&searchOrganism, "organism", "o", "", "Filter by organism")
	searchCmd.Flags().StringVar(&searchPlatform, "platform", "", "Filter by platform")
	searchCmd.Flags().StringVar(&searchStrategy, "strategy", "", "Filter by strategy")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 100, "Maximum results to return")
	searchCmd.Flags().StringVarP(&searchFormat, "format", "f", "table", "Output format (table|json|csv|tsv)")
	searchCmd.Flags().StringVar(&searchOutput, "output", "", "Save results to file")
	searchCmd.Flags().BoolVar(&searchNoHeader, "no-header", false, "Omit header in output")

	// The ingest command for data ingestion
	ingestCmd := cli.NewIngestCmd()

	// Metadata command flags
	metadataCmd.Flags().StringVarP(&metadataFormat, "format", "f", "table", "Output format (table|json|yaml)")
	metadataCmd.Flags().StringVar(&metadataFields, "fields", "", "Comma-separated list of fields")
	metadataCmd.Flags().BoolVar(&metadataExpand, "expand", false, "Expand nested structures")

	// Models download command flags
	modelsDownloadCmd.Flags().StringVar(&downloadVariant, "variant", "", "Model variant to download (quantized|fp16|full)")

	// Add commands to root
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(ingestCmd)
	rootCmd.AddCommand(metadataCmd)
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(embedCmd)

	// Add subcommands to db
	dbCmd.AddCommand(dbInfoCmd)

	// Add subcommands to models
	modelsCmd.AddCommand(modelsListCmd)
	modelsCmd.AddCommand(modelsDownloadCmd)
	modelsCmd.AddCommand(modelsTestCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
