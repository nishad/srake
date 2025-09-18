---
title: srake 🍶🧬
layout: hextra-home
---

{{< hextra/hero-badge >}}
  <div class="hx-w-2 hx-h-2 hx-rounded-full hx-bg-primary-400"></div>
  <span>Free, open source</span>
  {{< icon name="arrow-circle-right" attributes="height=14" >}}
{{< /hextra/hero-badge >}}

<div class="hx-mt-6 hx-mb-6">
{{< hextra/hero-headline >}}
  SRA Metadata&nbsp;<br class="sm:hx-block hx-hidden" />Processing Tool
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-subtitle >}}
  Process NCBI SRA metadata archives with&nbsp;<br class="sm:hx-block hx-hidden" />
  streaming, filtering, and resume capabilities
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx-mb-6">
{{< hextra/hero-button text="Get Started" link="docs/getting-started" >}}
{{< hextra/hero-button text="View on GitHub" link="https://github.com/nishad/srake" style="secondary" >}}
</div>

<div class="hx-mt-6"></div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Accession Conversion"
    subtitle="Convert between SRA, GEO, BioProject, and BioSample identifiers seamlessly"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
    image="images/hextra-doc.webp"
    imageClass="hx-top-[40%] hx-left-[24px] hx-w-[180%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(194,97,254,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Multi-Source Downloads"
    subtitle="Download from FTP, AWS, GCP with parallel transfers and Aspera support"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-lg:hx-min-h-[340px]"
    image="images/hextra-markdown.webp"
    imageClass="hx-top-[40%] hx-left-[36px] hx-w-[180%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(142,53,74,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Relationship Queries"
    subtitle="Navigate between studies, experiments, samples, and runs effortlessly"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
    image="images/hextra-search.webp"
    imageClass="hx-top-[40%] hx-left-[36px] hx-w-[110%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(221,210,59,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Smart Filtering"
    subtitle="Filter by taxonomy, platform, strategy, date ranges, and quality metrics"
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

## Quick Start {.hx-mt-12}

{{< tabs items="Install,Ingest,Search" >}}
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
  # Search the database
  srake search "homo sapiens"

  # Start API server
  srake server --port 8080

  # Query via API
  curl "localhost:8080/api/search?q=human"
  ```
  {{< /tab >}}
{{< /tabs >}}


## Learn More

{{< cards >}}
  {{< card link="docs" title="Documentation" icon="book-open" subtitle="Complete guides and references" >}}
  {{< card link="docs/getting-started" title="Getting Started" subtitle="Install and run in minutes" >}}
  {{< card link="docs/features" title="Features" icon="sparkles" subtitle="Explore all capabilities" >}}
  {{< card link="docs/api" title="API Reference" subtitle="REST API and Go library" >}}
{{< /cards >}}