package processor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nishad/srake/internal/database"
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

	return &FilteredProcessor{
		StreamProcessor: NewStreamProcessor(db),
		filters:         filters,
		stats:           NewFilterStats(),
	}, nil
}

// ProcessWithFilters processes data with filtering applied
func (fp *FilteredProcessor) ProcessWithFilters(ctx context.Context, source string) error {
	// Set up progress callback to include filter stats
	fp.SetProgressFunc(func(p Progress) {
		if fp.filters.Verbose {
			fmt.Printf("Progress: %.1f%% | Matched: %d/%d | Skipped: %d\n",
				p.PercentComplete,
				fp.stats.TotalMatched,
				fp.stats.TotalProcessed,
				fp.stats.TotalSkipped)
		}
	})

	// Start processing
	var err error
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		err = fp.ProcessURL(ctx, source)
	} else {
		err = fp.ProcessFile(ctx, source)
	}

	// Print final statistics
	if fp.filters.HasFilters() {
		fmt.Println("\n" + fp.stats.GetSummary())
	}

	return err
}

// Override processing methods to apply filters

// ProcessStudy applies filters to a study record
func (fp *FilteredProcessor) ProcessStudy(study *parser.Study) error {
	fp.stats.TotalProcessed++

	// Apply date filter for studies
	if !fp.shouldProcessStudyByDate(study) {
		fp.stats.SkippedByDate++
		fp.stats.TotalSkipped++
		return nil
	}

	// Apply center filter
	if !fp.shouldProcessByCenter(study.CenterName) {
		fp.stats.SkippedByCenter++
		fp.stats.TotalSkipped++
		return nil
	}

	// Apply study type filter
	if len(fp.filters.StudyTypes) > 0 && study.Descriptor.StudyType != nil {
		if !contains(fp.filters.StudyTypes, study.Descriptor.StudyType.ExistingStudyType) {
			fp.stats.TotalSkipped++
			return nil
		}
	}

	// If stats only mode, just count
	if fp.filters.StatsOnly {
		fp.stats.TotalMatched++
		fp.stats.UniqueStudies[study.Accession] = true
		return nil
	}

	// Process the study
	dbStudy := &database.Study{
		StudyAccession: study.Accession,
		StudyTitle:     study.Descriptor.StudyTitle,
		StudyAbstract:  study.Descriptor.StudyAbstract,
		StudyType:      getStudyType(study),
	}

	err := fp.db.InsertStudy(dbStudy)
	if err == nil {
		fp.stats.TotalMatched++
		fp.stats.UniqueStudies[study.Accession] = true
		fp.recordsInserted.Add(1)
	}

	return err
}

// ProcessExperiment applies filters to an experiment record
func (fp *FilteredProcessor) ProcessExperiment(exp *parser.Experiment) error {
	fp.stats.TotalProcessed++

	// Apply platform filter
	if !fp.shouldProcessByPlatform(exp) {
		fp.stats.SkippedByPlatform++
		fp.stats.TotalSkipped++
		return nil
	}

	// Apply strategy filter
	if !fp.shouldProcessByStrategy(exp) {
		fp.stats.SkippedByStrategy++
		fp.stats.TotalSkipped++
		return nil
	}

	// Apply instrument model filter
	if !fp.shouldProcessByInstrument(exp) {
		fp.stats.TotalSkipped++
		return nil
	}

	// If stats only mode, just count
	if fp.filters.StatsOnly {
		fp.stats.TotalMatched++
		fp.stats.UniqueExperiments[exp.Accession] = true
		return nil
	}

	// Process the experiment
	platform := extractPlatform(exp)
	instrument := extractInstrumentModel(exp.Platform)

	dbExp := &database.Experiment{
		ExperimentAccession: exp.Accession,
		StudyAccession:      exp.StudyRef.Accession,
		Title:               exp.Title,
		Platform:            platform,
		InstrumentModel:     instrument,
	}

	if exp.Design.LibraryDescriptor.LibraryStrategy != "" {
		dbExp.LibraryStrategy = exp.Design.LibraryDescriptor.LibraryStrategy
		dbExp.LibrarySource = exp.Design.LibraryDescriptor.LibrarySource
	}

	err := fp.db.InsertExperiment(dbExp)
	if err == nil {
		fp.stats.TotalMatched++
		fp.stats.UniqueExperiments[exp.Accession] = true
		fp.recordsInserted.Add(1)
	}

	return err
}

// ProcessSample applies filters to a sample record
func (fp *FilteredProcessor) ProcessSample(sample *parser.Sample) error {
	fp.stats.TotalProcessed++

	// Apply taxonomy filter
	if !fp.shouldProcessByTaxonomy(sample) {
		fp.stats.SkippedByTaxonomy++
		fp.stats.TotalSkipped++
		return nil
	}

	// Apply organism filter
	if !fp.shouldProcessByOrganism(sample) {
		fp.stats.SkippedByOrganism++
		fp.stats.TotalSkipped++
		return nil
	}

	// If stats only mode, just count
	if fp.filters.StatsOnly {
		fp.stats.TotalMatched++
		fp.stats.UniqueSamples[sample.Accession] = true
		return nil
	}

	// Process the sample
	dbSample := &database.Sample{
		SampleAccession: sample.Accession,
		Title:           sample.Title,
		Organism:        sample.SampleName.ScientificName,
		TaxonID:         sample.SampleName.TaxonID,
		Description:     sample.Description,
	}

	// Extract additional attributes if available
	if sample.SampleAttributes != nil {
		for _, attr := range sample.SampleAttributes.Attributes {
			switch strings.ToLower(attr.Tag) {
			case "tissue":
				dbSample.Tissue = attr.Value
			case "cell_type", "cell type":
				dbSample.CellType = attr.Value
			}
		}
	}

	err := fp.db.InsertSample(dbSample)
	if err == nil {
		fp.stats.TotalMatched++
		fp.stats.UniqueSamples[sample.Accession] = true
		fp.recordsInserted.Add(1)
	}

	return err
}

// ProcessRun applies filters to a run record
func (fp *FilteredProcessor) ProcessRun(run *parser.Run) error {
	fp.stats.TotalProcessed++

	// Apply read count filter
	if !fp.shouldProcessByReadCount(run) {
		fp.stats.SkippedByReads++
		fp.stats.TotalSkipped++
		return nil
	}

	// If stats only mode, just count
	if fp.filters.StatsOnly {
		fp.stats.TotalMatched++
		fp.stats.UniqueRuns[run.Accession] = true
		return nil
	}

	// Process the run
	dbRun := &database.Run{
		RunAccession:        run.Accession,
		ExperimentAccession: run.ExperimentRef.Accession,
	}

	if run.Statistics != nil {
		dbRun.TotalSpots = run.Statistics.TotalSpots
		dbRun.TotalBases = run.Statistics.TotalBases
	}

	err := fp.db.InsertRun(dbRun)
	if err == nil {
		fp.stats.TotalMatched++
		fp.stats.UniqueRuns[run.Accession] = true
		fp.recordsInserted.Add(1)
	}

	return err
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