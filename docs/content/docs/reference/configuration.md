---
title: Configuration
weight: 20
---

SRAKE follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html).

## Directory structure

| Directory | Default Path | Purpose |
|-----------|-------------|---------|
| Config | `~/.config/srake/` | Configuration file |
| Data | `~/.local/share/srake/` | Database and models |
| Cache | `~/.cache/srake/` | Downloads, search index |
| State | `~/.local/state/srake/` | Resume checkpoints |

```
~/.config/srake/
  config.yaml

~/.local/share/srake/
  srake.db
  models/

~/.cache/srake/
  downloads/
  index/
  embeddings/

~/.local/state/srake/
  resume/
```

## Environment variables

**Path overrides:**

| Variable | Default | Description |
|----------|---------|-------------|
| `SRAKE_DB_PATH` | `~/.local/share/srake/srake.db` | Database path |
| `SRAKE_INDEX_PATH` | adjacent to database | Search index path |
| `SRAKE_MODELS_PATH` | `~/.local/share/srake/models` | Models directory |
| `SRAKE_EMBEDDINGS_PATH` | adjacent to database | Embeddings directory |
| `SRAKE_CONFIG` | `~/.config/srake/config.yaml` | Config file path |

**XDG fallbacks** (used when SRAKE-specific vars are not set):

| Variable | Default |
|----------|---------|
| `XDG_CONFIG_HOME` | `~/.config` |
| `XDG_DATA_HOME` | `~/.local/share` |
| `XDG_CACHE_HOME` | `~/.cache` |
| `XDG_STATE_HOME` | `~/.local/state` |

**Search and output:**

| Variable | Description |
|----------|-------------|
| `SRAKE_MODEL_VARIANT` | Embedding model variant: full, quantized, fp16 |
| `SRAKE_SEARCH_BACKEND` | Search backend: tiered, bleve, sqlite |
| `NO_COLOR` | Disable colored output |

**Precedence** (highest to lowest):
1. Command-line flags
2. Environment variables
3. Config file
4. Built-in defaults

## Config file

Location: `~/.config/srake/config.yaml`

```yaml
data_directory: ~/.local/share/srake

database:
  path: ~/.local/share/srake/srake.db
  cache_size: 10000        # KB
  mmap_size: 268435456     # bytes (256MB)
  journal_mode: WAL

search:
  enabled: true
  backend: tiered          # tiered, bleve, or sqlite
  index_path: ~/.cache/srake/index/srake.bleve
  default_limit: 100
  batch_size: 1000

vectors:
  enabled: true
  similarity_metric: cosine
  use_quantized: true
  dimensions: 768

embeddings:
  enabled: true
  models_directory: ~/.local/share/srake/models
  default_model: Xenova/SapBERT-from-PubMedBERT-fulltext
  default_variant: quantized
  batch_size: 32
  num_threads: 4
  max_text_length: 512
  cache_embeddings: true
  combine_fields:
    - organism
    - library_strategy
    - title
    - abstract
```

## Examples

```bash
# Use fast storage for the database
SRAKE_DB_PATH=/mnt/nvme/srake.db srake ingest --auto

# Temporary cache for testing
SRAKE_CACHE_HOME=/tmp/srake srake index --build

# Shared database for multiple users
export SRAKE_DB_PATH=/shared/data/srake.db
```
