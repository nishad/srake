package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var metadataCmd = &cobra.Command{
	Use:   "metadata <accession> [accessions...]",
	Short: "Get metadata for specific accessions",
	Long: `Retrieve detailed metadata for one or more SRA accessions.

Supports SRX (experiment), SRR (run), SRP (project), and SRS (sample) accessions.`,
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
