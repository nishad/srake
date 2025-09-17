-- Comprehensive SRA Database Schema based on XSD analysis
-- This schema captures all critical fields from SRA XML schemas

-- ============ STUDIES TABLE ============
CREATE TABLE IF NOT EXISTS studies (
    study_accession TEXT PRIMARY KEY,

    -- NameGroup attributes
    alias TEXT,
    center_name TEXT,
    broker_name TEXT,

    -- Core fields
    study_title TEXT,
    study_type TEXT,
    study_abstract TEXT,
    study_description TEXT,
    center_project_name TEXT,

    -- Dates
    submission_date DATE,
    first_public DATE,
    last_update DATE,

    -- Identifiers
    primary_id TEXT,
    secondary_ids TEXT,  -- JSON array
    external_ids TEXT,   -- JSON array
    submitter_ids TEXT,  -- JSON array

    -- Links and attributes
    study_links TEXT,     -- JSON array
    study_attributes TEXT, -- JSON array
    related_studies TEXT,  -- JSON array

    -- Extracted organism (denormalized for fast queries)
    organism TEXT,

    -- Full metadata
    metadata JSON
);

-- ============ EXPERIMENTS TABLE ============
CREATE TABLE IF NOT EXISTS experiments (
    experiment_accession TEXT PRIMARY KEY,

    -- NameGroup attributes
    alias TEXT,
    center_name TEXT,
    broker_name TEXT,

    -- References
    study_accession TEXT REFERENCES studies(study_accession),
    sample_accession TEXT,  -- Can reference samples table

    -- Core fields
    title TEXT,
    design_description TEXT,

    -- Library information
    library_name TEXT,
    library_strategy TEXT,
    library_source TEXT,
    library_selection TEXT,
    library_layout TEXT,  -- 'SINGLE' or 'PAIRED'
    library_construction_protocol TEXT,

    -- Paired-end specific
    nominal_length INTEGER,
    nominal_sdev REAL,

    -- Platform information
    platform TEXT,
    instrument_model TEXT,

    -- Targeted sequencing
    targeted_loci TEXT,  -- JSON array

    -- Pooling information
    pool_member_count INTEGER,
    pool_info TEXT,  -- JSON object with pool details

    -- Links and attributes
    experiment_links TEXT,     -- JSON array
    experiment_attributes TEXT, -- JSON array

    -- Spot descriptor
    spot_length INTEGER,
    spot_decode_spec TEXT,  -- JSON object

    -- Full metadata
    metadata JSON
);

-- ============ SAMPLES TABLE ============
CREATE TABLE IF NOT EXISTS samples (
    sample_accession TEXT PRIMARY KEY,

    -- NameGroup attributes
    alias TEXT,
    center_name TEXT,
    broker_name TEXT,

    -- Core fields
    title TEXT,
    description TEXT,

    -- Taxonomy
    taxon_id INTEGER,
    scientific_name TEXT,
    common_name TEXT,
    organism TEXT,  -- Extracted/normalized

    -- Sample source information (from attributes)
    tissue TEXT,
    cell_type TEXT,
    cell_line TEXT,
    strain TEXT,
    sex TEXT,
    age TEXT,
    disease TEXT,
    treatment TEXT,

    -- Geographic/environmental
    geo_loc_name TEXT,
    lat_lon TEXT,
    collection_date TEXT,
    env_biome TEXT,
    env_feature TEXT,
    env_material TEXT,

    -- Links and attributes
    sample_links TEXT,     -- JSON array
    sample_attributes TEXT, -- JSON array with all attributes

    -- BioSample/BioProject references
    biosample_accession TEXT,
    bioproject_accession TEXT,

    -- Full metadata
    metadata JSON
);

-- ============ RUNS TABLE ============
CREATE TABLE IF NOT EXISTS runs (
    run_accession TEXT PRIMARY KEY,

    -- NameGroup attributes
    alias TEXT,
    center_name TEXT,
    broker_name TEXT,
    run_center TEXT,

    -- References
    experiment_accession TEXT REFERENCES experiments(experiment_accession),

    -- Core fields
    title TEXT,
    run_date TEXT,

    -- Statistics
    total_spots BIGINT,
    total_bases BIGINT,
    total_size BIGINT,
    load_done BOOLEAN,
    published TEXT,

    -- File information
    data_files TEXT,  -- JSON array of file objects

    -- Links and attributes
    run_links TEXT,       -- JSON array
    run_attributes TEXT,  -- JSON array

    -- Quality metrics (from attributes)
    quality_score_mean REAL,
    quality_score_std REAL,
    read_count_r1 BIGINT,
    read_count_r2 BIGINT,

    -- Full metadata
    metadata JSON
);

-- ============ SUBMISSIONS TABLE (optional but useful) ============
CREATE TABLE IF NOT EXISTS submissions (
    submission_accession TEXT PRIMARY KEY,

    -- NameGroup attributes
    alias TEXT,
    center_name TEXT,
    broker_name TEXT,
    lab_name TEXT,

    -- Core fields
    submission_date TEXT,
    submission_comment TEXT,

    -- Actions
    actions TEXT,  -- JSON array

    -- Files
    files TEXT,    -- JSON array

    -- Full metadata
    metadata JSON
);

