---
title: Examples
weight: 5
prev: /docs/features
next: /docs/api
---

# Examples and Use Cases

Real-world examples demonstrating common workflows with srake, including pipeline composition and automation patterns.

## Research Workflows

### Finding RNA-Seq Data for a Specific Organism

Find all human RNA-Seq experiments published in 2024:

```bash
# Build search index first
srake search index --build

# Search with advanced query syntax
srake search "organism:human AND library_strategy:RNA-Seq" --advanced \
  --date-from 2024-01-01 \
  --limit 100 \
  --format csv \
  --output human_rna_seq_2024.csv

# Search for specific disease studies
srake search "breast cancer AND organism:\"homo sapiens\" AND library_strategy:RNA-Seq" \
  --advanced \
  --spots-min 10000000 \
  --format json \
  --output breast_cancer_studies.json
```

### Using Advanced Search Features

Leverage boolean operators and field-specific queries:

```bash
# Complex boolean queries
srake search "(cancer OR tumor) AND organism:human NOT cell_type:hela" --advanced

# Wildcard searches
srake search "RNA* AND platform:ILLUMINA" --advanced

# Range queries for high-coverage data
srake search "spots:[10000000 TO *] AND bases:[1000000000 TO *]" --advanced

# Fuzzy search for typo tolerance
srake search "transciptome" --fuzzy  # Finds "transcriptome"
```

### Aggregation and Analytics

Analyze metadata distributions:

```bash
# Count studies by organism
srake search "RNA-Seq" --aggregate-by organism

# Get faceted results
srake search "cancer" --facets --format json | \
  jq '.facets.platform' | head -20

# Count total matching records
srake search "single cell" --count-only

# Group by library strategy
srake search --organism "homo sapiens" --aggregate-by library_strategy
```

### Downloading Data for a Published Study

Download all FASTQ files for a study mentioned in a paper:

```bash
# Convert GEO accession from paper to SRA
srake convert GSE123456 --to SRP

# Get all runs for the study
srake runs SRP123456 --format json --output runs.json

# Download all runs in parallel
srake download SRP123456 \
  --type fastq \
  --source aws \
  --parallel 4 \
  --output ./fastq_files/
```

### Cross-referencing Multiple Studies

Compare samples across different studies:

```bash
# Get samples from multiple studies
srake samples SRP001 --format json > study1_samples.json
srake samples SRP002 --format json > study2_samples.json
srake samples SRP003 --format json > study3_samples.json

# Or in batch
for study in SRP001 SRP002 SRP003; do
  srake samples $study --detailed --format json > ${study}_samples.json
done
```

## Unix Pipeline Integration

### Composable Commands

srake commands can be chained together using Unix pipes:

```bash
# Find experiments → Get runs → Download
srake search "CRISPR" --format tsv --no-header | \
  cut -f1 | \
  xargs -I {} srake runs {} --format tsv --no-header | \
  cut -f1 | \
  srake download --type fastq

# Convert accessions in bulk
cat geo_accessions.txt | \
  srake convert --to SRP --format tsv --no-header | \
  cut -f2 > sra_projects.txt

# Chain multiple conversions
echo "GSE123456" | \
  srake convert --to SRP | \
  grep SRP | \
  srake runs --format json
```

### Stream Processing

Process large datasets without intermediate files:

```bash
# Real-time filtering and conversion
srake search "RNA-Seq" --format tsv --no-header | \
  awk '$3 > 1000000' | \
  cut -f1 | \
  while read acc; do
    srake convert $acc --to GSE --quiet
  done

# Parallel processing with xargs
srake search "mouse" --limit 100 --format tsv --no-header | \
  cut -f1 | \
  xargs -P 4 -I {} srake metadata {} --format json --quiet
```

## Data Discovery

### Finding Related Experiments

Discover all experiments related to a sample:

```bash
# Start with a sample accession
SAMPLE="SRS123456"

# Get all experiments for this sample
srake experiments $SAMPLE --detailed

# Get the parent study
srake studies $SAMPLE

# Get all other samples in the study
STUDY=$(srake studies $SAMPLE --format json | jq -r '.[0].study_accession')
srake samples $STUDY --detailed
```

### Exploring Platform-Specific Data

Find all Oxford Nanopore sequencing data:

```bash
# Ingest only Nanopore data
srake ingest --auto \
  --platforms OXFORD_NANOPORE

# Search for specific applications
srake search "metagenome" \
  --platform OXFORD_NANOPORE \
  --limit 50
```

## Batch Operations

### Converting a List of Accessions

Convert multiple accessions from a publication supplementary table:

```bash
# Create accession list
cat > geo_accessions.txt << EOF
GSE111111
GSE222222
GSE333333
GSE444444
EOF

# Batch convert to SRA projects
srake convert --batch geo_accessions.txt \
  --to SRP \
  --format json \
  --output sra_projects.json

# Extract just the SRP IDs
cat sra_projects.json | jq -r '.[] | select(.error == null) | .targets[]' > srp_list.txt
```

### Bulk Download with Filtering

Download only high-quality runs from multiple experiments:

```bash
# Get runs with quality metrics
srake runs SRP123456 --detailed --format json | \
  jq '.[] | select(.total_bases > 10000000000) | .run_accession' > \
  high_quality_runs.txt

# Download filtered runs
srake download --list high_quality_runs.txt \
  --source aws \
  --parallel 4 \
  --threads 2
```

## Integration Examples

### Building a Local Index

Create a searchable index of specific data types:

```bash
# 1. Ingest filtered data
srake ingest --auto \
  --organisms "homo sapiens,mus musculus" \
  --strategies "RNA-Seq,ChIP-Seq,ATAC-Seq" \
  --min-reads 10000000 \
  --date-from 2023-01-01

# 2. Export metadata for indexing
srake search "*" --limit 0 --format json > all_metadata.json

# 3. Start API server for queries
srake server --port 8080 &

# 4. Query via API
curl "http://localhost:8080/api/search?q=transcription+factor&limit=20"
```

### Creating a Download Queue

Generate and process a download queue:

```bash
#!/bin/bash
# download_queue.sh

# Get all RNA-Seq runs from 2024
srake search "RNA-Seq" \
  --format json \
  --date-from 2024-01-01 | \
  jq -r '.results[].accession' > rna_seq_2024.txt

# Process in batches of 10
split -l 10 rna_seq_2024.txt batch_

# Download each batch
for batch in batch_*; do
  echo "Processing $batch"
  srake download --list $batch \
    --parallel 2 \
    --output ./downloads/
  sleep 60  # Pause between batches
done
```

### Metadata Analysis Pipeline

Extract and analyze metadata for a research domain:

```bash
# Get all cancer-related studies
srake search "cancer" --format json --limit 1000 > cancer_studies.json

# Extract platform distribution
cat cancer_studies.json | \
  jq -r '.results[].platform' | \
  sort | uniq -c | sort -rn

# Get temporal distribution
cat cancer_studies.json | \
  jq -r '.results[].published' | \
  cut -d'-' -f1 | \
  sort | uniq -c
```

## Advanced Filtering

### Multi-criteria Filtering

Complex filtering for specific research needs:

```bash
# Ingest single-cell RNA-seq from human brain
srake ingest --auto \
  --taxon-ids 9606 \
  --organisms "homo sapiens" \
  --strategies "RNA-Seq" \
  --min-reads 100000000 \
  --filter-verbose

# Search with additional criteria
srake search "brain OR neuron OR glia" \
  --format json | \
  jq '.results[] | select(.title | test("single.cell|sc.?RNA|10x"; "i"))'
```

### Quality Control Pipeline

Filter and validate high-quality datasets:

```bash
# Function to check data quality
check_quality() {
  local accession=$1

  # Get run information
  srake runs $accession --format json | \
    jq -r '.[] | "\(.run_accession):\(.total_bases):\(.total_spots)"' | \
    while IFS=: read -r run bases spots; do
      if [ $bases -gt 10000000000 ]; then
        echo "$run PASS"
      else
        echo "$run FAIL"
      fi
    done
}

# Check multiple studies
for study in SRP001 SRP002 SRP003; do
  echo "Checking $study"
  check_quality $study
done
```

## Performance Optimization

### Parallel Processing

Maximize throughput with parallel operations:

```bash
# Parallel conversion
cat accessions.txt | \
  parallel -j 4 'srake convert {} --to GSE --format json > {}.json'

# Parallel metadata fetch
cat studies.txt | \
  parallel -j 8 'srake metadata {} --format json' > all_metadata.jsonl

# Parallel download with resource limits
nice -n 10 srake download --list large_dataset.txt \
  --parallel 4 \
  --threads 2 \
  --output /data/downloads/
```

### Incremental Updates

Keep your database current with minimal overhead:

```bash
#!/bin/bash
# daily_update.sh

# Check last update
LAST_UPDATE=$(srake db info | grep "Last update" | cut -d: -f2)

# Ingest only new data
srake ingest --daily \
  --date-from "$LAST_UPDATE" \
  --no-progress

# Log update
echo "$(date): Updated from $LAST_UPDATE" >> update.log
```

## Troubleshooting Examples

### Debugging Failed Downloads

```bash
# Dry run to check URLs
srake download SRR123456 --dry-run --verbose

# Test different sources
for source in ftp aws gcp ncbi; do
  echo "Testing $source"
  srake download SRR123456 \
    --source $source \
    --dry-run
done

# Use verbose mode for debugging
srake download SRR123456 \
  --verbose \
  --retry 5
```

### Handling Large Result Sets

```bash
# Paginate through large results
OFFSET=0
LIMIT=1000

while true; do
  COUNT=$(srake search "human" \
    --offset $OFFSET \
    --limit $LIMIT \
    --format json | \
    jq '.results | length')

  if [ $COUNT -eq 0 ]; then
    break
  fi

  # Process batch
  srake search "human" \
    --offset $OFFSET \
    --limit $LIMIT \
    --format json > batch_$OFFSET.json

  OFFSET=$((OFFSET + LIMIT))
done
```

## Automation Scripts

### Daily Report Generator

```bash
#!/bin/bash
# daily_report.sh

DATE=$(date +%Y-%m-%d)
REPORT="report_$DATE.html"

cat > $REPORT << EOF
<html>
<head><title>SRA Daily Report - $DATE</title></head>
<body>
<h1>SRA Daily Report</h1>
<p>Generated: $(date)</p>
EOF

# Database statistics
echo "<h2>Database Statistics</h2><pre>" >> $REPORT
srake db info >> $REPORT
echo "</pre>" >> $REPORT

# New studies today
echo "<h2>New Studies</h2><pre>" >> $REPORT
srake search "*" --date-from $DATE --format table >> $REPORT
echo "</pre>" >> $REPORT

echo "</body></html>" >> $REPORT

# Email report
mail -s "SRA Daily Report $DATE" \
  -a $REPORT \
  team@example.com < /dev/null
```

## Next Steps

- Review the [CLI Reference](/docs/reference/cli) for detailed command options
- Explore the [API Documentation](/docs/api) for programmatic access
- Check [Performance Tips](/docs/performance) for optimization strategies