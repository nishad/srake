---
title: About SRAKE
---

SRAKE (SRA Knowledge Engine) is a command-line tool for ingesting, indexing, and querying NCBI Sequence Read Archive (SRA) metadata locally. Pronounced like Japanese sake (酒).

It processes the full SRA metadata XML archives (~14GB compressed) via streaming decompression directly into a local SQLite database, then provides full-text and vector similarity search over the data.

## Architecture

- **Streaming ingestion**: HTTP/file to gzip to tar to XML to SQLite in a single pass
- **SQLite + FTS5**: Primary storage with full-text search virtual tables
- **Bleve**: Full-text search engine with BM25 ranking
- **SapBERT embeddings**: Optional biomedical vector similarity search via ONNX Runtime
- **REST API**: HTTP server with JSON endpoints for search, metadata, and statistics
- **MCP**: Model Context Protocol support for AI assistant integration

## Status

SRAKE is a hackathon project developed at [BioHackathon 2025, Mie, Japan](https://2025.biohackathon.org/). It is experimental and not production-ready.

## Links

- [GitHub Repository](https://github.com/nishad/srake)
- MIT License
