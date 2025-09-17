---
title: Filtering System
weight: 1
---

srake includes a comprehensive filtering system that allows you to process only the data you need, saving time, storage, and computational resources.

## Overview

The filtering system operates during the streaming pipeline, applying filters before data is inserted into the database. This approach ensures:

- **Memory Efficiency**: Filters are applied during streaming without loading entire datasets
- **Early Rejection**: Unwanted records are discarded before expensive database operations
- **Minimal Overhead**: < 5% performance impact when filtering is enabled
- **Real-time Statistics**: Track filtering effectiveness as processing occurs

## Available Filters

### Taxonomy Filtering

Filter by NCBI taxonomy IDs to focus on specific organisms:

```bash
# Single taxonomy ID (e.g., human: 9606)
srake ingest --file archive.tar.gz --taxon-ids 9606

# Multiple taxonomy IDs
srake ingest --file archive.tar.gz --taxon-ids 9606,10090,7955

# Exclude specific taxonomy IDs
srake ingest --file archive.tar.gz --exclude-taxon-ids 32630,2697049

# Combine include and exclude
srake ingest --file archive.tar.gz \
  --taxon-ids 9606,10090 \
  --exclude-taxon-ids 562
```

Common taxonomy IDs:
- `9606` - Homo sapiens (human)
- `10090` - Mus musculus (mouse)
- `7955` - Danio rerio (zebrafish)
- `7227` - Drosophila melanogaster
- `562` - Escherichia coli

### Organism Name Filtering

Filter by scientific names when you don't know the taxonomy IDs:

```bash
# Single organism
srake ingest --file archive.tar.gz --organisms "homo sapiens"

# Multiple organisms
srake ingest --file archive.tar.gz \
  --organisms "homo sapiens,mus musculus,rattus norvegicus"
```

### Date Range Filtering

Filter by submission or publication dates:

```bash
# Data from 2024 only
srake ingest --file archive.tar.gz \
  --date-from 2024-01-01 \
  --date-to 2024-12-31

# Data from last 90 days
srake ingest --file archive.tar.gz \
  --date-from 2024-10-01

# Data up to a specific date
srake ingest --file archive.tar.gz \
  --date-to 2024-06-30
```

### Platform Filtering

Filter by sequencing platform:

```bash
# Illumina data only
srake ingest --file archive.tar.gz --platforms ILLUMINA

# Multiple platforms
srake ingest --file archive.tar.gz \
  --platforms ILLUMINA,OXFORD_NANOPORE,PACBIO_SMRT
```

Available platforms:
- `ILLUMINA`
- `OXFORD_NANOPORE`
- `PACBIO_SMRT`
- `ION_TORRENT`
- `LS454`
- `ABI_SOLID`
- `COMPLETE_GENOMICS`

### Library Strategy Filtering

Filter by sequencing library strategy:

```bash
# RNA-Seq data only
srake ingest --file archive.tar.gz --strategies RNA-Seq

# Multiple strategies
srake ingest --file archive.tar.gz \
  --strategies RNA-Seq,WGS,WES,ChIP-Seq
```

Common strategies:
- `RNA-Seq` - RNA sequencing
- `WGS` - Whole genome sequencing
- `WES` / `WXS` - Whole exome sequencing
- `ChIP-Seq` - Chromatin immunoprecipitation sequencing
- `ATAC-Seq` - Assay for transposase-accessible chromatin
- `Bisulfite-Seq` - Bisulfite sequencing
- `Hi-C` - Chromosome conformation capture
- `AMPLICON` - Amplicon sequencing

### Quality Filtering

Filter by read count or base count thresholds:

```bash
# Minimum 10 million reads
srake ingest --file archive.tar.gz --min-reads 10000000

# Between 10-100 million reads
srake ingest --file archive.tar.gz \
  --min-reads 10000000 \
  --max-reads 100000000

# Minimum 1 billion bases
srake ingest --file archive.tar.gz --min-bases 1000000000
```

## Combining Filters

Filters can be combined to create precise selection criteria:

