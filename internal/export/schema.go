package export

import (
	"fmt"
	"time"
)

// createSchema creates the SRAmetadb schema
func (e *Exporter) createSchema() error {
	schemas := []string{
		// metaInfo table
		`CREATE TABLE metaInfo (name varchar(50), value varchar(50))`,

		// submission table
		`CREATE TABLE submission (
			submission_ID REAL,
			submission_alias TEXT,
			submission_accession TEXT,
			submission_comment TEXT,
			files TEXT,
			broker_name TEXT,
			center_name TEXT,
			lab_name TEXT,
			submission_date TEXT,
			sra_link TEXT,
			submission_url_link TEXT,
			xref_link TEXT,
			submission_entrez_link TEXT,
			ddbj_link TEXT,
			ena_link TEXT,
			submission_attribute TEXT,
			sradb_updated TEXT
		)`,
		`CREATE INDEX submission_acc_idx ON submission (submission_accession)`,

		// study table
		`CREATE TABLE study (
			study_ID REAL,
			study_alias TEXT,
			study_accession TEXT,
			study_title TEXT,
			study_type TEXT,
			study_abstract TEXT,
			broker_name TEXT,
			center_name TEXT,
			center_project_name TEXT,
			study_description TEXT,
			related_studies TEXT,
			primary_study TEXT,
			sra_link TEXT,
			study_url_link TEXT,
			xref_link TEXT,
			study_entrez_link TEXT,
			ddbj_link TEXT,
			ena_link TEXT,
			study_attribute TEXT,
			submission_accession TEXT,
			sradb_updated TEXT
		)`,
		`CREATE INDEX study_acc_idx ON study (study_accession)`,

		// sample table
		`CREATE TABLE sample (
			sample_ID REAL,
			sample_alias TEXT,
			sample_accession TEXT,
			broker_name TEXT,
			center_name TEXT,
			taxon_id INTEGER,
			scientific_name TEXT,
			common_name TEXT,
			anonymized_name TEXT,
			individual_name TEXT,
			description TEXT,
			sra_link TEXT,
			sample_url_link TEXT,
			xref_link TEXT,
			sample_entrez_link TEXT,
			ddbj_link TEXT,
			ena_link TEXT,
			sample_attribute TEXT,
			submission_accession TEXT,
			sradb_updated TEXT
		)`,
		`CREATE INDEX sample_acc_idx ON sample (sample_accession)`,

		// experiment table
		`CREATE TABLE experiment (
			experiment_ID REAL,
			bamFile TEXT,
			fastqFTP TEXT,
			experiment_alias TEXT,
			experiment_accession TEXT,
			broker_name TEXT,
			center_name TEXT,
			title TEXT,
			study_name TEXT,
			study_accession TEXT,
			design_description TEXT,
			sample_name TEXT,
			sample_accession TEXT,
			sample_member TEXT,
			library_name TEXT,
			library_strategy TEXT,
			library_source TEXT,
			library_selection TEXT,
			library_layout TEXT,
			targeted_loci TEXT,
			library_construction_protocol TEXT,
			spot_length INTEGER,
			adapter_spec TEXT,
			read_spec TEXT,
			platform TEXT,
			instrument_model TEXT,
			platform_parameters TEXT,
			sequence_space TEXT,
			base_caller TEXT,
			quality_scorer TEXT,
			number_of_levels INTEGER,
			multiplier TEXT,
			qtype TEXT,
			sra_link TEXT,
			experiment_url_link TEXT,
			xref_link TEXT,
			experiment_entrez_link TEXT,
			ddbj_link TEXT,
			ena_link TEXT,
			experiment_attribute TEXT,
			submission_accession TEXT,
			sradb_updated TEXT
		)`,
		`CREATE INDEX experiment_acc_idx ON experiment (experiment_accession)`,

		// run table
		`CREATE TABLE run (
			run_ID REAL,
			bamFile TEXT,
			run_alias TEXT,
			run_accession TEXT,
			broker_name TEXT,
			instrument_name TEXT,
			run_date TEXT,
			run_file TEXT,
			run_center TEXT,
			total_data_blocks INTEGER,
			experiment_accession TEXT,
			experiment_name TEXT,
			sra_link TEXT,
			run_url_link TEXT,
			xref_link TEXT,
			run_entrez_link TEXT,
			ddbj_link TEXT,
			ena_link TEXT,
			run_attribute TEXT,
			submission_accession TEXT,
			sradb_updated TEXT
		)`,
		`CREATE INDEX run_acc_idx ON run (run_accession)`,

		// sra table (denormalized view)
		`CREATE TABLE sra (
			sra_ID REAL,
			SRR_bamFile TEXT,
			SRX_bamFile TEXT,
			SRX_fastqFTP TEXT,
			run_ID REAL,
			run_alias TEXT,
			run_accession TEXT,
			run_date TEXT,
			updated_date TEXT,
			spots REAL,
			bases REAL,
			run_center TEXT,
			experiment_name TEXT,
			run_url_link TEXT,
			run_entrez_link TEXT,
			run_attribute TEXT,
			experiment_ID REAL,
			experiment_alias TEXT,
			experiment_accession TEXT,
			experiment_title TEXT,
			study_name TEXT,
			sample_name TEXT,
			design_description TEXT,
			library_name TEXT,
			library_strategy TEXT,
			library_source TEXT,
			library_selection TEXT,
			library_layout TEXT,
			library_construction_protocol TEXT,
			adapter_spec TEXT,
			read_spec TEXT,
			platform TEXT,
			instrument_model TEXT,
			instrument_name TEXT,
			platform_parameters TEXT,
			sequence_space TEXT,
			base_caller TEXT,
			quality_scorer TEXT,
			number_of_levels INTEGER,
			multiplier TEXT,
			qtype TEXT,
			experiment_url_link TEXT,
			experiment_entrez_link TEXT,
			experiment_attribute TEXT,
			sample_ID REAL,
			sample_alias TEXT,
			sample_accession TEXT,
			taxon_id INTEGER,
			common_name TEXT,
			anonymized_name TEXT,
			individual_name TEXT,
			description TEXT,
			sample_url_link TEXT,
			sample_entrez_link TEXT,
			sample_attribute TEXT,
			study_ID REAL,
			study_alias TEXT,
			study_accession TEXT,
			study_title TEXT,
			study_type TEXT,
			study_abstract TEXT,
			center_project_name TEXT,
			study_description TEXT,
			study_url_link TEXT,
			study_entrez_link TEXT,
			study_attribute TEXT,
			related_studies TEXT,
			primary_study TEXT,
			submission_ID REAL,
			submission_accession TEXT,
			submission_comment TEXT,
			submission_center TEXT,
			submission_lab TEXT,
			submission_date TEXT,
			sradb_updated TEXT
		)`,

		// Indexes for sra table
		`CREATE INDEX sra_run_acc_idx ON sra (run_accession)`,
		`CREATE INDEX sra_experiment_acc_idx ON sra (experiment_accession)`,
		`CREATE INDEX sra_sample_acc_idx ON sra (sample_accession)`,
		`CREATE INDEX sra_study_acc_idx ON sra (study_accession)`,
		`CREATE INDEX sra_submission_acc_idx ON sra (submission_accession)`,

		// col_desc table
		`CREATE TABLE col_desc (
			col_desc_ID REAL,
			table_name TEXT,
			field_name TEXT,
			type TEXT,
			description TEXT,
			value_list TEXT,
			sradb_updated TEXT
		)`,
	}

	// Execute each schema statement
	for _, schema := range schemas {
		if _, err := e.targetDB.Exec(schema); err != nil {
			return fmt.Errorf("failed to execute schema: %w\nSQL: %s", err, schema)
		}
	}

	return nil
}

