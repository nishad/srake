package main

import (
	"fmt"
	"log"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/search"
)

func main() {
	// Load configuration
	cfg := config.DefaultConfig()

	// Print operational mode
	fmt.Printf("Operational Mode: %s\n", cfg.GetOperationalMode())
	fmt.Printf("Search Enabled: %v\n", cfg.IsSearchEnabled())
	fmt.Printf("Vectors Enabled: %v\n", cfg.IsVectorEnabled())

	// Test database connection
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open SQLite: %v", err)
	}
	db := &database.DB{DB: sqlDB}
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test search manager
	manager, err := search.NewManager(cfg, db)
	if err != nil {
		log.Fatalf("Failed to create search manager: %v", err)
	}
	defer manager.Close()

	// Get search stats
	stats, err := manager.GetStats()
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("\nSearch Stats:\n")
		fmt.Printf("  Backend: %s\n", stats.Backend)
		fmt.Printf("  Document Count: %d\n", stats.DocumentCount)
		fmt.Printf("  Vectors Enabled: %v\n", stats.VectorsEnabled)
		fmt.Printf("  Is Healthy: %v\n", stats.IsHealthy)
	}

	// Test search
	fmt.Println("\nTesting search functionality...")
	results, err := manager.Search("test", search.SearchOptions{
		Limit: 10,
	})
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Search returned %d results\n", results.TotalHits)
		fmt.Printf("Search mode: %s\n", results.Mode)
		fmt.Printf("Time: %dms\n", results.TimeMs)
	}

	fmt.Println("\nSystem is configured and operational!")
}