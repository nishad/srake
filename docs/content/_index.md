---
title: SRAKE - SRA Knowledge Engine
layout: hextra-home
---

<div class="hx:mx-auto hx:max-w-screen-xl hx:px-4 hx:py-8">

{{< hextra/hero-badge >}}
  <div class="hx-w-2 hx-h-2 hx-rounded-full hx-bg-primary-400"></div>
  <span>Free, open source</span>
  {{< icon name="arrow-circle-right" attributes="height=14" >}}
{{< /hextra/hero-badge >}}

<div class="hx-mt-12 hx-mb-10">
{{< hextra/hero-headline >}}
  SRA Knowledge Engine
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-subtitle >}}
  Process and query NCBI SRA metadata locally with streaming ingestion, full-text search, and vector similarity.<br />
  <em style="font-size: 0.9em; opacity: 0.9;">Pronounced like Japanese sake (酒)</em>
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx-mt-10 hx-mb-16">
{{< hextra/hero-button text="Get Started" link="docs/getting-started" >}}
{{< hextra/hero-button text="View on GitHub" link="https://github.com/nishad/srake" style="secondary" >}}
</div>

<div class="hx-mt-20 hx-mb-8"></div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Streaming Ingestion"
    subtitle="Process 14GB+ NCBI archives with minimal memory. HTTP to Gzip to Tar to XML to SQLite in a single pass."
  >}}
  {{< hextra/feature-card
    title="Multi-Backend Search"
    subtitle="Bleve full-text, SQLite FTS5, and SapBERT vector similarity search with configurable hybrid ranking."
  >}}
  {{< hextra/feature-card
    title="Metadata Filtering"
    subtitle="Filter by organism, platform, library strategy, date range, and sequencing metrics during ingestion and search."
  >}}
  {{< hextra/feature-card
    title="REST API & MCP"
    subtitle="HTTP API for programmatic access. Model Context Protocol support for AI assistant integration."
  >}}
  {{< hextra/feature-card
    title="SRAmetadb Export"
    subtitle="Export to classic SRAmetadb.sqlite format with FTS3 or FTS5 for compatibility with existing R and Python tools."
  >}}
  {{< hextra/feature-card
    title="XDG Compliant"
    subtitle="Follows XDG Base Directory Specification. All paths configurable via environment variables."
  >}}
{{< /hextra/feature-grid >}}

## Quick Start {.hx-mt-16 .hx-mb-8}

{{< tabs items="Install,Ingest,Search,Server" >}}
  {{< tab >}}
  ```bash
  # Build from source (requires Go 1.25+ and CGO)
  go install -tags "sqlite_fts5" github.com/nishad/srake/cmd/srake@latest
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Ingest latest SRA metadata
  srake ingest --auto

  # Ingest with filters
  srake ingest --auto \
    --taxon-ids 9606 \
    --platforms ILLUMINA \
    --strategies RNA-Seq
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Build search index
  srake index --build

  # Search
  srake search "breast cancer RNA-Seq"

  # Search with filters
  srake search "cancer" \
    --organism "homo sapiens" \
    --platform ILLUMINA
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Start API server with MCP
  srake server --port 8080

  # Query the API
  curl "http://localhost:8080/api/v1/search?q=cancer&limit=10"
  ```
  {{< /tab >}}
{{< /tabs >}}

## Documentation {.hx-mt-16 .hx-mb-8}

{{< cards >}}
  {{< card link="docs/getting-started" title="Getting Started" subtitle="Installation and first steps" >}}
  {{< card link="docs/reference/cli" title="CLI Reference" icon="terminal" subtitle="All commands and flags" >}}
  {{< card link="docs/features" title="Features" icon="sparkles" subtitle="Search, filtering, export" >}}
  {{< card link="docs/api" title="API Reference" subtitle="REST API endpoints" >}}
{{< /cards >}}

</div>
