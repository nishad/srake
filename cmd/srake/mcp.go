package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nishad/srake/internal/database"
	mcpserver "github.com/nishad/srake/internal/mcp"
	"github.com/nishad/srake/internal/paths"
	"github.com/nishad/srake/internal/service"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP (Model Context Protocol) server",
	Long: `Start an MCP server for AI assistants like Claude Desktop and VS Code.

Supports two transport modes:
  stdio  (default)  Communicates over stdin/stdout using JSON-RPC 2.0.
  http              Starts an HTTP server using Streamable HTTP transport.

Tools provided:
  search_sra       Search SRA metadata with filters
  get_metadata     Get metadata for a specific accession
  find_similar     Find similar studies via vector search
  export_results   Export search results as JSON/CSV/TSV

Resources provided:
  sra://stats                  Database statistics
  sra://search/recent          Recently added studies
  sra://studies/{accession}    Full study metadata graph

Prompts provided:
  biomedical_search   Guided search for disease-related data
  sample_selection    Guided sample selection criteria`,
	Example: `  # Run with stdio transport (default, used by MCP clients)
  srake mcp

  # Run with HTTP transport
  srake mcp --transport http --port 8081

  # HTTP transport on a specific host/port
  srake mcp --transport http --host 0.0.0.0 --port 9090

  # Claude Desktop configuration (~/.claude/claude_desktop_config.json):
  {
    "mcpServers": {
      "srake": {
        "command": "srake",
        "args": ["mcp"]
      }
    }
  }

  # With custom database path
  srake mcp --db /path/to/srake.db`,
	RunE: runMCP,
}

var (
	mcpDBPath    string
	mcpIndexPath string
	mcpTransport string
	mcpHost      string
	mcpPort      int
)

func init() {
	mcpCmd.Flags().StringVar(&mcpDBPath, "db", "", "Database path (default: uses SRAKE_DB_PATH)")
	mcpCmd.Flags().StringVar(&mcpIndexPath, "index", "", "Index path (default: uses SRAKE_INDEX_PATH)")
	mcpCmd.Flags().StringVar(&mcpTransport, "transport", "stdio", "Transport mode: stdio or http")
	mcpCmd.Flags().StringVar(&mcpHost, "host", "localhost", "HTTP server host (only used with --transport http)")
	mcpCmd.Flags().IntVar(&mcpPort, "port", 8081, "HTTP server port (only used with --transport http)")
}

func runMCP(cmd *cobra.Command, args []string) error {
	// Resolve database path: flag → env → default
	if mcpDBPath == "" {
		mcpDBPath = os.Getenv("SRAKE_DB_PATH")
		if mcpDBPath == "" {
			mcpDBPath = paths.GetDatabasePath()
		}
	}

	// Resolve index path: flag → env → default
	if mcpIndexPath == "" {
		mcpIndexPath = os.Getenv("SRAKE_INDEX_PATH")
		if mcpIndexPath == "" {
			mcpIndexPath = paths.GetIndexPath()
		}
	}

	// Validate database exists
	if _, err := os.Stat(mcpDBPath); os.IsNotExist(err) {
		return fmt.Errorf("database not found: %s", mcpDBPath)
	}

	// All logging to stderr — stdout is the MCP transport
	log.SetOutput(os.Stderr)

	// Initialize database
	log.Printf("[MCP] Opening database: %s", mcpDBPath)
	db, err := database.Initialize(mcpDBPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Initialize services
	log.Printf("[MCP] Initializing search service with index: %s", mcpIndexPath)
	searchService, err := service.NewSearchService(db, mcpIndexPath)
	if err != nil {
		return fmt.Errorf("failed to initialize search service: %w", err)
	}
	defer searchService.Close()

	metadataService := service.NewMetadataService(db)
	exportService := service.NewExportService(db, searchService)

	svc := &mcpserver.Services{
		Search:   searchService,
		Metadata: metadataService,
		Export:   exportService,
	}

	// Setup context with signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("[MCP] Shutting down...")
		cancel()
	}()

	log.Printf("[MCP] Server starting (version %s, transport %s)", version, mcpTransport)
	switch mcpTransport {
	case "stdio":
		return mcpserver.Run(ctx, version, svc)
	case "http":
		return mcpserver.RunHTTP(ctx, version, svc, mcpHost, mcpPort)
	default:
		return fmt.Errorf("unsupported transport: %s (use stdio or http)", mcpTransport)
	}
}