-- ============ ANALYSIS TABLE (for future expansion) ============
CREATE TABLE IF NOT EXISTS analyses (
    analysis_accession TEXT PRIMARY KEY,

    -- NameGroup attributes
    alias TEXT,
    center_name TEXT,
    broker_name TEXT,

    -- References
    study_accession TEXT,

    -- Core fields
    title TEXT,
    description TEXT,
    analysis_type TEXT,

    -- Links and attributes
    analysis_links TEXT,      -- JSON array
    analysis_attributes TEXT,  -- JSON array

    -- Full metadata
    metadata JSON
);

-- ============ INDEXES FOR PERFORMANCE ============

-- Study indexes
CREATE INDEX IF NOT EXISTS idx_study_organism ON studies(organism);
CREATE INDEX IF NOT EXISTS idx_study_date ON studies(submission_date);
CREATE INDEX IF NOT EXISTS idx_study_type ON studies(study_type);
CREATE INDEX IF NOT EXISTS idx_study_center ON studies(center_name);
CREATE INDEX IF NOT EXISTS idx_study_alias ON studies(alias);

-- Experiment indexes
CREATE INDEX IF NOT EXISTS idx_exp_study ON experiments(study_accession);
CREATE INDEX IF NOT EXISTS idx_exp_sample ON experiments(sample_accession);
CREATE INDEX IF NOT EXISTS idx_exp_strategy ON experiments(library_strategy);
CREATE INDEX IF NOT EXISTS idx_exp_source ON experiments(library_source);
CREATE INDEX IF NOT EXISTS idx_exp_platform ON experiments(platform);
CREATE INDEX IF NOT EXISTS idx_exp_layout ON experiments(library_layout);
CREATE INDEX IF NOT EXISTS idx_exp_center ON experiments(center_name);

-- Sample indexes
CREATE INDEX IF NOT EXISTS idx_sample_organism ON samples(organism);
CREATE INDEX IF NOT EXISTS idx_sample_taxon ON samples(taxon_id);
CREATE INDEX IF NOT EXISTS idx_sample_scientific ON samples(scientific_name);
CREATE INDEX IF NOT EXISTS idx_sample_tissue ON samples(tissue);
CREATE INDEX IF NOT EXISTS idx_sample_cell_type ON samples(cell_type);
CREATE INDEX IF NOT EXISTS idx_sample_biosample ON samples(biosample_accession);
CREATE INDEX IF NOT EXISTS idx_sample_center ON samples(center_name);

-- Run indexes
CREATE INDEX IF NOT EXISTS idx_run_experiment ON runs(experiment_accession);
CREATE INDEX IF NOT EXISTS idx_run_date ON runs(run_date);
CREATE INDEX IF NOT EXISTS idx_run_center ON runs(run_center);
CREATE INDEX IF NOT EXISTS idx_run_published ON runs(published);

-- Full-text search indexes (if using FTS5)
-- CREATE VIRTUAL TABLE studies_fts USING fts5(
--     study_title, study_abstract, study_description,
--     content=studies, content_rowid=rowid
-- );

-- ============ VIEWS FOR COMMON QUERIES ============

-- View combining experiment with study and sample info
CREATE VIEW IF NOT EXISTS experiment_full AS
SELECT
    e.*,
    s.study_title,
    s.study_type,
    s.organism as study_organism,
    sm.scientific_name,
    sm.taxon_id,
    sm.organism as sample_organism,
    sm.tissue,
    sm.cell_type
FROM experiments e
LEFT JOIN studies s ON e.study_accession = s.study_accession
LEFT JOIN samples sm ON e.sample_accession = sm.sample_accession;

-- View for run statistics
CREATE VIEW IF NOT EXISTS run_stats AS
SELECT
    r.*,
    e.library_strategy,
    e.library_source,
    e.platform,
    e.instrument_model,
    s.study_title,
    s.organism
FROM runs r
LEFT JOIN experiments e ON r.experiment_accession = e.experiment_accession
LEFT JOIN studies s ON e.study_accession = s.study_accession;

-- ============ TRACKING AND METADATA TABLES ============

-- Track data updates
CREATE TABLE IF NOT EXISTS data_updates (
    update_id INTEGER PRIMARY KEY AUTOINCREMENT,
    update_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    file_name TEXT,
    records_processed INTEGER,
    records_inserted INTEGER,
    records_updated INTEGER,
    processing_time_seconds REAL,
    status TEXT,
    error_message TEXT
);

-- Track extraction statistics
CREATE TABLE IF NOT EXISTS extraction_stats (
    stat_date DATE PRIMARY KEY,
    total_studies INTEGER,
    total_experiments INTEGER,
    total_samples INTEGER,
    total_runs INTEGER,
    total_analyses INTEGER,
    unique_organisms INTEGER,
    unique_platforms INTEGER,
    unique_library_strategies INTEGER,
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);