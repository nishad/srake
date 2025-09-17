---
title: Resume Capability
weight: 2
---

srake includes intelligent resume functionality that handles interruptions gracefully, allowing you to continue processing from where you left off.

## Overview

Processing large SRA metadata archives (14GB+) can take significant time. Network issues, system crashes, or user interruptions can occur during processing. srake's resume capability ensures you never have to start over from the beginning.

### Key Features

- **Automatic Progress Tracking**: Real-time tracking of download and processing progress
- **Database-Backed Persistence**: Progress stored in SQLite for reliability
- **Checkpoint System**: Periodic checkpoints for accurate recovery points
- **File-Level Deduplication**: Skip already-processed XML files on resume
- **HTTP Range Support**: Resume downloads from exact byte position
- **Smart Recovery**: Automatically detect and resume interrupted sessions

## How It Works

### Progress Tracking

srake tracks multiple aspects of progress:

1. **Download Progress**
   - Bytes downloaded vs. total size
   - HTTP range support for partial downloads
   - Network failure recovery

2. **Processing Progress**
   - Current tar position in archive
   - Last processed XML file
   - Records inserted into database

3. **Checkpoint System**
   - Periodic checkpoints (default: every 1000 records)
   - Safe points for recovery
   - Minimal performance impact (< 100ms)

### Resume Detection

When you run an ingest command, srake automatically:

1. Checks for existing progress records
2. Validates the source matches
3. Offers to resume or start fresh
4. Resumes from the last safe checkpoint

## Using Resume

### Automatic Resume

Simply run the same command again after interruption:

```bash
# Original command
srake ingest --file NCBI_SRA_Full_20250818.tar.gz

# If interrupted, run again
srake ingest --file NCBI_SRA_Full_20250818.tar.gz

# Output:
Previous ingestion found:
  Source: NCBI_SRA_Full_20250818.tar.gz
  Progress: 45.3% complete (6.3GB/14GB)
  Records: 1,234,567 processed
  Started: 2025-01-17 10:30:00

Resume from last position? (y/n): y
Resuming from: experiment_batch_042.xml
[====================>.................] 45.3% | 6.3GB/14GB | ETA: 15 min
```

### Force Fresh Start

Override existing progress and start from beginning:

```bash
srake ingest --file archive.tar.gz --force

# Warning shown:
⚠️  Existing progress will be discarded
Continue? (y/n): y
Starting fresh ingestion...
```

### Check Status

View current or last ingestion status:

```bash
srake ingest --status

# Output:
Current Ingestion Status
────────────────────────
Source: NCBI_SRA_Full_20250818.tar.gz
State: In Progress
Progress: 67.8% complete
Records Processed: 2,345,678
Start Time: 2025-01-17 10:30:00
Last Update: 2025-01-17 11:45:23
Estimated Time Remaining: 12 minutes
```

### Configure Checkpoints

Adjust checkpoint frequency for your needs:

```bash
# Checkpoint every 5000 records (less frequent)
srake ingest --file archive.tar.gz --checkpoint 5000

# Checkpoint every 100 records (more frequent, safer)
srake ingest --file archive.tar.gz --checkpoint 100
```

### Interactive Mode

Get prompted before resuming:

```bash
srake ingest --file archive.tar.gz --interactive

# Always prompts:
Previous ingestion found. Resume? (y/n):
```

## Resume with Filters

Resume works seamlessly with filtering:

```bash
# Original filtered ingestion
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --platforms ILLUMINA

# Resume with same filters applied
srake ingest --file archive.tar.gz \
  --taxon-ids 9606 \
  --platforms ILLUMINA

# Filters are preserved and reapplied
```

## Performance Impact

Resume capability adds minimal overhead:

| Aspect | Impact |
|--------|--------|
| Memory Usage | < 1MB for progress tracking |
| Processing Speed | < 5% reduction |
| Checkpoint Time | < 100ms per checkpoint |
| Resume Time | < 5 seconds to restart |
| Database Size | < 100KB for progress records |

## Architecture

### Progress Database Schema

