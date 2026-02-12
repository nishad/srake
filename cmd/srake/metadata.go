package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/paths"
	"github.com/spf13/cobra"
)

var metadataCmd = &cobra.Command{
	Use:   "metadata <accession> [accessions...]",
	Short: "Get metadata for specific accessions",
	Long: `Retrieve detailed metadata for one or more SRA accessions.

Supports SRX (experiment), SRR (run), SRP/DRP/ERP (study), and SRS/DRS/ERS (sample) accessions.`,
	Example: `  srake metadata SRX123456
  srake metadata SRX123456 SRX123457 --format json
  srake metadata SRR999999 --fields title,platform,strategy
  srake metadata SRP123456 --format json --output metadata.json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runMetadata,
}

var (
	metadataFormat string
	metadataFields string
	metadataExpand bool
	metadataOutput string
)

func init() {
	// Metadata command flags
	metadataCmd.Flags().StringVarP(&metadataFormat, "format", "f", "table", "Output format (table|json|yaml)")
	metadataCmd.Flags().StringVar(&metadataFields, "fields", "", "Comma-separated list of fields")
	metadataCmd.Flags().BoolVar(&metadataExpand, "expand", false, "Expand nested structures")
}

func runMetadata(cmd *cobra.Command, args []string) error {
	accessions := args

	// Initialize database
	db, err := database.Initialize(paths.GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	for _, acc := range accessions {
		accType := detectAccessionType(acc)
		var data interface{}
		var fetchErr error

		switch accType {
		case "study":
			data, fetchErr = db.GetStudy(acc)
		case "experiment":
			data, fetchErr = db.GetExperiment(acc)
		case "sample":
			data, fetchErr = db.GetSample(acc)
		case "run":
			data, fetchErr = db.GetRun(acc)
		default:
			printWarning("Unknown accession type for: %s", acc)
			continue
		}

		if fetchErr != nil {
			printError("Failed to get metadata for %s: %v", acc, fetchErr)
			continue
		}

		if metadataFormat == "json" {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			encoder.Encode(data)
		} else {
			printMetadataTable(acc, accType, data)
		}
	}

	return nil
}

// detectAccessionType determines the type of accession based on prefix
func detectAccessionType(acc string) string {
	acc = strings.ToUpper(acc)
	if strings.HasPrefix(acc, "SRP") || strings.HasPrefix(acc, "DRP") || strings.HasPrefix(acc, "ERP") {
		return "study"
	}
	if strings.HasPrefix(acc, "SRX") || strings.HasPrefix(acc, "DRX") || strings.HasPrefix(acc, "ERX") {
		return "experiment"
	}
	if strings.HasPrefix(acc, "SRS") || strings.HasPrefix(acc, "DRS") || strings.HasPrefix(acc, "ERS") {
		return "sample"
	}
	if strings.HasPrefix(acc, "SRR") || strings.HasPrefix(acc, "DRR") || strings.HasPrefix(acc, "ERR") {
		return "run"
	}
	return "unknown"
}

// printMetadataTable prints metadata in table format
func printMetadataTable(acc, accType string, data interface{}) {
	printInfo("Metadata for %s (%s):", colorize(colorCyan, acc), accType)

	switch v := data.(type) {
	case *database.Study:
		fmt.Printf("  Accession:  %s\n", v.StudyAccession)
		fmt.Printf("  Title:      %s\n", truncateStr(v.StudyTitle, 70))
		fmt.Printf("  Type:       %s\n", v.StudyType)
		fmt.Printf("  Organism:   %s\n", v.Organism)
		if v.StudyAbstract != "" {
			fmt.Printf("  Abstract:   %s\n", truncateStr(v.StudyAbstract, 100))
		}
	case *database.Experiment:
		fmt.Printf("  Accession:  %s\n", v.ExperimentAccession)
		fmt.Printf("  Study:      %s\n", v.StudyAccession)
		fmt.Printf("  Title:      %s\n", truncateStr(v.Title, 70))
		fmt.Printf("  Platform:   %s\n", v.Platform)
		fmt.Printf("  Instrument: %s\n", v.InstrumentModel)
		fmt.Printf("  Strategy:   %s\n", v.LibraryStrategy)
		fmt.Printf("  Source:     %s\n", v.LibrarySource)
	case *database.Sample:
		fmt.Printf("  Accession:  %s\n", v.SampleAccession)
		fmt.Printf("  Organism:   %s\n", v.Organism)
		fmt.Printf("  Scientific: %s\n", v.ScientificName)
		if v.Tissue != "" {
			fmt.Printf("  Tissue:     %s\n", v.Tissue)
		}
		if v.Description != "" {
			fmt.Printf("  Description:%s\n", truncateStr(v.Description, 70))
		}
	case *database.Run:
		fmt.Printf("  Accession:  %s\n", v.RunAccession)
		fmt.Printf("  Experiment: %s\n", v.ExperimentAccession)
		fmt.Printf("  Total Spots:%d\n", v.TotalSpots)
		fmt.Printf("  Total Bases:%d\n", v.TotalBases)
		fmt.Printf("  Published:  %s\n", v.Published)
	}
	fmt.Println()
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
