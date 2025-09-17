-- Enhanced SRAmetadb-compatible schema with full XSD support
-- This schema includes all missing tables identified in the comprehensive analysis
-- Maintains backward compatibility while adding complete XSD coverage

-- Include all original tables from schema_full.sql
-- (These would be imported from the original file in production)

-- ============================================
-- NEW TABLES FOR COMPLETE XSD SUPPORT
-- ============================================

-- Analysis table for secondary analysis results (completely missing before)
CREATE TABLE IF NOT EXISTS analysis (
    analysis_ID REAL,
    analysis_accession TEXT PRIMARY KEY,
    analysis_alias TEXT,
    broker_name TEXT,
    center_name TEXT,
    title TEXT,
    description TEXT,
    analysis_type TEXT, -- DE_NOVO_ASSEMBLY, REFERENCE_ALIGNMENT, SEQUENCE_ANNOTATION, etc.
    study_accession TEXT,
    sample_accession TEXT,
    run_accession TEXT,
    analysis_date TEXT,
    assembly_info TEXT, -- JSON: {name, partial, coverage, program, platform}
    targets TEXT, -- JSON array of target SRA objects
    processing_pipeline TEXT, -- JSON: pipeline steps and parameters
    data_blocks TEXT, -- JSON: files and their checksums
    analysis_links TEXT, -- JSON: url_links and xref_links
    analysis_attributes TEXT, -- JSON: custom key-value pairs
    submission_accession TEXT,
    sradb_updated TEXT,
    FOREIGN KEY (study_accession) REFERENCES study(study_accession),
    FOREIGN KEY (sample_accession) REFERENCES sample(sample_accession),
    FOREIGN KEY (run_accession) REFERENCES run(run_accession),
    FOREIGN KEY (submission_accession) REFERENCES submission(submission_accession)
);
CREATE INDEX IF NOT EXISTS analysis_acc_idx ON analysis (analysis_accession);
CREATE INDEX IF NOT EXISTS idx_analysis_study ON analysis(study_accession);
CREATE INDEX IF NOT EXISTS idx_analysis_type ON analysis(analysis_type);

-- Sample pool table for multiplex/pooling relationships
CREATE TABLE IF NOT EXISTS sample_pool (
    pool_id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_sample TEXT NOT NULL,
    member_sample TEXT NOT NULL,
    member_name TEXT,
    member_label TEXT,
    proportion REAL,
    read_label TEXT,
    barcode TEXT,
    barcode_read_group_tag TEXT,
    pool_proportion TEXT,
    FOREIGN KEY (parent_sample) REFERENCES sample(sample_accession),
    FOREIGN KEY (member_sample) REFERENCES sample(sample_accession),
    UNIQUE(parent_sample, member_sample)
);
CREATE INDEX IF NOT EXISTS idx_pool_parent ON sample_pool(parent_sample);
CREATE INDEX IF NOT EXISTS idx_pool_member ON sample_pool(member_sample);

-- Structured identifier storage for all ID types
CREATE TABLE IF NOT EXISTS identifiers (
    identifier_id INTEGER PRIMARY KEY AUTOINCREMENT,
    record_type TEXT NOT NULL, -- study, sample, experiment, run, analysis
    record_accession TEXT NOT NULL,
    id_type TEXT NOT NULL, -- PRIMARY_ID, SECONDARY_ID, EXTERNAL_ID, SUBMITTER_ID, UUID
    id_namespace TEXT,
    id_value TEXT NOT NULL,
    id_label TEXT,
    UNIQUE(record_accession, id_type, id_value),
    CHECK(record_type IN ('study', 'sample', 'experiment', 'run', 'analysis', 'submission'))
);
CREATE INDEX IF NOT EXISTS idx_identifiers_record ON identifiers(record_accession);
CREATE INDEX IF NOT EXISTS idx_identifiers_type ON identifiers(id_type);
CREATE INDEX IF NOT EXISTS idx_identifiers_value ON identifiers(id_value);

