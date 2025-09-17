package main

import (
	"context"
	"fmt"
	"log"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/processor"
)

func main() {
	// Example: Process SRA metadata from URL
	db, err := database.Open("example.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	proc := processor.NewStreamProcessor(db)

	// Process a small daily update
	ctx := context.Background()
	url := "https://ftp.ncbi.nlm.nih.gov/sra/reports/Metadata/NCBI_SRA_Metadata_YYYY_MM_DD.tar.gz"

	if err := proc.ProcessURL(ctx, url); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Processing complete!")
}
