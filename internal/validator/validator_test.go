package validator

import (
	"testing"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator(ValidationConfig{
		ValidateEnumerations: true,
		ValidateReferences:   true,
		ValidateRequired:     true,
		StrictMode:           false,
	})

	if v == nil {
		t.Fatal("NewValidator returned nil")
	}
}

func TestDefaultValidator(t *testing.T) {
	v := DefaultValidator()
	if v == nil {
		t.Fatal("DefaultValidator returned nil")
	}
	if !v.config.ValidateEnumerations {
		t.Error("expected ValidateEnumerations to be true")
	}
	if !v.config.ValidateReferences {
		t.Error("expected ValidateReferences to be true")
	}
	if !v.config.ValidateRequired {
		t.Error("expected ValidateRequired to be true")
	}
}

func TestDetectDocumentType(t *testing.T) {
	v := DefaultValidator()

	tests := []struct {
		name     string
		xml      string
		expected string
	}{
		{"study", `<STUDY accession="SRP000001"><DESCRIPTOR><STUDY_TITLE>Test</STUDY_TITLE></DESCRIPTOR></STUDY>`, "study"},
		{"study set", `<STUDY_SET><STUDY accession="SRP000001"></STUDY></STUDY_SET>`, "study"},
		{"sample", `<SAMPLE accession="SRS000001"><SAMPLE_NAME><TAXON_ID>9606</TAXON_ID></SAMPLE_NAME></SAMPLE>`, "sample"},
		{"sample set", `<SAMPLE_SET><SAMPLE accession="SRS000001"></SAMPLE></SAMPLE_SET>`, "sample"},
		{"experiment", `<EXPERIMENT accession="SRX000001"><TITLE>Test</TITLE></EXPERIMENT>`, "experiment"},
		{"experiment set", `<EXPERIMENT_SET><EXPERIMENT></EXPERIMENT></EXPERIMENT_SET>`, "experiment"},
		{"run", `<RUN accession="SRR000001"></RUN>`, "run"},
		{"run set", `<RUN_SET><RUN></RUN></RUN_SET>`, "run"},
		{"analysis", `<ANALYSIS accession="SRZ000001"><TITLE>Test</TITLE></ANALYSIS>`, "analysis"},
		{"submission", `<SUBMISSION accession="SRA000001"></SUBMISSION>`, "submission"},
		{"unknown", `<UNKNOWN>data</UNKNOWN>`, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docType := v.detectDocumentType([]byte(tt.xml))
			if docType != tt.expected {
				t.Errorf("detectDocumentType() = %q, want %q", docType, tt.expected)
			}
		})
	}
}

func TestValidateXMLStructure(t *testing.T) {
	v := DefaultValidator()

	// Valid XML
	validXML := `<STUDY accession="SRP000001"><DESCRIPTOR><STUDY_TITLE>Test</STUDY_TITLE><STUDY_TYPE existing_study_type="Other"/></DESCRIPTOR></STUDY>`
	result, err := v.ValidateXML([]byte(validXML))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if result.Stats.ElementsValidated == 0 {
		t.Error("expected elements to be validated")
	}

	// Invalid XML
	invalidXML := `<STUDY><BROKEN`
	result, err = v.ValidateXML([]byte(invalidXML))
	if err != nil {
		t.Fatalf("ValidateXML should not return error for invalid XML, but set result errors")
	}
	if result.IsValid {
		t.Error("expected invalid result for broken XML")
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors for broken XML")
	}
}

