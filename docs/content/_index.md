---
title: srake 🍶🧬
toc: false
---

# srake - Blazing-Fast SRA Metadata Processing

> Process multi-gigabyte NCBI SRA metadata archives with zero-copy streaming, intelligent filtering, and resume capabilities

## Quick Start

```bash
# Install srake
go install github.com/nishad/srake/cmd/srake@latest

# Ingest SRA metadata
srake ingest --auto

# Search the database
srake search "homo sapiens"
```

## Key Features

### ⚡ Lightning Fast
Process 20,000+ records per second with concurrent processing and optimized SQLite backend

### 💾 Memory Efficient
Constant < 500MB memory usage regardless of file size with zero-copy streaming architecture

### 🔍 Smart Filtering
Filter by taxonomy, organism, platform, date ranges, and quality metrics during ingestion

### 🔄 Resume Support
Intelligent resume from interruption with checkpoint system and progress tracking

### 📊 Full-Text Search
Query your database with optimized SQLite full-text search and smart indexing

### 🌐 REST API
Built-in API server for programmatic access to your SRA metadata

## Documentation

- [Getting Started](/docs/getting-started) - Installation and quick start guide
- [Features](/docs/features) - Explore all capabilities
- [API Reference](/docs/api) - REST API and Go library documentation
- [Examples](/docs/examples) - Real-world usage examples

## Performance

| Metric | Performance |
|--------|-------------|
| **Throughput** | 20,000+ records/second |
| **Memory Usage** | < 500MB constant |
| **Large File Support** | 14GB+ without disk storage |
| **Resume Time** | < 5 seconds |
| **Filter Overhead** | < 5% |

## Get Started

[Read the Documentation](/docs) or [View on GitHub](https://github.com/nishad/srake)