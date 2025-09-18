package export

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// exportSubmissions exports submission records
func (e *Exporter) exportSubmissions() error {
	// For now, create a default submission record since srake doesn't have submission table
	stmt, err := e.targetDB.Prepare(`INSERT INTO submission
		(submission_ID, submission_accession, center_name, submission_date, sradb_updated)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Insert a default submission
	_, err = stmt.Exec(
		1,
		"SRA000001",
		"NCBI",
		time.Now().Format("2006-01-02 15:04:05"),
		time.Now().Format("2006-01-02 15:04:05"),
	)

	e.stats.Submissions = 1
	return err
}

// exportStudies exports study records
func (e *Exporter) exportStudies() error {
	// Skip counting for performance - large databases may have millions of records
	// The count was only used for progress tracking which is handled elsewhere

	// Prepare insert statement
	stmt, err := e.targetDB.Prepare(`INSERT INTO study (
		study_ID, study_alias, study_accession, study_title, study_type,
		study_abstract, broker_name, center_name, center_project_name,
		study_description, related_studies, primary_study, sra_link,
		study_url_link, xref_link, study_entrez_link, ddbj_link, ena_link,
		study_attribute, submission_accession, sradb_updated
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Query source data
	rows, err := e.sourceDB.DB.Query(`
		SELECT
			study_accession, study_title, study_type, study_abstract,
			organism, submission_date, metadata
		FROM studies
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Begin transaction for performance
	tx, err := e.targetDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txStmt := tx.Stmt(stmt)
	studyID := 1

	for rows.Next() {
		var s struct {
			accession      string
			title          sql.NullString
			studyType      sql.NullString
			abstract       sql.NullString
			organism       sql.NullString
			submissionDate sql.NullTime
			metadata       sql.NullString
		}

		err := rows.Scan(
			&s.accession, &s.title, &s.studyType, &s.abstract,
			&s.organism, &s.submissionDate, &s.metadata,
		)
		if err != nil {
			return err
		}

		// Extract metadata fields if needed
		// For now, we'll use simplified mapping

		// Generate URLs
		sraLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/%s", s.accession)
		entrezLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/?term=%s", s.accession)

		_, err = txStmt.Exec(
			studyID,
			"", // alias
			s.accession,
			s.title.String,
			s.studyType.String,
			s.abstract.String,
			"", // broker_name
			"", // center_name
			"", // center_project_name
			s.abstract.String, // use abstract as description
			"", // related_studies
			"", // primary_study
			sraLink,
			"", // study_url_link
			"", // xref_link
			entrezLink,
			"", // ddbj_link
			"", // ena_link
			"", // attributes
			"SRA000001", // submission_accession
			time.Now().Format("2006-01-02 15:04:05"),
		)
		if err != nil {
			return err
		}

		studyID++
		e.stats.Studies++
	}

	return tx.Commit()
}

// exportSamples exports sample records
func (e *Exporter) exportSamples() error {
	stmt, err := e.targetDB.Prepare(`INSERT INTO sample (
		sample_ID, sample_alias, sample_accession, broker_name, center_name,
		taxon_id, scientific_name, common_name, anonymized_name, individual_name,
		description, sra_link, sample_url_link, xref_link, sample_entrez_link,
		ddbj_link, ena_link, sample_attribute, submission_accession, sradb_updated
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := e.sourceDB.DB.Query(`
		SELECT
			sample_accession, description, taxon_id, scientific_name,
			organism, tissue, cell_type, metadata
		FROM samples
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := e.targetDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txStmt := tx.Stmt(stmt)
	sampleID := 1

	for rows.Next() {
		var s struct {
			accession      string
			description    sql.NullString
			taxonID        sql.NullInt64
			scientificName sql.NullString
			organism       sql.NullString
			tissue         sql.NullString
			cellType       sql.NullString
			metadata       sql.NullString
		}

		err := rows.Scan(
			&s.accession, &s.description, &s.taxonID, &s.scientificName,
			&s.organism, &s.tissue, &s.cellType, &s.metadata,
		)
		if err != nil {
			return err
		}

		// Build attributes from tissue and cell_type if available
		var attributes []string
		if s.tissue.Valid && s.tissue.String != "" {
			attributes = append(attributes, fmt.Sprintf("tissue=%s", s.tissue.String))
		}
		if s.cellType.Valid && s.cellType.String != "" {
			attributes = append(attributes, fmt.Sprintf("cell_type=%s", s.cellType.String))
		}
		attributesStr := strings.Join(attributes, "|")

		sraLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/%s", s.accession)
		entrezLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/?term=%s", s.accession)

		_, err = txStmt.Exec(
			sampleID,
			"", // alias - not in simplified schema
			s.accession,
			"", // broker_name - not in simplified schema
			"", // center_name - not in simplified schema
			s.taxonID.Int64,
			s.scientificName.String,
			"", // common_name - not in simplified schema
			"", // anonymized_name
			"", // individual_name
			s.description.String,
			sraLink,
			"", // sample_url_link
			"", // xref_link
			entrezLink,
			"", // ddbj_link
			"", // ena_link
			attributesStr,
			"SRA000001",
			time.Now().Format("2006-01-02 15:04:05"),
		)
		if err != nil {
			return err
		}

		sampleID++
		e.stats.Samples++
	}

	return tx.Commit()
}

// exportExperiments exports experiment records
func (e *Exporter) exportExperiments() error {
	stmt, err := e.targetDB.Prepare(`INSERT INTO experiment (
		experiment_ID, bamFile, fastqFTP, experiment_alias, experiment_accession,
		broker_name, center_name, title, study_name, study_accession,
		design_description, sample_name, sample_accession, sample_member,
		library_name, library_strategy, library_source, library_selection,
		library_layout, targeted_loci, library_construction_protocol,
		spot_length, adapter_spec, read_spec, platform, instrument_model,
		platform_parameters, sequence_space, base_caller, quality_scorer,
		number_of_levels, multiplier, qtype, sra_link, experiment_url_link,
		xref_link, experiment_entrez_link, ddbj_link, ena_link,
		experiment_attribute, submission_accession, sradb_updated
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := e.sourceDB.DB.Query(`
		SELECT
			e.experiment_accession, e.study_accession, e.title,
			e.library_strategy, e.library_source, e.platform,
			e.instrument_model, e.metadata,
			es.sample_accession
		FROM experiments e
		LEFT JOIN experiment_samples es ON e.experiment_accession = es.experiment_accession
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := e.targetDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txStmt := tx.Stmt(stmt)
	expID := 1

	for rows.Next() {
		var exp struct {
			accession       string
			studyAccession  sql.NullString
			title           sql.NullString
			libraryStrategy sql.NullString
			librarySource   sql.NullString
			platform        sql.NullString
			instrumentModel sql.NullString
			metadata        sql.NullString
			sampleAccession sql.NullString
		}

		err := rows.Scan(
			&exp.accession, &exp.studyAccession, &exp.title,
			&exp.libraryStrategy, &exp.librarySource, &exp.platform,
			&exp.instrumentModel, &exp.metadata, &exp.sampleAccession,
		)
		if err != nil {
			return err
		}

		// Extract any additional info from metadata if available
		attributesStr := ""
		if exp.metadata.Valid {
			attributesStr = jsonToDelimited(exp.metadata.String, "|")
		}

		sraLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/%s", exp.accession)
		entrezLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/?term=%s", exp.accession)

		_, err = txStmt.Exec(
			expID,
			"", // bamFile
			"", // fastqFTP
			"", // alias - not in simplified schema
			exp.accession,
			"", // broker_name - not in simplified schema
			"", // center_name - not in simplified schema
			exp.title.String,
			"", // study_name/alias - not in simplified schema
			exp.studyAccession.String,
			"", // design_description - not in simplified schema
			"", // sample_name/alias - not in simplified schema
			exp.sampleAccession.String,
			"", // sample_member
			"", // library_name - not in simplified schema
			exp.libraryStrategy.String,
			exp.librarySource.String,
			"", // library_selection - not in simplified schema
			"", // library_layout - not in simplified schema
			"", // targeted_loci
			"", // library_construction_protocol - not in simplified schema
			0,  // spot_length - not in simplified schema
			"", // adapter_spec
			"", // read_spec
			exp.platform.String,
			exp.instrumentModel.String,
			"", // platform_parameters
			"", // sequence_space
			"", // base_caller
			"", // quality_scorer
			0,  // number_of_levels
			"", // multiplier
			"", // qtype
			sraLink,
			"", // experiment_url_link
			"", // xref_link
			entrezLink,
			"", // ddbj_link
			"", // ena_link
			attributesStr,
			"SRA000001",
			time.Now().Format("2006-01-02 15:04:05"),
		)
		if err != nil {
			return err
		}

		expID++
		e.stats.Experiments++
	}

	return tx.Commit()
}

// exportRuns exports run records
func (e *Exporter) exportRuns() error {
	stmt, err := e.targetDB.Prepare(`INSERT INTO run (
		run_ID, bamFile, run_alias, run_accession, broker_name,
		instrument_name, run_date, run_file, run_center, total_data_blocks,
		experiment_accession, experiment_name, sra_link, run_url_link,
		xref_link, run_entrez_link, ddbj_link, ena_link, run_attribute,
		submission_accession, sradb_updated
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := e.sourceDB.DB.Query(`
		SELECT
			r.run_accession, r.experiment_accession,
			r.total_spots, r.total_bases, r.published, r.metadata
		FROM runs r
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := e.targetDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txStmt := tx.Stmt(stmt)
	runID := 1

	for rows.Next() {
		var r struct {
			accession    string
			expAccession sql.NullString
			totalSpots   sql.NullInt64
			totalBases   sql.NullInt64
			published    sql.NullTime
			metadata     sql.NullString
		}

		err := rows.Scan(
			&r.accession, &r.expAccession,
			&r.totalSpots, &r.totalBases, &r.published, &r.metadata,
		)
		if err != nil {
			return err
		}

		// Extract any additional info from metadata if available
		attributesStr := ""
		if r.metadata.Valid {
			attributesStr = jsonToDelimited(r.metadata.String, "|")
		}

		sraLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/%s", r.accession)
		entrezLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/?term=%s", r.accession)

		runDateStr := ""
		if r.published.Valid {
			runDateStr = r.published.Time.Format("2006-01-02 15:04:05")
		}

		_, err = txStmt.Exec(
			runID,
			"", // bamFile
			"", // alias - not in simplified schema
			r.accession,
			"", // broker_name - not in simplified schema
			"", // instrument_name
			runDateStr, // using published date as run_date
			"", // run_file
			"", // run_center - not in simplified schema
			0,  // total_data_blocks
			r.expAccession.String,
			"", // experiment_name/alias - not in simplified schema
			sraLink,
			"", // run_url_link
			"", // xref_link
			entrezLink,
			"", // ddbj_link
			"", // ena_link
			attributesStr,
			"SRA000001",
			time.Now().Format("2006-01-02 15:04:05"),
		)
		if err != nil {
			return err
		}

		runID++
		e.stats.Runs++
	}

	return tx.Commit()
}

// Helper functions for JSON conversion

func jsonToDelimited(jsonStr, delimiter string) string {
	if jsonStr == "" {
		return ""
	}

	var data []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}

	var result []string
	for _, item := range data {
		for k, v := range item {
			result = append(result, fmt.Sprintf("%s=%v", k, v))
		}
	}

	return strings.Join(result, delimiter)
}

func jsonArrayToDelimited(jsonStr, delimiter string) string {
	if jsonStr == "" {
		return ""
	}

	var data []string
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}

	return strings.Join(data, delimiter)
}