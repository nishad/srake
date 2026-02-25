package processor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nishad/srake/internal/parser"
)

// FilteredProcessor extends StreamProcessor with filtering capabilities
type FilteredProcessor struct {
	*StreamProcessor
	filters FilterOptions
	stats   *FilterStats
}

// NewFilteredProcessor creates a new processor with filtering capabilities
func NewFilteredProcessor(db Database, filters FilterOptions) (*FilteredProcessor, error) {
	// Validate filters
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("invalid filter options: %w", err)
	}

	fp := &FilteredProcessor{
		StreamProcessor: NewStreamProcessor(db),
		filters:         filters,
		stats:           NewFilterStats(),
	}

	// Wire up the record filter so StreamProcessor calls our Filter* methods
	fp.StreamProcessor.SetRecordFilter(fp)

	return fp, nil
}

// ProcessWithFilters processes data with filtering applied
func (fp *FilteredProcessor) ProcessWithFilters(ctx context.Context, source string) error {
	// Only set a default progress func if the caller hasn't already registered one
	if fp.progressFunc == nil {
		fp.SetProgressFunc(func(p Progress) {
			if fp.filters.Verbose {
				fmt.Fprintf(os.Stderr, "Progress: %.1f%% | Matched: %d/%d | Skipped: %d\n",
					p.PercentComplete,
					fp.stats.TotalMatched,
					fp.stats.TotalProcessed,
					fp.stats.TotalSkipped)
			}
		})
	}

	// Start processing
	var err error
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		err = fp.ProcessURL(ctx, source)
	} else {
		err = fp.ProcessFile(ctx, source)
	}

	return err
}

// RecordFilter interface implementation — each method returns true to include
// the record, false to skip it.

// FilterStudy applies filters to a study record
func (fp *FilteredProcessor) FilterStudy(study *parser.Study) bool {
	fp.stats.TotalProcessed++

	// Apply date filter for studies
	if !fp.shouldProcessStudyByDate(study) {
		fp.stats.SkippedByDate++
		fp.stats.TotalSkipped++
		return false
	}

	// Apply center filter
	if !fp.shouldProcessByCenter(study.CenterName) {
		fp.stats.SkippedByCenter++
		fp.stats.TotalSkipped++
		return false
	}

	// Apply study type filter
	if len(fp.filters.StudyTypes) > 0 && study.Descriptor.StudyType != nil {
		if !contains(fp.filters.StudyTypes, study.Descriptor.StudyType.ExistingStudyType) {
			fp.stats.TotalSkipped++
			return false
		}
	}

	fp.stats.TotalMatched++
	fp.stats.UniqueStudies[study.Accession] = true

	// If stats only mode, count the match but don't insert
	if fp.filters.StatsOnly {
		return false
	}

	return true
}

// FilterExperiment applies filters to an experiment record
func (fp *FilteredProcessor) FilterExperiment(exp *parser.Experiment) bool {
	fp.stats.TotalProcessed++

	// Apply platform filter
	if !fp.shouldProcessByPlatform(exp) {
		fp.stats.SkippedByPlatform++
		fp.stats.TotalSkipped++
		return false
	}

	// Apply strategy filter
	if !fp.shouldProcessByStrategy(exp) {
		fp.stats.SkippedByStrategy++
		fp.stats.TotalSkipped++
		return false
	}

	// Apply instrument model filter
	if !fp.shouldProcessByInstrument(exp) {
		fp.stats.TotalSkipped++
		return false
	}

	fp.stats.TotalMatched++
	fp.stats.UniqueExperiments[exp.Accession] = true

	// If stats only mode, count the match but don't insert
	if fp.filters.StatsOnly {
		return false
	}

	return true
}

// FilterSample applies filters to a sample record
func (fp *FilteredProcessor) FilterSample(sample *parser.Sample) bool {
	fp.stats.TotalProcessed++

	// Apply taxonomy filter
	if !fp.shouldProcessByTaxonomy(sample) {
		fp.stats.SkippedByTaxonomy++
		fp.stats.TotalSkipped++
		return false
	}

	// Apply organism filter
	if !fp.shouldProcessByOrganism(sample) {
		fp.stats.SkippedByOrganism++
		fp.stats.TotalSkipped++
		return false
	}

	fp.stats.TotalMatched++
	fp.stats.UniqueSamples[sample.Accession] = true

	// If stats only mode, count the match but don't insert
	if fp.filters.StatsOnly {
		return false
	}

	return true
}

// FilterRun applies filters to a run record
func (fp *FilteredProcessor) FilterRun(run *parser.Run) bool {
	fp.stats.TotalProcessed++

	// Apply read count filter
	if !fp.shouldProcessByReadCount(run) {
		fp.stats.SkippedByReads++
		fp.stats.TotalSkipped++
		return false
	}

	fp.stats.TotalMatched++
	fp.stats.UniqueRuns[run.Accession] = true

	// If stats only mode, count the match but don't insert
	if fp.filters.StatsOnly {
		return false
	}

	return true
}

// Filter check methods

