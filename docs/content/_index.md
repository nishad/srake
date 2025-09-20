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
  Process NCBI SRA metadata archives with streaming, filtering, and resume capabilities.<br />
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
    title="Quality-Controlled Search"
    subtitle="Multiple search modes with similarity thresholds, confidence scoring, and vector embeddings"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
    image="images/hextra-search.webp"
    imageClass="hx-top-[40%] hx-left-[24px] hx-w-[180%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(194,97,254,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Comprehensive Filtering"
    subtitle="Filter by organism, platform, library details, date ranges, and sequencing metrics"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-lg:hx-min-h-[340px]"
    image="images/hextra-markdown.webp"
    imageClass="hx-top-[40%] hx-left-[36px] hx-w-[180%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(142,53,74,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Aggregation & Analytics"
    subtitle="Group results by field, get counts, and analyze metadata distributions"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
    image="images/hextra-doc.webp"
    imageClass="hx-top-[40%] hx-left-[36px] hx-w-[110%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(221,210,59,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="RESTful API & MCP"
    subtitle="HTTP API with OpenAPI spec and Model Context Protocol for AI assistant integration"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-lg:hx-min-h-[340px]"
    image="images/hextra-theme.webp"
    imageClass="hx-top-[40%] hx-left-[36px] hx-w-[110%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(120,119,198,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Streaming Architecture"
    subtitle="Process 14GB+ archives with minimal memory using zero-copy streaming"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-lg:hx-min-h-[340px]"
    image="images/hextra-theme.webp"
    imageClass="hx-top-[40%] hx-left-[36px] hx-w-[110%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(24,188,156,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Resume & Recovery"
    subtitle="Intelligent checkpoint system for resuming interrupted operations"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
    image="images/hextra-theme.webp"
    imageClass="hx-top-[40%] hx-left-[36px] hx-w-[110%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(34,184,207,0.15),hsla(0,0%,100%,0));"
  >}}
{{< /hextra/feature-grid >}}

## Quick Start with SRAKE {.hx-mt-16 .hx-mb-8}

{{< tabs items="Install,Ingest,Index,Search,API" >}}
  {{< tab >}}
  ```bash
  # Using Go
  go install github.com/nishad/srake/cmd/srake@latest

  # Using Homebrew
  brew tap nishad/srake
  brew install srake

  # Using Docker
  docker pull ghcr.io/nishad/srake:latest
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Auto-select and ingest
  srake ingest --auto

  # With filters
  srake ingest --file archive.tar.gz \
    --taxon-ids 9606 \
    --platforms ILLUMINA \
    --strategies RNA-Seq
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Build search index
  srake index --build --progress

  # Build with vector embeddings
  srake index --build --with-embeddings

  # Verify index
  srake index --stats
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Quality-controlled search
  srake search "breast cancer" \
    --similarity-threshold 0.7 \
    --show-confidence

  # Vector semantic search
  srake search "tumor gene expression" \
    --search-mode vector

  # Export results
  srake search "RNA-Seq" --format json
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Start API server
  srake server --port 8082 \
    --enable-cors \
    --enable-mcp

  # Test API
  curl "http://localhost:8082/api/v1/search?\
  query=cancer&similarity_threshold=0.7"
  ```
  {{< /tab >}}
{{< /tabs >}}


## Learn More {.hx-mt-16 .hx-mb-8}

{{< cards >}}
  {{< card link="docs" title="Documentation" icon="book-open" subtitle="Complete guides and references" >}}
  {{< card link="docs/getting-started" title="Getting Started" subtitle="Install and run in minutes" >}}
  {{< card link="docs/features" title="Features" icon="sparkles" subtitle="Explore all capabilities" >}}
  {{< card link="docs/api" title="API Reference" subtitle="REST API and Go library" >}}
{{< /cards >}}

</div>