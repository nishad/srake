#!/bin/bash
API_URL="http://localhost:8081/api/v1"

echo "=== Testing SRAKE API ==="
echo

echo "1. Test Search with different queries:"
echo "- Simple search:"
curl -s "$API_URL/search?query=cancer&limit=2" | python3 -m json.tool | head -20

echo
echo "- Search with ILLUMINA:"
curl -s "$API_URL/search?query=ILLUMINA&limit=2" | python3 -m json.tool | head -20

echo
echo "2. Test POST search with filters:"
curl -s -X POST "$API_URL/search" \
  -H "Content-Type: application/json" \
  -d '{"query":"sequencing","limit":2,"search_mode":"text"}' | python3 -m json.tool | head -20

echo
echo "3. Test metadata endpoints:"
echo "- Get study DRP000205:"
curl -s "$API_URL/studies/DRP000205" | python3 -m json.tool | head -15

echo
echo "4. Test statistics endpoint:"
curl -s "$API_URL/stats" | python3 -m json.tool

echo
echo "5. Test export (CSV):"
curl -s -X POST "$API_URL/export" \
  -H "Content-Type: application/json" \
  -d '{"query":"RNA-Seq","format":"csv","limit":3}' | head -5

echo
echo "6. Test MCP tool call:"
curl -s -X POST "http://localhost:8081/mcp" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_metadata","arguments":{"accession":"DRP000205"}},"id":2}' | python3 -m json.tool | head -20

echo
echo "=== All tests completed ==="
