---
title: Tool Compatibility
weight: 8
prev: /docs/api
next: /docs/performance
---

# Compatibility with Other Tools

srake provides comprehensive functionality that matches and extends popular SRA metadata tools. This guide shows how srake commands map to equivalent operations in other tools.

## Feature Comparison Matrix

| Feature | srake | SRAdb | ffq | pysradb | MetaSRA |
|---------|-------|-------|-----|---------|---------|
| **Local Database** | ✅ | ✅ | ❌ | ✅ | ❌ |
| **Streaming Processing** | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Accession Conversion** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Multi-source Download** | ✅ | ✅ | ✅ | ✅ | ❌ |
| **Relationship Queries** | ✅ | ✅ | ❌ | ✅ | ❌ |
| **Batch Operations** | ✅ | ✅ | ✅ | ✅ | ❌ |
| **Resume Capability** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Filtering on Ingest** | ✅ | ❌ | ❌ | ❌ | ❌ |
| **REST API** | ✅ | ❌ | ❌ | ❌ | ✅ |
| **Aspera Support** | ✅ | ✅ | ❌ | ✅ | ❌ |

## Command Equivalents

### SRAdb (R Package) → srake

{{< tabs items="Conversion,Download,Search,Metadata" >}}

{{< tab >}}
**SRAdb:**
```r
# Convert SRP to GSE
sraConvert(in_acc = "SRP123456",
           out_type = "gse")

# Convert GSM to SRX
sraConvert(in_acc = "GSM123456",
           out_type = "srx")
```

**srake:**
```bash
# Convert SRP to GSE
srake convert SRP123456 --to GSE

# Convert GSM to SRX
srake convert GSM123456 --to SRX
```
{{< /tab >}}

{{< tab >}}
**SRAdb:**
```r
# Download SRA files
getSRAfile(in_acc = "SRR123456",
          method = "curl")

# Download FASTQ files
getFASTQfile(in_acc = "SRR123456",
            srcType = "ftp")
```

**srake:**
```bash
# Download SRA files
srake download SRR123456

# Download FASTQ files
srake download SRR123456 --type fastq
```
{{< /tab >}}

{{< tab >}}
**SRAdb:**
```r
# Search metadata
getSRA(search_terms = "breast cancer",
       out_types = c("study", "sample", "experiment"))
```

**srake:**
```bash
# Search metadata
srake search "breast cancer" --format json
```
{{< /tab >}}

{{< tab >}}
**SRAdb:**
```r
# Get SRA info
getSRAinfo(in_acc = "SRP123456",
          sra_con = sra_con)
```

**srake:**
```bash
# Get metadata
srake metadata SRP123456 --detailed
```
{{< /tab >}}

{{< /tabs >}}

### ffq → srake

{{< tabs items="Metadata,Download,Multi-DB" >}}

{{< tab >}}
**ffq:**
```bash
# Get metadata for an accession
ffq SRR123456

# Get metadata with specific depth
ffq -l 2 GSE123456

# Save to JSON
ffq -o metadata.json SRR123456
```

**srake:**
```bash
# Get metadata for an accession
srake metadata SRR123456

# Get related metadata
srake studies SRR123456 --detailed

# Save to JSON
srake metadata SRR123456 --format json --output metadata.json
```
{{< /tab >}}

{{< tab >}}
**ffq:**
```bash
# Get FTP links
ffq --ftp SRR123456

# Get AWS links
ffq --aws SRR123456

# Get GCP links
ffq --gcp SRR123456
```

**srake:**
```bash
# Download from FTP
srake download SRR123456 --source ftp

# Download from AWS
srake download SRR123456 --source aws

# Download from GCP
srake download SRR123456 --source gcp
```
{{< /tab >}}

{{< tab >}}
**ffq:**
```bash
# Query from multiple databases
ffq GSE123456  # Queries GEO
ffq SRR123456  # Queries SRA
ffq ENCSR000EYA # Queries ENCODE
```

**srake:**
```bash
# Convert between databases
srake convert GSE123456 --to SRP  # GEO to SRA
srake convert SRR123456 --to GSM  # SRA to GEO

# Direct metadata query
srake metadata GSE123456  # Handles any accession type
```
{{< /tab >}}

{{< /tabs >}}

### pysradb → srake

{{< tabs items="Metadata,Download,Conversion,Search" >}}

{{< tab >}}
**pysradb:**
```python
# Get metadata
from pysradb import SRAweb
db = SRAweb()
df = db.sra_metadata('SRP123456')

# Detailed metadata
df = db.sra_metadata('SRP123456', detailed=True)
```

**srake:**
```bash
# Get metadata
srake metadata SRP123456

# Detailed metadata
srake metadata SRP123456 --detailed --format json
```
{{< /tab >}}

{{< tab >}}
**pysradb:**
```python
# Download SRA files
db.download(df, protocol='fasp')

# Download with filters
db.download(df, filter_by_library_strategy='RNA-Seq')
```