// createFTSIndex creates the full-text search index
func (e *Exporter) createFTSIndex() error {
	var ftsSQL string

	if e.cfg.FTSVersion == 3 {
		// FTS3 version for compatibility
		ftsSQL = `CREATE VIRTUAL TABLE sra_ft USING fts3 (
			SRR_bamFile,SRX_bamFile,SRX_fastqFTP,run_ID,run_alias,run_accession,run_date,
			updated_date,spots,bases,run_center,experiment_name,run_url_link,run_entrez_link,
			run_attribute,experiment_ID,experiment_alias,experiment_accession,experiment_title,
			study_name,sample_name,design_description,library_name,library_strategy,
			library_source,library_selection,library_layout,library_construction_protocol,
			adapter_spec,read_spec,platform,instrument_model,instrument_name,
			platform_parameters,sequence_space,base_caller,quality_scorer,number_of_levels,
			multiplier,qtype,experiment_url_link,experiment_entrez_link,experiment_attribute,
			sample_ID,sample_alias,sample_accession,taxon_id,common_name,anonymized_name,
			individual_name,description,sample_url_link,sample_entrez_link,sample_attribute,
			study_ID,study_alias,study_accession,study_title,study_type,study_abstract,
			center_project_name,study_description,study_url_link,study_entrez_link,
			study_attribute,related_studies,primary_study,submission_ID,submission_accession,
			submission_comment,submission_center,submission_lab,submission_date,sradb_updated
		)`
	} else {
		// FTS5 version (modern and efficient)
		ftsSQL = `CREATE VIRTUAL TABLE sra_ft USING fts5 (
			SRR_bamFile,SRX_bamFile,SRX_fastqFTP,run_ID,run_alias,run_accession,run_date,
			updated_date,spots,bases,run_center,experiment_name,run_url_link,run_entrez_link,
			run_attribute,experiment_ID,experiment_alias,experiment_accession,experiment_title,
			study_name,sample_name,design_description,library_name,library_strategy,
			library_source,library_selection,library_layout,library_construction_protocol,
			adapter_spec,read_spec,platform,instrument_model,instrument_name,
			platform_parameters,sequence_space,base_caller,quality_scorer,number_of_levels,
			multiplier,qtype,experiment_url_link,experiment_entrez_link,experiment_attribute,
			sample_ID,sample_alias,sample_accession,taxon_id,common_name,anonymized_name,
			individual_name,description,sample_url_link,sample_entrez_link,sample_attribute,
			study_ID,study_alias,study_accession,study_title,study_type,study_abstract,
			center_project_name,study_description,study_url_link,study_entrez_link,
			study_attribute,related_studies,primary_study,submission_ID,submission_accession,
			submission_comment,submission_center,submission_lab,submission_date,sradb_updated,
			tokenize='porter unicode61'
		)`
	}

	if _, err := e.targetDB.Exec(ftsSQL); err != nil {
		return fmt.Errorf("failed to create FTS index: %w", err)
	}

	// Populate FTS index from sra table
	insertSQL := `INSERT INTO sra_ft SELECT
		SRR_bamFile,SRX_bamFile,SRX_fastqFTP,run_ID,run_alias,run_accession,run_date,
		updated_date,spots,bases,run_center,experiment_name,run_url_link,run_entrez_link,
		run_attribute,experiment_ID,experiment_alias,experiment_accession,experiment_title,
		study_name,sample_name,design_description,library_name,library_strategy,
		library_source,library_selection,library_layout,library_construction_protocol,
		adapter_spec,read_spec,platform,instrument_model,instrument_name,
		platform_parameters,sequence_space,base_caller,quality_scorer,number_of_levels,
		multiplier,qtype,experiment_url_link,experiment_entrez_link,experiment_attribute,
		sample_ID,sample_alias,sample_accession,taxon_id,common_name,anonymized_name,
		individual_name,description,sample_url_link,sample_entrez_link,sample_attribute,
		study_ID,study_alias,study_accession,study_title,study_type,study_abstract,
		center_project_name,study_description,study_url_link,study_entrez_link,
		study_attribute,related_studies,primary_study,submission_ID,submission_accession,
		submission_comment,submission_center,submission_lab,submission_date,sradb_updated
		FROM sra`

	if _, err := e.targetDB.Exec(insertSQL); err != nil {
		return fmt.Errorf("failed to populate FTS index: %w", err)
	}

	// Optimize FTS index if FTS5
	if e.cfg.FTSVersion == 5 {
		if _, err := e.targetDB.Exec(`INSERT INTO sra_ft(sra_ft) VALUES('optimize')`); err != nil {
			// Non-fatal error
			if e.cfg.Debug {
				fmt.Printf("Warning: Failed to optimize FTS5 index: %v\n", err)
			}
		}
	}

	return nil
}

