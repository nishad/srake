---
title: Filtering System
weight: 1
prev: /docs/features
next: /docs/features/resume
---

The filtering system allows you to process only the data you need, reducing storage requirements.

## Overview

The filtering system operates during the streaming pipeline, applying filters before data is inserted into the database:

Key features:
- **Memory Efficient**: Filters applied during streaming
- **Early Rejection**: Unwanted records discarded before database operations
- **Statistics**: Track filtering effectiveness

## Filter Types

### Taxonomy Filtering

Filter by NCBI taxonomy IDs to focus on specific organisms:

{{< tabs items="Single Species,Multiple Species,Exclude Taxa,Combined" >}}
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
  # Exclude viruses and bacteria
  srake ingest --file archive.tar.gz \
    --exclude-taxon-ids 32630,2697049,562
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Mammals excluding E. coli contamination
  srake ingest --file archive.tar.gz \
    --taxon-ids 9606,10090 \
    --exclude-taxon-ids 562
  ```
  {{< /tab >}}
{{< /tabs >}}

{{< callout type="info" >}}
**Common Taxonomy IDs:**
- `9606` - Homo sapiens (human)
- `10090` - Mus musculus (mouse)
- `7955` - Danio rerio (zebrafish)
- `7227` - Drosophila melanogaster
- `562` - Escherichia coli
{{< /callout >}}

### Organism Name Filtering

Filter by scientific names when you don't know the taxonomy IDs:

```bash
# Single organism
srake ingest --file archive.tar.gz \
  --organisms "homo sapiens"

# Multiple organisms
srake ingest --file archive.tar.gz \
  --organisms "homo sapiens,mus musculus,rattus norvegicus"
```

### Date Range Filtering

Filter by submission or publication dates:

{{< tabs items="Year 2024,Last 90 Days,Historical Data" >}}
  {{< tab >}}
  ```bash
  srake ingest --file archive.tar.gz \
    --date-from 2024-01-01 \
    --date-to 2024-12-31
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  srake ingest --file archive.tar.gz \
    --date-from 2024-10-01
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  srake ingest --file archive.tar.gz \
    --date-to 2020-12-31
  ```
  {{< /tab >}}
{{< /tabs >}}

### Platform Filtering

{{< tabs items="Illumina,Long Reads,Multiple,All Platforms" >}}
  {{< tab >}}
  ```bash
  # Illumina data only
  srake ingest --file archive.tar.gz \
    --platforms ILLUMINA
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Long-read platforms
  srake ingest --file archive.tar.gz \
    --platforms OXFORD_NANOPORE,PACBIO_SMRT
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Multiple platforms
  srake ingest --file archive.tar.gz \
    --platforms ILLUMINA,ION_TORRENT
  ```
  {{< /tab >}}
  {{< tab >}}
  **Available Platforms:**
  - `ILLUMINA`
  - `OXFORD_NANOPORE`
  - `PACBIO_SMRT`
  - `ION_TORRENT`
  - `LS454`
  - `ABI_SOLID`
  - `COMPLETE_GENOMICS`
  {{< /tab >}}
{{< /tabs >}}

### Library Strategy Filtering

Filter by experimental strategy:

{{< tabs items="RNA-Seq,Genomics,Epigenomics,Multiple" >}}
  {{< tab >}}
  ```bash
  # RNA sequencing only
  srake ingest --file archive.tar.gz \
    --strategies RNA-Seq
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Whole genome sequencing
  srake ingest --file archive.tar.gz \
    --strategies WGS,WXS
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Epigenomics studies
  srake ingest --file archive.tar.gz \
    --strategies ChIP-Seq,ATAC-Seq,Bisulfite-Seq
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Multiple strategies
  srake ingest --file archive.tar.gz \
    --strategies RNA-Seq,WGS,ChIP-Seq
  ```
  {{< /tab >}}
{{< /tabs >}}

{{< callout type="info" >}}
**Common Strategies:**
- `RNA-Seq` - RNA sequencing
- `WGS` - Whole Genome Sequencing
- `WXS` - Whole Exome Sequencing
- `ChIP-Seq` - Chromatin IP
- `ATAC-Seq` - Chromatin accessibility
- `Bisulfite-Seq` - DNA methylation
- `Hi-C` - Chromosome conformation
{{< /callout >}}

### Quality Filtering

Filter by sequencing depth and quality:

{{< tabs items="High Coverage,Specific Range,Ultra Deep" >}}
  {{< tab >}}
  ```bash
  # Minimum 10M reads and 1GB bases
  srake ingest --file archive.tar.gz \
    --min-reads 10000000 \
    --min-bases 1000000000
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Between 5M and 50M reads
  srake ingest --file archive.tar.gz \
    --min-reads 5000000 \
    --max-reads 50000000
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Ultra-deep sequencing (10GB+ bases)
  srake ingest --file archive.tar.gz \
    --min-bases 10000000000
  ```
  {{< /tab >}}
{{< /tabs >}}

## Complex Filter Combinations

### Research-Specific Workflows

{{< tabs items="Cancer Research,Population Genetics,Microbiome,Single Cell" >}}
  {{< tab >}}
  ```bash
  # Human cancer RNA-Seq studies
  srake ingest --file archive.tar.gz \
    --taxon-ids 9606 \
    --strategies RNA-Seq,WGS \
    --date-from 2023-01-01 \
    --min-reads 20000000
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Population genomics data
  srake ingest --file archive.tar.gz \
    --taxon-ids 9606 \
    --strategies WGS \
    --platforms ILLUMINA \
    --min-bases 30000000000
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Microbiome studies
  srake ingest --file archive.tar.gz \
    --strategies AMPLICON,WGS \
    --platforms ILLUMINA,ION_TORRENT \
    --exclude-taxon-ids 9606,10090
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Single-cell RNA-Seq
  srake ingest --file archive.tar.gz \
    --taxon-ids 9606,10090 \
    --strategies RNA-Seq \
    --date-from 2022-01-01 \
    --platforms ILLUMINA
  ```
  {{< /tab >}}
{{< /tabs >}}

## Preview Mode

Test your filters without inserting data:

{{< steps >}}

### Run with --stats-only

```bash
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --platforms ILLUMINA \
  --strategies RNA-Seq \
  --stats-only