-- Platform enumeration table with instrument models
CREATE TABLE IF NOT EXISTS platforms (
    platform_id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform_name TEXT NOT NULL UNIQUE,
    platform_type TEXT NOT NULL, -- sequencing, array, etc.
    instrument_models TEXT, -- JSON array of valid instrument models
    CHECK(platform_name IN (
        'LS454', 'ILLUMINA', 'HELICOS', 'ABI_SOLID', 'COMPLETE_GENOMICS',
        'BGISEQ', 'OXFORD_NANOPORE', 'PACBIO_SMRT', 'ION_TORRENT',
        'VELA_DIAGNOSTICS', 'CAPILLARY', 'GENAPSYS', 'DNBSEQ', 'ELEMENT',
        'GENEMIND', 'ULTIMA', 'TAPESTRI', 'SALUS', 'GENEUS_TECH',
        'SINGULAR_GENOMICS', 'GENEXUS', 'REVOLOCITY'
    ))
);
CREATE INDEX IF NOT EXISTS idx_platform_name ON platforms(platform_name);

-- Structured link storage for all external references
CREATE TABLE IF NOT EXISTS links (
    link_id INTEGER PRIMARY KEY AUTOINCREMENT,
    record_type TEXT NOT NULL,
    record_accession TEXT NOT NULL,
    link_type TEXT NOT NULL, -- URL_LINK, XREF_LINK, ENTREZ_LINK
    db TEXT, -- database name for XREF_LINK
    id TEXT, -- identifier in external database
    label TEXT, -- optional label
    url TEXT, -- for URL_LINK
    query TEXT, -- for ENTREZ_LINK
    CHECK(record_type IN ('study', 'sample', 'experiment', 'run', 'analysis', 'submission')),
    CHECK(link_type IN ('URL_LINK', 'XREF_LINK', 'ENTREZ_LINK'))
);
CREATE INDEX IF NOT EXISTS idx_links_record ON links(record_accession);
CREATE INDEX IF NOT EXISTS idx_links_type ON links(link_type);
CREATE INDEX IF NOT EXISTS idx_links_db ON links(db);

-- Spot descriptor table for read structure details
CREATE TABLE IF NOT EXISTS spot_descriptor (
    descriptor_id INTEGER PRIMARY KEY AUTOINCREMENT,
    experiment_accession TEXT,
    run_accession TEXT,
    spot_decode_spec TEXT, -- SPOT_DECODE_SPEC details
    read_specs TEXT, -- JSON array of read specifications
    spot_length INTEGER,
    adapter_spec TEXT,
    FOREIGN KEY (experiment_accession) REFERENCES experiment(experiment_accession),
    FOREIGN KEY (run_accession) REFERENCES run(run_accession)
);
CREATE INDEX IF NOT EXISTS idx_spot_experiment ON spot_descriptor(experiment_accession);
CREATE INDEX IF NOT EXISTS idx_spot_run ON spot_descriptor(run_accession);

-- Processing/Pipeline information table
CREATE TABLE IF NOT EXISTS processing (
    processing_id INTEGER PRIMARY KEY AUTOINCREMENT,
    record_type TEXT NOT NULL,
    record_accession TEXT NOT NULL,
    pipeline_name TEXT,
    pipeline_version TEXT,
    processing_date TEXT,
    base_calls TEXT, -- JSON: base calling details
    quality_scores TEXT, -- JSON: quality scoring details
    sequence_processing TEXT, -- JSON: processing steps
    directives TEXT, -- JSON: processing directives
    CHECK(record_type IN ('experiment', 'run', 'analysis'))
);
CREATE INDEX IF NOT EXISTS idx_processing_record ON processing(record_accession);

-- Library strategy enumeration table
CREATE TABLE IF NOT EXISTS library_strategies (
    strategy_id INTEGER PRIMARY KEY AUTOINCREMENT,
    strategy_name TEXT NOT NULL UNIQUE,
    strategy_category TEXT, -- genomic, transcriptomic, epigenetic, etc.
    description TEXT,
    CHECK(strategy_name IN (
        'WGS', 'WGA', 'WXS', 'RNA-Seq', 'ssRNA-seq', 'miRNA-Seq', 'ncRNA-Seq',
        'FL-cDNA', 'EST', 'Hi-C', 'ATAC-seq', 'WCS', 'RAD-Seq', 'CLONE',
        'POOLCLONE', 'AMPLICON', 'CLONEEND', 'FINISHING', 'ChIP-Seq',
        'MNase-Seq', 'DNase-Hypersensitivity', 'Bisulfite-Seq', 'CTS',
        'MRE-Seq', 'MeDIP-Seq', 'MBD-Seq', 'Tn-Seq', 'VALIDATION', 'FAIRE-seq',
        'SELEX', 'RIP-Seq', 'ChIA-PET', 'Synthetic-Long-Read', 'Targeted-Capture',
        'Tethered Chromatin Conformation Capture', 'OTHER'
    ))
);

