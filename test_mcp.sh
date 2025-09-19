#!/bin/bash

echo "=== Testing MCP (Model Context Protocol) ==="
echo
API_URL="http://localhost:8081"

echo "1. Test MCP capabilities:"
curl -s "$API_URL/mcp/capabilities" | python3 -m json.tool
echo

echo "2. Test MCP list_tools:"
curl -s -X POST "$API_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}' | python3 -m json.tool
echo

echo "3. Test MCP list_prompts:"
curl -s -X POST "$API_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"prompts/list","params":{},"id":2}' | python3 -m json.tool
echo

echo "4. Test MCP search_sra tool:"
curl -s -X POST "$API_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"search_sra","arguments":{"query":"human RNA-Seq","limit":3}},"id":3}' | python3 -m json.tool | head -40
echo

echo "5. Test MCP find_similar tool:"
curl -s -X POST "$API_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"find_similar","arguments":{"text":"breast cancer transcriptome analysis","limit":2}},"id":4}' | python3 -m json.tool | head -30
echo

echo "6. Test MCP export_results tool:"
curl -s -X POST "$API_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"export_results","arguments":{"query":"ILLUMINA","format":"csv","limit":3}},"id":5}' | head -10
echo

echo "7. Test MCP get_metadata with experiment:"
# First find a valid experiment ID
EXPERIMENT_ID=$(curl -s "$API_URL/api/v1/search?query=RNA-Seq&limit=1" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['results'][0]['id'] if data['results'] else '')")
echo "Using experiment ID: $EXPERIMENT_ID"
curl -s -X POST "$API_URL/mcp" \
  -H "Content-Type: application/json" \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"get_metadata\",\"arguments\":{\"accession\":\"$EXPERIMENT_ID\"}},\"id\":6}" | python3 -m json.tool | head -30
echo

echo "8. Test MCP get_prompt (search_help):"
curl -s -X POST "$API_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"prompts/get","params":{"name":"search_help"},"id":7}' | python3 -m json.tool
echo

echo "9. Test MCP batch request:"
curl -s -X POST "$API_URL/mcp" \
  -H "Content-Type: application/json" \
  -d '[
    {"jsonrpc":"2.0","method":"tools/call","params":{"name":"search_sra","arguments":{"query":"cancer","limit":1}},"id":"batch1"},
    {"jsonrpc":"2.0","method":"tools/call","params":{"name":"search_sra","arguments":{"query":"mouse","limit":1}},"id":"batch2"}
  ]' | python3 -m json.tool | head -40
echo

echo "=== MCP Testing Complete ==="