func TestValidateStudyRequired(t *testing.T) {
	v := DefaultValidator()

	// Study missing required fields
	xml := `<STUDY accession="SRP000001"><DESCRIPTOR></DESCRIPTOR></STUDY>`
	result, err := v.ValidateXML([]byte(xml))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if result.IsValid {
		t.Error("expected invalid result for study missing required fields")
	}

	// Check for specific missing fields
	hasStudyTitle := false
	hasStudyType := false
	for _, e := range result.Errors {
		if e.Field == "STUDY_TITLE" {
			hasStudyTitle = true
		}
		if e.Field == "STUDY_TYPE" {
			hasStudyType = true
		}
	}
	if !hasStudyTitle {
		t.Error("expected error for missing STUDY_TITLE")
	}
	if !hasStudyType {
		t.Error("expected error for missing STUDY_TYPE")
	}
}

func TestValidateStudyComplete(t *testing.T) {
	v := DefaultValidator()

	xml := `<STUDY accession="SRP000001">
		<DESCRIPTOR>
			<STUDY_TITLE>Test Study</STUDY_TITLE>
			<STUDY_TYPE existing_study_type="Other"/>
		</DESCRIPTOR>
	</STUDY>`

	result, err := v.ValidateXML([]byte(xml))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if !result.IsValid {
		t.Errorf("expected valid result, got errors: %v", result.Errors)
	}
}

func TestValidateSampleRequired(t *testing.T) {
	v := DefaultValidator()

	// Sample missing TAXON_ID and SCIENTIFIC_NAME
	xml := `<SAMPLE accession="SRS000001"><SAMPLE_NAME></SAMPLE_NAME></SAMPLE>`
	result, err := v.ValidateXML([]byte(xml))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if result.IsValid {
		t.Error("expected invalid result for sample missing required fields")
	}
}

func TestValidateSampleTaxonIDFormat(t *testing.T) {
	v := DefaultValidator()

	// Valid TAXON_ID
	xml := `<SAMPLE accession="SRS000001"><SAMPLE_NAME><TAXON_ID>9606</TAXON_ID><SCIENTIFIC_NAME>Homo sapiens</SCIENTIFIC_NAME></SAMPLE_NAME></SAMPLE>`
	result, err := v.ValidateXML([]byte(xml))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if !result.IsValid {
		t.Errorf("expected valid result, got errors: %v", result.Errors)
	}

	// Invalid TAXON_ID (non-numeric)
	xml = `<SAMPLE accession="SRS000001"><SAMPLE_NAME><TAXON_ID>not_a_number</TAXON_ID><SCIENTIFIC_NAME>Test</SCIENTIFIC_NAME></SAMPLE_NAME></SAMPLE>`
	result, err = v.ValidateXML([]byte(xml))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if result.IsValid {
		t.Error("expected invalid result for non-numeric TAXON_ID")
	}
}

func TestValidateExperimentRequired(t *testing.T) {
	v := DefaultValidator()

	// Experiment missing required fields
	xml := `<EXPERIMENT accession="SRX000001"></EXPERIMENT>`
	result, err := v.ValidateXML([]byte(xml))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if result.IsValid {
		t.Error("expected invalid result for experiment missing required fields")
	}

	requiredFields := map[string]bool{
		"TITLE":            false,
		"STUDY_REF":        false,
		"DESIGN":           false,
		"PLATFORM":         false,
		"LIBRARY_STRATEGY": false,
	}

	for _, e := range result.Errors {
		if _, ok := requiredFields[e.Field]; ok {
			requiredFields[e.Field] = true
		}
	}

	for field, found := range requiredFields {
		if !found {
			t.Errorf("expected error for missing %s", field)
		}
	}
}

func TestValidateRunRequired(t *testing.T) {
	v := DefaultValidator()

	xml := `<RUN accession="SRR000001"></RUN>`
	result, err := v.ValidateXML([]byte(xml))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if result.IsValid {
		t.Error("expected invalid result for run missing EXPERIMENT_REF")
	}
}

func TestValidateAnalysisRequired(t *testing.T) {
	v := DefaultValidator()

	xml := `<ANALYSIS accession="SRZ000001"></ANALYSIS>`
	result, err := v.ValidateXML([]byte(xml))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if result.IsValid {
		t.Error("expected invalid result for analysis missing required fields")
	}
}

