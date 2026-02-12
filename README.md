# SRAKE - SRA Knowledge Engine

*Pronounced like Japanese sake (酒) — "srah-keh"*

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/nishad/srake)](https://github.com/nishad/srake/releases)

SRAKE is a tool for ingesting, searching, and serving [NCBI SRA](https://www.ncbi.nlm.nih.gov/sra) (Sequence Read Archive) metadata. It streams multi-gigabyte compressed archives directly into a local SQLite database, supports full-text and semantic vector search, and exposes results via CLI, REST API, or MCP.

> **Pre-alpha software** — developed at [BioHackathon 2025, Mie, Japan](https://2025.biohackathon.org/). Not production-ready. APIs, schemas, and behavior may change without notice.

## Installation

### From source (requires Go 1.25+, CGO, SQLite3)

```bash
git clone https://github.com/nishad/srake.git
cd srake
go build -tags "sqlite_fts5,search" -o srake ./cmd/srake
```

### Pre-built binaries

Download from the [releases page](https://github.com/nishad/srake/releases). Available for linux/amd64, linux/arm64, darwin/amd64, and darwin/arm64.

### Docker

```bash
docker pull ghcr.io/nishad/srake:latest
docker run -v $(pwd)/data:/data ghcr.io/nishad/srake:latest --help
```

## Quick start

```bash
# Ingest SRA metadata (auto-selects best source from NCBI)
srake ingest --auto

# Ingest a specific local archive
srake ingest --file /path/to/archive.tar.gz

# Search
srake search "homo sapiens" --limit 10

# Start the API server
srake server --port 8080

# Start MCP server for AI assistants
srake mcp --transport stdio
```

## Commands

### `srake ingest` — Ingest SRA metadata

Streams tar.gz archives from NCBI (or local files) directly into SQLite without intermediate extraction.

```bash
srake ingest --auto                # auto-select best source
srake ingest --daily               # latest daily update
srake ingest --monthly             # full monthly dataset
srake ingest --file archive.tar.gz # local file
srake ingest --list                # list available files on NCBI
```

Filtering during ingest:

```bash
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --platforms ILLUMINA \
  --strategies RNA-Seq \
  --date-from 2024-01-01 \
  --min-reads 1000000

# Preview what would be ingested without inserting
srake ingest --file archive.tar.gz --taxon-ids 9606 --stats-only
```

### `srake search` — Search metadata

Supports multiple search modes: `database` (SQLite FTS5), `text` (Bleve), `vector` (semantic embeddings), `hybrid` (combined), and `auto` (default).

```bash
srake search "breast cancer" --limit 20
srake search "RNA-Seq" --organism "homo sapiens" --platform ILLUMINA
srake search "tumor gene expression" --search-mode vector
srake search "mouse brain" --similarity-threshold 0.7 --show-confidence
srake search "covid" --format json --output results.json
```

Output formats: `table` (default), `json`, `csv`, `tsv`.

### `srake server` — REST API server

```bash
srake server --port 8080 --enable-cors
```

Endpoints:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/search?query=...` | Search metadata |
| GET | `/api/v1/studies/{accession}` | Get study |
| GET | `/api/v1/experiments/{accession}` | Get experiment |
| GET | `/api/v1/samples/{accession}` | Get sample |
| GET | `/api/v1/runs/{accession}` | Get run |
| GET | `/api/v1/stats` | Database statistics |
| GET | `/api/v1/health` | Health check |
| POST | `/api/v1/export` | Export results |

### `srake mcp` — MCP server for AI assistants

Implements [Model Context Protocol](https://modelcontextprotocol.io/) over stdio or HTTP.

```bash
srake mcp --transport stdio    # for Claude Desktop, etc.
srake mcp --transport http --port 8081
```

Provides tools: `search_sra`, `get_metadata`, `find_similar`, `export_results`.

### `srake metadata` — Accession lookup

```bash
srake metadata SRR12345678 --format json
srake metadata SRP123456 SRX654321 --format yaml
```

### `srake db` — Database management

```bash
srake db info                  # show database statistics
srake db stats --rebuild       # rebuild pre-computed statistics
srake db export -o out.sqlite  # export to SRAmetadb-compatible format
srake db export -o out.sqlite --fts-version 3  # FTS3 for legacy compatibility
```

### `srake models` — Embedding model management

```bash
srake models list
srake models download Xenova/SapBERT-from-PubMedBERT-fulltext
srake models test <model-id> "sample text"
```

## Vector search

SRAKE supports semantic search using biomedical embeddings (SapBERT via ONNX Runtime). This requires downloading a model and building an index with embeddings enabled.

```bash
srake models download Xenova/SapBERT-from-PubMedBERT-fulltext
srake search "metabolic pathway analysis" --search-mode vector
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `SRAKE_DB_PATH` | Path to metadata database |
| `SRAKE_INDEX_PATH` | Path to search index directory |
| `SRAKE_CONFIG_DIR` | Configuration directory (default: `~/.config/srake`) |
| `SRAKE_DATA_DIR` | Data directory (default: `~/.local/share/srake`) |
| `SRAKE_CACHE_DIR` | Cache directory (default: `~/.cache/srake`) |
| `SRAKE_MODEL_VARIANT` | Embedding model variant (`full`, `quantized`) |
| `NO_COLOR` | Disable colored output |

Follows [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/).

## Database schema

Core tables: `studies`, `experiments`, `samples`, `runs`, `submissions`, `analyses`.
Junction tables: `experiment_samples`, `statistics`.
Full-text search via SQLite FTS5 virtual tables.

## Development

```bash
go build -tags "sqlite_fts5,search" ./...
go test -tags "sqlite_fts5,search" ./...

# Build with version injection
go build -tags "sqlite_fts5,search" \
  -ldflags="-X main.Version=$(git describe --tags) -X main.Commit=$(git rev-parse --short HEAD) -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o srake ./cmd/srake
```

## License

[MIT](LICENSE) — Nishad Thalhath, 2025
