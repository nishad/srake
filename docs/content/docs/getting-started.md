---
title: Getting Started with SRAKE
weight: 1
prev: /docs
next: /docs/features
---

# Getting Started with SRAKE - SRA Knowledge Engine

*SRAKE pronunciation: Like Japanese sake (酒) — "srah-keh"*

## Installation

SRAKE (SRA Knowledge Engine) provides multiple installation methods to suit your needs:

{{< tabs items="Go,Docker,Binary,Source" >}}

  {{< tab >}}
  **Requirements**: Go 1.19 or later

  ```bash
  go install github.com/nishad/srake/cmd/srake@latest
  ```

  Verify the SRAKE installation:
  ```bash
  srake --version  # SRAKE - SRA Knowledge Engine
  ```
  {{< /tab >}}

  {{< tab >}}
  **Run in container**:

  ```bash
  # Pull the image
  docker pull ghcr.io/nishad/srake:latest

  # Run with volume mounts for persistence
  docker run -v ~/.local/share/srake:/data \
    -v ~/.cache/srake:/cache \
    ghcr.io/nishad/srake:latest \
    ingest --auto
  ```
  {{< /tab >}}

  {{< tab >}}
  **Download pre-built binaries**:

  ```bash
  # Linux/macOS
  wget https://github.com/nishad/srake/releases/latest/download/srake-$(uname -s)-$(uname -m).tar.gz
  tar -xzf srake-*.tar.gz
  sudo mv srake /usr/local/bin/
  ```

  Verify installation:
  ```bash
  srake --version
  ```
  {{< /tab >}}

  {{< tab >}}
  **Build from source**:

  ```bash
  git clone https://github.com/nishad/srake.git
  cd srake
  go build -o srake ./cmd/srake
  ./srake --help
  ```
  {{< /tab >}}

{{< /tabs >}}

## Quick Start

{{< steps >}}

### Check Installation

Verify srake is installed and check default paths:

```bash
# Check version
srake --version

# View configured paths
srake config paths
```

This will show where srake stores its data:
- **Database**: `~/.local/share/srake/srake.db`
- **Index**: `~/.cache/srake/index/srake.bleve`
- **Downloads**: `~/.cache/srake/downloads/`
- **Config**: `~/.config/srake/config.yaml`

### Ingest SRA Metadata

Let srake automatically select and download the appropriate archive:

```bash
srake ingest --auto
```

{{< callout type="tip" >}}
The `--auto` flag intelligently selects between daily updates or full datasets based on your database state.
Data is stored in `~/.local/share/srake/srake.db` by default. Use `SRAKE_DB_PATH` to override.
{{< /callout >}}

Alternative ingestion methods:

{{< tabs items="Daily,Monthly,Local File,Remote URL" >}}
  {{< tab >}}
  ```bash
  # Latest daily update
  srake ingest --daily
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Full monthly archive
  srake ingest --monthly
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Local archive file
  srake ingest --file /path/to/archive.tar.gz
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Direct from NCBI
  srake ingest --file https://ftp.ncbi.nlm.nih.gov/sra/reports/Metadata/archive.tar.gz
  ```
  {{< /tab >}}
{{< /tabs >}}

### Build Search Index

Before searching, build the search index using the root-level `index` command:

```bash
# Build initial index
srake index --build

# Build with progress display
srake index --build --progress

# Build with vector embeddings for semantic search
srake index --build --with-embeddings

# Verify index
srake index --stats
```

### Search Your Data

**Simple Search**:
```bash
srake search "homo sapiens"
```

**Quality-Controlled Search**:
```bash
# Search with similarity threshold
srake search "breast cancer" --similarity-threshold 0.7 --show-confidence

# Return only top results
srake search "RNA-Seq" --top-percentile 10

# Vector semantic search for related concepts
srake search "tumor gene expression" --search-mode vector
```

