---
title: API Reference
weight: 5
---

# REST API Reference

SRAKE provides a comprehensive RESTful API with OpenAPI 3.0 specification for programmatic access to all search and metadata functionality.

## Quick Start

```bash
# Start the API server
srake server --port 8082 --enable-cors --enable-mcp

# Test the API
curl "http://localhost:8082/api/v1/search?query=RNA-Seq&limit=5"

# Check API health
curl "http://localhost:8082/api/v1/health"
```

## OpenAPI Specification

The complete API specification is available in `openapi.yaml` which can be imported into:
- **Swagger UI** for interactive documentation
- **Postman** for testing
- **OpenAPI Generator** for client SDK generation

## Base URL

```
http://localhost:8082/api/v1
```

## Server Configuration

```bash
# Basic server start
srake server

# With custom configuration
srake server \
  --port 8082 \
  --host 0.0.0.0 \
  --enable-cors \
  --enable-mcp

# Using environment variables
SRAKE_DB_PATH=/path/to/db.sqlite \
SRAKE_INDEX_PATH=/path/to/index \
srake server
```

## Authentication

The API currently does not require authentication. For production deployments, authentication can be added via reverse proxy.

## Core Endpoints

### Search Operations

#### `GET /api/v1/search`

Perform advanced searches with quality control and multiple search modes.

**Query Parameters:**

| Parameter | Type | Description | Default | Example |
|-----------|------|-------------|---------|---------|
| `query` | string | Search query text | - | `RNA-Seq` |
| `limit` | integer | Maximum results (1-1000) | 20 | `10` |
| `offset` | integer | Pagination offset | 0 | `0` |
| `organism` | string | Filter by organism | - | `homo sapiens` |
| `library_strategy` | string | Filter by strategy | - | `RNA-Seq` |
| `platform` | string | Filter by platform | - | `ILLUMINA` |
| `search_mode` | string | Search mode: `database`, `fts`, `hybrid`, `vector` | `hybrid` | `hybrid` |
| `similarity_threshold` | float | Min similarity (0-1) | - | `0.7` |
| `min_score` | float | Minimum score | - | `2.0` |
| `top_percentile` | integer | Top N% of results | - | `10` |
| `show_confidence` | boolean | Include confidence level | false | `true` |

**Example Requests:**

```bash
# Simple search
curl "http://localhost:8082/api/v1/search?query=breast%20cancer"

# Advanced search with quality control
curl "http://localhost:8082/api/v1/search?query=RNA-Seq&\
similarity_threshold=0.7&\
show_confidence=true&\
organism=homo%20sapiens"

# Vector semantic search
curl "http://localhost:8082/api/v1/search?query=tumor%20gene%20expression&\
search_mode=vector&\
similarity_threshold=0.8"
```

**Response Example:**

```json
{
  "results": [
    {
      "id": "SRX22037872",
      "type": "experiment",
      "title": "RNA-Seq of breast cancer cells",
      "organism": "Homo sapiens",
      "platform": "ILLUMINA",
      "library_strategy": "RNA-Seq",
      "score": 8.5,
      "similarity": 0.92,
      "confidence": "high"
    }
  ],
  "total_results": 150,
  "query": "breast cancer",
  "time_taken_ms": 125,
  "search_mode": "hybrid"
}
```

#### `POST /api/v1/search`

Advanced search with complex filters in request body.

**Request Body:**

```json
{
  "query": "breast cancer transcriptome",
  "filters": {
    "organism": "homo sapiens",
    "library_strategy": "RNA-Seq",
    "platform": "ILLUMINA"
  },
  "limit": 10,
  "search_mode": "hybrid",
  "similarity_threshold": 0.8,
  "show_confidence": true
}
```

### Metadata Endpoints

#### `GET /api/v1/studies`

List all studies with pagination.

```bash
curl "http://localhost:8082/api/v1/studies?limit=10&offset=0"
```

#### `GET /api/v1/studies/{accession}`

Get detailed study information.

```bash
curl "http://localhost:8082/api/v1/studies/SRP259537"
```

#### `GET /api/v1/studies/{accession}/experiments`

Get all experiments for a study.

```bash
curl "http://localhost:8082/api/v1/studies/SRP259537/experiments"
```

#### `GET /api/v1/studies/{accession}/samples`

Get all samples for a study.

```bash
curl "http://localhost:8082/api/v1/studies/SRP259537/samples"
```

#### `GET /api/v1/studies/{accession}/runs`

Get all runs for a study.

