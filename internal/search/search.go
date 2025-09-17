package search

import (
	"database/sql"
	"fmt"
	"github.com/nishad/srake/internal/models"
	"sort"
	"strings"
)

// SearchEngine provides efficient text search without FTS
type SearchEngine struct {
	db *sql.DB
}

// New creates a new search engine
func New(db *sql.DB) *SearchEngine {
	return &SearchEngine{db: db}
}

// SearchResult represents a unified search result
type SearchResult struct {
	Type        string   `json:"type"` // study, experiment, run, sample
	Accession   string   `json:"accession"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Score       float64  `json:"score"`      // relevance score
	Highlights  []string `json:"highlights"` // matched snippets
}

// Search performs a multi-table search
func (se *SearchEngine) Search(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 100
	}

	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, fmt.Errorf("empty search query")
	}

	var results []SearchResult

	// Search studies
	studies, err := se.searchStudies(query, limit)
	if err == nil {
		results = append(results, studies...)
	}

	// Search experiments
	experiments, err := se.searchExperiments(query, limit)
	if err == nil {
		results = append(results, experiments...)
	}

	// Search samples
	samples, err := se.searchSamples(query, limit)
	if err == nil {
		results = append(results, samples...)
	}

	// Search runs
	runs, err := se.searchRuns(query, limit)
	if err == nil {
		results = append(results, runs...)
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (se *SearchEngine) searchStudies(query string, limit int) ([]SearchResult, error) {
	sql := `
		SELECT study_accession, study_title, study_abstract, study_type
		FROM study
		WHERE LOWER(study_accession) LIKE ?
		   OR LOWER(study_title) LIKE ?
		   OR LOWER(study_abstract) LIKE ?
		   OR LOWER(study_type) LIKE ?
		LIMIT ?
	`

	pattern := "%" + query + "%"
	rows, err := se.db.Query(sql, pattern, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var accession, title, abstract, studyType sql.NullString
		if err := rows.Scan(&accession, &title, &abstract, &studyType); err != nil {
			continue
		}

		result := SearchResult{
			Type:        "study",
			Accession:   accession.String,
			Title:       title.String,
			Description: abstract.String,
			Score:       se.calculateScore(query, accession.String, title.String, abstract.String),
		}
		result.Highlights = se.getHighlights(query, title.String, abstract.String)
		results = append(results, result)
	}

	return results, nil
}

func (se *SearchEngine) searchExperiments(query string, limit int) ([]SearchResult, error) {
	sql := `
		SELECT experiment_accession, title, library_strategy, platform
		FROM experiment
		WHERE LOWER(experiment_accession) LIKE ?
		   OR LOWER(title) LIKE ?
		   OR LOWER(library_strategy) LIKE ?
		   OR LOWER(platform) LIKE ?
		LIMIT ?
	`

	pattern := "%" + query + "%"
	rows, err := se.db.Query(sql, pattern, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var accession, title, strategy, platform sql.NullString
		if err := rows.Scan(&accession, &title, &strategy, &platform); err != nil {
			continue
		}

		desc := fmt.Sprintf("%s | %s", strategy.String, platform.String)
		result := SearchResult{
			Type:        "experiment",
			Accession:   accession.String,
			Title:       title.String,
			Description: desc,
			Score:       se.calculateScore(query, accession.String, title.String, desc),
		}
		result.Highlights = se.getHighlights(query, title.String, desc)
		results = append(results, result)
	}

	return results, nil
}

func (se *SearchEngine) searchSamples(query string, limit int) ([]SearchResult, error) {
	sql := `
		SELECT sample_accession, scientific_name, common_name, description
		FROM sample
		WHERE LOWER(sample_accession) LIKE ?
		   OR LOWER(scientific_name) LIKE ?
		   OR LOWER(common_name) LIKE ?
		   OR LOWER(description) LIKE ?
		LIMIT ?
	`

	pattern := "%" + query + "%"
	rows, err := se.db.Query(sql, pattern, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var accession, scientific, common, desc sql.NullString
		if err := rows.Scan(&accession, &scientific, &common, &desc); err != nil {
			continue
		}

		title := scientific.String
		if common.String != "" {
			title = fmt.Sprintf("%s (%s)", scientific.String, common.String)
		}

		result := SearchResult{
			Type:        "sample",
			Accession:   accession.String,
			Title:       title,
			Description: desc.String,
			Score:       se.calculateScore(query, accession.String, title, desc.String),
		}
		result.Highlights = se.getHighlights(query, title, desc.String)
		results = append(results, result)
	}

	return results, nil
}

func (se *SearchEngine) searchRuns(query string, limit int) ([]SearchResult, error) {
	sql := `
		SELECT run_accession, experiment_accession, spots, bases
		FROM run
		WHERE LOWER(run_accession) LIKE ?
		   OR LOWER(experiment_accession) LIKE ?
		LIMIT ?
	`

	pattern := "%" + query + "%"
	rows, err := se.db.Query(sql, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var accession, expAccession sql.NullString
		var spots, bases sql.NullFloat64
		if err := rows.Scan(&accession, &expAccession, &spots, &bases); err != nil {
			continue
		}

		desc := fmt.Sprintf("Experiment: %s | Spots: %.0f | Bases: %.0f",
			expAccession.String, spots.Float64, bases.Float64)

		result := SearchResult{
			Type:        "run",
			Accession:   accession.String,
			Title:       accession.String,
			Description: desc,
			Score:       se.calculateScore(query, accession.String, "", desc),
		}
		result.Highlights = se.getHighlights(query, accession.String, desc)
		results = append(results, result)
	}

	return results, nil
}

// calculateScore calculates relevance score based on match quality
func (se *SearchEngine) calculateScore(query string, fields ...string) float64 {
	query = strings.ToLower(query)
	score := 0.0

	for i, field := range fields {
		if field == "" {
			continue
		}

		field = strings.ToLower(field)
		weight := 1.0 / float64(i+1) // Earlier fields get higher weight

		// Exact match
		if field == query {
			score += 10.0 * weight
		} else if strings.HasPrefix(field, query) {
			score += 5.0 * weight
		} else if strings.Contains(field, query) {
			score += 2.0 * weight
		}

		// Word-level matching
		queryWords := strings.Fields(query)
		fieldWords := strings.Fields(field)
		for _, qw := range queryWords {
			for _, fw := range fieldWords {
				if fw == qw {
					score += 1.0 * weight
				} else if strings.HasPrefix(fw, qw) {
					score += 0.5 * weight
				}
			}
		}
	}

	return score
}

// getHighlights extracts matching snippets
func (se *SearchEngine) getHighlights(query string, fields ...string) []string {
	var highlights []string
	query = strings.ToLower(query)

	for _, field := range fields {
		if field == "" {
			continue
		}

		lower := strings.ToLower(field)
		index := strings.Index(lower, query)
		if index >= 0 {
			// Extract context around match
			start := index - 30
			if start < 0 {
				start = 0
			}
			end := index + len(query) + 30
			if end > len(field) {
				end = len(field)
			}

			snippet := field[start:end]
			if start > 0 {
				snippet = "..." + snippet
			}
			if end < len(field) {
				snippet = snippet + "..."
			}

			highlights = append(highlights, snippet)
			if len(highlights) >= 3 {
				break
			}
		}
	}

	return highlights
}

// SearchByOrganism searches specifically for organisms
func (se *SearchEngine) SearchByOrganism(organism string, limit int) ([]models.Sample, error) {
	query := `
		SELECT sample_accession, scientific_name, taxon_id, description
		FROM sample
		WHERE LOWER(scientific_name) LIKE ?
		   OR LOWER(common_name) LIKE ?
		LIMIT ?
	`

	pattern := "%" + strings.ToLower(organism) + "%"
	rows, err := se.db.Query(query, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []models.Sample
	for rows.Next() {
		var s models.Sample
		var desc sql.NullString
		var taxonID sql.NullInt64

		if err := rows.Scan(&s.Accession, &s.ScientificName, &taxonID, &desc); err != nil {
			continue
		}

		if taxonID.Valid {
			s.TaxonID = int(taxonID.Int64)
		}
		s.Description = desc.String
		samples = append(samples, s)
	}

	return samples, nil
}

// SearchByLibraryStrategy searches by library strategy
func (se *SearchEngine) SearchByLibraryStrategy(strategy string, limit int) ([]models.Experiment, error) {
	query := `
		SELECT experiment_accession, title, study_accession, sample_accession,
		       library_strategy, library_source, library_selection, library_layout,
		       platform, instrument_model
		FROM experiment
		WHERE library_strategy = ?
		LIMIT ?
	`

	rows, err := se.db.Query(query, strategy, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var experiments []models.Experiment
	for rows.Next() {
		var e models.Experiment
		var title, studyAcc, sampleAcc, source, selection, layout, platform, model sql.NullString

		if err := rows.Scan(&e.Accession, &title, &studyAcc, &sampleAcc,
			&e.LibraryStrategy, &source, &selection, &layout, &platform, &model); err != nil {
			continue
		}

		e.Title = title.String
		e.StudyAccession = studyAcc.String
		e.SampleAccession = sampleAcc.String
		e.LibrarySource = source.String
		e.LibrarySelection = selection.String
		e.LibraryLayout = layout.String
		e.Platform = platform.String
		e.InstrumentModel = model.String

		experiments = append(experiments, e)
	}

	return experiments, nil
}
