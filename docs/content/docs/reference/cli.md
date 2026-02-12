---
title: CLI Reference
weight: 10
---

# CLI Reference

## Global Flags

Available for all commands:

- `--no-color` -- Disable colored output (also respects `NO_COLOR` env var)
- `-v, --verbose` -- Verbose output
- `-q, --quiet` -- Suppress non-error output

---

## `srake ingest`

Ingest SRA metadata from NCBI or local archives.

```bash
srake ingest [flags]
```

**Source flags:**

| Flag | Description |
|------|-------------|
| `--auto` | Auto-select the best file from NCBI |
| `--daily` | Ingest the latest daily update |
| `--monthly` | Ingest the latest monthly dataset |
| `--file <path>` | Ingest a local or remote file |
| `--list` | List available files without ingesting |

**Filter flags:**

| Flag | Description |
|------|-------------|
| `--taxon-ids <ids>` | Comma-separated NCBI taxonomy IDs to include |
| `--exclude-taxon-ids <ids>` | Taxonomy IDs to exclude |
| `--organisms <names>` | Organism names to include |
| `--platforms <names>` | Sequencing platforms (e.g. ILLUMINA, PACBIO_SMRT) |
| `--strategies <names>` | Library strategies (e.g. RNA-Seq, WGS) |
| `--date-from <date>` | Include records from this date (YYYY-MM-DD) |
| `--date-to <date>` | Include records until this date |
| `--min-reads <n>` | Minimum read count |
| `--max-reads <n>` | Maximum read count |
| `--min-bases <n>` | Minimum base count |
| `--max-bases <n>` | Maximum base count |
| `--stats-only` | Preview filter results without inserting |

**Other flags:**

| Flag | Description |
|------|-------------|
| `--db <path>` | Database path |
| `--force` | Force re-ingestion |
| `--no-progress` | Disable progress bar |

```bash
# Examples
srake ingest --auto
srake ingest --file archive.tar.gz --taxon-ids 9606 --platforms ILLUMINA
srake ingest --auto --stats-only  # preview what would be imported
```

---

## `srake search`

Search SRA metadata using full-text, vector, or hybrid search.

```bash
srake search <query> [flags]
```

**Filter flags:**

| Flag | Description |
|------|-------------|
| `--organism <name>` | Filter by organism |
| `--platform <name>` | Filter by platform |
| `--library-strategy <name>` | Filter by library strategy |
| `--library-source <name>` | Filter by library source |
| `--library-selection <name>` | Filter by library selection |
| `--library-layout <name>` | Filter by library layout |
| `--study-type <name>` | Filter by study type |
| `--instrument-model <name>` | Filter by instrument model |
| `--date-from <date>` | Date range start |
| `--date-to <date>` | Date range end |
| `--spots-min <n>` | Minimum spots (reads) |
| `--spots-max <n>` | Maximum spots |
| `--bases-min <n>` | Minimum bases |
| `--bases-max <n>` | Maximum bases |

**Output flags:**

| Flag | Description |
|------|-------------|
| `--limit <n>` | Max results (default: 100) |
| `--offset <n>` | Skip N results |
| `--format <type>` | Output format: table, json, csv, tsv, accession |
| `--output <file>` | Write results to file |
| `--no-header` | Omit table header |
| `--fields <list>` | Comma-separated field list |

**Search mode flags:**

| Flag | Description |
|------|-------------|
| `--search-mode <mode>` | Search mode: auto, text, vector, hybrid, database |
| `--fuzzy` | Enable fuzzy matching |
| `--exact` | Require exact matches |
| `--advanced` | Enable advanced query syntax |
| `--similarity-threshold <f>` | Vector similarity threshold (0.0-1.0, default: 0.5) |
| `--min-score <f>` | Minimum BM25 score |
| `--show-confidence` | Show confidence scores |
| `--hybrid-weight <f>` | Hybrid weight (0.0=text, 1.0=vector, default: 0.7) |
| `--facets` | Include facet counts |
| `--stats` | Show search statistics |

```bash
# Examples
srake search "homo sapiens"
srake search "cancer" --organism "homo sapiens" --format json
srake search "tumor expression" --search-mode vector --show-confidence
srake search "RNA-Seq" --format accession --output accessions.txt
```

---

## `srake index`

Build and manage the search index.

```bash
srake index [flags]
```

| Flag | Description |
|------|-------------|
| `--build` | Build the search index |
| `--rebuild` | Rebuild from scratch |
| `--verify` | Verify index integrity |
| `--stats` | Show index statistics |
| `--resume` | Resume interrupted build |
| `--batch-size <n>` | Documents per batch (default: 500) |
| `--workers <n>` | Parallel workers (0 = auto) |
| `--path <dir>` | Index directory |
| `--backend <type>` | Backend: tiered (default), bleve |
| `--with-embeddings` | Include vector embeddings |
| `--embedding-model <name>` | Embedding model name |
| `--progress` | Show progress bar |

