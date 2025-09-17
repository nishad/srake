-- Full SRAmetadb-compatible schema (without FTS for efficiency)
-- This schema maintains compatibility with tools expecting original SRAmetadb structure
-- Full-text search is handled by Go application layer instead of SQLite FTS

-- Metadata table
CREATE TABLE IF NOT EXISTS metaInfo (
    name varchar(50),
    value varchar(50)
);

-- Submission table
CREATE TABLE IF NOT EXISTS submission (
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
);
CREATE INDEX IF NOT EXISTS submission_acc_idx ON submission (submission_accession);

-- Study table with all original fields
CREATE TABLE IF NOT EXISTS study (
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
);
CREATE INDEX IF NOT EXISTS study_acc_idx ON study (study_accession);

-- Sample table
CREATE TABLE IF NOT EXISTS sample (
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
);
CREATE INDEX IF NOT EXISTS sample_acc_idx ON sample (sample_accession);
CREATE INDEX IF NOT EXISTS idx_sample_organism ON sample(scientific_name);
CREATE INDEX IF NOT EXISTS idx_sample_taxon ON sample(taxon_id);

-- Experiment table
CREATE TABLE IF NOT EXISTS experiment (
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
);
CREATE INDEX IF NOT EXISTS experiment_acc_idx ON experiment (experiment_accession);
CREATE INDEX IF NOT EXISTS idx_experiment_study ON experiment(study_accession);
CREATE INDEX IF NOT EXISTS idx_experiment_sample ON experiment(sample_accession);
CREATE INDEX IF NOT EXISTS idx_experiment_strategy ON experiment(library_strategy);
CREATE INDEX IF NOT EXISTS idx_experiment_platform ON experiment(platform);

-- Run table
CREATE TABLE IF NOT EXISTS run (
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
    sradb_updated TEXT,
    spots REAL,
    bases REAL,
    spot_length INTEGER,
    published DATETIME
);
CREATE INDEX IF NOT EXISTS run_acc_idx ON run (run_accession);
CREATE INDEX IF NOT EXISTS idx_run_experiment ON run(experiment_accession);

-- Main denormalized sra table (kept for compatibility with legacy tools)
-- Consider making this a VIEW in future optimizations
CREATE TABLE IF NOT EXISTS sra (
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
);

-- Indexes for sra table
CREATE INDEX IF NOT EXISTS sra_run_acc_idx ON sra (run_accession);
CREATE INDEX IF NOT EXISTS sra_experiment_acc_idx ON sra (experiment_accession);
CREATE INDEX IF NOT EXISTS sra_sample_acc_idx ON sra (sample_accession);
CREATE INDEX IF NOT EXISTS sra_study_acc_idx ON sra (study_accession);
CREATE INDEX IF NOT EXISTS sra_submission_acc_idx ON sra (submission_accession);

-- Column descriptions table
CREATE TABLE IF NOT EXISTS col_desc (
    col_desc_ID REAL,
    table_name TEXT,
    field_name TEXT,
    type TEXT,
    description TEXT,
    value_list TEXT,
    sradb_updated TEXT
);

-- Metadata table for internal tracking (not part of original schema)
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);