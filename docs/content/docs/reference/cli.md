---
title: CLI Reference
weight: 10
prev: /docs/getting-started
next: /docs/reference/api
---

# CLI Reference

Complete reference for all srake commands and options.

## Global Flags

These flags are available for all commands:

- `--no-color` - Disable colored output
- `-v, --verbose` - Enable verbose output
- `-q, --quiet` - Suppress non-error output
- `-y, --yes` - Assume yes to all prompts (non-interactive mode)
- `--debug` - Enable debug output for troubleshooting
- `--help` - Show help for any command

## Commands

### `srake ingest`

Ingest SRA metadata from NCBI or local archives.

```bash
srake ingest [flags]
```

#### Flags
- `--auto` - Auto-select the best file from NCBI
- `--daily` - Ingest the latest daily update
- `--monthly` - Ingest the latest monthly dataset
- `--file <path>` - Ingest specific file (local or NCBI)
- `--list` - List available files without ingesting
- `--db <path>` - Database path (default: "~/.local/share/srake/srake.db")
- `--force` - Force ingestion even if data exists
- `--no-progress` - Disable progress bar

#### Filtering Flags
- `--taxon-ids <ids>` - Filter by taxonomy IDs (comma-separated)
- `--exclude-taxon-ids <ids>` - Exclude taxonomy IDs
- `--date-from <YYYY-MM-DD>` - Start date for filtering
- `--date-to <YYYY-MM-DD>` - End date for filtering
- `--organisms <names>` - Filter by organism names
- `--platforms <names>` - Filter by platforms (ILLUMINA, OXFORD_NANOPORE, etc.)
- `--strategies <names>` - Filter by library strategies (RNA-Seq, WGS, etc.)
- `--min-reads <n>` - Minimum read count filter
- `--max-reads <n>` - Maximum read count filter
- `--stats-only` - Only show statistics without inserting data

#### Examples
```bash
# Auto-ingest best file
srake ingest --auto

# Non-interactive ingest (no prompts)
srake ingest --auto --yes

# Ingest with filters
srake ingest --auto --taxon-ids 9606 --platforms ILLUMINA --strategies RNA-Seq

# List available files
srake ingest --list

# Debug mode to see detailed processing
srake ingest --auto --debug
```

---

### `srake search`

Search SRA metadata with quality control and multiple search modes.

```bash
srake search <query> [flags]
```

#### Search Flags
- `-o, --organism <name>` - Filter by organism
- `--platform <name>` - Filter by platform
- `--library-strategy <name>` - Filter by library strategy
- `-l, --limit <n>` - Maximum results (default: 100)
- `--offset <n>` - Pagination offset
- `--search-mode <mode>` - Search mode: database|fts|hybrid|vector (default: hybrid)

#### Quality Control Flags
- `--similarity-threshold <float>` - Minimum similarity score (0-1)
- `--min-score <float>` - Minimum absolute score
- `--top-percentile <int>` - Return only top N% of results
- `--show-confidence` - Include confidence level in results

#### Output Flags
- `-f, --format <type>` - Output format (table|json|csv|tsv|xml)
- `--output <file>` - Save results to file
- `--no-header` - Omit header in output
- `--fields <list>` - Comma-separated list of fields to include

#### Examples
```bash
# Basic search
srake search "breast cancer"

# Search with quality control
srake search "RNA-Seq" --similarity-threshold 0.7 --show-confidence

# Vector semantic search
srake search "tumor gene expression" --search-mode vector

# Advanced filtering
srake search "transcriptome" \
  --organism "homo sapiens" \
  --library-strategy RNA-Seq \
  --platform ILLUMINA \
  --top-percentile 10

# Export filtered results
srake search "cancer" --format csv --output results.csv
```

---

### `srake convert`

Convert between different accession types (SRA, GEO, BioProject, BioSample).

```bash
srake convert [<accession> ...] [flags]
```

#### Flags
- `--to <type>` - Target accession type (required)
  - Options: GSE, SRP, SRX, GSM, SRR, SRS, PRJNA, BIOSAMPLE
- `-f, --format <type>` - Output format (table|json|yaml|csv|tsv)
- `-o, --output <file>` - Save results to file
- `--batch <file>` - Read accessions from file
- `--dry-run` - Preview conversions without executing

