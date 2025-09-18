---
title: API Reference
weight: 5
---

# REST API Reference

The srake REST API provides programmatic access to all search and metadata functionality.

## Starting the Server

```bash
# Start with default settings
srake server

# Custom port and host
srake server --port 8080 --host 0.0.0.0

# With custom database path
srake server --db /path/to/SRAmetadb.sqlite

# Development mode with verbose logging
srake server --dev --log-level debug
```

## Base URL

```
http://localhost:8080/api
```

## Authentication

The API currently does not require authentication for read operations.

## Endpoints

### Search Endpoints

#### `GET /api/search`

Perform full-text search with advanced filtering capabilities.

**Query Parameters:**

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `q` | string | Search query (supports advanced syntax) | - |
| `advanced` | boolean | Enable advanced query parsing | false |
| `limit` | integer | Maximum results to return | 100 |
| `offset` | integer | Pagination offset | 0 |
| `organism` | string | Filter by organism | - |
| `platform` | string | Filter by platform | - |
| `library_strategy` | string | Filter by library strategy | - |
| `library_source` | string | Filter by library source | - |
| `library_selection` | string | Filter by library selection | - |
| `library_layout` | string | Filter by library layout | - |
| `instrument` | string | Filter by instrument model | - |
| `date_from` | string | Filter by date from (YYYY-MM-DD) | - |
| `date_to` | string | Filter by date to (YYYY-MM-DD) | - |
| `spots_min` | integer | Minimum number of spots | - |
| `spots_max` | integer | Maximum number of spots | - |
| `bases_min` | integer | Minimum number of bases | - |
| `bases_max` | integer | Maximum number of bases | - |
| `format` | string | Output format (json, csv, tsv, accession) | json |
| `fields` | string | Comma-separated list of fields to return | all |
| `aggregate_by` | string | Field to aggregate results by | - |
| `count_only` | boolean | Return only count | false |
| `facets` | boolean | Include facets in response | false |

**Example Request:**

```bash
# Basic search
curl "http://localhost:8080/api/search?q=homo+sapiens&limit=10"

# Advanced search with filters
curl "http://localhost:8080/api/search?q=cancer&organism=homo+sapiens&platform=ILLUMINA&library_strategy=RNA-Seq"

# Boolean query
curl "http://localhost:8080/api/search?q=organism:human+AND+library_strategy:RNA-Seq&advanced=true"

# Aggregation
curl "http://localhost:8080/api/search?q=RNA-Seq&aggregate_by=organism"

# Count only
curl "http://localhost:8080/api/search?q=cancer&count_only=true"
```

**Response Format (JSON):**

```json
{
  "total": 42156,
  "limit": 10,
  "offset": 0,
  "hits": [
    {
      "id": "SRR123456",
      "score": 1.5432,
      "fields": {
        "type": "run",
        "run_accession": "SRR123456",
        "experiment_accession": "SRX123456",
        "study_accession": "SRP123456",
        "organism": "Homo sapiens",
        "platform": "ILLUMINA",
        "library_strategy": "RNA-Seq",
        "spots": 25000000,
        "bases": 2500000000
      }
    }
  ],
  "facets": {
    "organism": {
      "Homo sapiens": 15234,
      "Mus musculus": 8921
    },
    "platform": {
      "ILLUMINA": 35678,
      "OXFORD_NANOPORE": 4521
    }
  },
  "aggregations": {
    "organism": [
      {"value": "Homo sapiens", "count": 15234},
      {"value": "Mus musculus", "count": 8921}
    ]
  }
}
```

#### `POST /api/search/index`

Manage the search index.

**Request Body:**

```json
{
  "action": "build|rebuild|verify|stats",
  "batch_size": 1000,
  "workers": 4
}
```

**Example Request:**

```bash
# Build index
curl -X POST "http://localhost:8080/api/search/index" \
  -H "Content-Type: application/json" \
  -d '{"action": "build"}'

# Get index stats
curl -X POST "http://localhost:8080/api/search/index" \
  -H "Content-Type: application/json" \
  -d '{"action": "stats"}'
```

### Metadata Endpoints

#### `GET /api/metadata/{accession}`

Get metadata for a specific SRA accession.

**Path Parameters:**
- `accession`: SRA accession (SRP, SRX, SRR, SRS, etc.)

**Query Parameters:**
- `format`: Output format (json, yaml, xml)
- `expand`: Include related entities (true/false)

**Example Request:**

```bash
# Get run metadata
curl "http://localhost:8080/api/metadata/SRR123456"

# Get study with expanded relations
curl "http://localhost:8080/api/metadata/SRP123456?expand=true"
```

**Response Format:**

```json
{
  "accession": "SRR123456",
  "type": "run",
  "metadata": {
    "run_accession": "SRR123456",
    "experiment_accession": "SRX123456",
    "study_accession": "SRP123456",
    "sample_accession": "SRS123456",
    "spots": 25000000,
    "bases": 2500000000,
    "published_date": "2024-01-15"
  },
  "relations": {
    "experiment": {...},
    "study": {...},
    "sample": {...}
  }
}
```

### Conversion Endpoints

#### `GET /api/convert/{accession}`

Convert between different accession types.

**Path Parameters:**
- `accession`: Source accession

**Query Parameters:**
- `to`: Target type (GSE, GSM, SRP, SRX, SRR, PRJNA, etc.)

**Example Request:**

```bash
# Convert SRA to GEO
curl "http://localhost:8080/api/convert/SRP123456?to=GSE"

# Convert GEO to SRA
curl "http://localhost:8080/api/convert/GSM123456?to=SRX"
```