-- Library source enumeration table
CREATE TABLE IF NOT EXISTS library_sources (
    source_id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL UNIQUE,
    description TEXT,
    CHECK(source_name IN (
        'GENOMIC', 'GENOMIC SINGLE CELL', 'TRANSCRIPTOMIC',
        'TRANSCRIPTOMIC SINGLE CELL', 'METAGENOMIC', 'METATRANSCRIPTOMIC',
        'SYNTHETIC', 'VIRAL RNA', 'OTHER'
    ))
);

-- Library selection enumeration table
CREATE TABLE IF NOT EXISTS library_selections (
    selection_id INTEGER PRIMARY KEY AUTOINCREMENT,
    selection_name TEXT NOT NULL UNIQUE,
    selection_category TEXT,
    description TEXT,
    CHECK(selection_name IN (
        'RANDOM', 'PCR', 'RANDOM PCR', 'RT-PCR', 'HMPR', 'MF', 'CF-S', 'CF-M',
        'CF-H', 'CF-T', 'MDA', 'MSLL', 'cDNA', 'cDNA_randomPriming',
        'cDNA_oligo_dT', 'PolyA', 'Oligo-dT', 'Inverse rRNA', 'Inverse rRNA selection',
        'ChIP', 'ChIP-Seq', 'MNase', 'DNase', 'Hybrid Selection',
        'Reduced Representation', 'Restriction Digest', '5-methylcytidine antibody',
        'MBD2 protein methyl-CpG binding domain', 'CAGE', 'RACE', 'size fractionation',
        'Padlock probes capture method', 'other', 'unspecified'
    ))
);

-- File types enumeration table
CREATE TABLE IF NOT EXISTS file_types (
    file_type_id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_type TEXT NOT NULL UNIQUE,
    file_extension TEXT,
    file_category TEXT, -- archive, alignment, native, etc.
    description TEXT,
    CHECK(file_type IN (
        'sra', 'srf', 'sff', 'fastq', 'fasta', 'tab', 'bam', 'bai', 'cram', 'crai',
        'vcf', 'bcf', 'vcf_aggregate', 'bcf_aggregate', 'gff', 'gtf', 'bed', 'bigwig',
        'wiggle', '454_native', 'Illumina_native', 'Illumina_native_qseq',
        'Illumina_native_scarf', 'Illumina_native_fastq', 'SOLiD_native',
        'SOLiD_native_csfasta', 'SOLiD_native_qual', 'PacBio_HDF5',
        'CompleteGenomics_native', 'OxfordNanopore_native', 'agp', 'unlocalised_list',
        'info', 'manifest', 'readme', 'phenotype_file', 'BioNano_native',
        'Bionano_native', 'chromosome_list', 'sample_list', 'other'
    ))
);

-- Analysis types enumeration table
CREATE TABLE IF NOT EXISTS analysis_types (
    type_id INTEGER PRIMARY KEY AUTOINCREMENT,
    type_name TEXT NOT NULL UNIQUE,
    type_category TEXT,
    description TEXT,
    CHECK(type_name IN (
        'REFERENCE_ALIGNMENT', 'SEQUENCE_VARIATION', 'SEQUENCE_ASSEMBLY',
        'SEQUENCE_ANNOTATION', 'REFERENCE_SEQUENCE', 'SAMPLE_PHENOTYPE',
        'TRANSCRIPTOME_ASSEMBLY', 'TAXONOMIC_REFERENCE_SET', 'DE_NOVO_ASSEMBLY',
        'GENOME_MAP', 'AMR_ANTIBIOGRAM', 'PATHOGEN_ANALYSIS',
        'PROCESSED_READS', 'SEQUENCE_FLATFILE'
    ))
);

-- Data block/file information
CREATE TABLE IF NOT EXISTS data_files (
    file_id INTEGER PRIMARY KEY AUTOINCREMENT,
    record_type TEXT NOT NULL,
    record_accession TEXT NOT NULL,
    filename TEXT NOT NULL,
    filetype TEXT NOT NULL,
    checksum TEXT,
    checksum_method TEXT, -- MD5, SHA-1, SHA-256
    unencrypted_checksum TEXT,
    file_size INTEGER,
    file_date TEXT,
    ascii_offset INTEGER,
    quality_encoding TEXT,
    quality_scoring_system TEXT,
    CHECK(record_type IN ('run', 'analysis')),
    FOREIGN KEY (filetype) REFERENCES file_types(file_type)
);
CREATE INDEX IF NOT EXISTS idx_files_record ON data_files(record_accession);
CREATE INDEX IF NOT EXISTS idx_files_type ON data_files(filetype);

