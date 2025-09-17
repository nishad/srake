---
title: Getting Started
weight: 1
next: /docs/features
---

Get up and running with srake in minutes.

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/nishad/srake/releases).

```bash
# Linux/macOS - download and install
wget https://github.com/nishad/srake/releases/latest/download/srake-$(uname -s)-$(uname -m).tar.gz
tar -xzf srake-*.tar.gz
sudo mv srake /usr/local/bin/

# Verify installation
srake --version
```

### Using Go Install

If you have Go 1.19+ installed:

```bash
go install github.com/nishad/srake/cmd/srake@latest
```

### From Source

```bash
git clone https://github.com/nishad/srake.git
cd srake
go build -o srake ./cmd/srake
```

### Docker

```bash
# Pull the image
docker pull ghcr.io/nishad/srake:latest

# Run with local data directory
docker run -v $(pwd)/data:/data ghcr.io/nishad/srake:latest ingest --auto
```

## Quick Start

### 1. Ingest SRA Metadata

The first step is to ingest SRA metadata into your local database. srake can download and process files directly from NCBI.

#### Auto-select (Recommended)
```bash
# Auto-select best option from NCBI
srake ingest --auto
```
This intelligently selects between daily updates or full datasets based on your database state.

#### Daily Updates
```bash
# Ingest latest daily update
srake ingest --daily
```
Daily updates contain incremental changes from the last 24 hours.

#### Monthly Full Dataset
```bash
# Ingest monthly full dataset
srake ingest --monthly
```
Monthly archives contain the complete SRA metadata (14GB+ compressed).

#### Local File
```bash
# Ingest a local archive
srake ingest --file /path/to/archive.tar.gz
```
Process a previously downloaded archive file.

### 2. Search the Database

Once data is ingested, you can search it:

```bash
# Simple text search
srake search "homo sapiens"

# Search with filters
srake search "cancer" --organism "homo sapiens" --platform ILLUMINA

# Export results to JSON
srake search "RNA-Seq" --format json --output results.json
```

### 3. Start the API Server

For programmatic access, start the REST API server:

```bash
# Start server on default port 8080
srake server

# Custom port
srake server --port 3000
```

Then access the API:

```bash
# Search via API
curl "http://localhost:8080/api/search?q=human&limit=10"

# Get specific metadata
curl "http://localhost:8080/api/metadata/SRR12345678"
```

## Using Filters

srake supports powerful filtering during ingestion to process only the data you need:

### Taxonomy Filtering

```bash
# Human data only (taxonomy ID 9606)
srake ingest --file archive.tar.gz --taxon-ids 9606

# Multiple species
srake ingest --file archive.tar.gz --taxon-ids 9606,10090,7955

# Exclude certain organisms
srake ingest --file archive.tar.gz --exclude-taxon-ids 32630,2697049
```

### Date Range Filtering

```bash
# Data from 2024 only
srake ingest --file archive.tar.gz \
  --date-from 2024-01-01 \
  --date-to 2024-12-31
```

### Platform and Strategy Filtering

```bash
# Illumina RNA-Seq data only
srake ingest --file archive.tar.gz \
  --platforms ILLUMINA \
  --strategies RNA-Seq
```

### Quality Filtering

```bash
# High-quality data only
srake ingest --file archive.tar.gz \
  --min-reads 10000000 \
  --min-bases 1000000000
```

### Preview Mode

Use `--stats-only` to preview what would be imported without actually inserting data:

```bash
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --platforms ILLUMINA \
  --stats-only
```

## Resume from Interruption

srake automatically tracks progress and can resume from interruptions:

```bash
# If interrupted, simply run the same command again
srake ingest --file archive.tar.gz

# Output:
# Previous ingestion found:
#   Source: archive.tar.gz
#   Progress: 45.3% complete
# Resume from last position? (y/n): y

# Force fresh start
srake ingest --file archive.tar.gz --force

# Check status
srake ingest --status
```

## Database Management

```bash
# View database statistics
srake db info

# Output:
# Database Information
# ────────────────────
# Path: ./data/metadata.db
# Size: 2048.50 MB
# Tables:
#   studies:     50,234
#   experiments: 120,456
#   samples:     89,123
#   runs:        145,678
```

## Configuration

srake uses sensible defaults but can be configured via command-line flags:

```bash
# Custom database location
srake ingest --file archive.tar.gz --db /path/to/custom.db

# Disable progress bar
srake ingest --file archive.tar.gz --no-progress

# Verbose output
srake ingest --file archive.tar.gz --verbose

# Set checkpoint frequency
srake ingest --file archive.tar.gz --checkpoint 5000
```

## Next Steps

- Learn about the [Filtering System](/docs/features/filtering)
- Understand [Resume Capability](/docs/features/resume)
- Explore the [API Reference](/docs/api)
- See [Examples and Tutorials](/docs/examples)

## Getting Help

If you encounter issues:

1. Check the [FAQ](/docs/faq)
2. Search [existing issues](https://github.com/nishad/srake/issues)
3. Join the [discussion](https://github.com/nishad/srake/discussions)
4. Report bugs on [GitHub](https://github.com/nishad/srake/issues/new)