func (fp *FilteredProcessor) shouldProcessByTaxonomy(sample *parser.Sample) bool {
	// Check include list
	if len(fp.filters.TaxonomyIDs) > 0 {
		found := false
		for _, taxID := range fp.filters.TaxonomyIDs {
			if taxID == sample.SampleName.TaxonID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check exclude list
	if len(fp.filters.ExcludeTaxIDs) > 0 {
		for _, taxID := range fp.filters.ExcludeTaxIDs {
			if taxID == sample.SampleName.TaxonID {
				return false
			}
		}
	}

	return true
}

func (fp *FilteredProcessor) shouldProcessByOrganism(sample *parser.Sample) bool {
	// Check include list
	if len(fp.filters.Organisms) > 0 {
		if !contains(fp.filters.Organisms, sample.SampleName.ScientificName) {
			return false
		}
	}

	// Check exclude list
	if len(fp.filters.ExcludeOrganisms) > 0 {
		if contains(fp.filters.ExcludeOrganisms, sample.SampleName.ScientificName) {
			return false
		}
	}

	return true
}

func (fp *FilteredProcessor) shouldProcessStudyByDate(study *parser.Study) bool {
	// Extract date from study attributes
	var studyDate time.Time

	// Look for submission date in attributes
	if study.StudyAttributes != nil {
		for _, attr := range study.StudyAttributes.Attributes {
			if strings.EqualFold(attr.Tag, "submission_date") ||
				strings.EqualFold(attr.Tag, "ENA-FIRST-PUBLIC") ||
				strings.EqualFold(attr.Tag, "ENA-LAST-UPDATE") {
				if parsed, err := time.Parse("2006-01-02", attr.Value); err == nil {
					studyDate = parsed
					break
				}
				if parsed, err := time.Parse("2006-01-02T15:04:05Z", attr.Value); err == nil {
					studyDate = parsed
					break
				}
			}
		}
	}

	// If no date found and we have date filters, skip
	if studyDate.IsZero() && (!fp.filters.DateFrom.IsZero() || !fp.filters.DateTo.IsZero()) {
		return false
	}

	// Check date range
	if !fp.filters.DateFrom.IsZero() && studyDate.Before(fp.filters.DateFrom) {
		return false
	}

	if !fp.filters.DateTo.IsZero() && studyDate.After(fp.filters.DateTo) {
		return false
	}

	return true
}

func (fp *FilteredProcessor) shouldProcessByPlatform(exp *parser.Experiment) bool {
	if len(fp.filters.Platforms) == 0 {
		return true
	}

	platform := extractPlatform(exp)
	return contains(fp.filters.Platforms, platform)
}

func (fp *FilteredProcessor) shouldProcessByStrategy(exp *parser.Experiment) bool {
	if len(fp.filters.Strategies) == 0 {
		return true
	}

	if exp.Design.LibraryDescriptor.LibraryStrategy == "" {
		return false
	}

	strategy := exp.Design.LibraryDescriptor.LibraryStrategy
	return contains(fp.filters.Strategies, strategy)
}

func (fp *FilteredProcessor) shouldProcessByInstrument(exp *parser.Experiment) bool {
	if len(fp.filters.InstrumentModels) == 0 {
		return true
	}

	instrument := extractInstrumentModel(exp.Platform)
	return contains(fp.filters.InstrumentModels, instrument)
}

func (fp *FilteredProcessor) shouldProcessByCenter(centerName string) bool {
	if len(fp.filters.Centers) == 0 {
		return true
	}

	return contains(fp.filters.Centers, centerName)
}

func (fp *FilteredProcessor) shouldProcessByReadCount(run *parser.Run) bool {
	if run.Statistics == nil {
		return true // No statistics available, include by default
	}

	// Check minimum reads
	if fp.filters.MinReads > 0 && run.Statistics.TotalSpots < fp.filters.MinReads {
		return false
	}

	// Check maximum reads
	if fp.filters.MaxReads > 0 && run.Statistics.TotalSpots > fp.filters.MaxReads {
		return false
	}

	// Check minimum bases
	if fp.filters.MinBases > 0 && run.Statistics.TotalBases < fp.filters.MinBases {
		return false
	}

	// Check maximum bases
	if fp.filters.MaxBases > 0 && run.Statistics.TotalBases > fp.filters.MaxBases {
		return false
	}

	return true
}

// Helper functions

func extractPlatform(exp *parser.Experiment) string {
	if exp.Platform.Illumina != nil {
		return "ILLUMINA"
	}
	if exp.Platform.OxfordNanopore != nil {
		return "OXFORD_NANOPORE"
	}
	if exp.Platform.PacBio != nil {
		return "PACBIO_SMRT"
	}
	if exp.Platform.IonTorrent != nil {
		return "ION_TORRENT"
	}
	if exp.Platform.LS454 != nil {
		return "LS454"
	}
	if exp.Platform.Solid != nil {
		return "ABI_SOLID"
	}
	if exp.Platform.CompleteGenomics != nil {
		return "COMPLETE_GENOMICS"
	}
	return ""
}

func getStudyType(study *parser.Study) string {
	if study.Descriptor.StudyType != nil {
		if study.Descriptor.StudyType.ExistingStudyType != "" {
			return study.Descriptor.StudyType.ExistingStudyType
		}
		return study.Descriptor.StudyType.NewStudyType
	}
	return ""
}

// GetStats returns the current filter statistics
func (fp *FilteredProcessor) GetStats() *FilterStats {
	return fp.stats
}
