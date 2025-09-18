# Vector Search Setup Guide

## Overview
srake supports semantic vector search using Bleve v2.5+ with FAISS backend. This enables finding biologically similar studies using embedding vectors generated from ONNX models.

## Prerequisites

### 1. Install FAISS Library
Vector search requires the FAISS library. Run the provided setup script:

```bash
./scripts/setup-faiss.sh
```

This script will:
- Install required dependencies (cmake, OpenBLAS)
- Clone and build Bleve's FAISS fork
- Install FAISS C API libraries to `/usr/local`

### 2. Build with Vector Support
After installing FAISS, build srake with the `vectors` tag:

```bash
go build -tags "search vectors" ./cmd/srake
```

Or update your Makefile:
```makefile
TAGS = -tags "search vectors"
```

## Configuration
Enable vectors in your config file (`~/.srake/config.yaml`):

```yaml
vectors:
  enabled: true

embeddings:
  enabled: true
  model_path: "Xenova/SapBERT-from-PubMedBERT-fulltext"
  cache_dir: "~/.srake/models"
```

## Troubleshooting

### FAISS Library Not Found
If you get errors about missing FAISS headers:
- Ensure FAISS is installed: `ls /usr/local/lib/libfaiss_c.*`
- On macOS: May need to set `DYLD_LIBRARY_PATH=/usr/local/lib`
- On Linux: Run `sudo ldconfig` after installation

### Build Failures
- Ensure you're using the `vectors` build tag
- Check Go version is 1.23+ (required by Bleve v2.5)
- Verify FAISS installation completed successfully

## Testing Vector Search
```go
// Example: Find similar studies
results, err := searchManager.FindSimilar("SRP123456", SearchOptions{
    Limit: 10,
})
```

## Performance Notes
- Vector indexing is ~2-3x slower than text-only indexing
- Requires ~768 float32 values per document (3KB per embedding)
- Hybrid search (text + vector) provides best results