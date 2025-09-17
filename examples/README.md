# srake Examples

This directory contains example code for using srake as a library.

## Basic Example

The `basic/` directory shows how to:
- Open a database connection
- Create a stream processor
- Process SRA metadata from a URL

Run the example:
```bash
go run examples/basic/main.go
```

## Using srake as a Library

```go
import (
    "github.com/nishad/srake/internal/database"
    "github.com/nishad/srake/internal/processor"
)
```

Note: The `internal/` packages are not meant for external use.
Future versions may expose a public API in `pkg/`.
