package processor

import (
	"fmt"
	"strings"
	"time"
)

// FilterOptions defines the filtering criteria for processing SRA metadata
type FilterOptions struct {
	// Taxonomy filters
	TaxonomyIDs   []int // List of NCBI taxonomy IDs to include
	ExcludeTaxIDs []int // Taxonomy IDs to exclude

	// Date filters
	DateFrom  time.Time // Start date (submission/published date)
	DateTo    time.Time // End date
	DateField string    // "submission", "published", or "last_updated"

	// Organism filters
	Organisms        []string // Scientific names to include
	ExcludeOrganisms []string // Scientific names to exclude

	// Technical filters
	Platforms        []string // Sequencing platforms (ILLUMINA, OXFORD_NANOPORE, etc.)
	Strategies       []string // Library strategies (RNA-Seq, WGS, WES, etc.)
	StudyTypes       []string // Study types
	InstrumentModels []string // Specific instrument models

	// Quality filters
	MinReads int64 // Minimum read count (total_spots)
	MaxReads int64 // Maximum read count
	MinBases int64 // Minimum base count (total_bases)
	MaxBases int64 // Maximum base count

	// Source filters
	Centers   []string // Submission centers
	Countries []string // Geographic origin (from attributes)

	// Control flags
	SkipIfNoMatch bool // Skip entire file if no matches
	StatsOnly     bool // Just count matches without inserting
	Verbose       bool // Print detailed filtering information
}

// FilterStats tracks statistics about filtered records
type FilterStats struct {
	// Overall counts
	TotalProcessed int64
	TotalMatched   int64
	TotalSkipped   int64

	// Skip reasons
	SkippedByTaxonomy int64
	SkippedByDate     int64
	SkippedByOrganism int64
	SkippedByPlatform int64
	SkippedByStrategy int64
	SkippedByReads    int64
	SkippedByCenter   int64

	// Unique record tracking
	UniqueStudies     map[string]bool
	UniqueExperiments map[string]bool
	UniqueSamples     map[string]bool
	UniqueRuns        map[string]bool

	// Performance metrics
	StartTime      time.Time
	ProcessingTime time.Duration
}

// NewFilterStats creates a new FilterStats instance
func NewFilterStats() *FilterStats {
	return &FilterStats{
		UniqueStudies:     make(map[string]bool),
		UniqueExperiments: make(map[string]bool),
		UniqueSamples:     make(map[string]bool),
		UniqueRuns:        make(map[string]bool),
		StartTime:         time.Now(),
	}
}

// Validate checks if the filter options are valid
func (f *FilterOptions) Validate() error {
	// Validate date range
	if !f.DateFrom.IsZero() && !f.DateTo.IsZero() {
		if f.DateFrom.After(f.DateTo) {
			return fmt.Errorf("date-from (%s) is after date-to (%s)",
				f.DateFrom.Format("2006-01-02"),
				f.DateTo.Format("2006-01-02"))
		}
	}

	// Validate date field
	if f.DateField != "" {
		validFields := []string{"submission", "published", "last_updated"}
		valid := false
		for _, field := range validFields {
			if f.DateField == field {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid date field: %s (must be one of: %s)",
				f.DateField, strings.Join(validFields, ", "))
		}
	} else if !f.DateFrom.IsZero() || !f.DateTo.IsZero() {
		// Default to submission date if date filters are set but field is not specified
		f.DateField = "submission"
	}

	// Validate read/base ranges
	if f.MinReads > 0 && f.MaxReads > 0 && f.MinReads > f.MaxReads {
		return fmt.Errorf("min-reads (%d) is greater than max-reads (%d)",
			f.MinReads, f.MaxReads)
	}

	if f.MinBases > 0 && f.MaxBases > 0 && f.MinBases > f.MaxBases {
		return fmt.Errorf("min-bases (%d) is greater than max-bases (%d)",
			f.MinBases, f.MaxBases)
	}

	// Normalize platform names
	for i, platform := range f.Platforms {
		f.Platforms[i] = strings.ToUpper(platform)
	}

	// Normalize strategy names
	for i, strategy := range f.Strategies {
		f.Strategies[i] = normalizeStrategy(strategy)
	}

	return nil
}