#### Examples
```bash
# Convert SRA Project to GEO Series
srake convert SRP123456 --to GSE

# Convert multiple accessions
srake convert SRP001 SRP002 SRP003 --to GSE

# Batch conversion from file
srake convert --batch accessions.txt --to SRX --output results.json

# Convert from stdin (pipe-friendly)
echo "SRP123456" | srake convert --to GSE
cat accession_list.txt | srake convert --to GSM --format json

# Preview conversion without executing
srake convert SRP123456 --to GSE --dry-run

# Debug mode to see conversion details
srake convert SRP123456 --to GSE --debug
```

#### Supported Conversions

| From | To | Description |
|------|-----|-------------|
| SRP | GSE, SRX, SRR, SRS, PRJNA | Study to related accessions |
| SRX | GSM, SRP, SRR, SRS | Experiment to related accessions |
| SRR | SRX, SRP, GSM | Run to parent accessions |
| SRS | SRX, GSM, BIOSAMPLE | Sample to related accessions |
| GSE | SRP, GSM | GEO Series to SRA/samples |
| GSM | SRX, SRR, GSE | GEO Sample to SRA/series |
| PRJNA | SRP | BioProject to SRA Project |
| SAMN | SRS | BioSample to SRA Sample |

---

### `srake runs`

Get all runs for a study, experiment, or sample.

```bash
srake runs <accession> [flags]
```

#### Flags
- `-d, --detailed` - Include detailed information
- `-f, --format <type>` - Output format (table|json|yaml|csv|tsv)
- `-o, --output <file>` - Save results to file
- `-l, --limit <n>` - Limit number of results
- `--fields <list>` - Comma-separated list of fields

#### Examples
```bash
# Get runs for a study
srake runs SRP123456

# Get detailed run information
srake runs SRX123456 --detailed

# Export as JSON
srake runs SRP123456 --format json --output runs.json
```

---

### `srake samples`

Get all samples for a study or experiment.

```bash
srake samples <accession> [flags]
```

#### Flags
- `-d, --detailed` - Include organism and taxonomy information
- `-f, --format <type>` - Output format (table|json|yaml|csv|tsv)
- `-o, --output <file>` - Save results to file
- `-l, --limit <n>` - Limit number of results

#### Examples
```bash
# Get samples for a study
srake samples SRP123456

# Get detailed sample information
srake samples SRP123456 --detailed

# Export as CSV
srake samples SRX123456 --format csv --output samples.csv
```

---

### `srake experiments`

Get all experiments for a study or sample.

```bash
srake experiments <accession> [flags]
```

#### Flags
- `-d, --detailed` - Include platform and library information
- `-f, --format <type>` - Output format (table|json|yaml|csv|tsv)
- `-o, --output <file>` - Save results to file
- `-l, --limit <n>` - Limit number of results

#### Examples
```bash
# Get experiments for a study
srake experiments SRP123456

# Get experiments for a sample
srake experiments SRS123456 --detailed
```

---

### `srake studies`

Get study information for any SRA accession.

```bash
srake studies <accession> [flags]
```

#### Flags
- `-d, --detailed` - Include abstract and full metadata
- `-f, --format <type>` - Output format (table|json|yaml|csv|tsv)
- `-o, --output <file>` - Save results to file

#### Examples
```bash
# Get study from an experiment
srake studies SRX123456

# Get study from a run with details
srake studies SRR123456 --detailed
```

---

### `srake download`

Download SRA data files from multiple sources.

```bash
srake download [<accession> ...] [flags]
```

#### Flags
- `-s, --source <type>` - Download source (auto|ftp|aws|gcp|ncbi)
- `-t, --type <type>` - File type (sra|fastq|fasta)
- `-o, --output <dir>` - Output directory (default: "./")
- `--threads <n>` - Download threads per file (default: 1)
- `-p, --parallel <n>` - Parallel downloads (default: 1)
- `--aspera` - Use Aspera for high-speed transfer
- `-l, --list <file>` - File containing accessions
- `--retry <n>` - Number of retry attempts (default: 3)
- `--validate` - Validate downloaded files (default: true)
- `--dry-run` - Show what would be downloaded

#### Examples
```bash
# Basic download
srake download SRR123456

# Download from AWS with parallel transfers
srake download SRR123456 --source aws --threads 4

# Download all runs for a study
srake download SRP123456 --type fastq --output ./data/

# Batch download from file
srake download --list runs.txt --parallel 4

# Download from stdin (pipe-friendly)
echo "SRR123456" | srake download --type fastq
srake runs SRP123456 | srake download --parallel 4

# High-speed Aspera transfer
srake download SRR123456 --aspera

# Dry run to preview downloads
srake download SRP123456 --dry-run

# Non-interactive download (no prompts)
srake download SRP123456 --yes

# Debug mode for troubleshooting
srake download SRR123456 --debug
```

