---
title: "SRAmetadb Export"
weight: 35
---

# SRAmetadb Export

Export your srake database to the classic SRAmetadb.sqlite format for compatibility with existing bioinformatics tools and workflows.

## Overview

The export feature creates a SQLite database that matches the original SRAmetadb schema, allowing seamless integration with tools and pipelines that expect the traditional format. This bridges the gap between srake's modern architecture and legacy systems.

## Key Features

### Features at a Glance

- **Dual FTS Support**: Choose between FTS3 (legacy compatibility) and FTS5 (modern performance)
- **Complete Schema Mapping**: Maps srake's modern schema to classic SRAmetadb tables
- **Compression Support**: Optional gzip compression for exported databases
- **Batch Processing**: Configurable batch sizes for efficient memory usage
- **Progress Tracking**: Real-time progress updates for large exports
- **Flexible Output**: Export to any location with custom naming

## Quick Start

### Basic Export

Export your srake database to SRAmetadb format:

```bash
# Export with modern FTS5 (recommended)
srake db export -o SRAmetadb.sqlite

# Export with FTS3 for legacy tool compatibility
srake db export -o SRAmetadb.sqlite --fts-version 3
```

### Export Options

```bash
# Export from specific database
srake db export --db /path/to/srake.db -o output.sqlite

# Export with compression
srake db export -o SRAmetadb.sqlite.gz --compress

# Large dataset with custom batch size
srake db export -o SRAmetadb.sqlite --batch-size 50000

# Force overwrite existing file
srake db export -o SRAmetadb.sqlite --force
```

## Output Schema

The exported database contains all the standard SRAmetadb tables and structures:

### Core Tables

| Table | Description |
|-------|-------------|
| **study** | Research studies and projects |
| **experiment** | Sequencing experiments |
| **sample** | Biological samples |
| **run** | Sequencing runs |
| **submission** | Data submissions |
| **sra** | Denormalized table joining all data |

### Additional Components

- **Full-text Search**: `sra_ft` virtual table for text searching
- **Metadata**: `metaInfo` table with version and creation information
- **Documentation**: `col_desc` table with column descriptions

## FTS Version Comparison

### FTS5 (Default - Recommended)

```bash
srake db export -o modern.sqlite --fts-version 5
```

**Advantages:**
- 🚀 Faster search performance
- 💾 Smaller index size
- 🌍 Better Unicode support
- 🔍 Advanced search features
- 📊 Built-in ranking functions

**Best for:**
- New projects
- Modern analysis pipelines
- Performance-critical applications

### FTS3 (Legacy Compatibility)

```bash
srake db export -o legacy.sqlite --fts-version 3
```

**Advantages:**
- ✅ 100% compatibility with older tools
- 📦 Works with legacy R packages
- 🔧 No modifications needed for existing scripts

**Best for:**
- Existing pipelines requiring FTS3
- Legacy bioinformatics tools
- Older R/Bioconductor packages

## Integration Examples

### With R/Bioconductor

```r
# Using SRAdb package
library(SRAdb)

# Load exported database
sqlfile <- "SRAmetadb.sqlite"
sra_con <- dbConnect(SQLite(), sqlfile)

# Query as usual
rs <- dbGetQuery(sra_con,
  "SELECT * FROM sra WHERE study_accession = 'SRP000001'")

# Full-text search
results <- dbGetQuery(sra_con,
  "SELECT * FROM sra_ft WHERE sra_ft MATCH 'cancer AND RNA-Seq'")
```

### With Python

```python
import sqlite3
import pandas as pd

# Connect to exported database
conn = sqlite3.connect('SRAmetadb.sqlite')

# Query with pandas
query = """
SELECT study_accession, study_title, organism
FROM study
WHERE organism = 'Homo sapiens'
"""
df = pd.read_sql_query(query, conn)

# Full-text search
fts_query = """
SELECT * FROM sra_ft
WHERE sra_ft MATCH 'single cell'
LIMIT 100
"""
results = pd.read_sql_query(fts_query, conn)
```

### Command Line

```bash
# Query exported database
sqlite3 SRAmetadb.sqlite "SELECT COUNT(*) FROM sra"

# Full-text search
sqlite3 SRAmetadb.sqlite \
  "SELECT run_accession FROM sra_ft WHERE sra_ft MATCH 'ChIP-Seq'"

# Export results to CSV
sqlite3 -header -csv SRAmetadb.sqlite \
  "SELECT * FROM experiment WHERE platform = 'ILLUMINA'" > results.csv
```

## Advanced Usage

### Automated Export Pipeline

