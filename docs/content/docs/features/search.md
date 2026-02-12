---
title: Search
weight: 10
---

SRAKE provides multi-backend search over local SRA metadata.

## Search backends

| Backend | Description | Use case |
|---------|-------------|----------|
| **Bleve** | Full-text search with BM25 ranking | General queries |
| **SQLite FTS5** | FTS5 virtual tables for high-volume data | Large datasets |
| **Vector** | SapBERT cosine similarity | Semantic/concept search |
| **Hybrid** | Weighted combination of text + vector | Default mode |

## Building the index

```bash
# Build Bleve index
srake index --build

# Include vector embeddings
srake index --build --with-embeddings

# Check index status
srake index --stats
```

## Basic search

```bash
srake search "homo sapiens"
srake search "breast cancer RNA-Seq"
```

## Search modes

```bash
# Full-text (Bleve BM25)
srake search "cancer" --search-mode text

# Vector similarity (SapBERT)
srake search "tumor gene expression" --search-mode vector

# Hybrid (default)
srake search "cancer" --search-mode hybrid

# Database only (SQLite)
srake search "SRP123456" --search-mode database
```

## Advanced query syntax

Enable with `--advanced`:

```bash
# Boolean operators
srake search "organism:human AND library_strategy:RNA-Seq" --advanced

# NOT
srake search "cancer NOT organism:mouse" --advanced

# Phrase search
srake search "\"breast cancer\"" --advanced

# Wildcards
srake search "RNA*" --advanced
```

Field aliases: `org` (organism), `plat` (platform), `lib`/`strat` (library_strategy), `inst` (instrument_model), `acc` (accession).

## Filtering

Combine text search with metadata filters:

```bash
srake search "cancer" \
  --organism "homo sapiens" \
  --platform ILLUMINA \
  --library-strategy RNA-Seq \
  --spots-min 1000000 \
  --date-from 2023-01-01
```

## Quality control

```bash
# Set similarity threshold for vector results
srake search "cancer" --similarity-threshold 0.7

# Show confidence scores
srake search "cancer" --show-confidence

# Adjust hybrid weight (0.0=text only, 1.0=vector only)
srake search "cancer" --hybrid-weight 0.5

# Fuzzy search (tolerates typos)
srake search "hmuan" --fuzzy
```

## Output formats

```bash
srake search "cancer" --format json
srake search "cancer" --format csv --output results.csv
srake search "cancer" --format tsv
srake search "cancer" --format accession  # accession numbers only
srake search "cancer" --fields "accession,organism,platform"
```

## Pagination

```bash
srake search "human" --limit 100 --offset 0
srake search "human" --limit 100 --offset 100
```

## Facets

```bash
srake search "cancer" --facets --format json
```

## API access

Search is also available via the REST API:

```bash
curl "http://localhost:8080/api/v1/search?q=cancer&limit=10"
curl "http://localhost:8080/api/v1/search?q=RNA-Seq&organism=homo+sapiens"
```

See [API Reference](/docs/api) for details.
