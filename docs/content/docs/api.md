---
title: API Reference
weight: 5
---

# API Reference

Start the server:

```bash
srake server --port 8080
```

All endpoints are prefixed with `/api/v1/`.

---

## Search

### `GET /api/v1/search`

Query parameters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `q` / `query` | string | Search query |
| `limit` | int | Max results (default: 20) |
| `offset` | int | Skip N results |
| `organism` | string | Filter by organism |
| `library_strategy` | string | Filter by library strategy |
| `platform` | string | Filter by platform |
| `similarity_threshold` | float | Vector similarity threshold (0.0-1.0) |
| `min_score` | float | Minimum BM25 score |
| `show_confidence` | bool | Include confidence scores |
| `mode` / `search_mode` | string | Search mode: text, vector, hybrid, database |
| `format` | string | Response format |

```bash
curl "http://localhost:8080/api/v1/search?q=cancer&limit=10"
curl "http://localhost:8080/api/v1/search?q=RNA-Seq&organism=homo+sapiens&platform=ILLUMINA"
```

### `POST /api/v1/search/advanced`

Accepts a JSON body with the same parameters as the search query.

---

## Studies

### `GET /api/v1/studies`

List studies with pagination. Parameters: `limit` (max 100, default 20), `offset`.

### `GET /api/v1/studies/{accession}`

Get a single study by accession.

### `GET /api/v1/studies/{accession}/metadata`

Get the full metadata graph for a study (experiments, samples, runs, summary counts).

### `GET /api/v1/studies/{accession}/experiments`

List experiments for a study.

### `GET /api/v1/studies/{accession}/samples`

List samples for a study.

### `GET /api/v1/studies/{accession}/runs`

List runs for a study. Parameter: `limit`.

---

## Experiments, Samples, Runs

### `GET /api/v1/experiments/{accession}`

Get experiment by accession.

### `GET /api/v1/samples/{accession}`

Get sample by accession.

### `GET /api/v1/runs/{accession}`

Get run by accession.

---

## Statistics

### `GET /api/v1/stats`

Database-wide counts (studies, experiments, samples, runs).

### `GET /api/v1/stats/organisms`

Organism distribution with counts.

### `GET /api/v1/stats/platforms`

Platform distribution with counts.

### `GET /api/v1/stats/strategies`

Library strategy distribution with counts.

---

## Export

### `POST /api/v1/export`

Export search results. JSON body with `query`, `format` (json, csv, tsv, xml, jsonl), `filters`, `limit`.

---

## Health

### `GET /api/v1/health`

```json
{"status": "healthy", "database": "ok", "timestamp": "2025-01-15T10:30:00Z"}
```

---

## MCP (Model Context Protocol)

MCP support is available via the `srake mcp` command, which runs a stdio-based MCP server
compatible with Claude Desktop, VS Code, and other MCP clients.

See [`srake mcp`](/docs/reference/cli#srake-mcp) in the CLI reference for configuration details.