#### Automatic Expansion
The download command automatically expands:
- SRP → all runs in the study
- SRX → all runs in the experiment
- SRS → all runs for the sample

---

### `srake metadata`

Get detailed metadata for specific accessions.

```bash
srake metadata <accession> [accessions...] [flags]
```

#### Flags
- `-f, --format <type>` - Output format (table|json|yaml)
- `--fields <list>` - Comma-separated list of fields
- `--expand` - Expand nested structures

#### Examples
```bash
# Get metadata for an experiment
srake metadata SRX123456

# Get multiple accessions as JSON
srake metadata SRX123456 SRX123457 --format json

# Select specific fields
srake metadata SRR999999 --fields title,platform,strategy
```

---

### `srake index`

Manage search index for fast full-text and vector search.

```bash
srake index [flags]
```

#### Index Operations
- `--build` - Build search index from database
- `--rebuild` - Rebuild index from scratch (removes existing)
- `--verify` - Verify index integrity
- `--stats` - Show index statistics
- `--resume` - Resume interrupted index building

#### Index Options
- `--batch-size <n>` - Documents per batch (default: 1000)
- `--workers <n>` - Number of parallel workers
- `--path <dir>` - Index directory path
- `--with-embeddings` - Build vector embeddings for semantic search
- `--embedding-model <name>` - Model for embeddings (default: SapBERT)
- `--progress` - Show progress bar
- `--progress-file <file>` - Save progress to file
- `--checkpoint-dir <dir>` - Directory for checkpoints

#### Examples
```bash
# Build search index with progress
srake index --build --progress

# Build with vector embeddings for semantic search
srake index --build --with-embeddings

# Build with custom batch size and path
srake index --build --batch-size 5000 --path /custom/index

# Build embeddings with quantized model (faster, less memory)
SRAKE_MODEL_VARIANT=quantized srake index --build --with-embeddings

# Resume interrupted build
srake index --resume

# Rebuild from scratch
srake index --rebuild

# Verify index integrity
srake index --verify

# Show index statistics
srake index --stats
```

---

### `srake server`

Start the API server for programmatic access and AI integration.

```bash
srake server [flags]
```

#### Flags
- `-p, --port <n>` - Port to listen on (default: 8080)
- `--host <addr>` - Host to bind to (default: localhost)
- `--enable-cors` - Enable CORS for web access
- `--enable-mcp` - Enable Model Context Protocol for AI assistants
- `--db <path>` - Database path
- `--index-path <path>` - Search index path
- `--log-level <level>` - Log level (debug|info|warn|error)

#### Examples
```bash
# Start server with all features
srake server --port 8082 --enable-cors --enable-mcp

# Custom database and index
srake server --db /path/to/db --index-path /path/to/index

# Production deployment
srake server --host 0.0.0.0 --port 80 --enable-cors

# With environment variables
SRAKE_DB_PATH=test.db SRAKE_INDEX_PATH=/tmp/index srake server
```

#### API Endpoints
- `/api/v1/search` - Search with quality control
- `/api/v1/stats` - Database statistics
- `/api/v1/studies/{id}` - Study metadata
- `/api/v1/export` - Export search results
- `/api/v1/health` - Service health check
- `/mcp` - MCP JSON-RPC endpoint
- `/mcp/capabilities` - MCP server capabilities

---

### `srake db`

Database management commands.

```bash
srake db <subcommand> [flags]
```

#### Subcommands
- `info` - Show database statistics and information
- `export` - Export database to SRAmetadb format

#### Examples
```bash
# Show database statistics
srake db info

# Export to SRAmetadb format
srake db export -o SRAmetadb.sqlite
```

---

### `srake db export`

Export the srake database to SRAmetadb.sqlite format for compatibility with tools expecting the original SRAmetadb schema.

```bash
srake db export [flags]
```

#### Flags
- `-o, --output <file>` - Output database file path (default: "SRAmetadb.sqlite")
- `--db <path>` - Source database path (defaults to ~/.local/share/srake/srake.db)
- `--fts-version <n>` - FTS version: 3 for compatibility, 5 for modern (default: 5)
- `--batch-size <n>` - Batch size for data transfer (default: 10000)
- `--progress` - Show progress bar (default: true)
- `--compress` - Compress output with gzip
- `-f, --force` - Overwrite existing output file

