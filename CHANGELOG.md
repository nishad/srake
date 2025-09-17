# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Resume Capability**: Comprehensive resume functionality for interrupted processing
  - Automatic progress tracking with database-backed persistence
  - Checkpoint system for reliable recovery points
  - File-level deduplication to skip already-processed XML files
  - HTTP Range request support for partial downloads
  - Interactive resume prompts with progress statistics
  - Force restart option to override existing progress
- **Progress Tracking**: Real-time progress monitoring
  - Live progress bars with ETA calculation
  - Detailed statistics (speed, records/sec, completion percentage)
  - Checkpoint creation at configurable intervals
  - Status command to check current/last ingestion
- **Error Recovery**: Automatic retry with exponential backoff
  - Network failure handling with configurable retry attempts
  - Corruption detection and recovery
  - Transaction-based safety for data integrity
- **Code Modularization**: Refactored main.go into separate command modules
  - helpers.go - utility functions
  - server.go - API server functionality
  - search.go - search command implementation
  - database.go - database management
  - metadata.go - metadata retrieval
  - models.go - ML model management
  - embed.go - embedding support

### Changed
- **BREAKING**: Renamed `download` command to `ingest` for semantic correctness
  - The `ingest` command now handles both remote NCBI files and local archives
  - Use `srake ingest --file /path/to/local.tar.gz` for local files
  - Use `srake ingest --auto/--daily/--monthly` for NCBI files
- Added `ProcessFile` method to StreamProcessor for local file processing
- Enhanced database module with GetSQLDB() method for progress tracking
- Improved error messages and user feedback throughout

### Removed
- Removed deprecated `download` command (replaced by `ingest`)

### Performance
- Resume capability adds < 5% overhead to normal processing
- Checkpoint creation takes < 100ms
- File lookup performance: 85ns per operation (16M ops/sec)
- Resume startup time: < 5 seconds

## [v0.0.1-alpha] - 2025-01-17

### Added
- Initial alpha release of srake - SRA metadata processor
- Zero-copy streaming architecture for processing large compressed archives
- Support for downloading NCBI SRA metadata (daily/monthly updates)
- SQLite database backend with optimized schema
- Full-text search capabilities
- Multiple output formats (table, JSON, CSV, TSV)
- REST API server for programmatic access
- Command-line interface with the following commands:
  - `download` - Download and process SRA metadata from NCBI
  - `search` - Search SRA metadata with filters
  - `metadata` - Get metadata for specific accessions
  - `server` - Start REST API server
  - `db info` - Show database statistics
  - `models` - Manage embedding models (experimental)
- Progress indicators for long-running operations
- Colored output with automatic terminal detection
- Configuration file support (YAML)
- 95%+ XSD schema compliance
- Memory-efficient processing (< 500MB constant usage)
- High performance (20,000+ records/second)
- Support for organism, platform, and strategy filtering
- Batch processing with transactions
- WAL mode for concurrent access
- Smart indexing for common queries
- Signal handling for graceful shutdown
- Comprehensive error handling and user-friendly messages

### Performance
- Process 14GB+ compressed archives without intermediate storage
- 20,000+ records/second throughput
- < 500MB constant memory usage
- < 100ms search response time for indexed queries
- < 10ms API response latency

### Documentation
- Comprehensive README with installation and usage instructions
- Contributing guidelines
- API documentation
- Command reference
- Performance benchmarks

### Infrastructure
- GitHub Actions CI/CD pipeline
- Multi-platform build support (Linux, macOS, Windows)
- Docker image support
- Automated testing and linting
- Security scanning with Gosec and Trivy
- Issue templates for bugs and features


---

## Release Types

- **Major (x.0.0)**: Incompatible API changes
- **Minor (0.x.0)**: Backwards-compatible functionality additions
- **Patch (0.0.x)**: Backwards-compatible bug fixes
- **Pre-release**: Alpha, beta, or release candidate versions

## Version History

| Version | Date | Type | Summary |
|---------|------|------|---------|
| v0.0.1-alpha | 2025-01-17 | Alpha | Initial alpha release with core functionality |

[Unreleased]: https://github.com/nishad/srake/compare/v0.0.1-alpha...HEAD
[v0.0.1-alpha]: https://github.com/nishad/srake/releases/tag/v0.0.1-alpha