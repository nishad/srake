---
title: Examples
weight: 5
---

# Examples

## Ingest human RNA-Seq data

```bash
srake ingest --auto \
  --taxon-ids 9606 \
  --platforms ILLUMINA \
  --strategies RNA-Seq \
  --min-reads 10000000
```

## Build search index with embeddings

```bash
srake index --build --with-embeddings --progress
```

## Search and export results

```bash
# Search for cancer studies, export as CSV
srake search "breast cancer" \
  --organism "homo sapiens" \
  --library-strategy RNA-Seq \
  --format csv \
  --output breast_cancer_rnaseq.csv

# Get accession list for downstream tools
srake search "single cell" \
  --platform ILLUMINA \
  --format accession \
  --output accessions.txt

# Semantic search for related concepts
srake search "tumor gene expression profiling" \
  --search-mode vector \
  --show-confidence \
  --format json
```

## Export for R/Bioconductor workflows

```bash
# Export to SRAmetadb format
srake db export -o SRAmetadb.sqlite

# Use with FTS3 for older SRAdb R package
srake db export -o SRAmetadb.sqlite --fts-version 3
```

Then in R:

```r
library(DBI)
con <- dbConnect(RSQLite::SQLite(), "SRAmetadb.sqlite")
dbGetQuery(con, "SELECT * FROM sra_ft WHERE sra_ft MATCH 'cancer RNA-Seq'")
```

## API server for analysis pipelines

```bash
# Start server
srake server --port 8080 &

# Query from scripts
curl -s "http://localhost:8080/api/v1/search?q=CRISPR&organism=homo+sapiens&limit=50" \
  | jq '.results[].accession'

# Get study metadata
curl -s "http://localhost:8080/api/v1/studies/SRP123456/metadata" | jq .

# Statistics
curl -s "http://localhost:8080/api/v1/stats/organisms" | jq '.organisms[:10]'
```

## Filtered ingestion for specific projects

```bash
# Model organisms only
srake ingest --auto \
  --taxon-ids 9606,10090,7955,7227,6239 \
  --date-from 2024-01-01

# Long-read sequencing data
srake ingest --auto \
  --platforms PACBIO_SMRT,OXFORD_NANOPORE \
  --strategies WGS \
  --min-bases 1000000000

# Preview what would be imported
srake ingest --auto \
  --taxon-ids 9606 \
  --strategies ChIP-Seq \
  --stats-only
```

## Resume interrupted ingestion

```bash
# Start a large ingestion
srake ingest --monthly --taxon-ids 9606

# If interrupted, run the same command again
srake ingest --monthly --taxon-ids 9606
# Automatically resumes from last checkpoint
```

## Use with Claude Desktop (MCP)

Add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "srake": {
      "command": "srake",
      "args": ["mcp"]
    }
  }
}
```

With a custom database path:

```json
{
  "mcpServers": {
    "srake": {
      "command": "srake",
      "args": ["mcp", "--db", "/path/to/srake.db"]
    }
  }
}
```

Once configured, Claude can search SRA metadata, retrieve study details, find similar
studies, and export results directly through the MCP tools.

## MCP over HTTP (remote/shared access)

Start the MCP server with HTTP transport for remote clients or shared lab infrastructure:

```bash
# Start on localhost:8081 (default)
srake mcp --transport http

# Start on a specific host/port for network access
srake mcp --transport http --host 0.0.0.0 --port 9090
```

Test the connection:

```bash
curl -X POST http://localhost:8081 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
```

## Custom database location

```bash
# Use fast SSD for database
SRAKE_DB_PATH=/mnt/nvme/srake.db srake ingest --auto

# Search from the same location
SRAKE_DB_PATH=/mnt/nvme/srake.db srake search "RNA-Seq"
```
