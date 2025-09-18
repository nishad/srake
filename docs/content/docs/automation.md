---
title: Automation & Scripting
weight: 50
prev: /docs/examples
next: /docs/compatibility
---

# Automation & Scripting Guide

srake is designed to work seamlessly in automated workflows and scripts, following [clig.dev](https://clig.dev) best practices for command-line interfaces.

## Non-Interactive Mode

When running srake in scripts or CI/CD pipelines, use the `--yes` flag to automatically accept all prompts:

```bash
#!/bin/bash
# Automated daily ingest script
srake ingest --auto --yes --quiet
```

## Pipeline Composition with stdin

srake commands support stdin input, making them perfect for Unix pipelines:

### Chaining Commands

```bash
# Find all RNA-Seq experiments and download them
srake search "RNA-Seq" --format tsv | \
  cut -f1 | \
  srake download --type fastq --parallel 4

# Convert a list of accessions
cat accessions.txt | srake convert --to GSE --format json > converted.json

# Process search results through multiple tools
srake search "homo sapiens" --limit 1000 | \
  grep "ILLUMINA" | \
  cut -f1 | \
  srake metadata --format json
```

### Batch Processing

```bash
# Process accessions from a file
while IFS= read -r accession; do
  srake convert "$accession" --to GSE --quiet
done < accessions.txt

# Or use stdin directly
cat accessions.txt | srake convert --to GSE --output results.json
```

## Dry Run Mode

Test your commands without making changes using `--dry-run`:

```bash
# Preview what would be downloaded
srake download SRP123456 --dry-run

# Check conversions before executing
echo -e "SRP001\nSRP002\nSRP003" | srake convert --to GSE --dry-run
```

## Debugging Scripts

Use the `--debug` flag to troubleshoot issues:

```bash
# Enable debug output for detailed logging
srake download SRR123456 --debug 2> debug.log

# Combine with verbose for maximum information
srake convert SRP123456 --to GSE --debug --verbose
```

## Error Handling

srake follows Unix conventions for exit codes:
- `0`: Success
- `1`: General error
- `2`: Command line usage error

```bash
#!/bin/bash
set -e  # Exit on any error

# Check if download succeeded
if srake download SRR123456 --yes --quiet; then
  echo "Download successful"
else
  echo "Download failed with exit code $?"
  exit 1
fi
```

## Output Formats for Scripts

Use structured output formats for easier parsing:

```bash
# JSON for complex processing
srake search "mouse" --format json | jq '.[] | .accession'

# TSV for simple column extraction
srake search "human" --format tsv --no-header | awk '{print $1}'

# CSV for spreadsheet tools
srake convert SRP123456 --to GSE --format csv > results.csv
```

## Parallel Processing

Leverage GNU parallel for large-scale processing:

```bash
# Download multiple accessions in parallel
cat accessions.txt | parallel -j 4 srake download {} --yes --quiet

# Convert accessions in parallel
parallel -j 8 srake convert {} --to GSE ::: SRP001 SRP002 SRP003
```

## Cron Jobs

Example cron job for automated daily ingestion:

```bash
# Daily SRA metadata update at 2 AM
0 2 * * * /usr/local/bin/srake ingest --daily --yes --quiet >> /var/log/srake.log 2>&1

# Weekly full ingest on Sundays
0 3 * * 0 /usr/local/bin/srake ingest --auto --yes --force >> /var/log/srake.log 2>&1
```

## Docker Integration

Run srake in containerized environments:

```bash
# Non-interactive Docker execution
docker run -v $(pwd)/data:/data \
  srake-image \
  srake ingest --auto --yes --db /data/metadata.db

# Pipe data into containerized srake
cat accessions.txt | docker run -i srake-image \
  srake convert --to GSE --format json
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Update SRA Metadata
on:
  schedule:
    - cron: '0 0 * * *'  # Daily at midnight
  workflow_dispatch:

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install srake
        run: go install github.com/nishad/srake/cmd/srake@latest

      - name: Update metadata
        run: |
          srake ingest --auto --yes --quiet
          srake db info

      - name: Upload database
        uses: actions/upload-artifact@v3
        with:
          name: sra-metadata
          path: ./data/metadata.db
```

### GitLab CI Example

```yaml
update-sra-metadata:
  stage: data
  script:
    - srake ingest --auto --yes --quiet
    - srake search "homo sapiens" --limit 100 --format json > latest_human.json
  artifacts:
    paths:
      - ./data/metadata.db
      - latest_human.json
  only:
    - schedules
```

## Shell Functions

Create helpful shell functions for common tasks:

```bash
# Add to ~/.bashrc or ~/.zshrc

# Quick SRA to GEO conversion
sra2geo() {
  echo "$1" | srake convert --to GSE --quiet | tail -1
}

# Download helper with defaults
sra_download() {
  srake download "$@" --type fastq --parallel 4 --yes
}

# Search and count results
sra_count() {
  srake search "$1" --format tsv --no-header | wc -l
}

# Usage
$ sra2geo SRP123456
GSE98765

$ sra_download SRR123456 SRR123457
# Downloads with optimized settings

$ sra_count "homo sapiens RNA-Seq"
1234
```

## Best Practices

1. **Always use `--yes` in scripts** to avoid hanging on prompts
2. **Use `--quiet` to suppress non-essential output** in production scripts
3. **Enable `--debug` when developing** to understand command behavior
4. **Test with `--dry-run` first** before running destructive operations
5. **Check exit codes** for proper error handling
6. **Use structured output formats** (JSON/TSV) for reliable parsing
7. **Leverage stdin** for composability with other Unix tools
8. **Set appropriate timeouts** for network operations in CI/CD

## Environment Variables

srake respects standard environment variables:

```bash
# Disable colored output
export NO_COLOR=1

# Custom database location
export SRAKE_DB=/custom/path/metadata.db

# Run with environment overrides
NO_COLOR=1 srake search "mouse" --format table
```

## Logging

Redirect output streams for logging:

```bash
# Log errors only
srake ingest --auto 2> errors.log

# Log everything
srake ingest --auto --verbose > output.log 2>&1

# Separate stdout and stderr
srake search "human" > results.txt 2> errors.log

# Tee for both console and file
srake ingest --auto 2>&1 | tee -a srake.log
```

This guide ensures your srake automation is robust, maintainable, and follows Unix philosophy for maximum interoperability.