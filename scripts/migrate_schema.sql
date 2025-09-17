-- SRA Metadata Database Schema Migration Script
-- Version: 1.0
-- Date: 2025-09-17
-- Purpose: Migrate existing database to fix schema issues and add missing tables

-- Enable foreign keys (must be done before transaction)
PRAGMA foreign_keys = ON;

BEGIN TRANSACTION;

-- Step 1: Fix column names in runs table
-- Check if old columns exist and rename them
ALTER TABLE runs RENAME COLUMN spots TO total_spots;
ALTER TABLE runs RENAME COLUMN bases TO total_bases;

-- Step 2: Create missing tables if they don't exist

-- Pool/multiplex relationships
CREATE TABLE IF NOT EXISTS sample_pool (
    pool_id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_sample TEXT REFERENCES samples(sample_accession),
    member_sample TEXT,
    member_name TEXT,
    proportion REAL,
    read_label TEXT,
    UNIQUE(parent_sample, member_sample)
);

-- Structured identifiers
CREATE TABLE IF NOT EXISTS identifiers (
    record_type TEXT NOT NULL,
    record_accession TEXT NOT NULL,
    id_type TEXT NOT NULL,
    id_namespace TEXT,
    id_value TEXT NOT NULL,
    id_label TEXT,
    PRIMARY KEY (record_type, record_accession, id_type, id_value)
);

-- External links
CREATE TABLE IF NOT EXISTS links (
    link_id INTEGER PRIMARY KEY AUTOINCREMENT,
    record_type TEXT NOT NULL,
    record_accession TEXT NOT NULL,
    link_type TEXT,
    db TEXT,
    id TEXT,
    label TEXT,
    url TEXT
);

-- Junction table for experiment-sample many-to-many relationship
CREATE TABLE IF NOT EXISTS experiment_samples (
    experiment_accession TEXT REFERENCES experiments(experiment_accession),
    sample_accession TEXT REFERENCES samples(sample_accession),
    PRIMARY KEY (experiment_accession, sample_accession)
);

-- Step 3: Create indexes for new tables
CREATE INDEX IF NOT EXISTS idx_pool_parent ON sample_pool(parent_sample);
CREATE INDEX IF NOT EXISTS idx_pool_member ON sample_pool(member_sample);
CREATE INDEX IF NOT EXISTS idx_identifier_value ON identifiers(id_value);
CREATE INDEX IF NOT EXISTS idx_identifier_record ON identifiers(record_type, record_accession);
CREATE INDEX IF NOT EXISTS idx_link_record ON links(record_type, record_accession);
CREATE INDEX IF NOT EXISTS idx_exp_sample_exp ON experiment_samples(experiment_accession);
CREATE INDEX IF NOT EXISTS idx_exp_sample_sample ON experiment_samples(sample_accession);

-- Step 4: Clean orphaned records
-- Delete experiments without valid studies
DELETE FROM experiments
WHERE study_accession IS NOT NULL
  AND study_accession != ''
  AND study_accession NOT IN (SELECT study_accession FROM studies);

-- Delete runs without valid experiments
DELETE FROM runs
WHERE experiment_accession IS NOT NULL
  AND experiment_accession != ''
  AND experiment_accession NOT IN (SELECT experiment_accession FROM experiments);

-- Step 5: Add missing columns to existing tables (if needed)
-- Note: SQLite doesn't support IF NOT EXISTS for columns, so we need to check manually
-- These would be added in application code that checks column existence first

-- Step 6: Migrate sample-experiment relationships to junction table
-- Only if experiment_accession exists in samples table
-- This preserves existing relationships while moving to proper many-to-many
INSERT OR IGNORE INTO experiment_samples (experiment_accession, sample_accession)
SELECT DISTINCT experiment_accession, sample_accession
FROM samples
WHERE experiment_accession IS NOT NULL AND experiment_accession != '';

-- Step 7: Verify migration
-- Create a migration_log table to track migration status
CREATE TABLE IF NOT EXISTS migration_log (
    migration_id INTEGER PRIMARY KEY AUTOINCREMENT,
    version TEXT NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

INSERT INTO migration_log (version, description)
VALUES ('1.0', 'Fixed column names, added missing tables, cleaned orphaned records');

COMMIT;

-- Post-migration verification queries
-- Run these to verify the migration was successful:
SELECT 'Tables created:' as check_type,
       COUNT(*) as count
FROM sqlite_master
WHERE type='table'
  AND name IN ('sample_pool', 'identifiers', 'links', 'experiment_samples');

SELECT 'Orphaned experiments:' as check_type,
       COUNT(*) as count
FROM experiments e
LEFT JOIN studies s ON e.study_accession = s.study_accession
WHERE s.study_accession IS NULL;

SELECT 'Orphaned runs:' as check_type,
       COUNT(*) as count
FROM runs r
LEFT JOIN experiments e ON r.experiment_accession = e.experiment_accession
WHERE e.experiment_accession IS NULL;

SELECT 'Run statistics available:' as check_type,
       COUNT(*) as total_runs,
       SUM(CASE WHEN total_spots > 0 THEN 1 ELSE 0 END) as runs_with_stats
FROM runs;

-- Rollback script (in case of issues)
-- BEGIN TRANSACTION;
-- ALTER TABLE runs RENAME COLUMN total_spots TO spots;
-- ALTER TABLE runs RENAME COLUMN total_bases TO bases;
-- DROP TABLE IF EXISTS sample_pool;
-- DROP TABLE IF EXISTS identifiers;
-- DROP TABLE IF EXISTS links;
-- DROP TABLE IF EXISTS experiment_samples;
-- DROP TABLE IF EXISTS migration_log;
-- COMMIT;