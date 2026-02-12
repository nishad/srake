package mcp

import (
	"context"
	"fmt"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPrompts registers all MCP prompt templates.
func registerPrompts(server *gomcp.Server) {
	// biomedical_search — guided search for disease-related data
	server.AddPrompt(
		&gomcp.Prompt{
			Name:        "biomedical_search",
			Description: "Search for biomedical research data related to a disease or condition",
			Arguments: []*gomcp.PromptArgument{
				{
					Name:        "disease",
					Description: "Disease or condition to search for",
					Required:    true,
				},
				{
					Name:        "organism",
					Description: "Organism (default: human)",
					Required:    false,
				},
			},
		},
		func(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
			disease := req.Params.Arguments["disease"]
			if disease == "" {
				return nil, fmt.Errorf("disease argument is required")
			}

			organism := req.Params.Arguments["organism"]
			if organism == "" {
				organism = "human"
			}

			text := fmt.Sprintf(
				"Search for %s studies in %s using RNA-Seq or similar high-throughput sequencing methods. "+
					"Focus on studies with good sample sizes and clear experimental designs. "+
					"Use the search_sra tool with organism filter set to %q and look for library strategies "+
					"like RNA-Seq, scRNA-Seq, or ATAC-Seq. Prioritize studies with multiple replicates.",
				disease, organism, organism,
			)

			return &gomcp.GetPromptResult{
				Description: fmt.Sprintf("Search for %s research data in %s", disease, organism),
				Messages: []*gomcp.PromptMessage{{
					Role:    "user",
					Content: &gomcp.TextContent{Text: text},
				}},
			}, nil
		},
	)

	// sample_selection — guided sample selection criteria
	server.AddPrompt(
		&gomcp.Prompt{
			Name:        "sample_selection",
			Description: "Select samples for analysis based on specific criteria",
			Arguments: []*gomcp.PromptArgument{
				{
					Name:        "criteria",
					Description: "Selection criteria for samples",
					Required:    true,
				},
			},
		},
		func(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
			criteria := req.Params.Arguments["criteria"]
			if criteria == "" {
				return nil, fmt.Errorf("criteria argument is required")
			}

			text := fmt.Sprintf(
				"Find samples matching: %s. "+
					"Focus on data quality, sample size, and experimental design. "+
					"Use the search_sra tool to find relevant studies, then use get_metadata to inspect "+
					"individual accessions. Consider filtering by organism, platform, and library strategy. "+
					"Evaluate the total number of reads/bases to ensure sufficient sequencing depth.",
				criteria,
			)

			return &gomcp.GetPromptResult{
				Description: fmt.Sprintf("Select samples matching: %s", criteria),
				Messages: []*gomcp.PromptMessage{{
					Role:    "user",
					Content: &gomcp.TextContent{Text: text},
				}},
			}, nil
		},
	)
}
