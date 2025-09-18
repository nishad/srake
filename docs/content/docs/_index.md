---
title: Documentation
linkTitle: Docs
cascade:
  type: docs
---

Welcome to the **srake** documentation! This guide will help you get started with processing NCBI SRA metadata efficiently.

## What is srake?

srake is a tool for processing and querying NCBI SRA (Sequence Read Archive) metadata. Built with a streaming architecture, srake can process large compressed archives without intermediate storage.

## Key Features

{{< cards >}}
  {{< card link="getting-started" title="Getting Started" subtitle="Install and run srake in minutes" >}}
  {{< card link="features/filtering" title="Filtering System" subtitle="Process only the data you need" >}}
  {{< card link="features/resume" title="Resume Capability" subtitle="Handle interruptions gracefully" >}}
  {{< card link="api" title="API Reference" subtitle="REST API and Go library" >}}
{{< /cards >}}

## Features

- **Performance**: Efficient record processing
- **Memory Management**: Streaming architecture for large files
- **Pipeline**: HTTP → Gzip → Tar → XML → Database streaming
- **Filtering**: Filter by taxonomy, organism, platform, and more
- **Resume Support**: Recovery from interruptions
- **Search**: Full-text search with SQLite backend

## Quick Example

```bash
# Install srake
go install github.com/nishad/srake/cmd/srake@latest

# Ingest SRA metadata with filters
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --platforms ILLUMINA \
  --strategies RNA-Seq

# Search the database
srake search "homo sapiens" --limit 10

# Start API server
srake server --port 8080
```
