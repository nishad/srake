//go:build sqlite_fts5

package search

import (
	"path/filepath"
	"testing"

	"github.com/nishad/srake/internal/database"
)

// TestSearchSQLiteFTSFallback verifies that the SQLite FTS5 fallback used by
// searchAll (when the Bleve index is unavailable) returns real results instead
// of the previous placeholder that always returned zero hits.
func TestSearchSQLiteFTSFallback(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fallback.db")
	db, err := database.Initialize(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	// Seed a couple of studies, then build the FTS5 tables the fallback queries.
	for _, s := range []*database.Study{
		{StudyAccession: "SRP000001", StudyTitle: "Lung adenocarcinoma transcriptome", StudyAbstract: "RNA-Seq of human lung tumors"},
		{StudyAccession: "SRP000002", StudyTitle: "Mouse brain development", StudyAbstract: "single cell sequencing of neurons"},
	} {
		if err := db.InsertStudy(s); err != nil {
			t.Fatalf("insert study: %v", err)
		}
	}
	if err := database.NewFTS5Manager(db).CreateFTSTables(); err != nil {
		t.Fatalf("create FTS5 tables: %v", err)
	}

	backend := &TieredSearchBackend{db: db}

	res, err := backend.searchSQLiteFTS("lung tumor", SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("searchSQLiteFTS: %v", err)
	}
	if res.Mode != "fts5" {
		t.Errorf("expected mode fts5, got %q", res.Mode)
	}
	if len(res.Hits) == 0 {
		t.Fatal("fallback returned 0 hits; the FTS5 fallback is not actually searching")
	}

	// The matching study should be present with its type populated.
	found := false
	for _, h := range res.Hits {
		if h.ID == "SRP000001" {
			found = true
			if h.Type != "study" {
				t.Errorf("expected type 'study', got %q", h.Type)
			}
		}
	}
	if !found {
		t.Errorf("expected SRP000001 in fallback results, got %d hits", len(res.Hits))
	}
}
