package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nishad/srake/internal/service"
)

// Services holds the service dependencies for the MCP server.
type Services struct {
	Search   *service.SearchService
	Metadata *service.MetadataService
	Export   *service.ExportService
}

// NewServer creates a configured MCP server with all tools, resources, and prompts.
func NewServer(version string, svc *Services) *gomcp.Server {
	server := gomcp.NewServer(
		&gomcp.Implementation{
			Name:    "srake",
			Version: version,
		},
		nil,
	)

	registerTools(server, svc)
	registerResources(server, svc)
	registerPrompts(server)

	return server
}

// Run creates and starts the MCP server on stdio transport.
// All log output is redirected to stderr so stdout remains clean for MCP JSON-RPC.
func Run(ctx context.Context, version string, svc *Services) error {
	// stdout is the MCP transport — all logging must go to stderr
	log.SetOutput(os.Stderr)

	server := NewServer(version, svc)
	return server.Run(ctx, &gomcp.StdioTransport{})
}

// RunHTTP creates and starts the MCP server on Streamable HTTP transport.
func RunHTTP(ctx context.Context, version string, svc *Services, host string, port int) error {
	server := NewServer(version, svc)

	handler := gomcp.NewStreamableHTTPHandler(
		func(r *http.Request) *gomcp.Server { return server },
		nil,
	)

	addr := fmt.Sprintf("%s:%d", host, port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("[MCP] HTTP server listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