```sql
CREATE TABLE ingest_progress (
    id INTEGER PRIMARY KEY,
    source_url TEXT NOT NULL,
    source_hash TEXT UNIQUE NOT NULL,
    total_bytes INTEGER,
    downloaded_bytes INTEGER,
    processed_bytes INTEGER,
    last_tar_position INTEGER,
    last_xml_file TEXT,
    records_processed INTEGER,
    state TEXT,
    started_at TIMESTAMP,
    updated_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT
);

CREATE TABLE processed_files (
    id INTEGER PRIMARY KEY,
    progress_id INTEGER,
    filename TEXT NOT NULL,
    processed_at TIMESTAMP,
    FOREIGN KEY (progress_id) REFERENCES ingest_progress(id)
);
```

### Recovery Process

1. **Validation Phase**
   - Verify source file exists/accessible
   - Check source hash matches
   - Validate database consistency

2. **Restoration Phase**
   - Seek to last tar position
   - Skip processed XML files
   - Restore counters and statistics

3. **Continuation Phase**
   - Resume normal processing
   - Continue checkpoint creation
   - Update progress tracking

## Troubleshooting

### Resume Not Working

If resume doesn't work as expected:

1. **Check source file hasn't changed**
   ```bash
   # View progress details
   srake ingest --status
   ```

2. **Verify database integrity**
   ```bash
   # Database location
   ls -la ./data/metadata.db
   ```

3. **Force restart if needed**
   ```bash
   srake ingest --file archive.tar.gz --force
   ```

### Corrupted Progress

If progress records are corrupted:

```bash
# Clean up progress records
srake ingest --cleanup

# Start fresh
srake ingest --file archive.tar.gz
```

### Network Issues

For unstable networks:

```bash
# Increase retry attempts
srake ingest --file https://ftp.ncbi.nlm.nih.gov/sra/archive.tar.gz \
  --max-retries 10 \
  --retry-delay 30
```

## Best Practices

1. **Let It Resume**: Don't use `--force` unless necessary
2. **Regular Checkpoints**: Default (1000 records) works well for most cases
3. **Monitor Progress**: Use `--verbose` to see detailed progress
4. **Keep Database**: Don't delete metadata.db during processing
5. **Network Stability**: For large files, ensure stable connection

## Examples

### Research Workflow

```bash
# Start large ingestion Friday evening
srake ingest --monthly \
  --taxon-ids 9606 \
  --strategies RNA-Seq

# System maintenance interrupts at 30%
# Resume Monday morning
srake ingest --monthly \
  --taxon-ids 9606 \
  --strategies RNA-Seq

# Continues from 30%, completes successfully
```

### Batch Processing

```bash
# Process multiple archives with resume support
for archive in *.tar.gz; do
    srake ingest --file "$archive"
    # Each file has independent resume tracking
done
```

### Filtered Resume

```bash
# Complex filter with resume
srake ingest --file huge_archive.tar.gz \
  --taxon-ids 9606,10090 \
  --platforms ILLUMINA \
  --strategies RNA-Seq,WGS \
  --date-from 2024-01-01 \
  --min-reads 10000000

# Power failure at 60%

# Resume with exact same filters
srake ingest --file huge_archive.tar.gz \
  --taxon-ids 9606,10090 \
  --platforms ILLUMINA \
  --strategies RNA-Seq,WGS \
  --date-from 2024-01-01 \
  --min-reads 10000000

# Continues from 60% with filters intact
```

## Technical Details

### File Identification

Files are identified by:
- Source URL/path
- SHA-256 hash of first 1MB
- File size
- Modification time

### State Management

Progress states:
- `pending`: Initialized but not started
- `downloading`: Downloading from remote source
- `processing`: Processing tar archive
- `completed`: Successfully finished
- `failed`: Error occurred
- `cancelled`: User cancelled

### Cleanup Policy

- Progress records kept for 30 days after completion
- Failed attempts kept for 7 days
- Manual cleanup available via `--cleanup`

## Next Steps

- Learn about [Performance Optimizations](/docs/features/performance)
- Explore [Architecture Details](/docs/features/architecture)
- See [Real-World Examples](/docs/examples)