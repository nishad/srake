#!/bin/bash

# Test script for export functionality

echo "Testing SRAmetadb export functionality"
echo "======================================"

# Test FTS5 export
echo -e "\n1. Testing FTS5 export..."
./srake db export --db test_import.db -o /tmp/test_fts5.sqlite --force --fts-version 5 --batch-size 1000 --quiet

if [ $? -eq 0 ]; then
    echo "✓ FTS5 export completed"
    echo "  Checking output..."
    sqlite3 /tmp/test_fts5.sqlite ".tables" | head -5
    echo "  Size: $(du -h /tmp/test_fts5.sqlite | cut -f1)"
else
    echo "✗ FTS5 export failed"
fi

# Test FTS3 export
echo -e "\n2. Testing FTS3 export..."
./srake db export --db test_import.db -o /tmp/test_fts3.sqlite --force --fts-version 3 --batch-size 1000 --quiet

if [ $? -eq 0 ]; then
    echo "✓ FTS3 export completed"
    echo "  Checking output..."
    sqlite3 /tmp/test_fts3.sqlite ".tables" | head -5
    echo "  Size: $(du -h /tmp/test_fts3.sqlite | cut -f1)"
else
    echo "✗ FTS3 export failed"
fi

# Test compressed export
echo -e "\n3. Testing compressed export..."
./srake db export --db test_import.db -o /tmp/test_compressed.sqlite --force --compress --quiet

if [ $? -eq 0 ]; then
    echo "✓ Compressed export completed"
    echo "  Size: $(du -h /tmp/test_compressed.sqlite.gz | cut -f1)"
else
    echo "✗ Compressed export failed"
fi

echo -e "\n======================================"
echo "Export tests completed"