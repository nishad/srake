package mcp

import (
	"encoding/json"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// detectAccessionType returns the SRA record type based on accession prefix.
func detectAccessionType(accession string) string {
	if strings.HasPrefix(accession, "SRP") || strings.HasPrefix(accession, "ERP") || strings.HasPrefix(accession, "DRP") {
		return "study"
	}
	if strings.HasPrefix(accession, "SRX") || strings.HasPrefix(accession, "ERX") || strings.HasPrefix(accession, "DRX") {
		return "experiment"
	}
	if strings.HasPrefix(accession, "SRS") || strings.HasPrefix(accession, "ERS") || strings.HasPrefix(accession, "DRS") {
		return "sample"
	}
	if strings.HasPrefix(accession, "SRR") || strings.HasPrefix(accession, "ERR") || strings.HasPrefix(accession, "DRR") {
		return "run"
	}
	return ""
}

// toJSON marshals v to a JSON string. On error it returns the error message.
func toJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return `{"error":"` + err.Error() + `"}`
	}
	return string(b)
}

// errResult returns a CallToolResult flagged as an error.
func errResult(msg string) *gomcp.CallToolResult {
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{&gomcp.TextContent{Text: msg}},
		IsError: true,
	}
}