```

### Review Statistics

```
Filter Statistics
─────────────────
Total XML files: 150,234
Files matching filters: 12,456 (8.3%)
Records that would be inserted:
  Studies:     3,234
  Experiments: 12,456
  Samples:     11,234
  Runs:        15,678
Estimated database size: 1.2 GB
Processing time estimate: 15 minutes
```

### Adjust and Apply

```bash
# Apply the filters for real
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --platforms ILLUMINA \
  --strategies RNA-Seq
```

{{< /steps >}}

## Filter Processing

Filters are applied during the streaming pipeline, allowing efficient processing of large datasets without loading everything into memory.

## Filter Configuration Files

For complex, reusable filter sets, use YAML configuration:

{{< tabs items="config.yaml,Usage" >}}
  {{< tab >}}
  ```yaml
  # filters.yaml
  taxonomy:
    include: [9606, 10090]
    exclude: [562]

  platforms:
    - ILLUMINA
    - OXFORD_NANOPORE

  strategies:
    - RNA-Seq
    - WGS

  date:
    from: "2024-01-01"
    to: "2024-12-31"

  quality:
    min_reads: 10000000
    min_bases: 1000000000
  ```
  {{< /tab >}}
  {{< tab >}}
  ```bash
  # Use configuration file
  srake ingest --file archive.tar.gz \
    --filter-config filters.yaml

  # Override specific settings
  srake ingest --file archive.tar.gz \
    --filter-config filters.yaml \
    --date-from 2025-01-01
  ```
  {{< /tab >}}
{{< /tabs >}}

## Best Practices

### Tips for Effective Filtering

1. **Start with --stats-only** to preview results
2. **Use taxonomy filters** for targeted datasets
3. **Combine filters** for precise data selection
4. **Save configurations** for reproducible workflows
5. **Monitor statistics** to verify filter effectiveness

### Common Patterns

{{< cards >}}
  {{< card title="Model Organisms" icon="beaker" subtitle="Focus on well-studied species for comparative analysis" >}}
  {{< card title="Recent Data" icon="calendar" subtitle="Filter by date for the latest sequencing technologies" >}}
  {{< card title="High Quality" icon="shield-check" subtitle="Use quality filters for publication-ready datasets" >}}
  {{< card title="Technology Specific" subtitle="Filter by platform for consistent processing pipelines" >}}
{{< /cards >}}

## Troubleshooting

### Filters Not Working?

{{< steps >}}

### Verify Filter Syntax
```bash
# Check your command
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --verbose
```

### Check Statistics
```bash
# Use stats-only mode
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --stats-only
```

### Enable Verbose Output
```bash
# See detailed filtering decisions
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --verbose --log-level debug
```

{{< /steps >}}

## Next Steps

{{< cards >}}
  {{< card link="/docs/features/resume" title="Resume Capability" subtitle="Handle interruptions gracefully" >}}
  {{< card link="/docs/features/performance" title="Performance" icon="lightning-bolt" subtitle="Optimization techniques" >}}
  {{< card link="/docs/examples" title="Examples" icon="academic-cap" subtitle="Real-world filtering scenarios" >}}
{{< /cards >}}