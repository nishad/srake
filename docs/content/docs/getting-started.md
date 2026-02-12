---
title: Getting Started
weight: 1
prev: /docs
next: /docs/features
---

## Installation

### Build from source

Requires Go 1.25+ with CGO enabled (for SQLite).

```bash
git clone https://github.com/nishad/srake.git
cd srake
go build -tags "sqlite_fts5" -o srake ./cmd/srake
./srake --help
```

### Go install

```bash
go install -tags "sqlite_fts5" github.com/nishad/srake/cmd/srake@latest
```

{{< callout type="info" >}}
The `sqlite_fts5` build tag is required for full-text search support.
{{< /callout >}}

## Quick Start

{{< steps >}}

### Ingest SRA metadata

```bash
# Auto-select the best archive from NCBI
srake ingest --auto

# Or ingest a local file
srake ingest --file /path/to/NCBI_SRA_Metadata.tar.gz

# Ingest with filters (human Illumina RNA-Seq only)
srake ingest --auto \
  --taxon-ids 9606 \
  --platforms ILLUMINA \
  --strategies RNA-Seq
```

### Build a search index

```bash
# Build Bleve full-text index
srake index --build

# Optionally include vector embeddings for semantic search
srake index --build --with-embeddings
```

### Search

```bash
# Basic search
srake search "homo sapiens"

# Search with filters
srake search "cancer" \
  --organism "homo sapiens" \
  --platform ILLUMINA \
  --library-strategy RNA-Seq

# Semantic vector search
srake search "tumor gene expression" --search-mode vector

# Export results as JSON
srake search "RNA-Seq" --format json --output results.json
```

### Start the API server

```bash
srake server --port 8080

# Query the API
curl "http://localhost:8080/api/v1/search?q=cancer&limit=10"
curl "http://localhost:8080/api/v1/stats"
```

{{< /steps >}}

## Data paths

SRAKE follows the XDG Base Directory Specification:

| Directory | Default Path | Purpose |
|-----------|-------------|---------|
| Config | `~/.config/srake/` | Configuration file |
| Data | `~/.local/share/srake/` | Database and models |
| Cache | `~/.cache/srake/` | Downloads and search index |
| State | `~/.local/state/srake/` | Resume checkpoints |

Override with environment variables:

```bash
SRAKE_DB_PATH=/fast/ssd/srake.db srake ingest --auto
SRAKE_CACHE_HOME=/tmp/srake srake index --build
```

See [Configuration](/docs/reference/configuration) for the full list.

## Next steps

{{< cards >}}
  {{< card link="/docs/reference/cli" title="CLI Reference" subtitle="All commands and flags" >}}
  {{< card link="/docs/features" title="Features" subtitle="Search, filtering, export" >}}
  {{< card link="/docs/api" title="API Reference" subtitle="REST API endpoints" >}}
{{< /cards >}}