```bash
curl "http://localhost:8082/api/v1/studies/SRP259537/runs?limit=100"
```

#### `GET /api/v1/experiments/{accession}`

Get experiment details.

```bash
curl "http://localhost:8082/api/v1/experiments/SRX22037872"
```

#### `GET /api/v1/samples/{accession}`

Get sample details.

```bash
curl "http://localhost:8082/api/v1/samples/SRS19840123"
```

#### `GET /api/v1/runs/{accession}`

Get run details.

```bash
curl "http://localhost:8082/api/v1/runs/SRR25889421"
```

### Statistics

#### `GET /api/v1/stats`

Get comprehensive database statistics.

```bash
curl "http://localhost:8082/api/v1/stats"
```

**Response Example:**

```json
{
  "total_documents": 1234567,
  "indexed_documents": 1234000,
  "index_size": 2147483648,
  "last_indexed": "2024-12-19T10:30:00Z",
  "top_organisms": [
    {"name": "Homo sapiens", "count": 450000, "percentage": 36.5},
    {"name": "Mus musculus", "count": 280000, "percentage": 22.7}
  ],
  "top_platforms": [
    {"name": "ILLUMINA", "count": 950000, "percentage": 77.0}
  ],
  "top_strategies": [
    {"name": "RNA-Seq", "count": 400000, "percentage": 32.4}
  ]
}
```

### Export

#### `POST /api/v1/export`

Export search results in various formats.

**Supported Formats:**
- `json` - Standard JSON
- `csv` - Comma-separated values
- `tsv` - Tab-separated values
- `xml` - XML format
- `jsonl` - Newline-delimited JSON

**Request Body:**

```json
{
  "query": "RNA-Seq human",
  "format": "csv",
  "limit": 100,
  "filters": {
    "organism": "homo sapiens"
  }
}
```

**Example:**

```bash
# Export as CSV
curl -X POST http://localhost:8082/api/v1/export \
  -H "Content-Type: application/json" \
  -d '{"query":"RNA-Seq","format":"csv","limit":100}' \
  -o results.csv
```

### Health Monitoring

#### `GET /api/v1/health`

Check service health status.

```bash
curl "http://localhost:8082/api/v1/health"
```

**Response:**

```json
{
  "status": "healthy",
  "timestamp": "2024-12-19T10:30:00Z",
  "search_service": "healthy",
  "metadata_service": "healthy"
}
```

## MCP (Model Context Protocol)

SRAKE supports MCP for AI assistant integration using JSON-RPC 2.0.

### MCP Capabilities

```bash
curl "http://localhost:8082/mcp/capabilities"
```

### MCP Tools

Available tools for AI assistants:
- `search_sra` - Search with quality control
- `get_metadata` - Get detailed metadata
- `find_similar` - Vector similarity search
- `export_results` - Export in various formats

**Example MCP Request:**

```bash
curl -X POST http://localhost:8082/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "search_sra",
      "arguments": {
        "query": "breast cancer RNA-Seq",
        "limit": 5,
        "similarity_threshold": 0.7
      }
    },
    "id": 1
  }'
```

## Search Modes

SRAKE supports multiple search modes for different use cases:

| Mode | Description | Use Case |
|------|-------------|----------|
| `database` | Direct SQL queries | Exact matching, filters only |
| `fts` | Full-text search with Bleve | Text search with highlights |
| `hybrid` | Combines database and FTS | Best of both worlds (default) |
| `vector` | Semantic search with embeddings | Conceptual similarity |

## Quality Control

Control search quality with these parameters:

- **Similarity Threshold** (0-1): Minimum similarity score for results
- **Min Score**: Minimum absolute score requirement
- **Top Percentile**: Return only top N% of results
- **Confidence Levels**: `high` (>0.8), `medium` (0.5-0.8), `low` (<0.5)

## Client Libraries

### Python

```python
import requests
import pandas as pd

# Search API
response = requests.get('http://localhost:8082/api/v1/search', params={
    'query': 'breast cancer',
    'organism': 'homo sapiens',
    'library_strategy': 'RNA-Seq',
    'similarity_threshold': 0.7,
    'show_confidence': True,
    'limit': 100
})

data = response.json()
df = pd.DataFrame(data['results'])
print(f"Found {data['total_results']} results")
print(f"Top result confidence: {df.iloc[0]['confidence']}")
```

### JavaScript/TypeScript

