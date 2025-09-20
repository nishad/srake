---
title: About SRAKE
type: about
---

# About SRAKE - SRA Knowledge Engine

*Pronunciation: Like Japanese sake (酒) — "srah-keh"*

## What is SRAKE?

SRAKE (SRA Knowledge Engine) is a blazing-fast, memory-efficient tool for processing and querying NCBI SRA (Sequence Read Archive) metadata. Built with a zero-copy streaming architecture, SRAKE can process multi-gigabyte compressed archives without intermediate storage, making it ideal for bioinformatics workflows and large-scale genomic data analysis.

## Key Features

- **Streaming Architecture**: Process 14GB+ compressed archives without intermediate storage
- **High Performance**: 20,000+ records/second throughput with concurrent processing
- **Memory Efficient**: Constant < 500MB memory usage regardless of file size
- **Resume Capability**: Intelligent resume from interruption point with progress tracking
- **SQLite Backend**: Optimized schema with full-text search and smart indexing
- **Quality-Controlled Search**: Multiple search modes with similarity thresholds and confidence scoring
- **Vector Embeddings**: Semantic search using SapBERT for biomedical concepts

## Project Status

⚠️ **Important Notice**: SRAKE is a hackathon project developed at [BioHackathon 2025, Mie, Japan](https://2025.biohackathon.org/). It is currently in pre-alpha stage and not production-ready. Please treat it as an experimental tool for exploration and testing only.

## Contributing

Bug reports and feature requests are welcome! Please visit our [GitHub repository](https://github.com/nishad/srake) to contribute or report issues.

## License

SRAKE is released under the MIT License. See the LICENSE file for details.