**Filtered Search**:
```bash
srake search "cancer" \
  --organism "homo sapiens" \
  --platform ILLUMINA \
  --library-strategy RNA-Seq \
  --limit 100
```

**Search Modes**:
```bash
# Database search (exact matching)
srake search "SRP123456" --search-mode database

# Full-text search (text matching)
srake search "breast cancer" --search-mode fts

# Hybrid search (best of both - default)
srake search "RNA-Seq human" --search-mode hybrid

# Vector search (semantic similarity)
srake search "metabolic pathway analysis" --search-mode vector
```

**Export Results**:
```bash
# JSON format
srake search "RNA-Seq" --format json --output results.json

# CSV with specific fields
srake search "cancer" --format csv --fields "accession,organism,platform"

# XML format
srake search "transcriptome" --format xml --output results.xml
```

### Convert Accessions

Convert between SRA, GEO, and BioProject identifiers:

```bash
# SRA to GEO
srake convert SRP123456 --to GSE

# GEO to SRA
srake convert GSM123456 --to SRX

# Batch conversion
srake convert SRP001 SRP002 --to GSE --format json
```

### Query Relationships

Navigate the relationships between SRA entities:

```bash
# Get all runs for a study
srake runs SRP123456

# Get samples for an experiment
srake samples SRX123456 --detailed

# Get parent study from any accession
srake studies SRR123456
```

### Download Data

Download SRA files from multiple sources:

```bash
# Basic download
srake download SRR123456

# Download from AWS
srake download SRR123456 --source aws --threads 4

# Download all runs for a study
srake download SRP123456 --type fastq --parallel 4
```

### Start API Server

Launch the REST API server with AI integration support:

```bash
# Start server with all features
srake server --port 8082 --enable-cors --enable-mcp

# Custom configuration
srake server --port 3000 --host 0.0.0.0 --enable-cors

# With specific database and index
SRAKE_DB_PATH=test.db SRAKE_INDEX_PATH=/tmp/index srake server
```

{{< callout type="info" >}}
**API Features**:
- RESTful API: `/api/v1/search`, `/api/v1/stats`, `/api/v1/export`
- MCP for AI assistants: `/mcp`, `/mcp/capabilities`
- Quality control: similarity thresholds, confidence scoring
- Multiple formats: JSON, CSV, TSV, XML

Example: `curl "http://localhost:8082/api/v1/search?query=cancer&similarity_threshold=0.7"`
{{< /callout >}}

### Manage Cache & Configuration

Control disk usage and customize settings:

```bash
# View cache usage
srake cache info

# Clean old downloads (free up space)
srake cache clean --older 30d

# Initialize configuration
srake config init

# Edit configuration
srake config edit

# Check all paths and environment variables
srake config paths
```

{{< callout type="tip" >}}
Use environment variables for custom paths:
- `SRAKE_DB_PATH=/fast/ssd/srake.db` - Use fast storage for database
- `SRAKE_CACHE_HOME=/tmp/srake` - Use temporary storage for cache
{{< /callout >}}

{{< /steps >}}

## Automation Features

