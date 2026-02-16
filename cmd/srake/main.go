package main

import (
	"fmt"
	"os"

	"github.com/nishad/srake/internal/cli"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	noColor bool
	noInput bool
	verbose bool
	quiet   bool
	debug   bool // Debug flag

	// Version information (injected via ldflags)
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Helper functions are defined in helpers.go

// Root command
var rootCmd = &cobra.Command{
	Use:   "srake",
	Short: "SRAKE - SRA Knowledge Engine",
	Long: colorize(colorBold, "SRAKE - SRA Knowledge Engine") + `

Pronounced like Japanese sake (酒) — "srah-keh"

SRAKE provides a unified interface for searching, downloading, and serving
SRA (Sequence Read Archive) metadata from NCBI.

ENVIRONMENT VARIABLES:
  SRAKE_DB_PATH          Path to the SRAKE metadata database
  SRAKE_INDEX_PATH       Path to the search index directory
  SRAKE_CONFIG_DIR       Configuration directory (default: ~/.config/srake)
  SRAKE_DATA_DIR         Data directory (default: ~/.local/share/srake)
  SRAKE_CACHE_DIR        Cache directory (default: ~/.cache/srake)
  SRAKE_MODEL_VARIANT    Model variant for embeddings (full|quantized)
  NO_COLOR               Disable colored output
  FORCE_COLOR            Force colored output even when not a TTY

The tool follows XDG Base Directory Specification and respects standard
environment variables like XDG_CONFIG_HOME, XDG_DATA_HOME, and XDG_CACHE_HOME.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildDate),
	Example: `  srake search "homo sapiens"
  srake server --port 8080
  srake db info
  srake ingest --file metadata.tar.gz`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// FORCE_COLOR takes priority — skip disabling color
		if os.Getenv("FORCE_COLOR") == "" {
			// Check NO_COLOR environment variable
			if os.Getenv("NO_COLOR") != "" {
				noColor = true
			}
			// Check TERM=dumb (no color support)
			if os.Getenv("TERM") == "dumb" {
				noColor = true
			}
		}
	},
}

// Server command is defined in server.go

// Search command
// Search command is defined in search.go

// runSearch is defined in search.go

// Database commands are defined in database.go

// Import the CLI package for download command
// (download command is now in cmd/cli/download.go)

// Metadata command is defined in metadata.go

// runMetadata is defined in metadata.go

// Models commands are defined in models.go

// runModelsList is defined in models.go

// Model commands are defined in models.go

// runModelsTest is defined in models.go

// Embed command is defined in embed.go

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVar(&noInput, "no-input", false, "Disable interactive prompts (fail instead of prompt)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")

	// Enable shell completion command (bash, zsh, fish, powershell)
	rootCmd.CompletionOptions.DisableDefaultCmd = false

	// The ingest command for data ingestion
	ingestCmd := cli.NewIngestCmd()

	// Add commands to root
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(ingestCmd)
	rootCmd.AddCommand(metadataCmd)
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(embedCmd)
	rootCmd.AddCommand(mcpCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
