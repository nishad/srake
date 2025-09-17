package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/nishad/srake/internal/database"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server",
	Long: `Start the srake API server for programmatic access to SRA metadata.

The server provides RESTful endpoints for searching and retrieving
SRA metadata from the local database.`,
	Example: `  srake server
  srake server --port 3000
  srake server --dev --log-level debug`,
	RunE: runServer,
}

var (
	serverPort     int
	serverHost     string
	serverDBPath   string
	serverLogLevel string
	serverDev      bool
)

func runServer(cmd *cobra.Command, args []string) error {
	// Initialize database
	db, err := database.Initialize(serverDBPath)
	if err != nil {
		printError("Failed to initialize database: %v", err)
		return err
	}
	defer db.Close()

	// Setup interrupt handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check database connection
		dbStatus := "healthy"
		if err := db.Ping(); err != nil {
			dbStatus = "unhealthy"
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status":   "healthy",
			"version":  version,
			"database": dbStatus,
		})
	})

	// Search endpoint
	router.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		query := r.URL.Query().Get("q")
		organism := r.URL.Query().Get("organism")
		strategy := r.URL.Query().Get("library_strategy")
		limitStr := r.URL.Query().Get("limit")

		limit := 100
		if limitStr != "" {
			fmt.Sscanf(limitStr, "%d", &limit)
		}

		var results interface{}
		var err error

		if query != "" {
			// Full-text search
			results, err = db.FullTextSearch(query)
		} else if organism != "" {
			// Search by organism
			results, err = db.SearchByOrganism(organism, limit)
		} else if strategy != "" {
			// Search by library strategy
			results, err = db.SearchByLibraryStrategy(strategy, limit)
		} else {
			http.Error(w, "No search parameters provided", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"query":   query + organism + strategy,
			"results": results,
			"limit":   limit,
		}
		json.NewEncoder(w).Encode(response)
	})

	// Get specific accession endpoints
	router.HandleFunc("/api/experiment/{accession}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		exp, err := db.GetExperiment(vars["accession"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(exp)
	})

	router.HandleFunc("/api/run/{accession}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		run, err := db.GetRun(vars["accession"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(run)
	})

	router.HandleFunc("/api/sample/{accession}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		sample, err := db.GetSample(vars["accession"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(sample)
	})

	router.HandleFunc("/api/study/{accession}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		study, err := db.GetStudy(vars["accession"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(study)
	})

	// Statistics endpoint
	router.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats, err := db.GetStats()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(stats)
	})

	// Setup server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", serverHost, serverPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start server in goroutine
	go func() {
		printInfo("Starting server on %s:%d", serverHost, serverPort)
		if serverDev {
			printInfo("Development mode enabled")
		}
		printSuccess("Server ready at http://%s:%d", serverHost, serverPort)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			printError("Server failed: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	printInfo("\nShutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %v", err)
	}

	printSuccess("Server stopped gracefully")
	return nil
}