package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/paths"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Relationship commands
var (
	runsCmd = &cobra.Command{
		Use:   "runs <accession>",
		Short: "Get all runs for a study, experiment, or sample",
		Long: `Retrieve all run accessions associated with a given study (SRP),
experiment (SRX), or sample (SRS) accession.`,
		Example: `  # Get all runs for a study
  srake runs SRP123456

  # Get runs with detailed information
  srake runs SRX123456 --detailed

  # Export as JSON
  srake runs SRP123456 --format json --output runs.json`,
		Args: cobra.ExactArgs(1),
		RunE: runGetRuns,
	}

	samplesCmd = &cobra.Command{
		Use:   "samples <accession>",
		Short: "Get all samples for a study or experiment",
		Long: `Retrieve all sample accessions associated with a given study (SRP)
or experiment (SRX) accession.`,
		Example: `  # Get all samples for a study
  srake samples SRP123456

  # Get samples with organism information
  srake samples SRX123456 --detailed`,
		Args: cobra.ExactArgs(1),
		RunE: runGetSamples,
	}

	experimentsCmd = &cobra.Command{
		Use:   "experiments <accession>",
		Short: "Get all experiments for a study or sample",
		Long: `Retrieve all experiment accessions associated with a given study (SRP)
or sample (SRS) accession.`,
		Example: `  # Get all experiments for a study
  srake experiments SRP123456

  # Get experiments with platform information
  srake experiments SRS123456 --detailed`,
		Args: cobra.ExactArgs(1),
		RunE: runGetExperiments,
	}

	studiesCmd = &cobra.Command{
		Use:   "studies <accession>",
		Short: "Get study information for any SRA accession",
		Long: `Retrieve the parent study (SRP) for any SRA accession type
(experiment, sample, run).`,
		Example: `  # Get study for an experiment
  srake studies SRX123456

  # Get study for a run
  srake studies SRR123456`,
		Args: cobra.ExactArgs(1),
		RunE: runGetStudies,
	}
)

// Common flags for relationship commands
var (
	relFormat   string
	relOutput   string
	relDetailed bool
	relLimit    int
	relFields   string
)

// RunInfo contains information about a run
type RunInfo struct {
	RunAccession string `json:"run_accession" yaml:"run_accession"`
	Experiment   string `json:"experiment_accession" yaml:"experiment_accession"`
	TotalSpots   int64  `json:"total_spots" yaml:"total_spots"`
	TotalBases   int64  `json:"total_bases" yaml:"total_bases"`
	Published    string `json:"published" yaml:"published"`
	Platform     string `json:"platform,omitempty" yaml:"platform,omitempty"`
	Strategy     string `json:"library_strategy,omitempty" yaml:"library_strategy,omitempty"`
}

func init() {
	// Add common flags to all relationship commands
	for _, cmd := range []*cobra.Command{runsCmd, samplesCmd, experimentsCmd, studiesCmd} {
		cmd.Flags().StringVarP(&relFormat, "format", "f", "table", "Output format (table|json|yaml|csv|tsv)")
		cmd.Flags().StringVarP(&relOutput, "output", "o", "", "Save results to file")
		cmd.Flags().BoolVarP(&relDetailed, "detailed", "d", false, "Include detailed information")
		cmd.Flags().IntVarP(&relLimit, "limit", "l", 0, "Limit number of results (0 = no limit)")
		cmd.Flags().StringVar(&relFields, "fields", "", "Comma-separated list of fields to include")
	}
}

