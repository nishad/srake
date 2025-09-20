package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nishad/srake/internal/api"
	"github.com/nishad/srake/internal/paths"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the SRAKE API server",
	Long: `Start the SRAKE (SRA Knowledge Engine) API server for programmatic access to SRA metadata.

The server provides:
- RESTful API endpoints for searching and retrieving metadata
- MCP (Model Context Protocol) support for AI assistants
- Export functionality in multiple formats
- CORS support for web applications`,
	Example: `  srake server
  srake server --port 3000
  srake server --enable-cors --enable-mcp`,
	RunE: runServer,
}

var (
	serverPort       int
	serverHost       string
	serverDBPath     string
	serverIndexPath  string
	serverEnableCORS bool
	serverEnableMCP  bool
)

func init() {
	// Server command flags
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "Port to listen on")
	serverCmd.Flags().StringVar(&serverHost, "host", "0.0.0.0", "Host to bind to")
	serverCmd.Flags().StringVar(&serverDBPath, "db", "", "Database path (default: uses SRAKE_DB_PATH)")
	serverCmd.Flags().StringVar(&serverIndexPath, "index", "", "Index path (default: uses SRAKE_INDEX_PATH)")
	serverCmd.Flags().BoolVar(&serverEnableCORS, "enable-cors", true, "Enable CORS for web access")
	serverCmd.Flags().BoolVar(&serverEnableMCP, "enable-mcp", true, "Enable MCP (Model Context Protocol) endpoints")
}

func runServer(cmd *cobra.Command, args []string) error {
	// Get database path
	if serverDBPath == "" {
		serverDBPath = os.Getenv("SRAKE_DB_PATH")
		if serverDBPath == "" {
			serverDBPath = paths.GetDatabasePath()
		}
	}

	// Get index path
	if serverIndexPath == "" {
		serverIndexPath = os.Getenv("SRAKE_INDEX_PATH")
		if serverIndexPath == "" {
			serverIndexPath = paths.GetIndexPath()
		}
	}

	// Validate database exists
	if _, err := os.Stat(serverDBPath); os.IsNotExist(err) {
		return fmt.Errorf("database not found: %s", serverDBPath)
	}

	// Create server configuration
	config := &api.Config{
		Host:         serverHost,
		Port:         serverPort,
		DatabasePath: serverDBPath,
		IndexPath:    serverIndexPath,
		EnableCORS:   serverEnableCORS,
		EnableMCP:    serverEnableMCP,
	}

	// Print initialization header
	printPhase("Initializing srake server")
	printInfo("Database: %s", serverDBPath)
	printInfo("Index: %s", serverIndexPath)

	// Initialize API server with spinner
	spinner := StartSpinner("Initializing server components")
	server, err := api.NewServer(config)
	if err != nil {
		spinner.Stop(false, "failed")
		return fmt.Errorf("failed to initialize server: %w", err)
	}
	spinner.Stop(true, "ready")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		printInfo("Starting API server on %s:%d", serverHost, serverPort)
		printInfo("Database: %s", serverDBPath)
		printInfo("Index: %s", serverIndexPath)

		if serverEnableCORS {
			printInfo("CORS enabled for web access")
		}
		if serverEnableMCP {
			printInfo("MCP endpoints enabled at /mcp")
		}

		printSuccess("\nServer ready at http://%s:%d", serverHost, serverPort)
		printInfo("API documentation at http://%s:%d/", serverHost, serverPort)

		if err := server.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt or server error
	select {
	case <-sigChan:
		printInfo("\nShutting down server...")
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
		return err
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	printSuccess("Server stopped gracefully")
	return nil
}