#### Examples
```bash
# Basic export with FTS5 (recommended)
srake db export -o SRAmetadb.sqlite

# Export with FTS3 for 100% compatibility
srake db export -o SRAmetadb.sqlite --fts-version 3

# Export from specific database
srake db export --db /path/to/srake.db -o SRAmetadb.sqlite

# Export with compression
srake db export -o SRAmetadb.sqlite.gz --compress

# Large dataset with custom batch size
srake db export -o SRAmetadb.sqlite --batch-size 50000
```

#### Output Schema
The exported database contains:
- **Standard tables**: `study`, `experiment`, `sample`, `run`, `submission`
- **Denormalized table**: `sra` (joins all tables for easy querying)
- **Full-text search**: `sra_ft` (FTS3 or FTS5 virtual table)
- **Metadata**: `metaInfo` (version and creation info)
- **Column descriptions**: `col_desc` (field documentation)

#### Compatibility Notes
- **FTS5** (default): Modern, faster, smaller index size, better Unicode support
- **FTS3**: Use for compatibility with older tools that require FTS3
- The export maps srake's modern schema to the classic SRAmetadb format
- JSON fields are converted to pipe-delimited strings
- Missing legacy fields are populated with appropriate defaults

### `srake config`

Configuration and path management commands.

```bash
srake config <subcommand> [flags]
```

#### Subcommands
- `paths` - Show all active paths and environment variables
- `show` - Display current configuration
- `init` - Initialize default configuration file
- `edit` - Open configuration in editor

#### Flags (init)
- `--force` - Overwrite existing configuration

#### Examples
```bash
# View all paths
srake config paths

# Initialize configuration
srake config init

# Edit configuration
srake config edit

# Show current config
srake config show
```

### `srake cache`

Cache management commands for controlling disk usage.

```bash
srake cache <subcommand> [flags]
```

#### Subcommands
- `info` - Show cache information and sizes
- `clean` - Remove cache files

#### Flags (clean)
- `--all` - Remove all cache including indices
- `--older <duration>` - Remove files older than duration (e.g., 30d, 24h)
- `--search` - Remove search result cache
- `--downloads` - Remove downloaded files
- `--index` - Remove search index (requires rebuild)

#### Examples
```bash
# View cache usage
srake cache info

# Clean downloads older than 30 days
srake cache clean --older 30d

# Remove all downloads
srake cache clean --downloads

# Clean everything (with confirmation)
srake cache clean --all
```

---

## Output Formats

Most commands support multiple output formats:

- **table** (default) - Human-readable table with colors
- **json** - JSON format for programmatic use
- **yaml** - YAML format
- **csv** - Comma-separated values
- **tsv** - Tab-separated values
- **xml** - XML format (convert command only)

## Environment Variables

### Path Configuration
- `SRAKE_CONFIG_HOME` - Override config directory (default: `~/.config/srake`)
- `SRAKE_DATA_HOME` - Override data directory (default: `~/.local/share/srake`)
- `SRAKE_CACHE_HOME` - Override cache directory (default: `~/.cache/srake`)
- `SRAKE_STATE_HOME` - Override state directory (default: `~/.local/state/srake`)
- `SRAKE_DB_PATH` - Override database path (default: `~/.local/share/srake/srake.db`)
- `SRAKE_INDEX_PATH` - Override search index path (default: `~/.cache/srake/index`)
- `SRAKE_MODELS_PATH` - Override models directory for embeddings

### Search Configuration
- `SRAKE_MODEL_VARIANT` - Model variant for embeddings: full|quantized (default: full)
- `SRAKE_DEFAULT_LIMIT` - Default search result limit
- `SRAKE_SEARCH_MODE` - Default search mode: database|fts|hybrid|vector

### Output Control
- `NO_COLOR` - Disable colored output globally
- `SRAKE_NO_COLOR` - Disable colored output for srake
- `SRAKE_DEBUG` - Enable debug output
- `SRAKE_VERBOSE` - Enable verbose output

### Cloud Configuration
- `AWS_REGION` - Affects download source auto-selection
- `GCP_PROJECT` - Affects download source auto-selection

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Command line usage error
- `130` - Interrupted (Ctrl+C)