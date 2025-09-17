package validator

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Validator validates SRA XML documents
type Validator struct {
	config ValidationConfig
}

// ValidationConfig holds validation configuration
type ValidationConfig struct {
	ValidateEnumerations bool
	ValidateReferences   bool
	ValidateRequired     bool
	StrictMode           bool
}

// NewValidator creates a new validator
func NewValidator(config ValidationConfig) *Validator {
	return &Validator{
		config: config,
	}
}

// DefaultValidator creates a validator with default settings
func DefaultValidator() *Validator {
	return &Validator{
		config: ValidationConfig{
			ValidateEnumerations: true,
			ValidateReferences:   true,
			ValidateRequired:     true,
			StrictMode:           false,
		},
	}
}

// ValidationResult contains validation results
type ValidationResult struct {
	IsValid  bool                `json:"is_valid"`
	DocType  string              `json:"doc_type"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
	Stats    ValidationStats     `json:"stats"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Type    string `json:"type"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Type    string `json:"type"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ValidationStats contains validation statistics
type ValidationStats struct {
	ElementsValidated int `json:"elements_validated"`
	AttributesChecked int `json:"attributes_checked"`
	ReferencesChecked int `json:"references_checked"`
}

// ValidateXML validates an XML document
func (v *Validator) ValidateXML(xmlData []byte) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		Stats:    ValidationStats{},
	}

	// Determine document type
	docType := v.detectDocumentType(xmlData)
	result.DocType = docType

	// Basic XML validation
	if err := v.validateXMLStructure(xmlData, result); err != nil {
		return result, err
	}

	// Type-specific validation
	switch docType {
	case "study":
		v.validateStudy(xmlData, result)
	case "sample":
		v.validateSample(xmlData, result)
	case "experiment":
		v.validateExperiment(xmlData, result)
	case "run":
		v.validateRun(xmlData, result)
	case "analysis":
		v.validateAnalysis(xmlData, result)
	case "submission":
		v.validateSubmission(xmlData, result)
	default:
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:    "UNKNOWN_TYPE",
			Message: fmt.Sprintf("Unknown document type: %s", docType),
		})
	}

	// Set overall validity
	result.IsValid = len(result.Errors) == 0

	return result, nil
}

// detectDocumentType determines the type of SRA document
func (v *Validator) detectDocumentType(xmlData []byte) string {
	if bytes.Contains(xmlData, []byte("<STUDY")) || bytes.Contains(xmlData, []byte("<STUDY_SET")) {
		return "study"
	} else if bytes.Contains(xmlData, []byte("<SAMPLE")) || bytes.Contains(xmlData, []byte("<SAMPLE_SET")) {
		return "sample"
	} else if bytes.Contains(xmlData, []byte("<EXPERIMENT")) || bytes.Contains(xmlData, []byte("<EXPERIMENT_SET")) {
		return "experiment"
	} else if bytes.Contains(xmlData, []byte("<RUN")) || bytes.Contains(xmlData, []byte("<RUN_SET")) {
		return "run"
	} else if bytes.Contains(xmlData, []byte("<ANALYSIS")) || bytes.Contains(xmlData, []byte("<ANALYSIS_SET")) {
		return "analysis"
	} else if bytes.Contains(xmlData, []byte("<SUBMISSION")) || bytes.Contains(xmlData, []byte("<SUBMISSION_SET")) {
		return "submission"
	}
	return "unknown"
}

// validateXMLStructure validates basic XML structure
func (v *Validator) validateXMLStructure(xmlData []byte, result *ValidationResult) error {
	decoder := xml.NewDecoder(bytes.NewReader(xmlData))

	elementCount := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    "XML_PARSE_ERROR",
				Message: fmt.Sprintf("XML parsing error: %v", err),
			})
			return nil
		}

		switch token.(type) {
		case xml.StartElement:
			elementCount++
		}
	}

	result.Stats.ElementsValidated = elementCount
	return nil
}

// validateStudy validates study-specific requirements
func (v *Validator) validateStudy(xmlData []byte, result *ValidationResult) {
	// Required fields
	if v.config.ValidateRequired {
		requiredFields := []string{"STUDY_TITLE", "STUDY_TYPE"}
		for _, field := range requiredFields {
			if !bytes.Contains(xmlData, []byte("<"+field)) {
				result.Errors = append(result.Errors, ValidationError{
					Type:    "MISSING_REQUIRED_FIELD",
					Field:   field,
					Message: fmt.Sprintf("%s is required for study records", field),
				})
			}
		}
	}

	// Study type validation
	if v.config.ValidateEnumerations {
		studyTypes := []string{
			"Whole Genome Sequencing", "Metagenomics", "Transcriptome Analysis",
			"Epigenetics", "Synthetic Genomics", "Forensic or Paleo-genomics",
			"Gene Regulation Study", "Cancer Genomics", "Population Genomics",
			"RNASeq", "Exome Sequencing", "Pooled Clone Sequencing",
			"Other",
		}

		studyType := v.extractFieldValue(xmlData, "existing_study_type")
		if studyType != "" && !v.contains(studyTypes, studyType) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "INVALID_STUDY_TYPE",
				Field:   "study_type",
				Message: fmt.Sprintf("Study type '%s' may not be standard", studyType),
			})
		}
	}
}

