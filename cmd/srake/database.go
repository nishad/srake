package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/paths"
	"github.com/spf13/cobra"
)

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

func init() {
	// Add subcommands to db
	dbCmd.AddCommand(dbInfoCmd)
}

func runDBInfo(cmd *cobra.Command, args []string) error {
	dbPath := serverDBPath
	if dbPath == "" {
		dbPath = paths.GetDatabasePath()
	}

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		printError("Database not found at %s", dbPath)
		fmt.Fprintf(os.Stderr, "\nIngest the database first:\n")
		fmt.Fprintf(os.Stderr, "  srake ingest --auto\n")
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