```bash
# Human RNA-Seq data from 2024 with high quality
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --strategies RNA-Seq \
  --platforms ILLUMINA \
  --date-from 2024-01-01 \
  --min-reads 10000000

# Mouse and human WGS data, excluding E. coli contamination
srake ingest --file archive.tar.gz \
  --taxon-ids 9606,10090 \
  --exclude-taxon-ids 562 \
  --strategies WGS \
  --platforms ILLUMINA,OXFORD_NANOPORE
```

## Stats-Only Mode

Preview what would be imported without actually inserting data:

```bash
# Preview filtering results
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --platforms ILLUMINA \
  --stats-only

# Output:
# Filter Statistics:
#   Total Processed: 1000000
#   Total Matched:   45234 (4.5%)
#   Total Skipped:   954766
#
# Skip Reasons:
#   By Taxonomy:  850000
#   By Platform:  104766
#
# Unique Records Matched:
#   Studies:     234
#   Experiments: 5678
#   Samples:     4321
#   Runs:        35001
```

## Filter Statistics

When filters are active, srake provides detailed statistics:

```bash
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --filter-verbose

# Real-time output:
# Progress: 23.4% | Matched: 12,456/234,567 | Skipped: 222,111
#
# Final statistics:
# Filter Statistics:
#   Total Processed: 1,000,000
#   Total Matched:   45,234 (4.5%)
#   Total Skipped:   954,766
#
# Skip Reasons:
#   By Taxonomy:  850,000
#   By Date:      50,000
#   By Platform:  30,000
#   By Reads:     24,766
```

## Performance Considerations

### Filter Order

Filters are applied in an optimized order for best performance:
1. Date filters (studies)
2. Taxonomy filters (samples)
3. Platform/strategy filters (experiments)
4. Quality filters (runs)

### Memory Usage

The filtering system maintains constant memory usage:
- Filter options: < 1KB
- Statistics tracking: < 10MB
- No additional buffers needed

### Processing Speed

Filter performance characteristics:
- **No filters**: 20,000+ records/second
- **With filters**: 19,000+ records/second (< 5% overhead)
- **Stats-only mode**: 25,000+ records/second

## Filter Profiles (Coming Soon)

Save and reuse filter configurations:

```yaml
# filters/human-rnaseq.yaml
taxonomy_ids: [9606]
strategies: [RNA-Seq]
platforms: [ILLUMINA]
min_reads: 10000000
date_from: 2024-01-01
```

```bash
# Use saved profile
srake ingest --file archive.tar.gz --filter-profile filters/human-rnaseq.yaml
```

## Use Cases

### Research-Specific Filtering

```bash
# Cancer research data
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --strategies RNA-Seq,WGS,WES \
  --min-reads 10000000

# Microbiome studies
srake ingest --file archive.tar.gz \
  --strategies AMPLICON,WGS \
  --platforms ILLUMINA \
  --exclude-taxon-ids 9606,10090

# Single-cell RNA-seq
srake ingest --file archive.tar.gz \
  --taxon-ids 9606,10090 \
  --strategies RNA-Seq \
  --min-reads 1000000
```

### Data Management

```bash
# Recent high-quality data only
srake ingest --file archive.tar.gz \
  --date-from 2024-01-01 \
  --min-reads 10000000 \
  --platforms ILLUMINA

# Specific research center data
srake ingest --file archive.tar.gz \
  --centers "Broad Institute,Sanger Institute"
```

## Best Practices

1. **Start with Stats-Only**: Use `--stats-only` to preview filter effectiveness
2. **Be Specific**: Combine multiple filters for precise data selection
3. **Monitor Progress**: Use `--filter-verbose` to track filtering in real-time
4. **Save Configurations**: Document successful filter combinations for reuse
5. **Consider Resources**: Filtering reduces database size and query time

## Troubleshooting

### No Matches Found

If filters result in no matches:
- Check taxonomy IDs are correct
- Verify date format (YYYY-MM-DD)
- Ensure platform/strategy names match exactly
- Try broader criteria first, then narrow down

### Unexpected Results

If filtering produces unexpected results:
- Use `--stats-only` to preview
- Check filter statistics with `--filter-verbose`
- Verify organism names match scientific nomenclature
- Review skip reasons in the statistics output

## Next Steps

- Learn about [Resume Capability](/docs/features/resume)
- Explore [Performance Optimizations](/docs/features/performance)
- See [Real-World Examples](/docs/examples)