-- Related studies junction table
CREATE TABLE IF NOT EXISTS related_studies (
    relation_id INTEGER PRIMARY KEY AUTOINCREMENT,
    study_accession TEXT NOT NULL,
    related_study_accession TEXT NOT NULL,
    is_primary INTEGER DEFAULT 0,
    relation_type TEXT, -- parent, child, related, supersedes, etc.
    FOREIGN KEY (study_accession) REFERENCES study(study_accession),
    FOREIGN KEY (related_study_accession) REFERENCES study(study_accession),
    UNIQUE(study_accession, related_study_accession)
);
CREATE INDEX IF NOT EXISTS idx_related_study ON related_studies(study_accession);
CREATE INDEX IF NOT EXISTS idx_related_target ON related_studies(related_study_accession);

-- XSD validation tracking table
CREATE TABLE IF NOT EXISTS xsd_validation (
    validation_id INTEGER PRIMARY KEY AUTOINCREMENT,
    record_type TEXT NOT NULL,
    record_accession TEXT NOT NULL,
    schema_version TEXT,
    validation_date TEXT,
    validation_status TEXT, -- VALID, INVALID, WARNING
    validation_errors TEXT, -- JSON array of errors
    CHECK(validation_status IN ('VALID', 'INVALID', 'WARNING'))
);
CREATE INDEX IF NOT EXISTS idx_validation_record ON xsd_validation(record_accession);
CREATE INDEX IF NOT EXISTS idx_validation_status ON xsd_validation(validation_status);

-- Insert platform definitions
INSERT OR IGNORE INTO platforms (platform_name, platform_type, instrument_models) VALUES
('ILLUMINA', 'sequencing', '["Illumina Genome Analyzer", "Illumina Genome Analyzer II", "Illumina Genome Analyzer IIx", "Illumina HiSeq 2500", "Illumina HiSeq 2000", "Illumina HiSeq 1500", "Illumina HiSeq 1000", "Illumina MiSeq", "Illumina HiScanSQ", "HiSeq X Ten", "NextSeq 500", "HiSeq X Five", "Illumina HiSeq 3000", "Illumina HiSeq 4000", "NextSeq 550", "Illumina NovaSeq 6000", "Illumina iSeq 100", "Illumina NovaSeq X", "Illumina NovaSeq X Plus", "NextSeq 1000", "NextSeq 2000", "Illumina MiniSeq", "Illumina HiSeq X", "UG 100"]'),
('OXFORD_NANOPORE', 'sequencing', '["MinION", "GridION", "PromethION", "PromethION 24", "PromethION 48", "flongle"]'),
('PACBIO_SMRT', 'sequencing', '["PacBio RS", "PacBio RS II", "Sequel", "Sequel II", "Sequel IIe", "Onso", "Revio"]'),
('ION_TORRENT', 'sequencing', '["Ion Torrent PGM", "Ion Torrent Proton", "Ion Torrent S5", "Ion Torrent S5 XL", "Ion GeneStudio S5", "Ion GeneStudio S5 Plus", "Ion GeneStudio S5 Prime"]'),
('LS454', 'sequencing', '["454 GS", "454 GS 20", "454 GS FLX", "454 GS FLX+", "454 GS FLX Titanium", "454 GS Junior"]'),
('COMPLETE_GENOMICS', 'sequencing', '["Complete Genomics"]'),
('BGISEQ', 'sequencing', '["BGISEQ-50", "BGISEQ-500", "MGISEQ-2000RS"]'),
('DNBSEQ', 'sequencing', '["DNBSEQ-G400", "DNBSEQ-T7", "DNBSEQ-G50", "DNBSEQ-G400 FAST"]'),
('ELEMENT', 'sequencing', '["Element AVITI"]'),
('GENEMIND', 'sequencing', '["GenoCare 1600", "GenoLab M"]'),
('ULTIMA', 'sequencing', '["UG 100"]'),
('TAPESTRI', 'sequencing', '["Tapestri"]'),
('VELA_DIAGNOSTICS', 'sequencing', '["Sentosa SQ301"]'),
('GENAPSYS', 'sequencing', '["Genapsys Sequencer"]'),
('SALUS', 'sequencing', '["Salus"]'),
('GENEUS_TECH', 'sequencing', '["Geneus Tech"]'),
('SINGULAR_GENOMICS', 'sequencing', '["Singular G4"]'),
('GENEXUS', 'sequencing', '["Genexus"]'),
('REVOLOCITY', 'sequencing', '["Revolocity"]'),
('HELICOS', 'sequencing', '["Helicos HeliScope"]'),
('ABI_SOLID', 'sequencing', '["AB SOLiD System", "AB SOLiD System 2.0", "AB SOLiD System 3.0", "AB SOLiD 3 Plus System", "AB SOLiD 4 System", "AB SOLiD 4hq System", "AB SOLiD PI System", "AB 5500 Genetic Analyzer", "AB 5500xl Genetic Analyzer", "AB 5500xl-W Genetic Analysis System"]'),
('CAPILLARY', 'sequencing', '["AB 3730xL Genetic Analyzer", "AB 3730 Genetic Analyzer", "AB 3500xL Genetic Analyzer", "AB 3500 Genetic Analyzer", "AB 3130xL Genetic Analyzer", "AB 3130 Genetic Analyzer", "AB 310 Genetic Analyzer"]');

