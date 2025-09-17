#!/bin/bash

# Safe Migration Script for SRA Metadata Database
# Handles both old and new schema gracefully

set -e

# Check if database file is provided
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <database_file>"
    exit 1
fi

DB_FILE="$1"

if [ ! -f "$DB_FILE" ]; then
    echo "Error: Database file '$DB_FILE' does not exist"
    exit 1
fi

# Backup the database first
BACKUP_FILE="${DB_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
echo "Creating backup: $BACKUP_FILE"
cp "$DB_FILE" "$BACKUP_FILE"

echo "Starting migration of $DB_FILE"

# Function to check if column exists
check_column_exists() {
    local table=$1
    local column=$2
    sqlite3 "$DB_FILE" "PRAGMA table_info($table);" | grep -q "$column"
}

# Check current schema
echo "Checking current schema..."

# Check if runs table has old or new column names
if check_column_exists "runs" "spots"; then
    echo "Found old column 'spots' - will rename to 'total_spots'"
    NEEDS_COLUMN_RENAME=1
elif check_column_exists "runs" "total_spots"; then
    echo "Column 'total_spots' already exists - skipping rename"
    NEEDS_COLUMN_RENAME=0
else
    echo "Warning: Neither 'spots' nor 'total_spots' found in runs table"
    NEEDS_COLUMN_RENAME=0
fi

# Create SQL migration script based on current state
cat > /tmp/migrate_temp.sql << 'EOF'
-- Temporarily disable foreign keys for migration
PRAGMA foreign_keys = OFF;

BEGIN TRANSACTION;

EOF

# Add column rename if needed
if [ "$NEEDS_COLUMN_RENAME" -eq 1 ]; then
    cat >> /tmp/migrate_temp.sql << 'EOF'
-- Fix column names in runs table
ALTER TABLE runs RENAME COLUMN spots TO total_spots;
ALTER TABLE runs RENAME COLUMN bases TO total_bases;

EOF
fi

# Add the rest of the migration
cat >> /tmp/migrate_temp.sql << 'EOF'
-- Create missing tables if they don't exist

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

-- Create indexes for new tables
CREATE INDEX IF NOT EXISTS idx_pool_parent ON sample_pool(parent_sample);
CREATE INDEX IF NOT EXISTS idx_pool_member ON sample_pool(member_sample);
CREATE INDEX IF NOT EXISTS idx_identifier_value ON identifiers(id_value);
CREATE INDEX IF NOT EXISTS idx_identifier_record ON identifiers(record_type, record_accession);
CREATE INDEX IF NOT EXISTS idx_link_record ON links(record_type, record_accession);
CREATE INDEX IF NOT EXISTS idx_exp_sample_exp ON experiment_samples(experiment_accession);
CREATE INDEX IF NOT EXISTS idx_exp_sample_sample ON experiment_samples(sample_accession);

-- Clean orphaned records
DELETE FROM experiments
WHERE study_accession IS NOT NULL
  AND study_accession != ''
  AND study_accession NOT IN (SELECT study_accession FROM studies);

DELETE FROM runs
WHERE experiment_accession IS NOT NULL
  AND experiment_accession != ''
  AND experiment_accession NOT IN (SELECT experiment_accession FROM experiments);

-- Migration log
CREATE TABLE IF NOT EXISTS migration_log (
    migration_id INTEGER PRIMARY KEY AUTOINCREMENT,
    version TEXT NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

INSERT INTO migration_log (version, description)
VALUES ('1.0', 'Fixed column names, added missing tables, cleaned orphaned records');

COMMIT;

-- Re-enable foreign keys after migration
PRAGMA foreign_keys = ON;
EOF

# Apply migration
echo "Applying migration..."
sqlite3 "$DB_FILE" < /tmp/migrate_temp.sql

if [ $? -eq 0 ]; then
    echo "Migration completed successfully!"

    # Run verification queries
    echo -e "\nVerification Results:"
    echo "====================="

    echo -e "\n1. New tables created:"
    sqlite3 "$DB_FILE" "SELECT name FROM sqlite_master WHERE type='table' AND name IN ('sample_pool', 'identifiers', 'links', 'experiment_samples');"

    echo -e "\n2. Orphaned records after cleanup:"
    sqlite3 "$DB_FILE" "SELECT 'Orphaned experiments: ' || COUNT(*) FROM experiments e LEFT JOIN studies s ON e.study_accession = s.study_accession WHERE s.study_accession IS NULL;"
    sqlite3 "$DB_FILE" "SELECT 'Orphaned runs: ' || COUNT(*) FROM runs r LEFT JOIN experiments e ON r.experiment_accession = e.experiment_accession WHERE e.experiment_accession IS NULL;"

    echo -e "\n3. Run statistics check:"
    sqlite3 "$DB_FILE" "SELECT 'Total runs: ' || COUNT(*) || ', Runs with stats: ' || SUM(CASE WHEN total_spots > 0 THEN 1 ELSE 0 END) FROM runs;"

    echo -e "\nBackup saved to: $BACKUP_FILE"
    echo "If you need to rollback, run: cp $BACKUP_FILE $DB_FILE"
else
    echo "Migration failed! Restoring from backup..."
    cp "$BACKUP_FILE" "$DB_FILE"
    echo "Database restored to original state"
    exit 1
fi

# Clean up temp file
rm -f /tmp/migrate_temp.sql

echo -e "\nMigration complete!"