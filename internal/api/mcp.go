package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nishad/srake/internal/service"
)

// MCPRequest represents a JSON-RPC 2.0 request for MCP
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

// MCPResponse represents a JSON-RPC 2.0 response for MCP
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// MCPError represents a JSON-RPC 2.0 error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Tools available to AI assistants
var mcpTools = []map[string]interface{}{
	{
		"name":        "search_sra",
		"description": "Search NCBI SRA metadata with quality control",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]string{
					"type":        "string",
					"description": "Search query",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results",
					"default":     20,
				},
				"organism": map[string]string{
					"type":        "string",
					"description": "Filter by organism",
				},
				"library_strategy": map[string]string{
					"type":        "string",
					"description": "Filter by library strategy (e.g., RNA-Seq)",
				},
				"similarity_threshold": map[string]interface{}{
					"type":        "number",
					"description": "Minimum similarity score (0-1)",
					"default":     0.5,
				},
			},
			"required": []string{"query"},
		},
	},
	{
		"name":        "get_metadata",
		"description": "Get detailed metadata for a specific accession",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"accession": map[string]string{
					"type":        "string",
					"description": "SRA accession (e.g., SRP123456)",
				},
			},
			"required": []string{"accession"},
		},
	},
	{
		"name":        "find_similar",
		"description": "Find similar studies using vector search",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]string{
					"type":        "string",
					"description": "Text to find similar studies for",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results",
					"default":     10,
				},
			},
			"required": []string{"query"},
		},
	},
	{
		"name":        "export_results",
		"description": "Export search results in various formats",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]string{
					"type":        "string",
					"description": "Search query",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Export format",
					"enum":        []string{"json", "csv", "tsv"},
					"default":     "json",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results",
					"default":     100,
				},
			},
			"required": []string{"query"},
		},
	},
}

// MCP Resources available
var mcpResources = []map[string]interface{}{
	{
		"uri":         "sra://search/recent",
		"name":        "Recent SRA Studies",
		"description": "Access recently added SRA studies",
		"mimeType":    "application/json",
	},
	{
		"uri":         "sra://stats",
		"name":        "Database Statistics",
		"description": "Get current database statistics",
		"mimeType":    "application/json",
	},
}

// MCP Prompts templates
var mcpPrompts = []map[string]interface{}{
	{
		"name":        "biomedical_search",
		"description": "Search for biomedical research data",
		"arguments": []map[string]string{
			{
				"name":        "disease",
				"description": "Disease or condition to search for",
				"required":    "true",
			},
			{
				"name":        "organism",
				"description": "Organism (default: human)",
				"required":    "false",
			},
		},
		"template": "Search for {{disease}} studies in {{organism|human}} using RNA-Seq or similar high-throughput methods",
	},
	{
		"name":        "sample_selection",
		"description": "Select samples for analysis",
		"arguments": []map[string]string{
			{
				"name":        "criteria",
				"description": "Selection criteria",
				"required":    "true",
			},
		},
		"template": "Find samples matching: {{criteria}}. Focus on data quality, sample size, and experimental design.",
	},
}

// handleMCP handles MCP JSON-RPC requests
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeMCPError(w, nil, -32700, "Parse error")
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		s.writeMCPError(w, req.ID, -32600, "Invalid Request")
		return
	}

	// Handle different MCP methods
	switch req.Method {
	case "tools/list":
		s.handleMCPListTools(w, req.ID)
	case "tools/call":
		s.handleMCPCallTool(w, req.ID, req.Params)
	case "resources/list":
		s.handleMCPListResources(w, req.ID)
	case "resources/read":
		s.handleMCPReadResource(w, req.ID, req.Params)
	case "prompts/list":
		s.handleMCPListPrompts(w, req.ID)
	case "prompts/get":
		s.handleMCPGetPrompt(w, req.ID, req.Params)
	default:
		s.writeMCPError(w, req.ID, -32601, "Method not found")
	}
}

// handleMCPCapabilities returns MCP server capabilities
func (s *Server) handleMCPCapabilities(w http.ResponseWriter, r *http.Request) {
	capabilities := map[string]interface{}{
		"name":    "SRAKE MCP Server",
		"version": "1.0.0",
		"capabilities": map[string]interface{}{
			"tools":     true,
			"resources": true,
			"prompts":   true,
		},
	}
	s.writeJSON(w, http.StatusOK, capabilities)
}

// MCP method handlers

func (s *Server) handleMCPListTools(w http.ResponseWriter, id interface{}) {
	s.writeMCPResponse(w, id, map[string]interface{}{
		"tools": mcpTools,
	})
}

func (s *Server) handleMCPCallTool(w http.ResponseWriter, id interface{}, params json.RawMessage) {
	var toolCall struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(params, &toolCall); err != nil {
		s.writeMCPError(w, id, -32602, "Invalid params")
		return
	}

	ctx := context.Background()
	var result interface{}
	var err error

	switch toolCall.Name {
	case "search_sra":
		result, err = s.mcpSearchSRA(ctx, toolCall.Arguments)
	case "get_metadata":
		result, err = s.mcpGetMetadata(ctx, toolCall.Arguments)
	case "find_similar":
		result, err = s.mcpFindSimilar(ctx, toolCall.Arguments)
	case "export_results":
		result, err = s.mcpExportResults(ctx, toolCall.Arguments)
	default:
		s.writeMCPError(w, id, -32602, "Unknown tool: "+toolCall.Name)
		return
	}

	if err != nil {
		s.writeMCPError(w, id, -32000, err.Error())
		return
	}

	s.writeMCPResponse(w, id, result)
}

