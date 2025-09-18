package main

import (
	"fmt"
	"os"

	"github.com/nishad/srake/internal/cli"
	"github.com/spf13/cobra"
)

// Version info
var (
	version = "0.0.1-alpha"
	commit  = "dev"
	date    = "unknown"
)

// Global flags
var (
	noColor bool
	quiet   bool
	verbose bool
	yes     bool
	debug   bool
)

// Root command
var rootCmd = &cobra.Command{
	Use:   "srake",
	Short: "SRA metadata processor",
	Long: `srake is a high-performance tool for processing and querying NCBI SRA metadata.

It provides a zero-copy streaming architecture to handle large compressed archives
efficiently, making it ideal for bioinformatics workflows and large-scale genomic
data analysis.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	Example: `  # Ingest SRA metadata
  srake ingest --auto

  # Search for experiments
  srake search "homo sapiens" --limit 10

  # Start API server
  srake server --port 8080

  # Get database statistics
  srake db info`,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
	rootCmd.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "Assume yes to all prompts (non-interactive mode)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")

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
	metadataCmd.Flags().StringVarP(&metadataOutput, "output", "o", "", "Save results to file")
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
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(runsCmd)
	rootCmd.AddCommand(samplesCmd)
	rootCmd.AddCommand(experimentsCmd)
	rootCmd.AddCommand(studiesCmd)
	rootCmd.AddCommand(downloadCmd)

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