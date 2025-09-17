package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

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