```javascript
// Using fetch API
const params = new URLSearchParams({
    query: 'cancer',
    organism: 'homo sapiens',
    search_mode: 'hybrid',
    similarity_threshold: '0.8',
    limit: '50'
});

fetch(`http://localhost:8082/api/v1/search?${params}`)
    .then(res => res.json())
    .then(data => {
        console.log(`Found ${data.total_results} results`);
        console.log(`Search took ${data.time_taken_ms}ms`);
        data.results.forEach(result => {
            console.log(`${result.id}: ${result.title} (${result.confidence})`);
        });
    });
```

### R

```r
library(httr)
library(jsonlite)

# Search with quality control
response <- GET("http://localhost:8082/api/v1/search",
                query = list(
                    query = "RNA-Seq",
                    organism = "homo sapiens",
                    similarity_threshold = 0.7,
                    show_confidence = TRUE,
                    limit = 100
                ))

data <- fromJSON(content(response, "text"))
print(paste("Total results:", data$total_results))
print(paste("Search mode:", data$search_mode))

# Convert to dataframe
df <- as.data.frame(data$results)
```

### curl/bash

```bash
#!/bin/bash

# Search and process results
curl -s "http://localhost:8082/api/v1/search?\
query=cancer&\
similarity_threshold=0.8&\
show_confidence=true&\
format=json" | \
jq -r '.results[] |
  select(.confidence == "high") |
  "\(.id)\t\(.organism)\t\(.library_strategy)\t\(.confidence)"' | \
while IFS=$'\t' read -r acc org strat conf; do
    echo "Processing $acc ($org, $strat) with $conf confidence"
    # Further processing...
done
```

## Error Handling

All endpoints return appropriate HTTP status codes:

| Status Code | Description |
|-------------|-------------|
| `200 OK` | Successful request |
| `400 Bad Request` | Invalid parameters |
| `404 Not Found` | Resource not found |
| `500 Internal Server Error` | Server error |
| `503 Service Unavailable` | Service unhealthy |

Error responses include detailed information:

```json
{
  "error": true,
  "message": "Invalid search parameters",
  "status": 400
}
```

## Performance Tips

1. **Use specific filters** to reduce result set size
2. **Enable pagination** with `limit` and `offset` for large results
3. **Set quality thresholds** to get only high-confidence results
4. **Use appropriate search mode** for your use case
5. **Cache frequently accessed data** on the client side

## CORS Configuration

CORS is enabled with the `--enable-cors` flag:

```bash
srake server --enable-cors
```

For production, configure specific allowed origins via reverse proxy.

## Rate Limiting

Rate limiting is not implemented in the core API. For production:
- Use a reverse proxy (nginx, Caddy)
- Implement API key authentication
- Add per-client rate limits

## Migration Guide

### From pysradb

```python
# pysradb (old)
from pysradb import SRAdb
db = SRAdb()
df = db.search("cancer", detailed=True)

# SRAKE API (new)
import requests
import pandas as pd

response = requests.get('http://localhost:8082/api/v1/search',
                        params={'query': 'cancer', 'limit': 1000})
df = pd.DataFrame(response.json()['results'])
```

### From SRAdb (R)

```r
# SRAdb (old)
library(SRAdb)
sqlfile <- getSRAdbFile()
con <- dbConnect(SQLite(), sqlfile)
rs <- dbGetQuery(con, "SELECT * FROM sra WHERE organism = 'Homo sapiens'")

# SRAKE API (new)
library(httr)
library(jsonlite)

response <- GET("http://localhost:8082/api/v1/search",
                query = list(organism = "homo sapiens"))
rs <- fromJSON(content(response, "text"))$results
```

## Advanced Features

### Hybrid Search Weighting

Control the balance between database and FTS results:

```bash
curl "http://localhost:8082/api/v1/search?\
query=cancer&\
search_mode=hybrid&\
hybrid_weight=0.7"  # 70% FTS, 30% database
```

### Field-Specific Export

Export only specific fields:

```json
{
  "query": "RNA-Seq",
  "format": "csv",
  "fields": ["id", "title", "organism", "platform"],
  "limit": 100
}
```

### Batch Processing

Process multiple queries efficiently:

```bash
#!/bin/bash
queries=("cancer" "diabetes" "alzheimer")
for q in "${queries[@]}"; do
  curl -s "http://localhost:8082/api/v1/search?query=$q&limit=10" \
    > "${q}_results.json"
done
```

## Support

- [GitHub Issues](https://github.com/nishad/srake/issues)
- [OpenAPI Specification](./openapi.yaml)
- [API Testing Guide](./TESTING_GUIDE.md)