---
title: Advanced Search
weight: 10
---

# Advanced Search Capabilities

SRake provides powerful full-text search with advanced query syntax, comprehensive filtering, and aggregation capabilities that match or exceed leading SRA tools.

## Quick Start

```bash
# Build the search index first
srake search index --build

# Basic search
srake search "homo sapiens"

# Advanced query
srake search "organism:human AND library_strategy:RNA-Seq" --advanced

# With filters
srake search --organism "homo sapiens" --platform ILLUMINA
```

## Search Features

### Full-Text Search with Bleve

SRake uses the [Bleve](https://blevesearch.com/) search engine for lightning-fast local search:

- **No network dependency**: All searches run locally
- **Sub-second response**: Most queries complete in <50ms
- **Automatic indexing**: Index updates during data ingestion
- **Fuzzy matching**: Tolerates typos and variations

### Advanced Query Syntax

Enable advanced queries with the `--advanced` flag:

#### Boolean Operators

```bash
# AND operator (both conditions must match)
srake search "organism:human AND library_strategy:RNA-Seq" --advanced

# OR operator (either condition matches)
srake search "platform:ILLUMINA OR platform:PACBIO" --advanced

# NOT operator (exclude matches)
srake search "cancer NOT organism:mouse" --advanced

# Complex combinations
srake search "(organism:human OR organism:mouse) AND library_strategy:RNA-Seq NOT cell_type:hela" --advanced
```

#### Field-Specific Searches

Target specific metadata fields:

```bash
# Search in specific fields
srake search "title:breast cancer" --advanced
srake search "organism:\"homo sapiens\"" --advanced
srake search "platform:ILLUMINA" --advanced

# Field aliases for convenience
srake search "org:human lib:RNA-Seq plat:ILLUMINA" --advanced
```

Available field aliases:
- `org` → organism
- `plat` → platform
- `lib` → library_strategy
- `strat` → library_strategy
- `inst` → instrument_model
- `acc` → accession

#### Wildcards

Use `*` for multiple characters, `?` for single character:

```bash
# Wildcard searches
srake search "RNA*" --advanced          # Matches RNA-Seq, RNA-seq, RNAseq
srake search "hum?n" --advanced         # Matches human, humin
srake search "SRP*123" --advanced       # Matches any SRP ending in 123
```

#### Range Queries

Search numeric and date ranges:

```bash
# Numeric ranges
srake search "spots:[1000000 TO 5000000]" --advanced
srake search "bases:[* TO 1000000000]" --advanced
srake search "spots:[1000000 TO *]" --advanced

# Combined with other queries
srake search "organism:human AND spots:[1000000 TO *]" --advanced
```

#### Phrase Searches

Exact phrase matching with quotes:

```bash
# Phrase searches
srake search "\"breast cancer\"" --advanced
srake search "\"single cell RNA sequencing\"" --advanced
```

## Comprehensive Filtering

### Biological Filters

```bash
# Organism
srake search --organism "homo sapiens"
srake search --organism "mus musculus"

# Platform and instrument
srake search --platform ILLUMINA
srake search --platform OXFORD_NANOPORE
srake search --instrument "Illumina HiSeq 2500"

# Library details
srake search --library-strategy RNA-Seq
srake search --library-strategy ChIP-Seq
srake search --library-source TRANSCRIPTOMIC
srake search --library-selection RANDOM
srake search --library-layout PAIRED

# Study type
srake search --study-type "Transcriptome Analysis"
```

### Date Range Filters

```bash
# Date range filtering
srake search --date-from 2023-01-01 --date-to 2023-12-31
srake search --date-from 2022-06-01
srake search --date-to 2024-01-01
```

### Sequencing Metrics

```bash
# Filter by spots (reads)
srake search --spots-min 1000000
srake search --spots-min 1000000 --spots-max 10000000

# Filter by bases
srake search --bases-min 1000000000  # 1 billion bases
srake search --bases-max 50000000000 # 50 billion bases
```

### Combining Filters

```bash
# Complex filter combinations
srake search "cancer" \
  --organism "homo sapiens" \
  --platform ILLUMINA \
  --library-strategy RNA-Seq \
  --spots-min 1000000 \
  --date-from 2023-01-01

# All RNA-Seq experiments from human samples with high coverage
srake search \
  --organism "homo sapiens" \
  --library-strategy RNA-Seq \
  --library-layout PAIRED \
  --spots-min 10000000
```

## Aggregation & Analytics

### Count Results

```bash
# Get total count only
srake search "cancer" --count-only
# Output: 42156

# Count with filters
srake search --organism "homo sapiens" --platform ILLUMINA --count-only

# JSON output for scripts
srake search "RNA-Seq" --count-only --format json
```

### Aggregate by Field

```bash
# Group by organism
srake search "RNA-Seq" --aggregate-by organism

# Output:
# Aggregation by organism
# ──────────────────────────────────────────────────
# Homo sapiens                         15234
# Mus musculus                         8921
# Rattus norvegicus                    3456
# ...

# Aggregate by platform
srake search --aggregate-by platform

# Aggregate by library strategy
srake search "cancer" --aggregate-by library_strategy
```

### Faceted Search

```bash
# Show facets (counts by category)
srake search "human" --facets

# Facets with JSON output for analysis
srake search "cancer" --facets --format json
```

## Output Formats

### Table (Default)

```bash
srake search "homo sapiens" --limit 10
```

### JSON

```bash
# Pretty-printed JSON
srake search "cancer" --format json

# Pipe to jq for processing
srake search "RNA-Seq" --format json | jq '.hits[].fields.organism' | sort | uniq -c
```

### CSV/TSV

```bash
# CSV format
srake search "human" --format csv > results.csv

# TSV format
srake search "mouse" --format tsv > results.tsv

# Without headers
srake search "cancer" --format csv --no-header
```

### Accession List

```bash
# Just accession numbers (useful for batch downloads)
srake search "RNA-Seq human" --format accession > accessions.txt

# Pipe to other commands
srake search "single cell" --format accession | head -20 | xargs srake metadata
```

### Custom Fields

```bash
# Select specific fields
srake search "human" --fields "accession,organism,platform,spots"

# Export specific fields as CSV
srake search "RNA-Seq" --fields "accession,title,organism" --format csv
```

## Search Modes

### Fuzzy Search

Tolerates typos and variations:

```bash
# Fuzzy search for typo tolerance
srake search "hmuan" --fuzzy        # Finds "human"
srake search "cancre" --fuzzy       # Finds "cancer"
srake search "rnaseq" --fuzzy       # Finds "RNA-Seq"
```

### Exact Matching

```bash
# Require exact phrase matching
srake search "RNA sequencing" --exact
```

## Performance Options

### Pagination

```bash
# Pagination for large result sets
srake search "human" --limit 100 --offset 0    # First 100
srake search "human" --limit 100 --offset 100  # Next 100
srake search "human" --limit 100 --offset 200  # Next 100
```

### Export to File

```bash
# Save results to file
srake search "cancer" --output results.json --format json
srake search "RNA-Seq" --output results.csv --format csv

# Large exports
srake search "human" --limit 10000 --output human_studies.json --format json
```

### Search Index Management

```bash
# Build or rebuild index
srake search index --build
srake search index --rebuild

# Custom batch size for large databases
srake search index --build --batch-size 1000

# Parallel indexing
srake search index --build --workers 4

# Verify index integrity
srake search index --verify

# Show index statistics
srake search index --stats
```

## Comparison with Other Tools

| Feature | SRake | SRAdb | pysradb | ffq | NCBI E-utilities |
|---------|-------|-------|---------|-----|------------------|
| Full-text search | ✅ Fast | ✅ Slow | ✅ Medium | ❌ | ✅ Online |
| Boolean operators | ✅ | ❌ | ❌ | ❌ | ✅ |
| Field queries | ✅ | ⚠️ | ⚠️ | ❌ | ✅ |
| Wildcards | ✅ | ⚠️ | ❌ | ❌ | ⚠️ |
| Range queries | ✅ | ❌ | ❌ | ❌ | ✅ |
| Fuzzy search | ✅ | ❌ | ❌ | ❌ | ❌ |
| Aggregations | ✅ | ⚠️ | ✅ | ❌ | ⚠️ |
| Offline mode | ✅ | ✅ | ❌ | ❌ | ❌ |
| Response time | <50ms | >1s | ~500ms | N/A | >1s |
| Rate limits | None | None | None | None | Yes |

## Examples by Use Case

### Finding RNA-Seq Studies

```bash
# All human RNA-Seq studies
srake search "organism:human AND library_strategy:RNA-Seq" --advanced

# High-quality RNA-Seq (>10M reads)
srake search --library-strategy RNA-Seq --spots-min 10000000

# Recent RNA-Seq studies
srake search --library-strategy RNA-Seq --date-from 2023-01-01
```

### Cancer Research

```bash
# All cancer studies
srake search "cancer OR tumor OR tumour OR carcinoma" --advanced

# Breast cancer specifically
srake search "\"breast cancer\" OR BRCA1 OR BRCA2" --advanced

# Human cancer RNA-Seq
srake search "cancer AND organism:human AND library_strategy:RNA-Seq" --advanced
```

### Single-Cell Studies

```bash
# Single-cell RNA-Seq
srake search "\"single cell\" OR scRNA-Seq OR \"10x Genomics\"" --advanced

# Recent single-cell studies
srake search "single cell" --date-from 2023-01-01 --platform ILLUMINA
```

### Quality Control

```bash
# Find low-quality samples
srake search --spots-max 100000

# Find high-coverage WGS
srake search --library-strategy WGS --bases-min 30000000000

# Platform-specific searches
srake search --platform PACBIO --library-strategy WGS
```

## Tips & Tricks

1. **Build index after ingestion**: Always rebuild the search index after ingesting new data
2. **Use field queries**: More precise than general text search
3. **Combine filters**: Multiple filters use AND logic
4. **Export for analysis**: Use JSON format with jq for complex analysis
5. **Check facets**: Use `--facets` to understand data distribution
6. **Fuzzy for variants**: Use fuzzy search for organism names and technical terms

## API Access

The search functionality is also available via the REST API:

```bash
# Start the server
srake server --port 8080

# Search via API
curl "localhost:8080/api/search?q=human&limit=10"

# With filters
curl "localhost:8080/api/search?q=cancer&organism=homo+sapiens&platform=ILLUMINA"

# JSON response
curl -H "Accept: application/json" "localhost:8080/api/search?q=RNA-Seq"
```

## Troubleshooting

### Index not found
```bash
# Build the index
srake search index --build
```

### Slow searches
```bash
# Rebuild index with optimization
srake search index --rebuild --batch-size 1000
```

### No results
```bash
# Try fuzzy search
srake search "your query" --fuzzy

# Check available data
srake search --stats
```