```bash
# Examples
srake index --build
srake index --build --with-embeddings --progress
srake index --rebuild --batch-size 1000
srake index --stats
```

---

## `srake server`

Start the REST API server.

```bash
srake server [flags]
```

| Flag | Description |
|------|-------------|
| `-p, --port <n>` | Port (default: 8080) |
| `--host <addr>` | Host to bind (default: 0.0.0.0) |
| `--enable-cors` | Enable CORS (default: true) |
| `--db <path>` | Database path |
| `--index <path>` | Index path |

```bash
# Examples
srake server --port 8080
srake server --port 3000 --host localhost
SRAKE_DB_PATH=/data/srake.db srake server
```

See [API Reference](/docs/api) for endpoint documentation.

---

## `srake mcp`

Start an MCP (Model Context Protocol) server for AI assistants.

```bash
srake mcp [flags]
```

| Flag | Description |
|------|-------------|
| `--db <path>` | Database path |
| `--index <path>` | Index path |
| `--transport <mode>` | Transport mode: `stdio` (default) or `http` |
| `--host <addr>` | HTTP server host (default: localhost, only with `--transport http`) |
| `--port <n>` | HTTP server port (default: 8081, only with `--transport http`) |

**Stdio mode** (default): communicates over stdin/stdout using JSON-RPC 2.0. All diagnostic output goes to stderr.

**HTTP mode**: starts a Streamable HTTP server. MCP clients connect over the network using the Streamable HTTP transport spec.

**Tools:** `search_sra`, `get_metadata`, `find_similar`, `export_results`

**Resources:** `sra://stats`, `sra://search/recent`, `sra://studies/{accession}`

**Prompts:** `biomedical_search`, `sample_selection`

```bash
# Claude Desktop configuration (~/.claude/claude_desktop_config.json):
{
  "mcpServers": {
    "srake": {
      "command": "srake",
      "args": ["mcp"]
    }
  }
}

# With custom database
srake mcp --db /path/to/srake.db

# HTTP transport for remote access
srake mcp --transport http --port 8081

# HTTP transport on all interfaces
srake mcp --transport http --host 0.0.0.0 --port 9090
```

---

## `srake metadata`

Retrieve metadata for specific accessions.

```bash
srake metadata <accession> [accessions...] [flags]
```

| Flag | Description |
|------|-------------|
| `-f, --format <type>` | Output format: table, json, yaml |
| `--fields <list>` | Comma-separated field list |
| `--expand` | Expand nested structures |

Supports SRP/DRP/ERP (study), SRX/DRX/ERX (experiment), SRS/DRS/ERS (sample), and SRR/DRR/ERR (run) accessions.

```bash
# Examples
srake metadata SRX123456
srake metadata SRP000001 --format json
```

---

## `srake db`

Database management commands.

### `srake db info`

Show database statistics.

```bash
srake db info
```

### `srake db stats`

Manage pre-computed statistics.

```bash
srake db stats --show
srake db stats --rebuild
```

### `srake db export`

Export the database to SRAmetadb.sqlite format.

```bash
srake db export [flags]
```

| Flag | Description |
|------|-------------|
| `-o, --output <file>` | Output path (default: SRAmetadb.sqlite) |
| `--db <path>` | Source database path |
| `--fts-version <n>` | FTS version: 3 or 5 (default: 5) |
| `--batch-size <n>` | Batch size (default: 10000) |
| `--progress` | Show progress (default: true) |
| `--compress` | Gzip compress output |
| `-f, --force` | Overwrite existing file |

```bash
# Examples
srake db export -o SRAmetadb.sqlite
srake db export -o legacy.sqlite --fts-version 3
srake db export -o SRAmetadb.sqlite.gz --compress
```

See [SRAmetadb Export](/docs/features/export) for details.

---

## `srake models`

Manage embedding models for vector search.

### `srake models list`

List installed models.

### `srake models download <model-id>`

Download a model. Flag: `--variant` (quantized, fp16, full).

### `srake models test <model-id> <text>`

Test a model with sample text.

```bash
# Examples
srake models list
srake models download Xenova/SapBERT-from-PubMedBERT-fulltext --variant quantized
srake models test Xenova/SapBERT-from-PubMedBERT-fulltext "breast cancer"
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `SRAKE_DB_PATH` | Database path |
| `SRAKE_INDEX_PATH` | Search index path |
| `SRAKE_MODELS_PATH` | Models directory |
| `SRAKE_EMBEDDINGS_PATH` | Embeddings directory |
| `SRAKE_MODEL_VARIANT` | Model variant: full, quantized, fp16 |
| `SRAKE_SEARCH_BACKEND` | Search backend: tiered, bleve, sqlite |
| `SRAKE_CONFIG` | Config file path |
| `NO_COLOR` | Disable colored output |
| `XDG_CONFIG_HOME` | XDG config directory |
| `XDG_DATA_HOME` | XDG data directory |
| `XDG_CACHE_HOME` | XDG cache directory |
| `XDG_STATE_HOME` | XDG state directory |

See [Configuration](/docs/reference/configuration) for full details.