-- Insert library strategy definitions
INSERT OR IGNORE INTO library_strategies (strategy_name, strategy_category) VALUES
('WGS', 'genomic'),
('WGA', 'genomic'),
('WXS', 'genomic'),
('WCS', 'genomic'),
('CLONE', 'genomic'),
('POOLCLONE', 'genomic'),
('AMPLICON', 'genomic'),
('CLONEEND', 'genomic'),
('FINISHING', 'genomic'),
('RAD-Seq', 'genomic'),
('RNA-Seq', 'transcriptomic'),
('ssRNA-seq', 'transcriptomic'),
('miRNA-Seq', 'transcriptomic'),
('ncRNA-Seq', 'transcriptomic'),
('FL-cDNA', 'transcriptomic'),
('EST', 'transcriptomic'),
('ChIP-Seq', 'epigenetic'),
('ATAC-seq', 'epigenetic'),
('Bisulfite-Seq', 'epigenetic'),
('MeDIP-Seq', 'epigenetic'),
('MBD-Seq', 'epigenetic'),
('MRE-Seq', 'epigenetic'),
('MNase-Seq', 'epigenetic'),
('DNase-Hypersensitivity', 'epigenetic'),
('FAIRE-seq', 'epigenetic'),
('Hi-C', 'structural'),
('ChIA-PET', 'structural'),
('Tethered Chromatin Conformation Capture', 'structural'),
('CTS', 'other'),
('Tn-Seq', 'other'),
('VALIDATION', 'other'),
('SELEX', 'other'),
('RIP-Seq', 'other'),
('Synthetic-Long-Read', 'other'),
('Targeted-Capture', 'other'),
('OTHER', 'other');

-- Views for backward compatibility and convenience
CREATE VIEW IF NOT EXISTS experiment_pool_view AS
SELECT
    e.experiment_accession,
    e.sample_accession as primary_sample,
    sp.member_sample,
    sp.member_name,
    sp.barcode,
    sp.proportion
FROM experiment e
LEFT JOIN sample_pool sp ON e.sample_accession = sp.parent_sample;

CREATE VIEW IF NOT EXISTS study_relationships AS
SELECT
    s.study_accession,
    s.study_title,
    rs.related_study_accession,
    rs2.study_title as related_study_title,
    rs.relation_type,
    rs.is_primary
FROM study s
LEFT JOIN related_studies rs ON s.study_accession = rs.study_accession
LEFT JOIN study rs2 ON rs.related_study_accession = rs2.study_accession;

-- Statistics view for quick system overview
CREATE VIEW IF NOT EXISTS sra_statistics AS
SELECT
    (SELECT COUNT(*) FROM study) as total_studies,
    (SELECT COUNT(*) FROM sample) as total_samples,
    (SELECT COUNT(*) FROM experiment) as total_experiments,
    (SELECT COUNT(*) FROM run) as total_runs,
    (SELECT COUNT(*) FROM analysis) as total_analyses,
    (SELECT COUNT(DISTINCT platform) FROM experiment) as unique_platforms,
    (SELECT COUNT(DISTINCT library_strategy) FROM experiment) as unique_strategies,
    (SELECT COUNT(DISTINCT scientific_name) FROM sample) as unique_organisms,
    (SELECT SUM(bases) FROM run) as total_bases,
    (SELECT MAX(sradb_updated) FROM study) as last_update;