func TestValidatePlatform(t *testing.T) {
	v := DefaultValidator()

	// Test platform validation directly
	xml := `<PLATFORM><ILLUMINA><INSTRUMENT_MODEL>Illumina NovaSeq 6000</INSTRUMENT_MODEL></ILLUMINA></PLATFORM>`
	result := &ValidationResult{IsValid: true, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	v.validatePlatform([]byte(xml), result)

	if result.Stats.AttributesChecked == 0 {
		t.Error("expected platform to be validated")
	}

	// Test unknown platform
	xml = `<PLATFORM><UNKNOWN_PLATFORM/></PLATFORM>`
	result = &ValidationResult{IsValid: true, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	v.validatePlatform([]byte(xml), result)

	if len(result.Warnings) == 0 {
		t.Error("expected warning for unknown platform")
	}
}

func TestValidateStudyReference(t *testing.T) {
	v := DefaultValidator()

	// Valid study reference - test helper directly
	xml := `<STUDY_REF accession="SRP000001"/>`
	result := &ValidationResult{IsValid: true, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	v.validateStudyReference([]byte(xml), result)

	if result.Stats.ReferencesChecked == 0 {
		t.Error("expected references to be checked")
	}
	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings for valid reference: %v", result.Warnings)
	}
}

func TestValidateInvalidStudyReference(t *testing.T) {
	v := DefaultValidator()

	// Invalid study reference
	xml := `<STUDY_REF accession="INVALID"/>`
	result := &ValidationResult{IsValid: true, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	v.validateStudyReference([]byte(xml), result)

	hasRefWarning := false
	for _, w := range result.Warnings {
		if w.Type == "INVALID_REFERENCE" {
			hasRefWarning = true
		}
	}
	if !hasRefWarning {
		t.Error("expected warning for invalid study reference")
	}
}

func TestIsValidAccession(t *testing.T) {
	v := DefaultValidator()

	tests := []struct {
		accession string
		prefixes  []string
		valid     bool
	}{
		{"SRP000001", []string{"SRP"}, true},
		{"SRP123456789", []string{"SRP"}, true},
		{"ERP000001", []string{"SRP", "ERP", "DRP"}, true},
		{"DRP000001", []string{"SRP", "ERP", "DRP"}, true},
		{"INVALID", []string{"SRP"}, false},
		{"SRP1", []string{"SRP"}, false},       // too short
		{"SRP1234567890", []string{"SRP"}, false}, // too long
	}

	for _, tt := range tests {
		t.Run(tt.accession, func(t *testing.T) {
			result := v.isValidAccession(tt.accession, tt.prefixes)
			if result != tt.valid {
				t.Errorf("isValidAccession(%q, %v) = %v, want %v", tt.accession, tt.prefixes, result, tt.valid)
			}
		})
	}
}

func TestContains(t *testing.T) {
	v := DefaultValidator()

	if !v.contains([]string{"a", "b", "c"}, "b") {
		t.Error("expected contains to return true for existing element")
	}

	if v.contains([]string{"a", "b", "c"}, "d") {
		t.Error("expected contains to return false for missing element")
	}

	if v.contains([]string{}, "a") {
		t.Error("expected contains to return false for empty slice")
	}
}

func TestExtractFieldValue(t *testing.T) {
	v := DefaultValidator()

	xml := `<STUDY_TITLE>Test Study Title</STUDY_TITLE>`
	value := v.extractFieldValue([]byte(xml), "STUDY_TITLE")
	if value != "Test Study Title" {
		t.Errorf("expected 'Test Study Title', got %q", value)
	}

	// Missing field
	value = v.extractFieldValue([]byte(xml), "NONEXISTENT")
	if value != "" {
		t.Errorf("expected empty string for missing field, got %q", value)
	}
}

func TestExtractAttribute(t *testing.T) {
	v := DefaultValidator()

	xml := `<FILE filename="test.fastq" filetype="fastq"/>`
	value := v.extractAttribute([]byte(xml), "filetype")
	if value != "fastq" {
		t.Errorf("expected 'fastq', got %q", value)
	}
}

func TestExtractAccessionFromElement(t *testing.T) {
	v := DefaultValidator()

	xml := `<STUDY_REF accession="SRP000001"/>`
	value := v.extractAccessionFromElement([]byte(xml), "STUDY_REF")
	if value != "SRP000001" {
		t.Errorf("expected 'SRP000001', got %q", value)
	}
}

func TestValidateLibraryStrategy(t *testing.T) {
	v := DefaultValidator()

	// Valid strategy
	xml := `<EXPERIMENT><LIBRARY_STRATEGY>RNA-Seq</LIBRARY_STRATEGY></EXPERIMENT>`
	result := &ValidationResult{IsValid: true, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	v.validateLibraryStrategy([]byte(xml), result)

	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings for valid library strategy: %v", result.Warnings)
	}

	// Invalid strategy
	xml = `<EXPERIMENT><LIBRARY_STRATEGY>INVALID_STRATEGY</LIBRARY_STRATEGY></EXPERIMENT>`
	result = &ValidationResult{IsValid: true, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	v.validateLibraryStrategy([]byte(xml), result)

	if len(result.Warnings) == 0 {
		t.Error("expected warning for invalid library strategy")
	}
}

func TestValidateLibrarySource(t *testing.T) {
	v := DefaultValidator()

	// Valid source
	xml := `<LIBRARY_SOURCE>GENOMIC</LIBRARY_SOURCE>`
	result := &ValidationResult{IsValid: true, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	v.validateLibrarySource([]byte(xml), result)

	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}

	// Invalid source
	xml = `<LIBRARY_SOURCE>INVALID</LIBRARY_SOURCE>`
	result = &ValidationResult{IsValid: true, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	v.validateLibrarySource([]byte(xml), result)

	if len(result.Warnings) == 0 {
		t.Error("expected warning for invalid library source")
	}
}

func TestValidateRunFiles(t *testing.T) {
	v := DefaultValidator()

	// Valid file info
	xml := `<RUN><FILES><FILE filename="test.fastq" filetype="fastq"/></FILES></RUN>`
	result := &ValidationResult{IsValid: true, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	v.validateRunFiles([]byte(xml), result)

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}

	// Missing filename attribute
	xml = `<RUN><FILES><FILE filetype="fastq"/></FILES></RUN>`
	result = &ValidationResult{IsValid: true, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	v.validateRunFiles([]byte(xml), result)

	if len(result.Errors) == 0 {
		t.Error("expected error for missing filename attribute")
	}
}

func TestValidationDisabledChecks(t *testing.T) {
	// Create validator with all checks disabled
	v := NewValidator(ValidationConfig{
		ValidateEnumerations: false,
		ValidateReferences:   false,
		ValidateRequired:     false,
		StrictMode:           false,
	})

	// Even a minimal study should pass
	xml := `<STUDY accession="SRP000001"></STUDY>`
	result, err := v.ValidateXML([]byte(xml))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if !result.IsValid {
		t.Errorf("expected valid with checks disabled, got errors: %v", result.Errors)
	}
}

func TestDocTypeField(t *testing.T) {
	v := DefaultValidator()

	xml := `<STUDY accession="SRP000001"><DESCRIPTOR><STUDY_TITLE>Test</STUDY_TITLE><STUDY_TYPE existing_study_type="Other"/></DESCRIPTOR></STUDY>`
	result, err := v.ValidateXML([]byte(xml))
	if err != nil {
		t.Fatalf("ValidateXML failed: %v", err)
	}
	if result.DocType != "study" {
		t.Errorf("expected DocType 'study', got %q", result.DocType)
	}
}
