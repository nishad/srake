# Utility Scripts

This directory contains utility scripts for srake users and contributors.

## Database Migration

### safe_migrate.sh

Safely migrates existing SRA databases to the latest schema version. This script:
- Creates automatic backups before migration
- Validates the database integrity
- Applies schema updates
- Preserves all existing data

**Usage:**
```bash
./scripts/safe_migrate.sh /path/to/your/database.db
```

**Example:**
```bash
# Migrate the default database
./scripts/safe_migrate.sh ./data/SRAmetadb.sqlite

# Migrate a custom database location
./scripts/safe_migrate.sh ~/research/sra_data.db
```

The script will:
1. Create a backup at `database.db.backup_YYYYMMDD_HHMMSS`
2. Apply the migration from `migrate_schema.sql`
3. Verify the migration succeeded
4. Report statistics about the migrated data

## Release Building

### build-release.sh

Builds release binaries for multiple platforms. This script is primarily for maintainers and contributors who want to build custom releases.

**Usage:**
```bash
./scripts/build-release.sh [version]
```

**Examples:**
```bash
# Build with automatic version detection from git
./scripts/build-release.sh

# Build a specific version
./scripts/build-release.sh v0.2.0

# Build a pre-release
./scripts/build-release.sh v0.2.0-beta.1
```

**Output:**
The script generates:
- Linux binaries (amd64, arm64)
- macOS binaries (amd64, arm64)
- Windows binaries (amd64)
- SHA256 checksums file
- Release notes template

All artifacts are created in the current directory with the naming pattern:
- `srake-{version}-{os}-{arch}.tar.gz` (Linux/macOS)
- `srake-{version}-windows-{arch}.zip` (Windows)
- `srake-{version}-SHA256SUMS.txt` (Checksums)

## SQL Schema

### migrate_schema.sql

The SQL schema migration file used by `safe_migrate.sh`. This file contains:
- Table creation statements
- Index definitions
- Schema updates from older versions
- Data transformation queries

This file should not be run directly. Use `safe_migrate.sh` instead to ensure proper backup and validation.

## Development

For development and maintenance scripts, see the project's contribution guidelines. Internal development scripts are maintained separately from these user-facing utilities.

## Requirements

- **safe_migrate.sh**: Requires SQLite3 command-line tool
- **build-release.sh**: Requires Go 1.19+ and standard Unix tools (tar, zip, sha256sum)

## Support

If you encounter issues with these scripts:
1. Check that you have the required tools installed
2. Ensure the scripts have execute permissions: `chmod +x scripts/*.sh`
3. Report issues at: https://github.com/nishad/srake/issues