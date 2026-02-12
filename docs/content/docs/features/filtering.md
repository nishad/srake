---
title: Filtering
weight: 1
---

Filters are applied during ingestion to control which records are imported into the database. This reduces database size and processing time.

## Taxonomy filters

```bash
# Human only (NCBI taxonomy ID 9606)
srake ingest --auto --taxon-ids 9606

# Multiple species
srake ingest --auto --taxon-ids 9606,10090,7955

# By organism name
srake ingest --auto --organisms "Homo sapiens"

# Exclude specific taxa
srake ingest --auto --exclude-taxon-ids 32630
```

## Platform and library filters

```bash
# Illumina RNA-Seq
srake ingest --auto --platforms ILLUMINA --strategies RNA-Seq

# Multiple platforms
srake ingest --auto --platforms ILLUMINA,PACBIO_SMRT

# Multiple strategies
srake ingest --auto --strategies RNA-Seq,WGS,ChIP-Seq
```

Supported platforms: ILLUMINA, PACBIO_SMRT, OXFORD_NANOPORE, ION_TORRENT, ABI_SOLID, LS454, BGISEQ, COMPLETE_GENOMICS, HELICOS, CAPILLARY.

## Date range filters

```bash
# Records from 2024 only
srake ingest --auto --date-from 2024-01-01 --date-to 2024-12-31

# Everything after a date
srake ingest --auto --date-from 2023-06-01
```

## Sequencing metrics

```bash
# Minimum 10 million reads
srake ingest --auto --min-reads 10000000

# Read count range
srake ingest --auto --min-reads 1000000 --max-reads 100000000

# Minimum 1 billion bases
srake ingest --auto --min-bases 1000000000
```

## Combining filters

All filters use AND logic:

```bash
srake ingest --auto \
  --taxon-ids 9606 \
  --platforms ILLUMINA \
  --strategies RNA-Seq \
  --date-from 2024-01-01 \
  --min-reads 10000000
```

## Preview mode

Use `--stats-only` to see what would be imported without actually inserting data:

```bash
srake ingest --auto --taxon-ids 9606 --stats-only
```

Use `--filter-verbose` to see per-record filter decisions during ingestion.
