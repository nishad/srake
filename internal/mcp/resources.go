package mcp

import (
	"context"
	"fmt"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nishad/srake/internal/service"
)

// registerResources registers all MCP resources and resource templates.
func registerResources(server *gomcp.Server, svc *Services) {
	// sra://stats — database statistics
	server.AddResource(
		&gomcp.Resource{
			URI:      "sra://stats",
			Name:     "Database Statistics",
			MIMEType: "application/json",
		},
		func(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
			stats, err := svc.Search.GetStats(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get stats: %w", err)
			}
			return &gomcp.ReadResourceResult{
				Contents: []*gomcp.ResourceContents{{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     toJSON(stats),
				}},
			}, nil
		},
	)

	// sra://search/recent — recently added studies
	server.AddResource(
		&gomcp.Resource{
			URI:      "sra://search/recent",
			Name:     "Recent SRA Studies",
			MIMEType: "application/json",
		},
		func(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
			resp, err := svc.Search.Search(ctx, &service.SearchRequest{Limit: 10})
			if err != nil {
				return nil, fmt.Errorf("failed to get recent studies: %w", err)
			}
			return &gomcp.ReadResourceResult{
				Contents: []*gomcp.ResourceContents{{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     toJSON(resp),
				}},
			}, nil
		},
	)

	// sra://studies/{accession} — full study metadata graph
	server.AddResourceTemplate(
		&gomcp.ResourceTemplate{
			URITemplate: "sra://studies/{accession}",
			Name:        "Study Metadata",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
			// Extract accession from URI: sra://studies/SRP123456
			uri := req.Params.URI
			accession := uri[strings.LastIndex(uri, "/")+1:]
			if accession == "" {
				return nil, fmt.Errorf("accession required in URI")
			}

			metadata, err := svc.Metadata.GetStudyMetadata(ctx, accession)
			if err != nil {
				return nil, fmt.Errorf("failed to get study metadata: %w", err)
			}

			return &gomcp.ReadResourceResult{
				Contents: []*gomcp.ResourceContents{{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     toJSON(metadata),
				}},
			}, nil
		},
	)
}
