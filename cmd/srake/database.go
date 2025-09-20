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
	Long:  `Manage the local SRAKE (SRA Knowledge Engine) metadata database.`,
	Example: `  srake db info
  srake db update
  srake db vacuum`,
}

// Database info subcommand
var dbInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show database statistics",
	Long:  `Display information about the local SRAKE metadata database.`,
	RunE:  runDBInfo,
}

// Database stats subcommand
var dbStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Manage database statistics",
	Long:  `Manage the pre-computed database statistics table.`,
	Example: `  srake db stats --rebuild  # Rebuild statistics from scratch
  srake db stats --show     # Show current statistics`,
}

var (
	statsRebuild bool
	statsShow    bool
)

func init() {
	// Add subcommands to db
	dbCmd.AddCommand(dbInfoCmd)
	dbCmd.AddCommand(dbStatsCmd)

	// Add flags to stats command
	dbStatsCmd.Flags().BoolVar(&statsRebuild, "rebuild", false, "Rebuild statistics table")
	dbStatsCmd.Flags().BoolVar(&statsShow, "show", false, "Show statistics table contents")
	dbStatsCmd.RunE = runDBStats
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

func runDBStats(cmd *cobra.Command, args []string) error {
	// Validate flags - must specify exactly one action
	if !statsRebuild && !statsShow {
		return fmt.Errorf("must specify either --rebuild or --show")
	}
	if statsRebuild && statsShow {
		return fmt.Errorf("cannot specify both --rebuild and --show")
	}

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

	// Open database
	db, err := database.Initialize(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	if statsRebuild {
		printInfo("Rebuilding database statistics...")

		// Count and update statistics
		if err := db.UpdateStatistics(); err != nil {
			return fmt.Errorf("failed to rebuild statistics: %v", err)
		}

		printSuccess("Statistics rebuilt successfully")

		// Show the new statistics
		stats, _ := db.GetStatistics()
		fmt.Println()
		fmt.Printf("%s\n", colorize(colorBold, "Updated Statistics:"))
		fmt.Printf("  studies:     %s\n", colorize(colorCyan, fmt.Sprintf("%d", stats["studies"])))
		fmt.Printf("  experiments: %s\n", colorize(colorCyan, fmt.Sprintf("%d", stats["experiments"])))
		fmt.Printf("  samples:     %s\n", colorize(colorCyan, fmt.Sprintf("%d", stats["samples"])))
		fmt.Printf("  runs:        %s\n", colorize(colorCyan, fmt.Sprintf("%d", stats["runs"])))
		fmt.Printf("  submissions: %s\n", colorize(colorCyan, fmt.Sprintf("%d", stats["submissions"])))
		fmt.Printf("  analyses:    %s\n", colorize(colorCyan, fmt.Sprintf("%d", stats["analyses"])))
	} else if statsShow {
		// Get cached statistics
		stats, err := db.GetStatistics()
		if err != nil || len(stats) == 0 {
			printWarning("No statistics found. Run 'srake db stats --rebuild' to create them.")
			return nil
		}

		printInfo("Cached Database Statistics")
		fmt.Println(colorize(colorGray, strings.Repeat("─", 40)))

		// Display statistics
		fmt.Printf("%s\n", colorize(colorBold, "Table Counts:"))
		for table, count := range stats {
			fmt.Printf("  %-12s %s\n", table+":", colorize(colorCyan, fmt.Sprintf("%d", count)))
		}

		// Check when statistics were last updated
		var lastUpdate string
		row := db.QueryRow("SELECT last_updated FROM statistics LIMIT 1")
		row.Scan(&lastUpdate)
		if lastUpdate != "" {
			fmt.Printf("\n%s %s\n", colorize(colorBold, "Last Updated:"), lastUpdate)
		}
	}

	return nil
}
