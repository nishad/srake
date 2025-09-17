package processor

import (
	"fmt"
	"time"

	"github.com/nishad/srake/internal/validator"
)

// NewComprehensiveExtractor creates a new comprehensive extractor
func NewComprehensiveExtractor(db Database, options ExtractionOptions) *ComprehensiveExtractor {
	if options.BatchSize == 0 {
		options.BatchSize = 1000
	}

	var v *validator.Validator
	if options.ValidateXML {
		validatorConfig := validator.ValidationConfig{
			ValidateEnumerations: true,
			ValidateReferences:   true,
			ValidateRequired:     true,
			StrictMode:           options.StrictValidation,
		}
		v = validator.NewValidator(validatorConfig)
	}

	return &ComprehensiveExtractor{
		db:                db,
		options:           options,
		validator:         v,
		platformHandler:   NewPlatformHandler(),
		poolHandler:       NewPoolHandler(db),
		identifierHandler: NewIdentifierHandler(db),
		stats: ExtractionStats{
			StartTime: time.Now(),
		},
	}
}

// validateXMLData validates XML data if validation is enabled
func (ce *ComprehensiveExtractor) validateXMLData(xmlData []byte) (bool, *validator.ValidationResult) {
	if ce.validator == nil || !ce.options.ValidateXML {
		return true, nil
	}

	result, err := ce.validator.ValidateXML(xmlData)
	if err != nil {
		ce.stats.Errors = append(ce.stats.Errors, fmt.Sprintf("Validation error: %v", err))
		return false, nil
	}

	// Update statistics
	ce.stats.ValidationErrors += len(result.Errors)
	ce.stats.ValidationWarnings += len(result.Warnings)

	// Log validation issues if enabled
	if ce.options.LogValidationIssues {
		for _, e := range result.Errors {
			ce.stats.Errors = append(ce.stats.Errors,
				fmt.Sprintf("Validation Error [%s] %s: %s", result.DocType, e.Field, e.Message))
		}
		for _, w := range result.Warnings {
			ce.stats.Errors = append(ce.stats.Errors,
				fmt.Sprintf("Validation Warning [%s] %s: %s", result.DocType, w.Field, w.Message))
		}
	}

	// Determine if we should proceed
	shouldProceed := true
	if ce.options.StrictValidation && len(result.Errors) > 0 {
		shouldProceed = false
	}

	return shouldProceed, result
}

// PrintStats prints extraction statistics
func (ce *ComprehensiveExtractor) PrintStats() {
	elapsed := time.Since(ce.stats.StartTime)
	fmt.Printf("\n=== Extraction Statistics ===\n")
	fmt.Printf("Time elapsed: %v\n", elapsed)
	fmt.Printf("\nProcessed:\n")
	fmt.Printf("  Studies:     %d\n", ce.stats.StudiesProcessed)
	fmt.Printf("  Experiments: %d\n", ce.stats.ExperimentsProcessed)
	fmt.Printf("  Samples:     %d\n", ce.stats.SamplesProcessed)
	fmt.Printf("  Runs:        %d\n", ce.stats.RunsProcessed)
	fmt.Printf("  Analyses:    %d\n", ce.stats.AnalysesProcessed)
	fmt.Printf("  Submissions: %d\n", ce.stats.SubmissionsProcessed)
	fmt.Printf("\nExtracted:\n")
	fmt.Printf("  Studies:     %d\n", ce.stats.StudiesExtracted)
	fmt.Printf("  Experiments: %d\n", ce.stats.ExperimentsExtracted)
	fmt.Printf("  Samples:     %d\n", ce.stats.SamplesExtracted)
	fmt.Printf("  Runs:        %d\n", ce.stats.RunsExtracted)
	fmt.Printf("  Analyses:    %d\n", ce.stats.AnalysesExtracted)
	fmt.Printf("  Submissions: %d\n", ce.stats.SubmissionsExtracted)

	if ce.options.ValidateXML {
		fmt.Printf("\nValidation:\n")
		fmt.Printf("  Errors:      %d\n", ce.stats.ValidationErrors)
		fmt.Printf("  Warnings:    %d\n", ce.stats.ValidationWarnings)
	}

	if len(ce.stats.Errors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(ce.stats.Errors))
		for i, err := range ce.stats.Errors {
			if i >= 10 {
				fmt.Printf("  ... and %d more\n", len(ce.stats.Errors)-10)
				break
			}
			fmt.Printf("  - %s\n", err)
		}
	}
}

// GetStats returns the extraction statistics
func (ce *ComprehensiveExtractor) GetStats() ExtractionStats {
	return ce.stats
}
