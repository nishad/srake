---
title: Resume
weight: 2
---

# Resume

SRAKE tracks ingestion progress and can resume from the last checkpoint after interruption.

## How it works

During ingestion, SRAKE periodically saves progress (processed files, record counts, byte offsets) to the state directory (`~/.local/state/srake/resume/`). If the process is interrupted, running the same command again detects the previous progress and offers to resume.

## Usage

```bash
# Start ingestion
srake ingest --file archive.tar.gz

# If interrupted, run the same command again
srake ingest --file archive.tar.gz
# Detects previous progress and resumes from last checkpoint
```

### Force a fresh start

```bash
srake ingest --file archive.tar.gz --force
```

## Resume with filters

Filters are preserved across resume:

```bash
# Original command
srake ingest --auto --taxon-ids 9606 --platforms ILLUMINA

# Resume applies the same filters
srake ingest --auto --taxon-ids 9606 --platforms ILLUMINA
```

## State files

Resume state is stored in `~/.local/state/srake/resume/` (or `$SRAKE_STATE_HOME/resume/`). These files are safe to delete if you want to force a clean start without using `--force`.