// HasFilters returns true if any filters are set
func (f *FilterOptions) HasFilters() bool {
	return len(f.TaxonomyIDs) > 0 ||
		len(f.ExcludeTaxIDs) > 0 ||
		!f.DateFrom.IsZero() ||
		!f.DateTo.IsZero() ||
		len(f.Organisms) > 0 ||
		len(f.ExcludeOrganisms) > 0 ||
		len(f.Platforms) > 0 ||
		len(f.Strategies) > 0 ||
		len(f.StudyTypes) > 0 ||
		len(f.InstrumentModels) > 0 ||
		f.MinReads > 0 ||
		f.MaxReads > 0 ||
		f.MinBases > 0 ||
		f.MaxBases > 0 ||
		len(f.Centers) > 0 ||
		len(f.Countries) > 0
}

// String returns a human-readable description of the filters
func (f *FilterOptions) String() string {
	var parts []string

	if len(f.TaxonomyIDs) > 0 {
		parts = append(parts, fmt.Sprintf("TaxIDs=%v", f.TaxonomyIDs))
	}
	if len(f.ExcludeTaxIDs) > 0 {
		parts = append(parts, fmt.Sprintf("ExcludeTaxIDs=%v", f.ExcludeTaxIDs))
	}
	if !f.DateFrom.IsZero() {
		parts = append(parts, fmt.Sprintf("DateFrom=%s", f.DateFrom.Format("2006-01-02")))
	}
	if !f.DateTo.IsZero() {
		parts = append(parts, fmt.Sprintf("DateTo=%s", f.DateTo.Format("2006-01-02")))
	}
	if len(f.Organisms) > 0 {
		parts = append(parts, fmt.Sprintf("Organisms=%v", f.Organisms))
	}
	if len(f.Platforms) > 0 {
		parts = append(parts, fmt.Sprintf("Platforms=%v", f.Platforms))
	}
	if len(f.Strategies) > 0 {
		parts = append(parts, fmt.Sprintf("Strategies=%v", f.Strategies))
	}
	if f.MinReads > 0 {
		parts = append(parts, fmt.Sprintf("MinReads=%d", f.MinReads))
	}
	if f.MaxReads > 0 {
		parts = append(parts, fmt.Sprintf("MaxReads=%d", f.MaxReads))
	}

	if len(parts) == 0 {
		return "No filters"
	}
	return strings.Join(parts, ", ")
}

// GetSummary returns a summary of the filter statistics
func (s *FilterStats) GetSummary() string {
	s.ProcessingTime = time.Since(s.StartTime)

	matchRate := float64(0)
	if s.TotalProcessed > 0 {
		matchRate = float64(s.TotalMatched) * 100 / float64(s.TotalProcessed)
	}

	summary := fmt.Sprintf(`Filter Statistics:
  Total Processed: %d
  Total Matched:   %d (%.1f%%)
  Total Skipped:   %d

Skip Reasons:
  By Taxonomy:  %d
  By Date:      %d
  By Organism:  %d
  By Platform:  %d
  By Strategy:  %d
  By Reads:     %d
  By Center:    %d

Unique Records Matched:
  Studies:     %d
  Experiments: %d
  Samples:     %d
  Runs:        %d

Processing Time: %s`,
		s.TotalProcessed,
		s.TotalMatched, matchRate,
		s.TotalSkipped,
		s.SkippedByTaxonomy,
		s.SkippedByDate,
		s.SkippedByOrganism,
		s.SkippedByPlatform,
		s.SkippedByStrategy,
		s.SkippedByReads,
		s.SkippedByCenter,
		len(s.UniqueStudies),
		len(s.UniqueExperiments),
		len(s.UniqueSamples),
		len(s.UniqueRuns),
		s.ProcessingTime.Round(time.Second))

	return summary
}

// Helper functions

func normalizeStrategy(strategy string) string {
	// Normalize common variations
	replacements := map[string]string{
		"rnaseq":    "RNA-Seq",
		"rna seq":   "RNA-Seq",
		"wgs":       "WGS",
		"wes":       "WES",
		"wxs":       "WXS",
		"chipseq":   "ChIP-Seq",
		"chip seq":  "ChIP-Seq",
		"atacseq":   "ATAC-Seq",
		"atac seq":  "ATAC-Seq",
		"amplicon":  "AMPLICON",
		"targeted":  "Targeted-Capture",
		"bisulfite": "Bisulfite-Seq",
		"hic":       "Hi-C",
		"hi c":      "Hi-C",
		"mirnaseq":  "miRNA-Seq",
		"mirna seq": "miRNA-Seq",
	}

	lower := strings.ToLower(strings.TrimSpace(strategy))
	if normalized, exists := replacements[lower]; exists {
		return normalized
	}
	return strategy
}
