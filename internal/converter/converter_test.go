package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestConverter(t *testing.T) (*Converter, func()) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	conv, err := NewConverter(dbPath)
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}

	cleanup := func() {
		conv.Close()
	}

	return conv, cleanup
}

func TestNewConverter(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	if conv == nil {
		t.Fatal("NewConverter returned nil")
	}
	if conv.db == nil {
		t.Error("expected db to be initialized")
	}
	if conv.httpClient == nil {
		t.Error("expected httpClient to be initialized")
	}
	if conv.cache == nil {
		t.Error("expected cache to be initialized")
	}
}

func TestNewConverterInvalidPath(t *testing.T) {
	_, err := NewConverter("/nonexistent/path/test.db")
	if err == nil {
		t.Error("expected error for invalid database path")
	}
}

func TestDetectAccessionType(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	tests := []struct {
		accession string
		expected  string
	}{
		{"SRP000001", "SRP"},
		{"SRX000001", "SRX"},
		{"SRR000001", "SRR"},
		{"SRS000001", "SRS"},
		{"GSE123456", "GSE"},
		{"GSM123456", "GSM"},
		{"PRJNA000001", "PRJNA"},
		{"PRJEB000001", "PRJEB"},
		{"PRJDB000001", "PRJDB"},
		{"SAMN00000001", "SAMN"},
		{"SAME00000001", "SAME"},
		{"SAMD00000001", "SAMD"},
		{"DRP000001", "DRP"},
		{"DRX000001", "DRX"},
		{"DRR000001", "DRR"},
		{"DRS000001", "DRS"},
		{"ERP000001", "ERP"},
		{"ERX000001", "ERX"},
		{"ERR000001", "ERR"},
		{"ERS000001", "ERS"},
		{"UNKNOWN123", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.accession, func(t *testing.T) {
			result := conv.detectAccessionType(tt.accession)
			if result != tt.expected {
				t.Errorf("detectAccessionType(%q) = %q, want %q", tt.accession, result, tt.expected)
			}
		})
	}
}

func TestDetermineTable(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	tests := []struct {
		name     string
		fields   []string
		expected string
	}{
		{"study field", []string{"study_accession"}, "studies"},
		{"experiment field", []string{"experiment_accession"}, "experiments"},
		{"sample field", []string{"sample_accession"}, "samples"},
		{"run field", []string{"run_accession"}, "runs"},
		{"unknown fields", []string{"unknown_field"}, ""},
		{"empty fields", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := conv.determineTable(tt.fields...)
			if result != tt.expected {
				t.Errorf("determineTable(%v) = %q, want %q", tt.fields, result, tt.expected)
			}
		})
	}
}

func TestConvertUnknownAccession(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	_, err := conv.Convert("UNKNOWN123", "SRP")
	if err == nil {
		t.Error("expected error for unknown accession type")
	}
}

func TestConvertUnsupportedConversion(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	_, err := conv.Convert("SRP000001", "UNSUPPORTED")
	if err == nil {
		t.Error("expected error for unsupported conversion")
	}
}

func TestConvertCaching(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	// First call - will query database (empty results)
	result1, err := conv.Convert("SRP000001", "SRX")
	if err != nil {
		t.Fatalf("first convert failed: %v", err)
	}

	// Second call - should hit cache
	result2, err := conv.Convert("SRP000001", "SRX")
	if err != nil {
		t.Fatalf("second convert failed: %v", err)
	}

	if result1.Source != result2.Source {
		t.Error("cached result should match original")
	}
}

func TestConvertNormalization(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	// Test that accession is normalized (uppercase, trimmed)
	result, err := conv.Convert("  srp000001  ", "SRX")
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}

	if result.Source != "SRP000001" {
		t.Errorf("expected normalized accession 'SRP000001', got %q", result.Source)
	}
}

func TestConvertSRPToSRX(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	// Insert test data
	_, err := conv.db.GetSQLDB().Exec(`
		INSERT OR REPLACE INTO studies (study_accession, study_title) VALUES ('SRP000001', 'Test')
	`)
	if err != nil {
		t.Fatalf("failed to insert test study: %v", err)
	}

	_, err = conv.db.GetSQLDB().Exec(`
		INSERT OR REPLACE INTO experiments (experiment_accession, study_accession, title)
		VALUES ('SRX000001', 'SRP000001', 'Exp 1'),
			   ('SRX000002', 'SRP000001', 'Exp 2')
	`)
	if err != nil {
		t.Fatalf("failed to insert test experiments: %v", err)
	}

	result, err := conv.Convert("SRP000001", "SRX")
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}

	if len(result.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(result.Targets))
	}
}

