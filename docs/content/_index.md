---
title: SRAKE
layout: hextra-home
---

{{< hextra/hero-badge >}}
  <div class="hx:w-2 hx:h-2 hx:rounded-full hx:bg-primary-400"></div>
  <span>Free, open source</span>
  {{< icon name="arrow-circle-right" attributes="height=14" >}}
{{< /hextra/hero-badge >}}

<div class="hx:mt-6 hx:mb-6">
{{< hextra/hero-headline >}}
  SRA Knowledge Engine
{{< /hextra/hero-headline >}}
</div>

<div class="hx:mb-12">
{{< hextra/hero-subtitle >}}
  Process and query NCBI SRA metadata locally&nbsp;<br class="hx:sm:block hx:hidden" />with streaming ingestion, full-text search, and vector similarity
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx:mb-6">
{{< hextra/hero-button text="Get Started" link="docs/getting-started" >}}
</div>

<div class="hx:mt-6"></div>

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
    subtitle="Export to classic SRAmetadb.sqlite format for compatibility with existing R and Python tools."
  >}}
  {{< hextra/feature-card
    title="XDG Compliant"
    subtitle="Follows XDG Base Directory Specification. All paths configurable via environment variables."
  >}}
{{< /hextra/feature-grid >}}
