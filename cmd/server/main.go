package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nishad/srake/internal/api"
	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/paths"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	var (
		port        = flag.Int("port", 8080, "Server port")
		host        = flag.String("host", "0.0.0.0", "Server host")
		dbPath      = flag.String("db", "", "Database path")
		configPath  = flag.String("config", "", "Configuration file path")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("srake server %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
		os.Exit(0)
	}

	// Load configuration
	cfg := config.DefaultConfig()
	if *configPath != "" {
		// TODO: Add config file loading if needed
		log.Printf("Config file loading not yet implemented")
	}

	// Set database path
	if *dbPath == "" {
		*dbPath = paths.GetDatabasePath()
	}

	// Initialize database
	log.Printf("Initializing database at %s...", *dbPath)
	db, err := database.Initialize(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create API handler
	handler, err := api.NewHandler(db, cfg)
	if err != nil {
		log.Fatalf("Failed to create API handler: %v", err)
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", *host, *port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