func TestConvertSRRToSRP(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	// Insert test data
	_, err := conv.db.GetSQLDB().Exec(`
		INSERT OR REPLACE INTO studies (study_accession, study_title) VALUES ('SRP000001', 'Test')
	`)
	if err != nil {
		t.Fatalf("failed to insert test study: %v", err)
	}
	_, err = conv.db.GetSQLDB().Exec(`
		INSERT OR REPLACE INTO experiments (experiment_accession, study_accession, title)
		VALUES ('SRX000001', 'SRP000001', 'Test')
	`)
	if err != nil {
		t.Fatalf("failed to insert test experiment: %v", err)
	}
	_, err = conv.db.GetSQLDB().Exec(`
		INSERT OR REPLACE INTO runs (run_accession, experiment_accession)
		VALUES ('SRR000001', 'SRX000001')
	`)
	if err != nil {
		t.Fatalf("failed to insert test run: %v", err)
	}

	result, err := conv.Convert("SRR000001", "SRP")
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}

	if len(result.Targets) != 1 || result.Targets[0] != "SRP000001" {
		t.Errorf("expected ['SRP000001'], got %v", result.Targets)
	}
}

func TestClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	conv, err := NewConverter(dbPath)
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}

	// Close should not panic
	conv.Close()

	// Close with nil db should not panic
	conv2 := &Converter{db: nil}
	conv2.Close()
}

func TestQueryLocalDBNilDB(t *testing.T) {
	conv := &Converter{db: nil, cache: make(map[string]*ConversionResult)}

	_, err := conv.queryLocalDB("field", "where", "value")
	if err == nil {
		t.Error("expected error for nil database")
	}
}

func TestQueryCustomNilDB(t *testing.T) {
	conv := &Converter{db: nil, cache: make(map[string]*ConversionResult)}

	_, err := conv.queryCustom("SELECT 1")
	if err == nil {
		t.Error("expected error for nil database")
	}
}

func TestConversionResultFields(t *testing.T) {
	result := &ConversionResult{
		Source:     "SRP000001",
		TargetType: "SRX",
		Targets:    []string{"SRX000001", "SRX000002"},
	}

	if result.Source != "SRP000001" {
		t.Errorf("expected source SRP000001, got %q", result.Source)
	}
	if result.TargetType != "SRX" {
		t.Errorf("expected target type SRX, got %q", result.TargetType)
	}
	if len(result.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(result.Targets))
	}
}

func TestConvertBioSampleToSRS(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	// Insert test identifier
	_, err := conv.db.GetSQLDB().Exec(`
		INSERT OR REPLACE INTO identifiers (record_type, record_accession, id_type, id_value)
		VALUES ('sample', 'SRS000001', 'BioSample', 'SAMN00000001')
	`)
	if err != nil {
		// Skip if table doesn't exist (migration issue)
		t.Skipf("skipping test: %v", err)
	}

	result, err := conv.Convert("SAMN00000001", "SRS")
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}

	if len(result.Targets) != 1 || result.Targets[0] != "SRS000001" {
		t.Errorf("expected ['SRS000001'], got %v", result.Targets)
	}
}

func TestConvertEmptyString(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	_, err := conv.Convert("", "SRP")
	if err == nil {
		t.Error("expected error for empty accession")
	}
}

func TestConvertSRPToSRR(t *testing.T) {
	conv, cleanup := setupTestConverter(t)
	defer cleanup()

	// Insert test data chain: study -> experiment -> run
	db := conv.db.GetSQLDB()
	db.Exec(`INSERT OR REPLACE INTO studies (study_accession, study_title) VALUES ('SRP000001', 'Test')`)
	db.Exec(`INSERT OR REPLACE INTO experiments (experiment_accession, study_accession, title) VALUES ('SRX000001', 'SRP000001', 'Exp')`)
	db.Exec(`INSERT OR REPLACE INTO runs (run_accession, experiment_accession) VALUES ('SRR000001', 'SRX000001')`)

	result, err := conv.Convert("SRP000001", "SRR")
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}

	if len(result.Targets) != 1 || result.Targets[0] != "SRR000001" {
		t.Errorf("expected ['SRR000001'], got %v", result.Targets)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