// runGetRuns retrieves all runs for a given accession
func runGetRuns(cmd *cobra.Command, args []string) error {
	accession := strings.ToUpper(args[0])

	// Resolve database path
	dbPath := serverDBPath
	if dbPath == "" {
		dbPath = paths.GetDatabasePath()
	}

	// Initialize database
	db, err := database.Initialize(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Determine query based on accession type
	var query string
	switch {
	case strings.HasPrefix(accession, "SRP"):
		// Get runs for a study
		query = `
			SELECT r.run_accession, r.experiment_accession, r.total_spots,
			       r.total_bases, r.published, e.platform, e.library_strategy
			FROM runs r
			JOIN experiments e ON r.experiment_accession = e.experiment_accession
			WHERE e.study_accession = ?`

	case strings.HasPrefix(accession, "SRX"):
		// Get runs for an experiment
		query = `
			SELECT r.run_accession, r.experiment_accession, r.total_spots,
			       r.total_bases, r.published, e.platform, e.library_strategy
			FROM runs r
			JOIN experiments e ON r.experiment_accession = e.experiment_accession
			WHERE r.experiment_accession = ?`

	case strings.HasPrefix(accession, "SRS"):
		// Get runs for a sample
		query = `
			SELECT r.run_accession, r.experiment_accession, r.total_spots,
			       r.total_bases, r.published, e.platform, e.library_strategy
			FROM runs r
			JOIN experiments e ON r.experiment_accession = e.experiment_accession
			JOIN experiment_samples es ON e.experiment_accession = es.experiment_accession
			WHERE es.sample_accession = ?`

	default:
		return fmt.Errorf("unsupported accession type: %s", accession)
	}

	// Add limit if specified
	if relLimit > 0 {
		query += fmt.Sprintf(" LIMIT %d", relLimit)
	}

	// Execute query
	rows, err := db.GetSQLDB().Query(query, accession)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Collect results
	runs := []RunInfo{}
	for rows.Next() {
		var run RunInfo
		var published, platform, strategy *string

		err := rows.Scan(&run.RunAccession, &run.Experiment, &run.TotalSpots,
			&run.TotalBases, &published, &platform, &strategy)
		if err != nil {
			continue
		}

		if published != nil {
			run.Published = *published
		}
		if platform != nil {
			run.Platform = *platform
		}
		if strategy != nil {
			run.Strategy = *strategy
		}

		runs = append(runs, run)
	}

	// Output results
	return outputRelationshipResults(runs, "runs", relDetailed)
}

// runGetSamples retrieves all samples for a given accession
func runGetSamples(cmd *cobra.Command, args []string) error {
	accession := strings.ToUpper(args[0])

	// Resolve database path
	dbPath := serverDBPath
	if dbPath == "" {
		dbPath = paths.GetDatabasePath()
	}

	// Initialize database
	db, err := database.Initialize(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Determine query based on accession type
	var query string
	switch {
	case strings.HasPrefix(accession, "SRP"):
		// Get samples for a study
		query = `
			SELECT DISTINCT s.sample_accession, s.organism, s.scientific_name,
			       s.taxon_id, s.description
			FROM samples s
			JOIN experiment_samples es ON s.sample_accession = es.sample_accession
			JOIN experiments e ON es.experiment_accession = e.experiment_accession
			WHERE e.study_accession = ?`

	case strings.HasPrefix(accession, "SRX"):
		// Get samples for an experiment
		query = `
			SELECT s.sample_accession, s.organism, s.scientific_name,
			       s.taxon_id, s.description
			FROM samples s
			JOIN experiment_samples es ON s.sample_accession = es.sample_accession
			WHERE es.experiment_accession = ?`

	default:
		return fmt.Errorf("unsupported accession type: %s", accession)
	}

	// Add limit if specified
	if relLimit > 0 {
		query += fmt.Sprintf(" LIMIT %d", relLimit)
	}

	// Execute query
	rows, err := db.GetSQLDB().Query(query, accession)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Collect results
	type SampleInfo struct {
		SampleAccession string `json:"sample_accession" yaml:"sample_accession"`
		Organism        string `json:"organism,omitempty" yaml:"organism,omitempty"`
		ScientificName  string `json:"scientific_name,omitempty" yaml:"scientific_name,omitempty"`
		TaxonID         int    `json:"taxon_id,omitempty" yaml:"taxon_id,omitempty"`
		Description     string `json:"description,omitempty" yaml:"description,omitempty"`
	}

	samples := []SampleInfo{}
	for rows.Next() {
		var sample SampleInfo
		var organism, sciName, desc *string
		var taxonID *int

		err := rows.Scan(&sample.SampleAccession, &organism, &sciName, &taxonID, &desc)
		if err != nil {
			continue
		}

		if organism != nil {
			sample.Organism = *organism
		}
		if sciName != nil {
			sample.ScientificName = *sciName
		}
		if taxonID != nil {
			sample.TaxonID = *taxonID
		}
		if desc != nil {
			sample.Description = *desc
		}

		samples = append(samples, sample)
	}

	// Output results
	return outputRelationshipResults(samples, "samples", relDetailed)
}

// runGetExperiments retrieves all experiments for a given accession
func runGetExperiments(cmd *cobra.Command, args []string) error {
	accession := strings.ToUpper(args[0])

	// Resolve database path
	dbPath := serverDBPath
	if dbPath == "" {
		dbPath = paths.GetDatabasePath()
	}

	// Initialize database
	db, err := database.Initialize(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Determine query based on accession type
	var query string
	switch {
	case strings.HasPrefix(accession, "SRP"):
		// Get experiments for a study
		query = `
			SELECT experiment_accession, title, library_strategy, library_source,
			       platform_name, instrument_model
			FROM experiments
			WHERE study_accession = ?`

	case strings.HasPrefix(accession, "SRS"):
		// Get experiments for a sample
		query = `
			SELECT e.experiment_accession, e.title, e.library_strategy, e.library_source,
			       e.platform_name, e.instrument_model
			FROM experiments e
			JOIN experiment_samples es ON e.experiment_accession = es.experiment_accession
			WHERE es.sample_accession = ?`

	default:
		return fmt.Errorf("unsupported accession type: %s", accession)
	}

	// Add limit if specified
	if relLimit > 0 {
		query += fmt.Sprintf(" LIMIT %d", relLimit)
	}

	// Execute query
	rows, err := db.GetSQLDB().Query(query, accession)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Collect results
	type ExperimentInfo struct {
		ExperimentAccession string `json:"experiment_accession" yaml:"experiment_accession"`
		Title               string `json:"title,omitempty" yaml:"title,omitempty"`
		LibraryStrategy     string `json:"library_strategy,omitempty" yaml:"library_strategy,omitempty"`
		LibrarySource       string `json:"library_source,omitempty" yaml:"library_source,omitempty"`
		Platform            string `json:"platform,omitempty" yaml:"platform,omitempty"`
		InstrumentModel     string `json:"instrument_model,omitempty" yaml:"instrument_model,omitempty"`
	}

	experiments := []ExperimentInfo{}
	for rows.Next() {
		var exp ExperimentInfo
		var title, strategy, source, platform, model *string

		err := rows.Scan(&exp.ExperimentAccession, &title, &strategy, &source, &platform, &model)
		if err != nil {
			continue
		}

		if title != nil {
			exp.Title = *title
		}
		if strategy != nil {
			exp.LibraryStrategy = *strategy
		}
		if source != nil {
			exp.LibrarySource = *source
		}
		if platform != nil {
			exp.Platform = *platform
		}
		if model != nil {
			exp.InstrumentModel = *model
		}

		experiments = append(experiments, exp)
	}

	// Output results
	return outputRelationshipResults(experiments, "experiments", relDetailed)
}

// runGetStudies retrieves study information for any accession
func runGetStudies(cmd *cobra.Command, args []string) error {
	accession := strings.ToUpper(args[0])

	// Resolve database path
	dbPath := serverDBPath
	if dbPath == "" {
		dbPath = paths.GetDatabasePath()
	}

	// Initialize database
	db, err := database.Initialize(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Determine query based on accession type
	var query string
	switch {
	case strings.HasPrefix(accession, "SRP"):
		// Direct study query
		query = `
			SELECT study_accession, study_title, study_abstract, study_type, organism
			FROM studies
			WHERE study_accession = ?`

	case strings.HasPrefix(accession, "SRX"):
		// Get study from experiment
		query = `
			SELECT s.study_accession, s.study_title, s.study_abstract, s.study_type, s.organism
			FROM studies s
			JOIN experiments e ON s.study_accession = e.study_accession
			WHERE e.experiment_accession = ?`

	case strings.HasPrefix(accession, "SRR"):
		// Get study from run
		query = `
			SELECT s.study_accession, s.study_title, s.study_abstract, s.study_type, s.organism
			FROM studies s
			JOIN experiments e ON s.study_accession = e.study_accession
			JOIN runs r ON e.experiment_accession = r.experiment_accession
			WHERE r.run_accession = ?`

	case strings.HasPrefix(accession, "SRS"):
		// Get study from sample
		query = `
			SELECT DISTINCT s.study_accession, s.study_title, s.study_abstract, s.study_type, s.organism
			FROM studies s
			JOIN experiments e ON s.study_accession = e.study_accession
			JOIN experiment_samples es ON e.experiment_accession = es.experiment_accession
			WHERE es.sample_accession = ?`

	default:
		return fmt.Errorf("unsupported accession type: %s", accession)
	}

	// Execute query
	rows, err := db.GetSQLDB().Query(query, accession)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Collect results
	type StudyInfo struct {
		StudyAccession string `json:"study_accession" yaml:"study_accession"`
		Title          string `json:"title,omitempty" yaml:"title,omitempty"`
		Abstract       string `json:"abstract,omitempty" yaml:"abstract,omitempty"`
		Type           string `json:"type,omitempty" yaml:"type,omitempty"`
		Organism       string `json:"organism,omitempty" yaml:"organism,omitempty"`
	}

	studies := []StudyInfo{}
	for rows.Next() {
		var study StudyInfo
		var title, abstract, studyType, organism *string

		err := rows.Scan(&study.StudyAccession, &title, &abstract, &studyType, &organism)
		if err != nil {
			continue
		}

		if title != nil {
			study.Title = *title
		}
		if abstract != nil && relDetailed {
			study.Abstract = *abstract
		}
		if studyType != nil {
			study.Type = *studyType
		}
		if organism != nil {
			study.Organism = *organism
		}

		studies = append(studies, study)
	}

	// Output results
	return outputRelationshipResults(studies, "studies", relDetailed)
}

// outputRelationshipResults handles output formatting for relationship queries
func outputRelationshipResults(data interface{}, dataType string, detailed bool) error {
	switch relFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if relOutput != "" {
			file, err := os.Create(relOutput)
			if err != nil {
				return err
			}
			defer file.Close()
			encoder = json.NewEncoder(file)
			encoder.SetIndent("", "  ")
		}
		return encoder.Encode(data)

	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		if relOutput != "" {
			file, err := os.Create(relOutput)
			if err != nil {
				return err
			}
			defer file.Close()
			encoder = yaml.NewEncoder(file)
		}
		return encoder.Encode(data)

	case "csv", "tsv":
		sep := ","
		if relFormat == "tsv" {
			sep = "\t"
		}
		return outputRelationshipCSV(data, dataType, sep)

	default: // table format
		return outputRelationshipTable(data, dataType, detailed)
	}
}

// outputRelationshipTable outputs results in table format
func outputRelationshipTable(data interface{}, dataType string, detailed bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	switch dataType {
	case "runs":
		runs := data.([]RunInfo)

		if len(runs) == 0 {
			fmt.Println("No runs found")
			return nil
		}

		// Header
		if detailed {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				colorize(colorBold, "RUN"),
				colorize(colorBold, "EXPERIMENT"),
				colorize(colorBold, "SPOTS"),
				colorize(colorBold, "BASES"),
				colorize(colorBold, "PLATFORM"),
				colorize(colorBold, "STRATEGY"))
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				colorize(colorBold, "RUN"),
				colorize(colorBold, "SPOTS"),
				colorize(colorBold, "BASES"))
		}

		// Data
		for _, run := range runs {
			if detailed {
				fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\n",
					colorize(colorCyan, run.RunAccession),
					run.Experiment,
					run.TotalSpots,
					run.TotalBases,
					run.Platform,
					run.Strategy)
			} else {
				fmt.Fprintf(w, "%s\t%d\t%d\n",
					colorize(colorCyan, run.RunAccession),
					run.TotalSpots,
					run.TotalBases)
			}
		}

		w.Flush()
		fmt.Printf("\n%s\n", colorize(colorGray, fmt.Sprintf("Total: %d runs", len(runs))))

	// Add similar cases for samples, experiments, studies...
	}

	return nil
}

// outputRelationshipCSV outputs results in CSV/TSV format
func outputRelationshipCSV(data interface{}, dataType string, separator string) error {
	// Implementation for CSV/TSV output
	// This would serialize the data structures to CSV format
	return nil
}