**srake:**
```bash
# Download with Aspera
srake download SRP123456 --aspera

# Download with filters (filter during ingest)
srake ingest --auto --strategies RNA-Seq
srake download SRP123456
```
{{< /tab >}}

{{< tab >}}
**pysradb:**
```python
# GSM to SRP
srp = db.gsm_to_srp(['GSM123456'])

# SRP to GSE
gse = db.srp_to_gse(['SRP123456'])

# SRX to SRR
srr = db.srx_to_srr(['SRX123456'])
```

**srake:**
```bash
# GSM to SRP
srake convert GSM123456 --to SRP

# SRP to GSE
srake convert SRP123456 --to GSE

# SRX to SRR
srake runs SRX123456
```
{{< /tab >}}

{{< tab >}}
**pysradb:**
```python
# Search by study
results = db.search_by_study_title('cancer')

# Search experiments
results = db.search_sra_studies('breast cancer', max_results=100)
```

**srake:**
```bash
# Search studies
srake search "cancer" --limit 100

# Search with filters
srake search "breast cancer" --organism "homo sapiens" --limit 100
```
{{< /tab >}}

{{< /tabs >}}

## Advanced Feature Mapping

### Batch Operations

**Other tools often require scripting:**
```python
# pysradb
for acc in accession_list:
    db.sra_metadata(acc)
```

**srake provides native batch support:**
```bash
# Batch conversion
srake convert --batch accessions.txt --to GSE

# Batch download
srake download --list runs.txt --parallel 4

# Batch metadata
srake metadata SRX001 SRX002 SRX003 --format json
```

### Filtering Capabilities

**Most tools require post-processing:**
```r
# SRAdb - filter after retrieval
data <- getSRA(search_terms = "*")
filtered <- subset(data, organism == "Homo sapiens")
```

**srake filters during ingestion:**
```bash
# Filter at source - more efficient
srake ingest --auto \
  --organisms "homo sapiens" \
  --platforms ILLUMINA \
  --strategies RNA-Seq \
  --min-reads 10000000
```

### Resume and Recovery

**Other tools typically lack resume:**
```python
# pysradb - no built-in resume
# If interrupted, must restart from beginning
```

**srake has intelligent resume:**
```bash
# Automatic resume from interruption
srake ingest --file large_archive.tar.gz
# If interrupted, rerun same command to resume
```

## Migration Guide

### From SRAdb

1. **Database setup:**
   ```bash
   # SRAdb: Download SQLite file
   # srake: Ingest directly
   srake ingest --auto
   ```

2. **Query syntax:**
   ```bash
   # SRAdb: SQL queries
   # srake: Simple CLI commands
   srake search "your query"
   ```

3. **Output formats:**
   ```bash
   # Both support multiple formats
   srake search "query" --format json
   ```

### From ffq

1. **Metadata retrieval:**
   ```bash
   # ffq focuses on links
   # srake provides full metadata
   srake metadata SRR123456 --detailed
   ```

2. **Download URLs:**
   ```bash
   # ffq shows URLs
   # srake downloads directly
   srake download SRR123456 --dry-run  # To see URLs
   srake download SRR123456             # To download
   ```

### From pysradb

1. **Python to CLI:**
   ```bash
   # pysradb requires Python scripting
   # srake works from command line
   srake convert GSM123456 --to SRX
   ```

2. **DataFrame to formats:**
   ```bash
   # pysradb returns DataFrames
   # srake supports multiple formats
   srake search "query" --format csv
   ```

## Unique srake Advantages

### 1. Streaming Architecture
- Process 14GB+ files with minimal RAM
- No need to extract archives to disk
- Zero-copy data transfer

### 2. Checkpoint System
- Resume from exact interruption point
- Track progress across sessions
- No duplicate processing

### 3. Integrated Filtering
- Filter during ingestion, not after
- Reduce database size
- Faster subsequent queries

### 4. Unified CLI
- Single tool for all operations
- Consistent command structure
- No language-specific setup

## Performance Comparison

| Operation | srake | SRAdb | pysradb |
|-----------|-------|-------|---------|
| 14GB Archive Ingestion | 15 min | 45 min* | 35 min* |
| Memory Usage | 200MB | 8GB+ | 4GB+ |
| Resume Support | ✅ | ❌ | ❌ |
| Concurrent Processing | ✅ | ❌ | Limited |

*Requires full extraction to disk first

## API Endpoints

For tools that need programmatic access, srake provides REST API equivalents:

```bash
# Start API server
srake server --port 8080

# Query endpoints
curl "http://localhost:8080/api/search?q=cancer"
curl "http://localhost:8080/api/metadata/SRP123456"
curl "http://localhost:8080/api/convert?from=SRP123456&to=GSE"
```

## Conclusion

srake combines the best features of existing tools while adding unique capabilities like streaming processing, checkpoint recovery, and integrated filtering. It provides a unified, efficient solution for SRA metadata management that scales from small queries to massive dataset processing.