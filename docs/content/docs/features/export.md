---
title: SRAmetadb Export
weight: 35
---

# SRAmetadb Export

Export the SRAKE database to the classic SRAmetadb.sqlite format for use with existing bioinformatics tools.

## Basic usage

```bash
# Export with FTS5 (recommended)
srake db export -o SRAmetadb.sqlite

# Export with FTS3 for legacy tool compatibility
srake db export -o SRAmetadb.sqlite --fts-version 3
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `-o, --output` | SRAmetadb.sqlite | Output file path |
| `--db` | auto-detected | Source database path |
| `--fts-version` | 5 | FTS version (3 or 5) |
| `--batch-size` | 10000 | Records per batch |
| `--compress` | false | Gzip compress output |
| `-f, --force` | false | Overwrite existing file |

## Output schema

The exported database contains:

| Table | Description |
|-------|-------------|
| `study` | Research studies |
| `experiment` | Sequencing experiments |
| `sample` | Biological samples |
| `run` | Sequencing runs |
| `submission` | Data submissions |
| `sra` | Denormalized join of all tables |
| `sra_ft` | Full-text search virtual table (FTS3 or FTS5) |
| `metaInfo` | Version and creation metadata |
| `col_desc` | Column descriptions |

## FTS version choice

**FTS5** (default): Faster queries, smaller index, better Unicode support. Use for new projects.

**FTS3**: Use when tools specifically require FTS3 (e.g., older R/Bioconductor SRAdb package).

{{< callout type="info" >}}
FTS5 support requires building SRAKE with the `sqlite_fts5` build tag.
{{< /callout >}}

## Usage with R

```r
library(DBI)
con <- dbConnect(RSQLite::SQLite(), "SRAmetadb.sqlite")
dbGetQuery(con, "SELECT * FROM study WHERE organism = 'Homo sapiens' LIMIT 10")
dbGetQuery(con, "SELECT * FROM sra_ft WHERE sra_ft MATCH 'cancer AND RNA-Seq'")
```

## Usage with Python

```python
import sqlite3
import pandas as pd

conn = sqlite3.connect("SRAmetadb.sqlite")
df = pd.read_sql("SELECT * FROM study WHERE organism = 'Homo sapiens'", conn)
```

## Schema mapping

The export handles these transformations:

- JSON arrays to pipe-delimited strings (`["A","B"]` to `A|B`)
- Nested metadata flattened to columns
- Missing legacy fields populated with defaults
- SRA/Entrez URLs generated automatically