func (s *Server) handleMCPListResources(w http.ResponseWriter, id interface{}) {
	s.writeMCPResponse(w, id, map[string]interface{}{
		"resources": mcpResources,
	})
}

func (s *Server) handleMCPReadResource(w http.ResponseWriter, id interface{}, params json.RawMessage) {
	var resourceReq struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(params, &resourceReq); err != nil {
		s.writeMCPError(w, id, -32602, "Invalid params")
		return
	}

	ctx := context.Background()
	var content interface{}
	var err error

	switch resourceReq.URI {
	case "sra://search/recent":
		// Get recent studies
		req := &service.SearchRequest{
			Limit: 10,
		}
		content, err = s.searchService.Search(ctx, req)

	case "sra://stats":
		// Get statistics
		content, err = s.searchService.GetStats(ctx)

	default:
		// Try to parse as study/experiment/sample/run URI
		if accession := extractAccessionFromURI(resourceReq.URI); accession != "" {
			content, err = s.metadataService.GetMetadata(ctx, &service.MetadataRequest{
				Accession: accession,
				Type:      detectAccessionType(accession),
			})
		} else {
			s.writeMCPError(w, id, -32602, "Unknown resource URI")
			return
		}
	}

	if err != nil {
		s.writeMCPError(w, id, -32000, err.Error())
		return
	}

	s.writeMCPResponse(w, id, map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"uri":      resourceReq.URI,
				"mimeType": "application/json",
				"text":     content,
			},
		},
	})
}

func (s *Server) handleMCPListPrompts(w http.ResponseWriter, id interface{}) {
	s.writeMCPResponse(w, id, map[string]interface{}{
		"prompts": mcpPrompts,
	})
}

func (s *Server) handleMCPGetPrompt(w http.ResponseWriter, id interface{}, params json.RawMessage) {
	var promptReq struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}

	if err := json.Unmarshal(params, &promptReq); err != nil {
		s.writeMCPError(w, id, -32602, "Invalid params")
		return
	}

	// Find the prompt
	for _, prompt := range mcpPrompts {
		if prompt["name"] == promptReq.Name {
			// Expand template with arguments
			template := prompt["template"].(string)
			for key, value := range promptReq.Arguments {
				template = strings.ReplaceAll(template, "{{"+key+"}}", value)
				template = strings.ReplaceAll(template, "{{"+key+"|", "{{") // Remove defaults
			}
			// Replace any remaining defaults
			template = strings.ReplaceAll(template, "{{", "")
			template = strings.ReplaceAll(template, "}}", "")

			s.writeMCPResponse(w, id, map[string]interface{}{
				"messages": []map[string]interface{}{
					{
						"role":    "user",
						"content": template,
					},
				},
			})
			return
		}
	}

	s.writeMCPError(w, id, -32602, "Unknown prompt: "+promptReq.Name)
}

// MCP tool implementations

func (s *Server) mcpSearchSRA(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	req := &service.SearchRequest{
		Query: getString(args, "query", ""),
		Limit: getInt(args, "limit", 20),
		SimilarityThreshold: getFloat32(args, "similarity_threshold", 0.5),
		ShowConfidence: true,
	}

	if organism := getString(args, "organism", ""); organism != "" {
		req.Filters = map[string]string{"organism": organism}
	}

	if strategy := getString(args, "library_strategy", ""); strategy != "" {
		if req.Filters == nil {
			req.Filters = make(map[string]string)
		}
		req.Filters["library_strategy"] = strategy
	}

	return s.searchService.Search(ctx, req)
}

func (s *Server) mcpGetMetadata(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	accession := getString(args, "accession", "")
	if accession == "" {
		return nil, fmt.Errorf("accession required")
	}

	// Detect type from accession prefix
	accType := detectAccessionType(accession)
	if accType == "" {
		return nil, fmt.Errorf("unknown accession type")
	}

	return s.metadataService.GetMetadata(ctx, &service.MetadataRequest{
		Accession: accession,
		Type:      accType,
	})
}

func (s *Server) mcpFindSimilar(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	req := &service.SearchRequest{
		Query:      getString(args, "query", ""),
		Limit:      getInt(args, "limit", 10),
		SearchMode: "vector",
		UseVectors: true,
		ShowConfidence: true,
	}

	return s.searchService.Search(ctx, req)
}

func (s *Server) mcpExportResults(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	req := &service.ExportRequest{
		Query:  getString(args, "query", ""),
		Format: getString(args, "format", "json"),
		Limit:  getInt(args, "limit", 100),
	}

	// Export to memory buffer
	var buf strings.Builder
	if err := s.exportService.Export(ctx, req, &buf); err != nil {
		return nil, err
	}

	return buf.String(), nil
}

// Helper functions

func (s *Server) writeMCPResponse(w http.ResponseWriter, id interface{}, result interface{}) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) writeMCPError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return defaultValue
}

func getFloat32(m map[string]interface{}, key string, defaultValue float32) float32 {
	if v, ok := m[key].(float64); ok {
		return float32(v)
	}
	if v, ok := m[key].(float32); ok {
		return v
	}
	return defaultValue
}

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

func extractAccessionFromURI(uri string) string {
	// Extract accession from URIs like sra://studies/SRP123456
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}