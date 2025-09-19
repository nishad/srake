package vectors

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// VectorStore manages vector embeddings using sqlite-vec
type VectorStore struct {
	db   *sql.DB
	path string
}

// NewVectorStore creates a new vector store
func NewVectorStore(dataDir string) (*VectorStore, error) {
	dbPath := filepath.Join(dataDir, "vectors.db")

	// Open database with sqlite-vec support
	// Note: sqlite-vec must be loaded as an extension
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open vector database: %w", err)
	}

	// Try to load sqlite-vec extension
	// This requires the vec0.so/dll file to be available
	_, err = db.Exec("SELECT load_extension('vec0')")
	if err != nil {
		// If sqlite-vec is not available, continue with basic functionality
		fmt.Printf("Warning: sqlite-vec extension not loaded: %v\n", err)
		fmt.Println("Vector similarity search will be limited. Install sqlite-vec for full functionality.")
	}

	// Performance optimizations
	pragmas := []string{
		"PRAGMA cache_size = 10000",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	vs := &VectorStore{
		db:   db,
		path: dbPath,
	}

	// Create schema
	if err := vs.createSchema(); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return vs, nil
}

func (vs *VectorStore) createSchema() error {
	schema := `
	-- Project vectors (study-level embeddings)
	CREATE TABLE IF NOT EXISTS project_vectors (
		project_id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		abstract TEXT,
		organism TEXT,
		study_type TEXT,
		sample_count INTEGER,
		experiment_count INTEGER,
		embedding BLOB,           -- Full precision (384 dims * 4 bytes)
		embedding_int8 BLOB       -- Quantized for speed (384 dims * 1 byte)
	);

	-- Sample vectors
	CREATE TABLE IF NOT EXISTS sample_vectors (
		sample_id TEXT PRIMARY KEY,
		project_id TEXT,
		description TEXT,
		organism TEXT,
		tissue TEXT,
		cell_type TEXT,
		embedding BLOB,
		embedding_int8 BLOB,
		FOREIGN KEY(project_id) REFERENCES project_vectors(project_id)
	);

	-- Embedding metadata for tracking
	CREATE TABLE IF NOT EXISTS embedding_metadata (
		entity_id TEXT PRIMARY KEY,
		entity_type TEXT,  -- 'project' or 'sample'
		model_id TEXT,
		model_variant TEXT,
		embedding_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		text_hash TEXT
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_proj_organism ON project_vectors(organism);
	CREATE INDEX IF NOT EXISTS idx_proj_type ON project_vectors(study_type);
	CREATE INDEX IF NOT EXISTS idx_sample_organism ON sample_vectors(organism);
	CREATE INDEX IF NOT EXISTS idx_sample_tissue ON sample_vectors(tissue);
	CREATE INDEX IF NOT EXISTS idx_sample_project ON sample_vectors(project_id);
	`

	_, err := vs.db.Exec(schema)
	return err
}

// ProjectVector represents a study/project with its embedding
type ProjectVector struct {
	ProjectID       string    `json:"project_id"`
	Title           string    `json:"title"`
	Abstract        string    `json:"abstract"`
	Organism        string    `json:"organism"`
	StudyType       string    `json:"study_type"`
	SampleCount     int       `json:"sample_count"`
	ExperimentCount int       `json:"experiment_count"`
	Embedding       []float32 `json:"embedding,omitempty"`
}

// SampleVector represents a sample with its embedding
type SampleVector struct {
	SampleID    string    `json:"sample_id"`
	ProjectID   string    `json:"project_id"`
	Description string    `json:"description"`
	Organism    string    `json:"organism"`
	Tissue      string    `json:"tissue"`
	CellType    string    `json:"cell_type"`
	Embedding   []float32 `json:"embedding,omitempty"`
}

// SimilarityResult represents a similarity search result
type SimilarityResult struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Organism string  `json:"organism"`
	Distance float32 `json:"distance"`
	Score    float32 `json:"score"` // 1 - distance for cosine similarity
}