// validateSample validates sample-specific requirements
func (v *Validator) validateSample(xmlData []byte, result *ValidationResult) {
	// Required fields
	if v.config.ValidateRequired {
		if !bytes.Contains(xmlData, []byte("<TAXON_ID")) {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "MISSING_REQUIRED_FIELD",
				Field:   "TAXON_ID",
				Message: "TAXON_ID is required for sample records",
			})
		}

		if !bytes.Contains(xmlData, []byte("<SCIENTIFIC_NAME")) {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "MISSING_REQUIRED_FIELD",
				Field:   "SCIENTIFIC_NAME",
				Message: "SCIENTIFIC_NAME is required for sample records",
			})
		}
	}

	// Validate TAXON_ID format (should be numeric)
	taxonID := v.extractFieldValue(xmlData, "TAXON_ID")
	if taxonID != "" {
		if !regexp.MustCompile(`^\d+$`).MatchString(taxonID) {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "INVALID_FORMAT",
				Field:   "TAXON_ID",
				Message: fmt.Sprintf("TAXON_ID must be numeric, got: %s", taxonID),
			})
		}
	}
}

// validateExperiment validates experiment-specific requirements
func (v *Validator) validateExperiment(xmlData []byte, result *ValidationResult) {
	// Required fields
	if v.config.ValidateRequired {
		requiredFields := []string{"TITLE", "STUDY_REF", "DESIGN", "PLATFORM"}
		for _, field := range requiredFields {
			if !bytes.Contains(xmlData, []byte("<"+field)) {
				result.Errors = append(result.Errors, ValidationError{
					Type:    "MISSING_REQUIRED_FIELD",
					Field:   field,
					Message: fmt.Sprintf("%s is required for experiment records", field),
				})
			}
		}

		// Library descriptor requirements
		if !bytes.Contains(xmlData, []byte("<LIBRARY_STRATEGY")) {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "MISSING_REQUIRED_FIELD",
				Field:   "LIBRARY_STRATEGY",
				Message: "LIBRARY_STRATEGY is required for experiment records",
			})
		}
	}

	// Validate enumerations
	if v.config.ValidateEnumerations {
		// Platform validation
		v.validatePlatform(xmlData, result)

		// Library strategy validation
		v.validateLibraryStrategy(xmlData, result)

		// Library source validation
		v.validateLibrarySource(xmlData, result)

		// Library selection validation
		v.validateLibrarySelection(xmlData, result)
	}

	// Validate references
	if v.config.ValidateReferences {
		v.validateStudyReference(xmlData, result)
		v.validateSampleReference(xmlData, result)
	}
}

// validateRun validates run-specific requirements
func (v *Validator) validateRun(xmlData []byte, result *ValidationResult) {
	// Required fields
	if v.config.ValidateRequired {
		requiredFields := []string{"EXPERIMENT_REF"}
		for _, field := range requiredFields {
			if !bytes.Contains(xmlData, []byte("<"+field)) {
				result.Errors = append(result.Errors, ValidationError{
					Type:    "MISSING_REQUIRED_FIELD",
					Field:   field,
					Message: fmt.Sprintf("%s is required for run records", field),
				})
			}
		}
	}

	// Validate file information
	if bytes.Contains(xmlData, []byte("<FILES")) {
		v.validateRunFiles(xmlData, result)
	}

	// Validate references
	if v.config.ValidateReferences {
		v.validateExperimentReference(xmlData, result)
	}
}

// validateAnalysis validates analysis-specific requirements
func (v *Validator) validateAnalysis(xmlData []byte, result *ValidationResult) {
	// Required fields
	if v.config.ValidateRequired {
		requiredFields := []string{"TITLE", "STUDY_REF", "DESCRIPTION", "ANALYSIS_TYPE"}
		for _, field := range requiredFields {
			if !bytes.Contains(xmlData, []byte("<"+field)) {
				result.Errors = append(result.Errors, ValidationError{
					Type:    "MISSING_REQUIRED_FIELD",
					Field:   field,
					Message: fmt.Sprintf("%s is required for analysis records", field),
				})
			}
		}
	}

	// Validate analysis type
	if v.config.ValidateEnumerations {
		validTypes := []string{
			"DE_NOVO_ASSEMBLY", "REFERENCE_ALIGNMENT",
			"SEQUENCE_ANNOTATION", "ABUNDANCE_MEASUREMENT",
		}

		hasValidType := false
		for _, aType := range validTypes {
			if bytes.Contains(xmlData, []byte("<"+aType)) {
				hasValidType = true
				break
			}
		}

		if !hasValidType && bytes.Contains(xmlData, []byte("<ANALYSIS_TYPE")) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "UNKNOWN_ANALYSIS_TYPE",
				Field:   "ANALYSIS_TYPE",
				Message: "Analysis type not recognized",
			})
		}
	}
}