Create a script for regular exports:

```bash
#!/bin/bash
# export_pipeline.sh

# Set paths
SRAKE_DB="/data/srake/srake.db"
EXPORT_DIR="/data/exports"
DATE=$(date +%Y%m%d)

# Export with both FTS versions
srake db export \
  --db "$SRAKE_DB" \
  -o "$EXPORT_DIR/SRAmetadb_fts5_$DATE.sqlite" \
  --fts-version 5 \
  --batch-size 50000

srake db export \
  --db "$SRAKE_DB" \
  -o "$EXPORT_DIR/SRAmetadb_fts3_$DATE.sqlite" \
  --fts-version 3 \
  --batch-size 50000

# Compress for archival
gzip "$EXPORT_DIR/SRAmetadb_fts5_$DATE.sqlite"
gzip "$EXPORT_DIR/SRAmetadb_fts3_$DATE.sqlite"

echo "Export completed: $DATE"
```

### Performance Optimization

For large databases, optimize the export process:

```bash
# Use larger batch size for better performance
srake db export -o output.sqlite --batch-size 100000

# Export without progress bar for scripts
srake db export -o output.sqlite --no-progress --quiet

# Monitor memory usage during export
srake db export -o output.sqlite --debug
```

### Validation

Verify the exported database:

```bash
# Check table counts
sqlite3 SRAmetadb.sqlite <<EOF
SELECT 'Studies:', COUNT(*) FROM study;
SELECT 'Experiments:', COUNT(*) FROM experiment;
SELECT 'Samples:', COUNT(*) FROM sample;
SELECT 'Runs:', COUNT(*) FROM run;
SELECT 'SRA Records:', COUNT(*) FROM sra;
EOF

# Test FTS functionality
sqlite3 SRAmetadb.sqlite \
  "SELECT COUNT(*) FROM sra_ft WHERE sra_ft MATCH 'test'"

# Verify schema
sqlite3 SRAmetadb.sqlite ".schema study"
```

## Schema Mapping

The export process handles the mapping between srake's modern schema and the classic SRAmetadb format:

### Field Transformations

| srake Field | SRAmetadb Field | Transformation |
|------------|-----------------|----------------|
| JSON arrays | Pipe-delimited strings | `["A","B"]` → `"A\|B"` |
| Nested metadata | Flattened columns | Extracted to individual fields |
| Modern timestamps | Legacy date format | ISO 8601 → `YYYY-MM-DD HH:MM:SS` |
| Missing fields | Default values | Populated with appropriate defaults |

### Data Integrity

- **Relationships preserved**: Foreign keys maintained between tables
- **Accessions unchanged**: All SRA accessions remain identical
- **Metadata complete**: All essential metadata included
- **URLs generated**: SRA and Entrez links created automatically

## Troubleshooting

### Common Issues

{{< callout type="info" >}}
**Large Database Export**

For databases over 10GB, use larger batch sizes and consider disabling progress:
```bash
srake db export -o output.sqlite --batch-size 100000 --no-progress
```
{{< /callout >}}

{{< callout type="warning" >}}
**Memory Usage**

Export process requires approximately 2x the batch size in RAM. Adjust batch size if encountering memory issues.
{{< /callout >}}

{{< callout type="tip" >}}
**Verification**

Always verify the export completed successfully:
```bash
sqlite3 output.sqlite "SELECT COUNT(*) FROM sra"
```
{{< /callout >}}

### Error Messages

| Error | Solution |
|-------|----------|
| `database not found` | Ensure source database exists and path is correct |
| `output file exists` | Use `--force` to overwrite or choose different name |
| `invalid FTS version` | Use either 3 or 5 for `--fts-version` |
| `out of memory` | Reduce `--batch-size` value |

## Best Practices

1. **Choose appropriate FTS version**
   - Use FTS5 for new projects
   - Use FTS3 only when required for compatibility

2. **Optimize batch size**
   - Start with default (10000)
   - Increase for better performance if RAM allows
   - Decrease if encountering memory issues

3. **Regular exports**
   - Automate exports after data updates
   - Keep versioned exports for reproducibility
   - Compress old exports to save space

4. **Validation**
   - Always verify record counts after export
   - Test FTS functionality
   - Check critical queries work as expected

## Next Steps

- **[API Access](/docs/api)**: Learn about programmatic database queries
- **[Automation Guide](/docs/automation)**: Explore scripted workflows
- **[Examples](/docs/examples)**: View real-world use cases
- **[CLI Reference](/docs/reference/cli#srake-db-export)**: Complete command documentation