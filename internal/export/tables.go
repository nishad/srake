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
	// Count total studies
	var count int
	err := e.sourceDB.DB.QueryRow(`SELECT COUNT(*) FROM studies`).Scan(&count)
	if err != nil {
		return err
	}

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
			sample_accession, alias, center_name, broker_name,
			title, description, taxon_id, scientific_name,
			common_name, organism, sample_attributes
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
			alias          sql.NullString
			centerName     sql.NullString
			brokerName     sql.NullString
			title          sql.NullString
			description    sql.NullString
			taxonID        sql.NullInt64
			scientificName sql.NullString
			commonName     sql.NullString
			organism       sql.NullString
			attributes     sql.NullString
		}

		err := rows.Scan(
			&s.accession, &s.alias, &s.centerName, &s.brokerName,
			&s.title, &s.description, &s.taxonID, &s.scientificName,
			&s.commonName, &s.organism, &s.attributes,
		)
		if err != nil {
			return err
		}

		attributesStr := jsonToDelimited(s.attributes.String, "|")
		sraLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/%s", s.accession)
		entrezLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/?term=%s", s.accession)

		_, err = txStmt.Exec(
			sampleID,
			s.alias.String,
			s.accession,
			s.brokerName.String,
			s.centerName.String,
			s.taxonID.Int64,
			s.scientificName.String,
			s.commonName.String,
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
			exp.experiment_accession, e.alias, e.center_name, e.broker_name,
			exp.study_accession, e.sample_accession, e.title, e.design_description,
			exp.library_name, e.library_strategy, e.library_source, e.library_selection,
			exp.library_layout, e.library_construction_protocol, e.platform,
			exp.instrument_model, e.spot_length, e.experiment_attributes,
			s.alias as study_alias, sa.alias as sample_alias
		FROM experiments e
		LEFT JOIN studies s ON e.study_accession = s.study_accession
		LEFT JOIN samples sa ON e.sample_accession = sa.sample_accession
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
			accession        string
			alias            sql.NullString
			centerName       sql.NullString
			brokerName       sql.NullString
			studyAccession   sql.NullString
			sampleAccession  sql.NullString
			title            sql.NullString
			designDesc       sql.NullString
			libraryName      sql.NullString
			libraryStrategy  sql.NullString
			librarySource    sql.NullString
			librarySelection sql.NullString
			libraryLayout    sql.NullString
			libraryProtocol  sql.NullString
			platform         sql.NullString
			instrumentModel  sql.NullString
			spotLength       sql.NullInt64
			attributes       sql.NullString
			studyAlias       sql.NullString
			sampleAlias      sql.NullString
		}

		err := rows.Scan(
			&exp.accession, &exp.alias, &exp.centerName, &exp.brokerName,
			&exp.studyAccession, &exp.sampleAccession, &exp.title, &exp.designDesc,
			&exp.libraryName, &exp.libraryStrategy, &exp.librarySource, &exp.librarySelection,
			&exp.libraryLayout, &exp.libraryProtocol, &exp.platform,
			&exp.instrumentModel, &exp.spotLength, &exp.attributes,
			&exp.studyAlias, &exp.sampleAlias,
		)
		if err != nil {
			return err
		}

		attributesStr := jsonToDelimited(exp.attributes.String, "|")
		sraLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/%s", exp.accession)
		entrezLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/?term=%s", exp.accession)

		_, err = txStmt.Exec(
			expID,
			"", // bamFile
			"", // fastqFTP
			exp.alias.String,
			exp.accession,
			exp.brokerName.String,
			exp.centerName.String,
			exp.title.String,
			exp.studyAlias.String,
			exp.studyAccession.String,
			exp.designDesc.String,
			exp.sampleAlias.String,
			exp.sampleAccession.String,
			"", // sample_member
			exp.libraryName.String,
			exp.libraryStrategy.String,
			exp.librarySource.String,
			exp.librarySelection.String,
			exp.libraryLayout.String,
			"", // targeted_loci
			exp.libraryProtocol.String,
			exp.spotLength.Int64,
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
			r.run_accession, r.alias, r.center_name, r.broker_name,
			r.run_center, r.experiment_accession, r.run_date,
			r.total_spots, r.total_bases, r.run_attributes,
			exp.alias as exp_alias
		FROM runs r
		LEFT JOIN experiments e ON r.experiment_accession = e.experiment_accession
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
			alias        sql.NullString
			centerName   sql.NullString
			brokerName   sql.NullString
			runCenter    sql.NullString
			expAccession sql.NullString
			runDate      sql.NullTime
			totalSpots   sql.NullInt64
			totalBases   sql.NullInt64
			attributes   sql.NullString
			expAlias     sql.NullString
		}

		err := rows.Scan(
			&r.accession, &r.alias, &r.centerName, &r.brokerName,
			&r.runCenter, &r.expAccession, &r.runDate,
			&r.totalSpots, &r.totalBases, &r.attributes,
			&r.expAlias,
		)
		if err != nil {
			return err
		}

		attributesStr := jsonToDelimited(r.attributes.String, "|")
		sraLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/%s", r.accession)
		entrezLink := fmt.Sprintf("https://www.ncbi.nlm.nih.gov/sra/?term=%s", r.accession)

		runDateStr := ""
		if r.runDate.Valid {
			runDateStr = r.runDate.Time.Format("2006-01-02 15:04:05")
		}

		_, err = txStmt.Exec(
			runID,
			"", // bamFile
			r.alias.String,
			r.accession,
			r.brokerName.String,
			"", // instrument_name
			runDateStr,
			"", // run_file
			r.runCenter.String,
			0, // total_data_blocks
			r.expAccession.String,
			r.expAlias.String,
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