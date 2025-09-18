package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/nishad/srake/internal/converter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var convertCmd = &cobra.Command{
	Use:   "convert [<accession> ...]",
	Short: "Convert between different accession types",
	Long: `Convert between different biological research identifiers.

Supports conversion between:
  - SRP (Study) ↔ GSE (GEO Series)
  - SRX (Experiment) ↔ GSM (GEO Sample)
  - SRR (Run) ↔ GSM
  - SRS (Sample) ↔ GSM
  - And more cross-database mappings`,
	Example: `  # Convert SRA Project to GEO Series
  srake convert SRP123456 --to GSE

  # Convert GEO Sample to SRA accessions
  srake convert GSM123456 --to SRX

  # Convert multiple accessions
  srake convert SRP123 SRP124 SRP125 --to GSE

  # Output as JSON
  srake convert GSE123456 --to SRP --format json

  # Convert from stdin
  echo "SRP123456" | srake convert --to GSE

  # Convert from file
  srake convert --batch accessions.txt --to GSE`,
	Args: cobra.MinimumNArgs(0),
	RunE: runConvert,
}

var (
	convertTo     string
	convertFormat string
	convertOutput string
	convertBatch  string
	convertDryRun bool
)

func init() {
	convertCmd.Flags().StringVar(&convertTo, "to", "", "Target accession type (GSE, SRP, SRX, GSM, SRR, SRS)")
	convertCmd.Flags().StringVarP(&convertFormat, "format", "f", "table", "Output format (table|json|yaml|csv|tsv)")
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "Save results to file")
	convertCmd.Flags().StringVar(&convertBatch, "batch", "", "Read accessions from file (one per line)")
	convertCmd.Flags().BoolVar(&convertDryRun, "dry-run", false, "Preview conversions without executing")
	convertCmd.MarkFlagRequired("to")
}

func runConvert(cmd *cobra.Command, args []string) error {
	// Collect all accessions
	accessions := args
	printDebug("Starting convert command with %d args", len(args))

	// Read from stdin if no arguments provided and stdin is available
	if len(args) == 0 && convertBatch == "" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Data is being piped to stdin
			printDebug("Reading accessions from stdin")
			stdinAccessions, err := readAccessionsFromReader(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			accessions = stdinAccessions
			printDebug("Read %d accessions from stdin", len(stdinAccessions))
		}
	}

	// Read from batch file if specified
	if convertBatch != "" {
		batchAccessions, err := readAccessionFile(convertBatch)
		if err != nil {
			return fmt.Errorf("failed to read batch file: %w", err)
		}
		accessions = append(accessions, batchAccessions...)
	}

	if len(accessions) == 0 {
		return fmt.Errorf("no accessions provided")
	}

	// Handle dry-run mode
	if convertDryRun {
		printInfo("DRY RUN: Showing what would be converted")
		for _, acc := range accessions {
			fmt.Printf("Would convert: %s → %s\n", acc, convertTo)
		}
		return nil
	}

	// Initialize converter
	conv := converter.NewConverter(serverDBPath)
	defer conv.Close()

	// Convert each accession
	results := make([]converter.ConversionResult, 0)

	for _, acc := range accessions {
		if verbose {
			printInfo("Converting %s to %s...", acc, convertTo)
		}
		printDebug("Processing accession: %s -> %s", acc, convertTo)

		result, err := conv.Convert(acc, convertTo)
		if err != nil {
			if !quiet {
				printError("Failed to convert %s: %v", acc, err)
			}
			// Add failed result
			results = append(results, converter.ConversionResult{
				Source: acc,
				Error:  err.Error(),
			})
			continue
		}

		results = append(results, *result)
	}

	// Output results
	return outputConversionResults(results)
}

func outputConversionResults(results []converter.ConversionResult) error {
	var output string

	switch convertFormat {
	case "json":
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		output = string(data)

	case "yaml":
		data, err := yaml.Marshal(results)
		if err != nil {
			return err
		}
		output = string(data)

	case "xml":
		type Results struct {
			XMLName xml.Name                     `xml:"results"`
			Results []converter.ConversionResult `xml:"result"`
		}
		data, err := xml.MarshalIndent(Results{Results: results}, "", "  ")
		if err != nil {
			return err
		}
		output = string(data)

	case "csv", "tsv":
		sep := ","
		if convertFormat == "tsv" {
			sep = "\t"
		}

		// Header
		output = fmt.Sprintf("source%starget_type%starget_accessions%sstatus\n", sep, sep, sep)

		// Data rows
		for _, r := range results {
			status := "success"
			targets := strings.Join(r.Targets, ";")
			if r.Error != "" {
				status = "failed"
				targets = r.Error
			}
			output += fmt.Sprintf("%s%s%s%s%s%s%s\n",
				r.Source, sep,
				r.TargetType, sep,
				targets, sep,
				status)
		}

	default: // table format
		if len(results) == 0 {
			printInfo("No results found")
			return nil
		}

		// Print header
		fmt.Printf("%-15s %-12s %-50s %s\n",
			colorize(colorBold, "SOURCE"),
			colorize(colorBold, "TARGET TYPE"),
			colorize(colorBold, "CONVERTED ACCESSIONS"),
			colorize(colorBold, "STATUS"))

		// Print separator
		if isTerminal() && !noColor {
			fmt.Println(colorize(colorGray, strings.Repeat("─", 90)))
		}

		// Print results
		for _, r := range results {
			if r.Error != "" {
				fmt.Printf("%-15s %-12s %-50s %s\n",
					r.Source,
					r.TargetType,
					"-",
					colorize(colorRed, "Error: "+r.Error))
			} else {
				targetsStr := strings.Join(r.Targets, ", ")
				if len(targetsStr) > 47 {
					targetsStr = targetsStr[:44] + "..."
				}

				fmt.Printf("%-15s %-12s %-50s %s\n",
					colorize(colorCyan, r.Source),
					r.TargetType,
					targetsStr,
					colorize(colorGreen, "✓"))
			}
		}

		// Summary
		successful := 0
		for _, r := range results {
			if r.Error == "" {
				successful++
			}
		}

		fmt.Printf("\n%s\n", colorize(colorGray,
			fmt.Sprintf("Successfully converted %d/%d accessions", successful, len(results))))

		return nil
	}

	// Write to file or stdout
	if convertOutput != "" {
		return os.WriteFile(convertOutput, []byte(output), 0644)
	}

	fmt.Print(output)
	return nil
}