// InsertProjectVector adds or updates a project vector
func (vs *VectorStore) InsertProjectVector(pv *ProjectVector) error {
	if len(pv.Embedding) != 384 {
		return fmt.Errorf("embedding must have 384 dimensions, got %d", len(pv.Embedding))
	}

	embeddingBlob := floatsToBytes(pv.Embedding)
	embeddingInt8 := quantizeToInt8(pv.Embedding)

	query := `
		INSERT OR REPLACE INTO project_vectors (
			project_id, title, abstract, organism, study_type,
			sample_count, experiment_count, embedding, embedding_int8
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := vs.db.Exec(query,
		pv.ProjectID, pv.Title, pv.Abstract, pv.Organism, pv.StudyType,
		pv.SampleCount, pv.ExperimentCount, embeddingBlob, embeddingInt8)

	if err != nil {
		return fmt.Errorf("failed to insert project vector: %w", err)
	}

	// Update metadata
	metaQuery := `
		INSERT OR REPLACE INTO embedding_metadata (
			entity_id, entity_type, model_id, model_variant, text_hash
		) VALUES (?, 'project', ?, ?, ?)
	`
	_, err = vs.db.Exec(metaQuery, pv.ProjectID, "SapBERT", "quantized", hashText(pv.Title+pv.Abstract))

	return err
}

// InsertSampleVector adds or updates a sample vector
func (vs *VectorStore) InsertSampleVector(sv *SampleVector) error {
	if len(sv.Embedding) != 384 {
		return fmt.Errorf("embedding must have 384 dimensions, got %d", len(sv.Embedding))
	}

	embeddingBlob := floatsToBytes(sv.Embedding)
	embeddingInt8 := quantizeToInt8(sv.Embedding)

	query := `
		INSERT OR REPLACE INTO sample_vectors (
			sample_id, project_id, description, organism,
			tissue, cell_type, embedding, embedding_int8
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := vs.db.Exec(query,
		sv.SampleID, sv.ProjectID, sv.Description, sv.Organism,
		sv.Tissue, sv.CellType, embeddingBlob, embeddingInt8)

	if err != nil {
		return fmt.Errorf("failed to insert sample vector: %w", err)
	}

	// Update metadata
	metaQuery := `
		INSERT OR REPLACE INTO embedding_metadata (
			entity_id, entity_type, model_id, model_variant, text_hash
		) VALUES (?, 'sample', ?, ?, ?)
	`
	_, err = vs.db.Exec(metaQuery, sv.SampleID, "SapBERT", "quantized", hashText(sv.Description))

	return err
}

// FindSimilarProjects finds similar projects using vector similarity
func (vs *VectorStore) FindSimilarProjects(projectID string, limit int, filters map[string]interface{}) ([]SimilarityResult, error) {
	// Get the target project's embedding
	var embeddingBlob []byte
	query := "SELECT embedding FROM project_vectors WHERE project_id = ?"
	err := vs.db.QueryRow(query, projectID).Scan(&embeddingBlob)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	targetEmbedding := bytesToFloats(embeddingBlob)

	// Search for similar projects
	return vs.searchSimilarProjects(targetEmbedding, projectID, limit, filters)
}

// SearchSimilar searches for similar items using a query embedding
func (vs *VectorStore) SearchSimilar(embedding []float32, limit int) ([]SimilarityResult, error) {
	return vs.searchSimilarProjects(embedding, "", limit, nil)
}

// searchSimilarProjects internal method for similarity search
func (vs *VectorStore) searchSimilarProjects(embedding []float32, excludeID string, limit int, filters map[string]interface{}) ([]SimilarityResult, error) {
	// Build query with filters
	query := `
		SELECT project_id, title, organism, embedding
		FROM project_vectors
		WHERE 1=1
	`
	args := []interface{}{}

	if excludeID != "" {
		query += " AND project_id != ?"
		args = append(args, excludeID)
	}

	// Add filters
	if organism, ok := filters["organism"].(string); ok && organism != "" {
		query += " AND organism = ?"
		args = append(args, organism)
	}

	if studyType, ok := filters["study_type"].(string); ok && studyType != "" {
		query += " AND study_type = ?"
		args = append(args, studyType)
	}

	rows, err := vs.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SimilarityResult
	for rows.Next() {
		var projectID, title, organism string
		var embeddingBlob []byte

		err := rows.Scan(&projectID, &title, &organism, &embeddingBlob)
		if err != nil {
			continue
		}

		candidateEmbedding := bytesToFloats(embeddingBlob)
		distance := cosineDistance(embedding, candidateEmbedding)

		results = append(results, SimilarityResult{
			ID:       projectID,
			Title:    title,
			Organism: organism,
			Distance: distance,
			Score:    1 - distance, // Convert distance to similarity score
		})
	}

	// Sort by distance (ascending) and limit
	sortByDistance(results)
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// FindSimilarSamples finds similar samples
func (vs *VectorStore) FindSimilarSamples(sampleID string, limit int) ([]SimilarityResult, error) {
	// Get the target sample's embedding
	var embeddingBlob []byte
	query := "SELECT embedding FROM sample_vectors WHERE sample_id = ?"
	err := vs.db.QueryRow(query, sampleID).Scan(&embeddingBlob)
	if err != nil {
		return nil, fmt.Errorf("sample not found: %w", err)
	}

	targetEmbedding := bytesToFloats(embeddingBlob)

	// Search for similar samples
	searchQuery := `
		SELECT sample_id, description, organism, embedding
		FROM sample_vectors
		WHERE sample_id != ?
	`

	rows, err := vs.db.Query(searchQuery, sampleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SimilarityResult
	for rows.Next() {
		var sampleID, description, organism string
		var embeddingBlob []byte

		err := rows.Scan(&sampleID, &description, &organism, &embeddingBlob)
		if err != nil {
			continue
		}

		candidateEmbedding := bytesToFloats(embeddingBlob)
		distance := cosineDistance(targetEmbedding, candidateEmbedding)

		results = append(results, SimilarityResult{
			ID:       sampleID,
			Title:    description,
			Organism: organism,
			Distance: distance,
			Score:    1 - distance,
		})
	}

	// Sort by distance and limit
	sortByDistance(results)
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Utility functions

// floatsToBytes converts float32 slice to bytes
func floatsToBytes(floats []float32) []byte {
	bytes := make([]byte, len(floats)*4)
	for i, f := range floats {
		bits := math.Float32bits(f)
		binary.LittleEndian.PutUint32(bytes[i*4:], bits)
	}
	return bytes
}

// bytesToFloats converts bytes to float32 slice
func bytesToFloats(bytes []byte) []float32 {
	floats := make([]float32, len(bytes)/4)
	for i := 0; i < len(floats); i++ {
		bits := binary.LittleEndian.Uint32(bytes[i*4:])
		floats[i] = math.Float32frombits(bits)
	}
	return floats
}

// quantizeToInt8 converts float32 embeddings to int8 for faster search
func quantizeToInt8(floats []float32) []byte {
	// Find min and max for scaling
	var min, max float32
	for i, f := range floats {
		if i == 0 || f < min {
			min = f
		}
		if i == 0 || f > max {
			max = f
		}
	}

	// Scale to int8 range
	scale := float32(255) / (max - min)
	bytes := make([]byte, len(floats))
	for i, f := range floats {
		scaled := (f - min) * scale
		bytes[i] = byte(scaled)
	}

	return bytes
}

// cosineDistance calculates cosine distance between two vectors
func cosineDistance(a, b []float32) float32 {
	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 1 // Maximum distance
	}

	similarity := dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
	return 1 - similarity // Convert to distance
}

// sortByDistance sorts results by distance (ascending)
func sortByDistance(results []SimilarityResult) {
	// Simple insertion sort for small datasets
	for i := 1; i < len(results); i++ {
		key := results[i]
		j := i - 1
		for j >= 0 && results[j].Distance > key.Distance {
			results[j+1] = results[j]
			j--
		}
		results[j+1] = key
	}
}

// hashText creates a simple hash of text for tracking
func hashText(text string) string {
	// Simple hash for tracking text changes
	hash := uint32(0)
	for _, c := range text {
		hash = hash*31 + uint32(c)
	}
	return fmt.Sprintf("%x", hash)
}

// GetStats returns statistics about the vector store
func (vs *VectorStore) GetStats() (map[string]int, error) {
	stats := make(map[string]int)

	var count int
	if err := vs.db.QueryRow("SELECT COUNT(*) FROM project_vectors").Scan(&count); err != nil {
		return nil, fmt.Errorf("failed to count project_vectors: %w", err)
	}
	stats["project_vectors"] = count

	if err := vs.db.QueryRow("SELECT COUNT(*) FROM sample_vectors").Scan(&count); err != nil {
		return nil, fmt.Errorf("failed to count sample_vectors: %w", err)
	}
	stats["sample_vectors"] = count

	if err := vs.db.QueryRow("SELECT COUNT(*) FROM embedding_metadata").Scan(&count); err != nil {
		return nil, fmt.Errorf("failed to count embedding_metadata: %w", err)
	}
	stats["embeddings"] = count

	return stats, nil
}

// Close closes the vector store database
func (vs *VectorStore) Close() error {
	return vs.db.Close()
}