// createMetaInfo creates and populates the metaInfo table
func (e *Exporter) createMetaInfo() error {
	// Insert metadata
	metaData := []struct{ name, value string }{
		{"schema version", "1.1"},
		{"creation date", time.Now().Format("2006-01-02")},
		{"srake version", "1.0.0"},
		{"fts version", fmt.Sprintf("%d", e.cfg.FTSVersion)},
		{"source", "NCBI SRA"},
	}

	stmt, err := e.targetDB.Prepare(`INSERT INTO metaInfo (name, value) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, meta := range metaData {
		if _, err := stmt.Exec(meta.name, meta.value); err != nil {
			return err
		}
	}

	return nil
}

// createColDesc creates and populates the col_desc table
func (e *Exporter) createColDesc() error {
	// Sample column descriptions
	colDescs := []struct {
		table_name, field_name, type_, description string
	}{
		{"study", "study_accession", "TEXT", "SRA study accession"},
		{"study", "study_title", "TEXT", "Title of the study"},
		{"study", "study_abstract", "TEXT", "Abstract describing the study"},
		{"experiment", "experiment_accession", "TEXT", "SRA experiment accession"},
		{"experiment", "library_strategy", "TEXT", "Sequencing library strategy"},
		{"experiment", "platform", "TEXT", "Sequencing platform"},
		{"sample", "sample_accession", "TEXT", "SRA sample accession"},
		{"sample", "taxon_id", "INTEGER", "NCBI taxonomy ID"},
		{"sample", "scientific_name", "TEXT", "Scientific name of the organism"},
		{"run", "run_accession", "TEXT", "SRA run accession"},
		{"run", "spots", "REAL", "Number of spots"},
		{"run", "bases", "REAL", "Number of bases"},
	}

	stmt, err := e.targetDB.Prepare(`INSERT INTO col_desc
		(col_desc_ID, table_name, field_name, type, description, value_list, sradb_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, desc := range colDescs {
		if _, err := stmt.Exec(
			i+1,
			desc.table_name,
			desc.field_name,
			desc.type_,
			desc.description,
			"",
			time.Now().Format("2006-01-02 15:04:05"),
		); err != nil {
			return err
		}
	}

	return nil
}