// validateSubmission validates submission-specific requirements
func (v *Validator) validateSubmission(xmlData []byte, result *ValidationResult) {
	// Submission-specific validation
	if bytes.Contains(xmlData, []byte("<ACTIONS")) {
		v.validateSubmissionActions(xmlData, result)
	}
}

// Platform validation helpers
func (v *Validator) validatePlatform(xmlData []byte, result *ValidationResult) {
	platforms := []string{
		"ILLUMINA", "ION_TORRENT", "PACBIO_SMRT", "OXFORD_NANOPORE",
		"LS454", "ABI_SOLID", "BGISEQ", "DNBSEQ", "ELEMENT", "ULTIMA",
		"COMPLETE_GENOMICS", "HELICOS", "CAPILLARY", "GENAPSYS",
		"GENEMIND", "TAPESTRI", "VELA_DIAGNOSTICS", "SALUS",
		"GENEUS_TECH", "SINGULAR_GENOMICS", "GENEXUS", "REVOLOCITY",
	}

	foundPlatform := false
	for _, platform := range platforms {
		if bytes.Contains(xmlData, []byte("<"+platform)) {
			foundPlatform = true
			result.Stats.AttributesChecked++
			break
		}
	}

	if !foundPlatform && bytes.Contains(xmlData, []byte("<PLATFORM")) {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:    "UNKNOWN_PLATFORM",
			Field:   "PLATFORM",
			Message: "Platform type not recognized",
		})
	}
}

// Library strategy validation
func (v *Validator) validateLibraryStrategy(xmlData []byte, result *ValidationResult) {
	strategies := []string{
		"WGS", "WGA", "WXS", "RNA-Seq", "ssRNA-seq", "miRNA-Seq",
		"ncRNA-Seq", "FL-cDNA", "EST", "Hi-C", "ATAC-seq", "WCS",
		"RAD-Seq", "CLONE", "POOLCLONE", "AMPLICON", "CLONEEND",
		"FINISHING", "ChIP-Seq", "MNase-Seq", "DNase-Hypersensitivity",
		"Bisulfite-Seq", "CTS", "MRE-Seq", "MeDIP-Seq", "MBD-Seq",
		"Tn-Seq", "VALIDATION", "FAIRE-seq", "SELEX", "RIP-Seq",
		"ChIA-PET", "Synthetic-Long-Read", "Targeted-Capture",
		"Tethered Chromatin Conformation Capture", "OTHER",
	}

	strategy := v.extractFieldValue(xmlData, "LIBRARY_STRATEGY")
	if strategy != "" {
		result.Stats.AttributesChecked++
		if !v.contains(strategies, strategy) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "INVALID_LIBRARY_STRATEGY",
				Field:   "LIBRARY_STRATEGY",
				Message: fmt.Sprintf("Library strategy '%s' not in standard list", strategy),
			})
		}
	}
}

// Library source validation
func (v *Validator) validateLibrarySource(xmlData []byte, result *ValidationResult) {
	sources := []string{
		"GENOMIC", "GENOMIC SINGLE CELL", "TRANSCRIPTOMIC",
		"TRANSCRIPTOMIC SINGLE CELL", "METAGENOMIC",
		"METATRANSCRIPTOMIC", "SYNTHETIC", "VIRAL RNA", "OTHER",
	}

	source := v.extractFieldValue(xmlData, "LIBRARY_SOURCE")
	if source != "" {
		result.Stats.AttributesChecked++
		if !v.contains(sources, source) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "INVALID_LIBRARY_SOURCE",
				Field:   "LIBRARY_SOURCE",
				Message: fmt.Sprintf("Library source '%s' not in standard list", source),
			})
		}
	}
}

// Library selection validation
func (v *Validator) validateLibrarySelection(xmlData []byte, result *ValidationResult) {
	selections := []string{
		"RANDOM", "PCR", "RANDOM PCR", "RT-PCR", "HMPR", "MF",
		"CF-S", "CF-M", "CF-H", "CF-T", "MDA", "MSLL", "cDNA",
		"cDNA_randomPriming", "cDNA_oligo_dT", "PolyA", "Oligo-dT",
		"Inverse rRNA", "Inverse rRNA selection", "ChIP", "ChIP-Seq",
		"MNase", "DNase", "Hybrid Selection", "Reduced Representation",
		"Restriction Digest", "5-methylcytidine antibody",
		"MBD2 protein methyl-CpG binding domain", "CAGE", "RACE",
		"size fractionation", "Padlock probes capture method",
		"other", "unspecified",
	}

	selection := v.extractFieldValue(xmlData, "LIBRARY_SELECTION")
	if selection != "" {
		result.Stats.AttributesChecked++
		if !v.contains(selections, selection) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "INVALID_LIBRARY_SELECTION",
				Field:   "LIBRARY_SELECTION",
				Message: fmt.Sprintf("Library selection '%s' not in standard list", selection),
			})
		}
	}
}

