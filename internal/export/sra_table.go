package export

// createSRATable creates and populates the denormalized sra table
func (e *Exporter) createSRATable() error {
	// For now, skip populating the SRA table with simplified schema
	// The SRA table structure is created but left empty
	// This allows tools expecting the table to exist to work
	// Note: SRA table remains empty but structure is created for compatibility
	return nil

	// Original code below - kept for reference when full schema is available
	/*
		// This is a complex denormalized view that joins all tables
		// We'll populate it from our existing tables

		insertSQL := `INSERT INTO sra (
			sra_ID, SRR_bamFile, SRX_bamFile, SRX_fastqFTP,
			run_ID, run_alias, run_accession, run_date, updated_date,
			spots, bases, run_center,
			experiment_name, run_url_link, run_entrez_link, run_attribute,
			experiment_ID, experiment_alias, experiment_accession, experiment_title,
			study_name, sample_name, design_description,
			library_name, library_strategy, library_source, library_selection,
			library_layout, library_construction_protocol,
			adapter_spec, read_spec, platform, instrument_model, instrument_name,
			platform_parameters, sequence_space, base_caller, quality_scorer,
			number_of_levels, multiplier, qtype,
			experiment_url_link, experiment_entrez_link, experiment_attribute,
			sample_ID, sample_alias, sample_accession, taxon_id, common_name,
			anonymized_name, individual_name, description,
			sample_url_link, sample_entrez_link, sample_attribute,
			study_ID, study_alias, study_accession, study_title, study_type,
			study_abstract, center_project_name, study_description,
			study_url_link, study_entrez_link, study_attribute,
			related_studies, primary_study,
			submission_ID, submission_accession, submission_comment,
			submission_center, submission_lab, submission_date, sradb_updated
		)
		SELECT
			ROW_NUMBER() OVER (ORDER BY r.run_accession) as sra_ID,
			r.bamFile as SRR_bamFile,
			e.bamFile as SRX_bamFile,
			e.fastqFTP as SRX_fastqFTP,
			r.run_ID,
			r.run_alias,
			r.run_accession,
			r.run_date,
			r.sradb_updated as updated_date,
			CAST(NULL AS REAL) as spots,  -- Will update separately
			CAST(NULL AS REAL) as bases,  -- Will update separately
			r.run_center,
			r.experiment_name,
			r.run_url_link,
			r.run_entrez_link,
			r.run_attribute,
			e.experiment_ID,
			e.experiment_alias,
			e.experiment_accession,
			e.title as experiment_title,
			e.study_name,
			e.sample_name,
			e.design_description,
			e.library_name,
			e.library_strategy,
			e.library_source,
			e.library_selection,
			e.library_layout,
			e.library_construction_protocol,
			e.adapter_spec,
			e.read_spec,
			e.platform,
			e.instrument_model,
			r.instrument_name,
			e.platform_parameters,
			e.sequence_space,
			e.base_caller,
			e.quality_scorer,
			e.number_of_levels,
			e.multiplier,
			e.qtype,
			e.experiment_url_link,
			e.experiment_entrez_link,
			e.experiment_attribute,
			sa.sample_ID,
			sa.sample_alias,
			sa.sample_accession,
			sa.taxon_id,
			sa.common_name,
			sa.anonymized_name,
			sa.individual_name,
			sa.description,
			sa.sample_url_link,
			sa.sample_entrez_link,
			sa.sample_attribute,
			st.study_ID,
			st.study_alias,
			st.study_accession,
			st.study_title,
			st.study_type,
			st.study_abstract,
			st.center_project_name,
			st.study_description,
			st.study_url_link,
			st.study_entrez_link,
			st.study_attribute,
			st.related_studies,
			st.primary_study,
			sub.submission_ID,
			sub.submission_accession,
			sub.submission_comment,
			sub.center_name as submission_center,
			sub.lab_name as submission_lab,
			sub.submission_date,
			r.sradb_updated
		FROM run r
		JOIN experiment e ON r.experiment_accession = e.experiment_accession
		JOIN sample sa ON e.sample_accession = sa.sample_accession
		JOIN study st ON e.study_accession = st.study_accession
		CROSS JOIN submission sub`

		result, err := e.targetDB.Exec(insertSQL)
		if err != nil {
			return fmt.Errorf("failed to populate sra table: %w", err)
		}

		count, _ := result.RowsAffected()
		e.stats.SRARecords = int(count)

		// Now update spots and bases from source database
		if err := e.updateSRAMetrics(); err != nil {
			return fmt.Errorf("failed to update SRA metrics: %w", err)
		}

		return nil
	*/
}

// updateSRAMetrics updates spots and bases in the sra table
func (e *Exporter) updateSRAMetrics() error {
	// Skip for simplified schema - SRA table is not populated
	return nil

	// Original code below - kept for reference
	/*
		// Query spots and bases from source database
		rows, err := e.sourceDB.DB.Query(`
			SELECT run_accession, total_spots, total_bases
			FROM runs
			WHERE total_spots IS NOT NULL OR total_bases IS NOT NULL
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		// Prepare update statement
		stmt, err := e.targetDB.Prepare(`
			UPDATE sra
			SET spots = ?, bases = ?
			WHERE run_accession = ?
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		// Begin transaction for performance
		tx, err := e.targetDB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		txStmt := tx.Stmt(stmt)

		for rows.Next() {
			var accession string
			var spots, bases sql.NullInt64

			if err := rows.Scan(&accession, &spots, &bases); err != nil {
				return err
			}

			spotsVal := float64(0)
			basesVal := float64(0)

			if spots.Valid {
				spotsVal = float64(spots.Int64)
			}
			if bases.Valid {
				basesVal = float64(bases.Int64)
			}

			if _, err := txStmt.Exec(spotsVal, basesVal, accession); err != nil {
				return err
			}
		}

		return tx.Commit()
	*/
}
