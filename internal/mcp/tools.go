package mcp

import (
	"context"
	"fmt"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nishad/srake/internal/service"
)

// Tool argument structs — field tags drive automatic JSON schema generation.

// SearchSRAArgs are the arguments for the search_sra tool.
type SearchSRAArgs struct {
	Query               string  `json:"query" jsonschema:"search query"`
	Limit               int     `json:"limit,omitempty" jsonschema:"maximum results, default 20"`
	Organism            string  `json:"organism,omitempty" jsonschema:"filter by organism"`
	LibraryStrategy     string  `json:"library_strategy,omitempty" jsonschema:"filter by library strategy, e.g. RNA-Seq"`
	SimilarityThreshold float32 `json:"similarity_threshold,omitempty" jsonschema:"minimum similarity score 0-1, default 0.5"`
}

// GetMetadataArgs are the arguments for the get_metadata tool.
type GetMetadataArgs struct {
	Accession string `json:"accession" jsonschema:"SRA accession, e.g. SRP123456"`
}

// FindSimilarArgs are the arguments for the find_similar tool.
type FindSimilarArgs struct {
	Query string `json:"query" jsonschema:"text to find similar studies for"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum results, default 10"`
}

// ExportResultsArgs are the arguments for the export_results tool.
type ExportResultsArgs struct {
	Query  string `json:"query" jsonschema:"search query"`
	Format string `json:"format,omitempty" jsonschema:"export format: json, csv, or tsv (default json)"`
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum results, default 100"`
}

// registerTools registers all MCP tools on the server.
func registerTools(server *gomcp.Server, svc *Services) {
	// search_sra
	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "search_sra",
		Description: "Search NCBI SRA metadata with quality control. Returns studies matching the query with optional organism and library strategy filters.",
	}, func(ctx context.Context, req *gomcp.CallToolRequest, args SearchSRAArgs) (*gomcp.CallToolResult, any, error) {
		if args.Query == "" {
			return errResult("query is required"), nil, nil
		}

		limit := args.Limit
		if limit <= 0 {
			limit = 20
		}
		threshold := args.SimilarityThreshold
		if threshold <= 0 {
			threshold = 0.5
		}

		searchReq := &service.SearchRequest{
			Query:               args.Query,
			Limit:               limit,
			SimilarityThreshold: threshold,
			ShowConfidence:      true,
		}

		if args.Organism != "" || args.LibraryStrategy != "" {
			searchReq.Filters = make(map[string]string)
			if args.Organism != "" {
				searchReq.Filters["organism"] = args.Organism
			}
			if args.LibraryStrategy != "" {
				searchReq.Filters["library_strategy"] = args.LibraryStrategy
			}
		}

		resp, err := svc.Search.Search(ctx, searchReq)
		if err != nil {
			return errResult(fmt.Sprintf("search failed: %v", err)), nil, nil
		}

		return &gomcp.CallToolResult{
			Content: []gomcp.Content{&gomcp.TextContent{Text: toJSON(resp)}},
		}, nil, nil
	})

	// get_metadata
	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "get_metadata",
		Description: "Get detailed metadata for a specific SRA accession. Supports study (SRP/ERP/DRP), experiment (SRX/ERX/DRX), sample (SRS/ERS/DRS), and run (SRR/ERR/DRR) accessions.",
	}, func(ctx context.Context, req *gomcp.CallToolRequest, args GetMetadataArgs) (*gomcp.CallToolResult, any, error) {
		if args.Accession == "" {
			return errResult("accession is required"), nil, nil
		}

		accType := detectAccessionType(args.Accession)
		if accType == "" {
			return errResult("unknown accession type: " + args.Accession), nil, nil
		}

		resp, err := svc.Metadata.GetMetadata(ctx, &service.MetadataRequest{
			Accession: args.Accession,
			Type:      accType,
		})
		if err != nil {
			return errResult(fmt.Sprintf("metadata lookup failed: %v", err)), nil, nil
		}

		return &gomcp.CallToolResult{
			Content: []gomcp.Content{&gomcp.TextContent{Text: toJSON(resp)}},
		}, nil, nil
	})

	// find_similar
	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "find_similar",
		Description: "Find similar studies using vector/semantic search. Use this when looking for conceptually related studies rather than exact keyword matches.",
	}, func(ctx context.Context, req *gomcp.CallToolRequest, args FindSimilarArgs) (*gomcp.CallToolResult, any, error) {
		if args.Query == "" {
			return errResult("query is required"), nil, nil
		}

		limit := args.Limit
		if limit <= 0 {
			limit = 10
		}

		searchReq := &service.SearchRequest{
			Query:          args.Query,
			Limit:          limit,
			SearchMode:     "vector",
			UseVectors:     true,
			ShowConfidence: true,
		}

		resp, err := svc.Search.Search(ctx, searchReq)
		if err != nil {
			return errResult(fmt.Sprintf("similarity search failed: %v", err)), nil, nil
		}

		return &gomcp.CallToolResult{
			Content: []gomcp.Content{&gomcp.TextContent{Text: toJSON(resp)}},
		}, nil, nil
	})

	// export_results
	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "export_results",
		Description: "Export search results in JSON, CSV, or TSV format. Returns the formatted content as text.",
	}, func(ctx context.Context, req *gomcp.CallToolRequest, args ExportResultsArgs) (*gomcp.CallToolResult, any, error) {
		if args.Query == "" {
			return errResult("query is required"), nil, nil
		}

		format := args.Format
		if format == "" {
			format = "json"
		}
		limit := args.Limit
		if limit <= 0 {
			limit = 100
		}

		exportReq := &service.ExportRequest{
			Query:  args.Query,
			Format: format,
			Limit:  limit,
		}

		var buf strings.Builder
		if err := svc.Export.Export(ctx, exportReq, &buf); err != nil {
			return errResult(fmt.Sprintf("export failed: %v", err)), nil, nil
		}

		return &gomcp.CallToolResult{
			Content: []gomcp.Content{&gomcp.TextContent{Text: buf.String()}},
		}, nil, nil
	})
}
