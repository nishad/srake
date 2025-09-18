---
title: "Configuration & Paths"
weight: 20
---

# Configuration & Paths

SRake follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) and [clig.dev](https://clig.dev) best practices for organizing its files and configuration.

## Directory Structure

SRake uses a clean, standardized directory structure that separates configuration, data, cache, and state:

### Base Directories

| Directory | Default Path | Purpose |
|-----------|-------------|---------|
| **Config** | `~/.config/srake/` | User configuration files |
| **Data** | `~/.local/share/srake/` | Essential persistent data (database, models) |
| **Cache** | `~/.cache/srake/` | Deletable cache data (downloads, indices) |
| **State** | `~/.local/state/srake/` | Application state (resume files, history) |

### Specific Paths

```bash
~/.config/srake/
└── config.yaml              # User configuration

~/.local/share/srake/
├── srake.db                 # Primary SQLite database
└── models/                  # ML models (persistent)
    └── Xenova/              # HuggingFace models
        └── SapBERT-from-PubMedBERT-fulltext/

~/.cache/srake/
├── downloads/               # Downloaded NCBI archives
│   ├── NCBI_SRA_Metadata_20250915.tar.gz
│   └── checksums.sha256
├── index/                   # Search indices (can be rebuilt)
│   └── srake.bleve/         # Bleve search index
├── embeddings/              # Computed embeddings cache
└── search/                  # Search result cache

~/.local/state/srake/
├── resume/                  # Resume/checkpoint files
│   └── checkpoint-*.json
└── history.log              # Operation history
```

## Environment Variables

SRake respects environment variables for customizing paths. These follow a clear precedence order:

### Precedence Order (highest to lowest)
1. Command-line flags (`--db`, `--cache`, etc.)
2. Environment variables
3. Config file (`~/.config/srake/config.yaml`)
4. Built-in defaults

### Available Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SRAKE_CONFIG_HOME` | Override config directory | `$XDG_CONFIG_HOME/srake` or `~/.config/srake` |
| `SRAKE_DATA_HOME` | Override data directory | `$XDG_DATA_HOME/srake` or `~/.local/share/srake` |
| `SRAKE_CACHE_HOME` | Override cache directory | `$XDG_CACHE_HOME/srake` or `~/.cache/srake` |
| `SRAKE_STATE_HOME` | Override state directory | `$XDG_STATE_HOME/srake` or `~/.local/state/srake` |
| `SRAKE_DB_PATH` | Override database location | `$SRAKE_DATA_HOME/srake.db` |
| `SRAKE_INDEX_PATH` | Override index location | `$SRAKE_CACHE_HOME/index/srake.bleve` |
| `SRAKE_MODELS_PATH` | Override models location | `$SRAKE_DATA_HOME/models` |
| `SRAKE_NO_COLOR` | Disable colored output | Also respects `NO_COLOR` |
| `SRAKE_DEBUG` | Enable debug logging | |
| `SRAKE_VERBOSE` | Enable verbose output | |

### Examples

```bash
# Use a different database location
SRAKE_DB_PATH=/mnt/ssd/srake.db srake search "RNA-Seq"

# Use temporary cache for testing
SRAKE_CACHE_HOME=/tmp/srake-cache srake ingest --daily

# Override all base directories
export SRAKE_DATA_HOME=/data/srake
export SRAKE_CACHE_HOME=/cache/srake
srake ingest --auto
```

## Configuration Management

### View Current Paths

Display all active paths and any environment variable overrides:

```bash
srake config paths
```

Output:
```
SRake Paths
────────────────────────────────────────
Base Directories:
  Config:   /Users/alice/.config/srake
  Data:     /Users/alice/.local/share/srake
  Cache:    /Users/alice/.cache/srake
  State:    /Users/alice/.local/state/srake

Specific Paths:
  Database:  /Users/alice/.local/share/srake/srake.db
  Index:     /Users/alice/.cache/srake/index/srake.bleve
  Models:    /Users/alice/.local/share/srake/models
  Downloads: /Users/alice/.cache/srake/downloads

Environment Variables:
  SRAKE_DB_PATH = /mnt/fast/srake.db

Path Status:
  Config Dir:  ✓ exists
  Data Dir:    ✓ exists
  Database:    ✓ exists
  Index:       ✗ not found
```

### Initialize Configuration

Create a default configuration file:

```bash
# Create default config
srake config init

# Force overwrite existing config
srake config init --force
```

### View Configuration

Display the current configuration settings:

```bash
srake config show
```

### Edit Configuration

Open the configuration file in your default editor:

```bash
srake config edit
```

This uses the `$EDITOR` environment variable, falling back to `vi` if not set.

## Configuration File

The configuration file is located at `~/.config/srake/config.yaml` and uses YAML format:

```yaml
# Data directory for persistent files
data_directory: /home/alice/.local/share/srake

# Database configuration
database:
  path: /home/alice/.local/share/srake/srake.db
  cache_size: 10000      # in KB (40MB)
  mmap_size: 268435456   # in bytes (256MB)
  journal_mode: WAL

# Search configuration
search:
  enabled: true
  backend: bleve
  index_path: /home/alice/.cache/srake/index/srake.bleve
  rebuild_on_start: false
  auto_sync: true
  sync_interval: 300     # seconds
  default_limit: 100
  batch_size: 1000
  use_cache: true
  cache_ttl: 3600        # seconds

# Vector search configuration
vectors:
  enabled: true
  requires_search: true
  similarity_metric: cosine
  use_quantized: true
  dimensions: 768
  optimization: memory_efficient

# Embedding configuration
embeddings:
  enabled: true
  models_directory: /home/alice/.local/share/srake/models
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

## Cache Management

SRake provides comprehensive cache management commands to control disk usage:

### View Cache Information

```bash
srake cache info
```

Shows cache directories, sizes, and file counts:
```
Cache Directories
────────────────────────────────────────
Cache Subdirectories:
  Downloads:      /Users/alice/.cache/srake/downloads (1.2 GB, 5 files)
  Search Index:   /Users/alice/.cache/srake/index (450 MB, 1 files)
  Embeddings:     /Users/alice/.cache/srake/embeddings (120 MB, 1523 files)
  Search Cache:   /Users/alice/.cache/srake/search (15 MB, 203 files)

Total cache size: 1785.00 MB
```

### Clean Cache

Remove cache files to free up disk space:

```bash
# Remove downloads older than 30 days
srake cache clean --older 30d

# Remove all downloads
srake cache clean --downloads

# Remove search result cache
srake cache clean --search

# Remove search index (will need rebuild)
srake cache clean --index

# Remove everything (requires confirmation)
srake cache clean --all
```

Options:
- `--all`: Remove all cache including indices
- `--older <duration>`: Remove files older than specified duration (e.g., 30d, 24h)
- `--search`: Remove cached search results
- `--downloads`: Remove downloaded files
- `--index`: Remove search index (requires rebuild with `srake search index --build`)

## Best Practices

### Backup

Important data to backup:
- `~/.local/share/srake/srake.db` - Your database
- `~/.config/srake/config.yaml` - Your configuration
- `~/.local/share/srake/models/` - Downloaded models (optional, can be re-downloaded)

### Performance Optimization

1. **Fast SSD for Database**: Place the database on fast storage
   ```bash
   SRAKE_DB_PATH=/mnt/nvme/srake.db srake ingest --auto
   ```

2. **Separate Cache Drive**: Use a different drive for cache
   ```bash
   SRAKE_CACHE_HOME=/mnt/scratch/srake-cache srake search "human"
   ```

3. **RAM Disk for Temporary Work**: Use tmpfs for intensive operations
   ```bash
   SRAKE_CACHE_HOME=/dev/shm/srake srake search index --rebuild
   ```

### Multi-User Setup

For shared systems, each user automatically gets their own paths:
- Alice: `~alice/.local/share/srake/`
- Bob: `~bob/.local/share/srake/`

For a shared database, set:
```bash
export SRAKE_DB_PATH=/shared/data/srake.db
export SRAKE_INDEX_PATH=/shared/cache/srake.bleve
```

### Docker/Container Usage

When using SRake in containers, mount volumes for persistence:

```dockerfile
# Dockerfile example
ENV SRAKE_DATA_HOME=/data
ENV SRAKE_CACHE_HOME=/cache
ENV SRAKE_CONFIG_HOME=/config

VOLUME ["/data", "/cache", "/config"]
```

```bash
# Docker run example
docker run -v /host/data:/data \
           -v /host/cache:/cache \
           -v /host/config:/config \
           srake:latest search "RNA-Seq"
```

## Troubleshooting

### Permission Issues

If you encounter permission errors:

```bash
# Check directory permissions
ls -la ~/.local/share/srake/

# Fix permissions
chmod -R u+rwX ~/.local/share/srake/
chmod -R u+rwX ~/.cache/srake/
```

### Path Not Found

If paths don't exist, SRake creates them automatically on startup. To manually create:

```bash
# SRake will create all necessary directories
srake config paths
```

### Reset to Defaults

To completely reset SRake:

```bash
# Remove all SRake data (careful!)
rm -rf ~/.config/srake
rm -rf ~/.local/share/srake
rm -rf ~/.cache/srake
rm -rf ~/.local/state/srake

# Start fresh
srake config init
srake ingest --auto
```