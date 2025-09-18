---
title: Documentation
linkTitle: Docs
cascade:
  type: docs
---

Welcome to the **srake** documentation! This guide will help you get started with processing NCBI SRA metadata efficiently.

## What is srake?

srake (pronounced "ess-RAH-keh" like Japanese sake 酒) is a blazing-fast, memory-efficient tool for processing and querying NCBI SRA (Sequence Read Archive) metadata. Built with a zero-copy streaming architecture, srake can process multi-gigabyte compressed archives without intermediate storage.

## Key Features

{{< cards >}}
  {{< card link="getting-started" title="Getting Started" icon="rocket" subtitle="Install and run srake in minutes" >}}
  {{< card link="features/filtering" title="Filtering System" icon="funnel" subtitle="Process only the data you need" >}}
  {{< card link="features/resume" title="Resume Capability" icon="arrow-path" subtitle="Handle interruptions gracefully" >}}
  {{< card link="api" title="API Reference" icon="code-bracket" subtitle="REST API and Go library" >}}
{{< /cards >}}

## Why srake?

- **🚀 Performance**: Process 20,000+ records per second
- **💾 Memory Efficient**: Constant < 500MB memory usage
- **🔄 Zero-Copy**: Direct HTTP → Gzip → Tar → XML → Database streaming
- **📊 Smart Filtering**: Filter by taxonomy, organism, platform, and more
- **✅ Resume Support**: Intelligent recovery from interruptions
- **🔍 Full-Text Search**: Query with optimized SQLite backend

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
