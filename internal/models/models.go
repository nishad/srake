package models

import (
	"time"
)

// Study represents an SRA study - simplified for internal use
type Study struct {
	ID                  int       `json:"id" db:"study_ID"`
	Accession           string    `json:"accession" db:"study_accession"`
	Title               string    `json:"title" db:"study_title"`
	Type                string    `json:"type" db:"study_type"`
	Abstract            string    `json:"abstract" db:"study_abstract"`
	SubmissionAccession string    `json:"submission_accession" db:"submission_accession"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// Experiment represents an SRA experiment - simplified for internal use
type Experiment struct {
	ID               int       `json:"id" db:"experiment_ID"`
	Accession        string    `json:"accession" db:"experiment_accession"`
	Title            string    `json:"title" db:"title"`
	Alias            string    `json:"alias" db:"experiment_alias"`
	StudyAccession   string    `json:"study_accession" db:"study_accession"`
	SampleAccession  string    `json:"sample_accession" db:"sample_accession"`
	LibraryStrategy  string    `json:"library_strategy" db:"library_strategy"`
	LibrarySource    string    `json:"library_source" db:"library_source"`
	LibrarySelection string    `json:"library_selection" db:"library_selection"`
	LibraryLayout    string    `json:"library_layout" db:"library_layout"`
	Platform         string    `json:"platform" db:"platform"`
	InstrumentModel  string    `json:"instrument_model" db:"instrument_model"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Run represents an SRA run - simplified for internal use
type Run struct {
	ID                  int       `json:"id" db:"run_ID"`
	Accession           string    `json:"accession" db:"run_accession"`
	ExperimentAccession string    `json:"experiment_accession" db:"experiment_accession"`
	Alias               string    `json:"alias" db:"run_alias"`
	Spots               int64     `json:"spots" db:"spots"`
	Bases               int64     `json:"bases" db:"bases"`
	SpotLength          int       `json:"spot_length" db:"spot_length"`
	PublishedDate       time.Time `json:"published_date" db:"published"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// Sample represents an SRA sample - simplified for internal use
type Sample struct {
	ID             int       `json:"id" db:"sample_ID"`
	Accession      string    `json:"accession" db:"sample_accession"`
	Alias          string    `json:"alias" db:"sample_alias"`
	ScientificName string    `json:"scientific_name" db:"scientific_name"`
	TaxonID        int       `json:"taxon_id" db:"taxon_id"`
	Description    string    `json:"description" db:"description"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// SearchResult represents a search result
type SearchResult struct {
	Type      string  `json:"type"`
	Accession string  `json:"accession"`
	Title     string  `json:"title"`
	Snippet   string  `json:"snippet"`
	Score     float64 `json:"score"`
}

// DownloadProgress represents download progress
type DownloadProgress struct {
	BytesDownloaded int64   `json:"bytes_downloaded"`
	TotalBytes      int64   `json:"total_bytes"`
	Percentage      float64 `json:"percentage"`
	Speed           float64 `json:"speed_mbps"`
	ETA             int     `json:"eta_seconds"`
}

// ProcessingProgress represents processing progress
type ProcessingProgress struct {
	FilesProcessed       int    `json:"files_processed"`
	CurrentFile          string `json:"current_file"`
	ExperimentsProcessed int    `json:"experiments_processed"`
	RunsProcessed        int    `json:"runs_processed"`
	SamplesProcessed     int    `json:"samples_processed"`
	StudiesProcessed     int    `json:"studies_processed"`
	ErrorCount           int    `json:"error_count"`
}

// DatabaseStats represents database statistics
type DatabaseStats struct {
	TotalStudies     int64     `json:"total_studies"`
	TotalExperiments int64     `json:"total_experiments"`
	TotalRuns        int64     `json:"total_runs"`
	TotalSamples     int64     `json:"total_samples"`
	DatabaseSize     int64     `json:"database_size_bytes"`
	LastUpdate       time.Time `json:"last_update"`
}
