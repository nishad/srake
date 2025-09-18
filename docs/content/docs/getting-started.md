---
title: Getting Started
weight: 1
prev: /docs
next: /docs/features
---

{{< callout type="info" >}}
**Quick Setup**: srake can be installed in less than a minute and process your first SRA metadata in seconds!
{{< /callout >}}

## Installation

srake provides multiple installation methods to suit your needs:

{{< tabs items="Go,Docker,Binary,Source" >}}

  {{< tab >}}
  **Requirements**: Go 1.19 or later

  ```bash
  go install github.com/nishad/srake/cmd/srake@latest
  ```

  Verify the installation:
  ```bash
  srake --version
  ```
  {{< /tab >}}

  {{< tab >}}
  **Run in container**:

  ```bash
  # Pull the image
  docker pull ghcr.io/nishad/srake:latest

  # Run with volume mount
  docker run -v $(pwd)/data:/data \
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

### Ingest SRA Metadata

Let srake automatically select and download the appropriate archive:

```bash
srake ingest --auto
```

{{< callout type="tip" >}}
The `--auto` flag intelligently selects between daily updates or full datasets based on your database state
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

### Search Your Data

**Simple Search**:
```bash
srake search "homo sapiens"
```

**Filtered Search**:
```bash
srake search "cancer" \
  --organism "homo sapiens" \
  --platform ILLUMINA
```

**Export Results**:
```bash
srake search "RNA-Seq" \
  --format json \
  --output results.json
```

### Start API Server

Launch the REST API server for programmatic access:

```bash
# Start server on default port 8080
srake server

# Custom port
srake server --port 3000
```

{{< callout type="info" >}}
**API Access**: Query via `curl "http://localhost:8080/api/search?q=human&limit=10"`
{{< /callout >}}

{{< /steps >}}

## Filtering Options

{{< callout type="success" >}}
**Performance**: Filtering adds < 5% overhead while reducing database size by up to 99%
{{< /callout >}}

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
  {{< card link="/docs/features/filtering" title="Filtering System" icon="funnel" subtitle="Learn advanced filtering techniques" />}}
  {{< card link="/docs/features/resume" title="Resume Capability" icon="arrow-path" subtitle="Handle large files reliably" />}}
  {{< card link="/docs/api" title="API Reference" icon="code-bracket" subtitle="Programmatic access guide" />}}
  {{< card link="/docs/examples" title="Examples" icon="academic-cap" subtitle="Real-world use cases" />}}
{{< /cards >}}

## Getting Help

{{< callout type="info" >}}
Need assistance? Check these resources:
- 📚 [FAQ](/docs/faq)
- 💬 [GitHub Discussions](https://github.com/nishad/srake/discussions)
- 🐛 [Report Issues](https://github.com/nishad/srake/issues/new)
- 📖 [Full Documentation](/docs)
{{< /callout >}}