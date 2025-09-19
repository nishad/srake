package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/paths"
	"github.com/nishad/srake/internal/service"
)

// Server represents the HTTP API server
type Server struct {
	router          *mux.Router
	server          *http.Server
	searchService   *service.SearchService
	metadataService *service.MetadataService
	exportService   *service.ExportService
	db              *database.DB
}

// Config holds server configuration
type Config struct {
	Host         string
	Port         int
	DatabasePath string
	IndexPath    string
	EnableCORS   bool
	EnableMCP    bool // Enable Model Context Protocol endpoints
}

// NewServer creates a new API server instance
func NewServer(cfg *Config) (*Server, error) {
	// Open database
	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize services
	indexPath := cfg.IndexPath
	if indexPath == "" {
		indexPath = paths.GetIndexPath()
	}

	searchService, err := service.NewSearchService(db, indexPath)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize search service: %w", err)
	}

	metadataService := service.NewMetadataService(db)
	exportService := service.NewExportService(db, searchService)

	// Create server
	s := &Server{
		router:          mux.NewRouter(),
		searchService:   searchService,
		metadataService: metadataService,
		exportService:   exportService,
		db:              db,
	}

	// Setup routes
	s.setupRoutes(cfg.EnableMCP)

	// Setup middleware
	if cfg.EnableCORS {
		s.router.Use(corsMiddleware)
	}
	s.router.Use(loggingMiddleware)
	s.router.Use(jsonMiddleware)

	// Create HTTP server
	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes(enableMCP bool) {
	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Search endpoints
	api.HandleFunc("/search", s.handleSearch).Methods("GET", "POST")
	api.HandleFunc("/search/advanced", s.handleAdvancedSearch).Methods("POST")

	// Metadata endpoints
	api.HandleFunc("/studies/{accession}", s.handleGetStudy).Methods("GET")
	api.HandleFunc("/experiments/{accession}", s.handleGetExperiment).Methods("GET")
	api.HandleFunc("/samples/{accession}", s.handleGetSample).Methods("GET")
	api.HandleFunc("/runs/{accession}", s.handleGetRun).Methods("GET")

	// Batch metadata endpoints
	api.HandleFunc("/studies", s.handleListStudies).Methods("GET")
	api.HandleFunc("/studies/{accession}/metadata", s.handleGetStudyMetadata).Methods("GET")
	api.HandleFunc("/studies/{accession}/experiments", s.handleGetStudyExperiments).Methods("GET")
	api.HandleFunc("/studies/{accession}/samples", s.handleGetStudySamples).Methods("GET")
	api.HandleFunc("/studies/{accession}/runs", s.handleGetStudyRuns).Methods("GET")

	// Statistics endpoints
	api.HandleFunc("/stats", s.handleGetStats).Methods("GET")
	api.HandleFunc("/stats/organisms", s.handleGetOrganismStats).Methods("GET")
	api.HandleFunc("/stats/platforms", s.handleGetPlatformStats).Methods("GET")
	api.HandleFunc("/stats/strategies", s.handleGetStrategyStats).Methods("GET")

	// Export endpoints
	api.HandleFunc("/export", s.handleExport).Methods("POST")

	// Health check
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

	// MCP endpoints (if enabled)
	if enableMCP {
		s.router.HandleFunc("/mcp", s.handleMCP).Methods("POST")
		s.router.HandleFunc("/mcp/capabilities", s.handleMCPCapabilities).Methods("GET")
	}

	// Root endpoint
	s.router.HandleFunc("/", s.handleRoot).Methods("GET")
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting API server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API server...")

	// Shutdown HTTP server
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	// Close services
	if s.searchService != nil {
		s.searchService.Close()
	}
	if s.exportService != nil {
		s.exportService.Close()
	}
	if s.metadataService != nil {
		s.metadataService.Close()
	}

	// Close database
	if s.db != nil {
		return s.db.Close()
	}

	return nil
}

// Middleware functions

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// Helper functions

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]interface{}{
		"error":  true,
		"message": message,
		"status": status,
	})
}

// handleRoot returns API information
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"name":    "SRAKE API",
		"version": "1.0.0",
		"description": "SRA Knowledgebase Engine API",
		"endpoints": map[string]string{
			"search":   "/api/v1/search",
			"studies":  "/api/v1/studies",
			"stats":    "/api/v1/stats",
			"health":   "/api/v1/health",
		},
	}
	s.writeJSON(w, http.StatusOK, info)
}

// handleHealth returns health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	health := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Check search service
	if err := s.searchService.Health(ctx); err != nil {
		health["status"] = "unhealthy"
		health["search_service"] = err.Error()
	} else {
		health["search_service"] = "healthy"
	}

	// Check metadata service
	if err := s.metadataService.Health(ctx); err != nil {
		health["status"] = "unhealthy"
		health["metadata_service"] = err.Error()
	} else {
		health["metadata_service"] = "healthy"
	}

	status := http.StatusOK
	if health["status"] != "healthy" {
		status = http.StatusServiceUnavailable
	}

	s.writeJSON(w, status, health)
}