srake follows [clig.dev](https://clig.dev) best practices for CLI design, making it perfect for automation:

### Non-Interactive Mode
```bash
# Use --yes flag to skip all prompts
srake ingest --auto --yes
srake download SRP123456 --yes
```

### Pipeline Composition
```bash
# Commands accept stdin for easy chaining
echo "SRP123456" | srake convert --to GSE
cat accessions.txt | srake download --parallel 4
srake search "RNA-Seq" | srake download --type fastq
```

### Dry Run & Debug
```bash
# Preview actions without executing
srake download SRP123456 --dry-run

# Enable debug output for troubleshooting
srake convert SRP123456 --to GSE --debug
```

### Structured Output
```bash
# Export in various formats for processing
srake search "human" --format json | jq '.results[].accession'
srake convert SRP123456 --to GSE --format csv > results.csv
```

See the [Automation Guide](/docs/automation) for more advanced scripting examples.

## Filtering Options

Filtering helps reduce database size by processing only the data you need.

### Filter by Taxonomy

{{< tabs items="Single Species,Multiple Species,Exclude Species" >}}
  {{< tab >}}
  ```bash
  # Human data only (taxonomy ID 9606)
  srake ingest --file archive.tar.gz \
    --taxon-ids 9606
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Human, mouse, and zebrafish
  srake ingest --file archive.tar.gz \
    --taxon-ids 9606,10090,7955
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Exclude viruses
  srake ingest --file archive.tar.gz \
    --exclude-taxon-ids 32630,2697049
  ```
  {{< /tab >}}
{{< /tabs >}}

### Filter by Date Range

```bash
# Data from 2024 only
srake ingest --file archive.tar.gz \
  --date-from 2024-01-01 \
  --date-to 2024-12-31
```

### Filter by Platform & Strategy

**Illumina RNA-Seq**:
```bash
srake ingest --file archive.tar.gz \
  --platforms ILLUMINA \
  --strategies RNA-Seq
```

**Oxford Nanopore WGS**:
```bash
srake ingest --file archive.tar.gz \
  --platforms OXFORD_NANOPORE \
  --strategies WGS
```

### Quality Filtering

```bash
# High-quality data only
srake ingest --file archive.tar.gz \
  --min-reads 10000000 \
  --min-bases 1000000000
```

{{< callout type="warning" >}}
**Preview Mode**: Use `--stats-only` to see what would be imported without actually inserting data
{{< /callout >}}

## Resume Capability

srake automatically tracks progress and can resume from interruptions:

{{< steps >}}

### Automatic Resume

```bash
# If interrupted, run the same command again
srake ingest --file archive.tar.gz

# Output:
# Previous ingestion found:
#   Progress: 45.3% complete
# Resume? (y/n): y
```

### Check Status

```bash
srake ingest --status

# Current Ingestion Status
# ────────────────────────
# Progress: 67.8% complete
# Records: 2,345,678
# ETA: 12 minutes
```

### Force Restart

```bash
# Start fresh (ignore existing progress)
srake ingest --file archive.tar.gz --force
```

{{< /steps >}}

## Database Management

**Database Info**:
```bash
srake db info

# Shows:
# • Database size
# • Table counts
# • Index status
```

**Custom Location**:
```bash
srake ingest \
  --file archive.tar.gz \
  --db /custom/path/db.sqlite
```

**Verbose Mode**:
```bash
srake ingest \
  --file archive.tar.gz \
  --verbose
```

## Configuration Options

### Performance Tuning

{{< tabs items="Checkpoints,Progress,Concurrency" >}}
  {{< tab >}}
  ```bash
  # Adjust checkpoint frequency
  srake ingest --file archive.tar.gz \
    --checkpoint 5000
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Disable progress bar
  srake ingest --file archive.tar.gz \
    --no-progress
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Set worker count
  srake ingest --file archive.tar.gz \
    --workers 8
  ```
  {{< /tab >}}
{{< /tabs >}}

## Next Steps

{{< cards >}}
  {{< card link="/docs/features/filtering" title="Filtering System" subtitle="Learn advanced filtering techniques" >}}
  {{< card link="/docs/features/resume" title="Resume Capability" subtitle="Handle large files reliably" >}}
  {{< card link="/docs/api" title="API Reference" subtitle="Programmatic access guide" >}}
  {{< card link="/docs/examples" title="Examples" subtitle="Real-world use cases" >}}
{{< /cards >}}

## Getting Help

{{< callout type="info" >}}
Need assistance? Check these resources:
- 📚 [FAQ](/docs/faq)
- 💬 [GitHub Discussions](https://github.com/nishad/srake/discussions)
- 🐛 [Report Issues](https://github.com/nishad/srake/issues/new)
- 📖 [Full Documentation](/docs)
{{< /callout >}}