// Reference validation helpers
func (v *Validator) validateStudyReference(xmlData []byte, result *ValidationResult) {
	ref := v.extractAccessionFromElement(xmlData, "STUDY_REF")
	if ref != "" {
		result.Stats.ReferencesChecked++
		if !v.isValidAccession(ref, []string{"SRP", "ERP", "DRP"}) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "INVALID_REFERENCE",
				Field:   "STUDY_REF",
				Message: fmt.Sprintf("Study reference '%s' has invalid format", ref),
			})
		}
	}
}

func (v *Validator) validateSampleReference(xmlData []byte, result *ValidationResult) {
	ref := v.extractAccessionFromElement(xmlData, "SAMPLE_DESCRIPTOR")
	if ref != "" {
		result.Stats.ReferencesChecked++
		if !v.isValidAccession(ref, []string{"SRS", "ERS", "DRS"}) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "INVALID_REFERENCE",
				Field:   "SAMPLE_REF",
				Message: fmt.Sprintf("Sample reference '%s' has invalid format", ref),
			})
		}
	}
}

func (v *Validator) validateExperimentReference(xmlData []byte, result *ValidationResult) {
	ref := v.extractAccessionFromElement(xmlData, "EXPERIMENT_REF")
	if ref != "" {
		result.Stats.ReferencesChecked++
		if !v.isValidAccession(ref, []string{"SRX", "ERX", "DRX"}) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "INVALID_REFERENCE",
				Field:   "EXPERIMENT_REF",
				Message: fmt.Sprintf("Experiment reference '%s' has invalid format", ref),
			})
		}
	}
}

// File validation
func (v *Validator) validateRunFiles(xmlData []byte, result *ValidationResult) {
	// Check for required file attributes
	if !bytes.Contains(xmlData, []byte(`filename="`)) {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "MISSING_FILE_INFO",
			Field:   "FILES",
			Message: "Run files must have filename attribute",
		})
	}

	// Validate file types
	fileTypes := []string{
		"sra", "srf", "sff", "fastq", "fasta", "tab",
		"bam", "bai", "cram", "crai", "vcf", "bcf",
	}

	fileType := v.extractAttribute(xmlData, "filetype")
	if fileType != "" && v.config.ValidateEnumerations {
		if !v.contains(fileTypes, fileType) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "UNKNOWN_FILE_TYPE",
				Field:   "filetype",
				Message: fmt.Sprintf("File type '%s' may not be supported", fileType),
			})
		}
	}
}

// Submission action validation
func (v *Validator) validateSubmissionActions(xmlData []byte, result *ValidationResult) {
	validActions := []string{
		"ADD", "MODIFY", "SUPPRESS", "HOLD", "RELEASE", "PROTECT", "VALIDATE",
	}

	for _, action := range validActions {
		if bytes.Contains(xmlData, []byte("<"+action)) {
			result.Stats.AttributesChecked++
		}
	}
}

// Helper functions
func (v *Validator) extractFieldValue(xmlData []byte, fieldName string) string {
	re := regexp.MustCompile(`<` + fieldName + `>([^<]+)</` + fieldName + `>`)
	matches := re.FindSubmatch(xmlData)
	if len(matches) > 1 {
		return string(matches[1])
	}
	return ""
}

func (v *Validator) extractAttribute(xmlData []byte, attrName string) string {
	re := regexp.MustCompile(attrName + `="([^"]+)"`)
	matches := re.FindSubmatch(xmlData)
	if len(matches) > 1 {
		return string(matches[1])
	}
	return ""
}

func (v *Validator) extractAccessionFromElement(xmlData []byte, elementName string) string {
	re := regexp.MustCompile(`<` + elementName + `[^>]*accession="([^"]+)"`)
	matches := re.FindSubmatch(xmlData)
	if len(matches) > 1 {
		return string(matches[1])
	}
	return ""
}

func (v *Validator) isValidAccession(accession string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(accession, prefix) {
			// Check format: prefix + 6-9 digits
			numPart := strings.TrimPrefix(accession, prefix)
			if matched, _ := regexp.MatchString(`^\d{6,9}$`, numPart); matched {
				return true
			}
		}
	}
	return false
}

func (v *Validator) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