**Response Format:**

```json
{
  "source": "SRP123456",
  "source_type": "SRP",
  "target": "GSE98765",
  "target_type": "GSE",
  "status": "success"
}
```

### Relationship Endpoints

#### `GET /api/runs/{accession}`

Get all runs for a study, experiment, or sample.

**Example Request:**

```bash
# Get runs for a study
curl "http://localhost:8080/api/runs/SRP123456"

# Get runs for an experiment
curl "http://localhost:8080/api/runs/SRX123456"
```

#### `GET /api/samples/{accession}`

Get all samples for a study or experiment.

```bash
# Get samples for a study
curl "http://localhost:8080/api/samples/SRP123456"
```

#### `GET /api/experiments/{accession}`

Get all experiments for a study.

```bash
# Get experiments for a study
curl "http://localhost:8080/api/experiments/SRP123456"
```

#### `GET /api/studies/{accession}`

Get study information from any related accession.

```bash
# Get study from run
curl "http://localhost:8080/api/studies/SRR123456"

# Get study from sample
curl "http://localhost:8080/api/studies/SRS123456"
```

### Statistics Endpoints

#### `GET /api/stats`

Get database statistics.

**Example Request:**

```bash
curl "http://localhost:8080/api/stats"
```

**Response Format:**

```json
{
  "database": {
    "size": 4567890123,
    "path": "/data/SRAmetadb.sqlite"
  },
  "tables": {
    "study": 456789,
    "experiment": 2345678,
    "sample": 3456789,
    "run": 12345678
  },
  "index": {
    "documents": 18625590,
    "size": 1234567890,
    "last_updated": "2024-01-15T10:30:00Z"
  }
}
```

## Error Handling

All endpoints return appropriate HTTP status codes:

- `200 OK`: Successful request
- `400 Bad Request`: Invalid parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

Error responses include a JSON body:

```json
{
  "error": "Invalid query syntax",
  "message": "Unmatched quotes in query string",
  "code": "INVALID_QUERY"
}
```

## Rate Limiting

The API currently does not implement rate limiting. For production deployments, consider adding a reverse proxy with rate limiting capabilities.

## CORS

CORS headers are enabled by default in development mode. For production, configure appropriate CORS policies based on your deployment needs.

## Client Libraries

### Python

```python
import requests

# Search API
response = requests.get('http://localhost:8080/api/search', params={
    'q': 'homo sapiens',
    'platform': 'ILLUMINA',
    'library_strategy': 'RNA-Seq',
    'limit': 100
})

results = response.json()
for hit in results['hits']:
    print(f"{hit['fields']['run_accession']}: {hit['fields']['organism']}")
```

### JavaScript/Node.js

```javascript
// Using fetch API
const params = new URLSearchParams({
    q: 'cancer',
    organism: 'homo sapiens',
    platform: 'ILLUMINA'
});

fetch(`http://localhost:8080/api/search?${params}`)
    .then(res => res.json())
    .then(data => {
        console.log(`Found ${data.total} results`);
        data.hits.forEach(hit => {
            console.log(hit.fields.run_accession);
        });
    });
```

### R

```r
library(httr)
library(jsonlite)

# Search for RNA-Seq data
response <- GET("http://localhost:8080/api/search",
                query = list(
                    q = "RNA-Seq",
                    organism = "homo sapiens",
                    limit = 100
                ))

data <- fromJSON(content(response, "text"))
print(paste("Total results:", data$total))
```

### curl/bash

```bash
#!/bin/bash

# Search and save results
curl -s "http://localhost:8080/api/search?q=cancer&format=json" \
  | jq '.hits[].fields.run_accession' \
  > accessions.txt

# Batch download using results
while read -r acc; do
    echo "Processing $acc"
    srake download "$acc"
done < accessions.txt
```

## Webhooks

Webhooks are not currently supported but are planned for future releases.

## WebSocket Support

Real-time updates via WebSocket are planned for future releases.

## GraphQL API

A GraphQL endpoint is under consideration for complex relationship queries.

## Performance Tips

1. **Use specific filters**: Narrow down results with filters to reduce response size
2. **Pagination**: Use `limit` and `offset` for large result sets
3. **Field selection**: Use `fields` parameter to return only needed data
4. **Aggregations**: Use `aggregate_by` for analytics instead of fetching all data
5. **Caching**: Implement client-side caching for frequently accessed data

## Migration from Other Tools

### From pysradb

```python
# pysradb
from pysradb import SRAdb
db = SRAdb()
df = db.search("cancer", detailed=True)

# srake API equivalent
import pandas as pd
import requests

response = requests.get('http://localhost:8080/api/search',
                        params={'q': 'cancer', 'format': 'json'})
df = pd.DataFrame([hit['fields'] for hit in response.json()['hits']])
```

### From SRAdb (R)

```r
# SRAdb
library(SRAdb)
sqlfile <- getSRAdbFile()
con <- dbConnect(SQLite(), sqlfile)
rs <- dbGetQuery(con, "SELECT * FROM sra WHERE organism = 'Homo sapiens'")

# srake API equivalent
library(httr)
library(jsonlite)

response <- GET("http://localhost:8080/api/search",
                query = list(organism = "homo sapiens"))
rs <- fromJSON(content(response, "text"))$hits
```

## Support

For API issues or feature requests, please visit:
- [GitHub Issues](https://github.com/nishad/srake/issues)
- [API Documentation](https://nishad.github.io/srake/api)