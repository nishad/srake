package testutil

import (
	"context"
	"sync"
)

// MockSearchBackend is a mock implementation of the search backend.
type MockSearchBackend struct {
	mu       sync.Mutex
	indexed  []interface{}
	deleted  []string
	searches []string

	// Configurable return values
	SearchResult interface{}
	SearchErr    error
	IndexErr     error
	DeleteErr    error
}

// Index records an index operation.
func (m *MockSearchBackend) Index(doc interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.IndexErr != nil {
		return m.IndexErr
	}
	m.indexed = append(m.indexed, doc)
	return nil
}

// IndexBatch records a batch index operation.
func (m *MockSearchBackend) IndexBatch(docs []interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.IndexErr != nil {
		return m.IndexErr
	}
	m.indexed = append(m.indexed, docs...)
	return nil
}

// Delete records a delete operation.
func (m *MockSearchBackend) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.deleted = append(m.deleted, id)
	return nil
}

// Search records a search operation and returns configured results.
func (m *MockSearchBackend) Search(query string) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searches = append(m.searches, query)
	return m.SearchResult, m.SearchErr
}

// IndexedDocs returns all indexed documents.
func (m *MockSearchBackend) IndexedDocs() []interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]interface{}, len(m.indexed))
	copy(result, m.indexed)
	return result
}

// DeletedIDs returns all deleted IDs.
func (m *MockSearchBackend) DeletedIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.deleted))
	copy(result, m.deleted)
	return result
}

// SearchQueries returns all search queries.
func (m *MockSearchBackend) SearchQueries() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.searches))
	copy(result, m.searches)
	return result
}

// Reset clears all recorded operations.
func (m *MockSearchBackend) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.indexed = nil
	m.deleted = nil
	m.searches = nil
}

// MockEmbedder is a mock implementation of the embedder interface.
type MockEmbedder struct {
	mu         sync.Mutex
	embeddings map[string][]float32

	// Configurable return values
	DefaultEmbedding []float32
	EmbedErr         error
}

// NewMockEmbedder creates a new mock embedder.
func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{
		embeddings:       make(map[string][]float32),
		DefaultEmbedding: make([]float32, 384), // Default embedding dimension
	}
}

// Embed returns a mock embedding for the given text.
func (m *MockEmbedder) Embed(text string) ([]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.EmbedErr != nil {
		return nil, m.EmbedErr
	}
	if emb, ok := m.embeddings[text]; ok {
		return emb, nil
	}
	return m.DefaultEmbedding, nil
}

// EmbedBatch returns mock embeddings for multiple texts.
func (m *MockEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.EmbedErr != nil {
		return nil, m.EmbedErr
	}
	result := make([][]float32, len(texts))
	for i, text := range texts {
		if emb, ok := m.embeddings[text]; ok {
			result[i] = emb
		} else {
			result[i] = m.DefaultEmbedding
		}
	}
	return result, nil
}

// SetEmbedding sets a specific embedding for a text.
func (m *MockEmbedder) SetEmbedding(text string, embedding []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embeddings[text] = embedding
}

// MockHTTPClient is a mock HTTP client for testing.
type MockHTTPClient struct {
	mu        sync.Mutex
	responses map[string][]byte
	errors    map[string]error
	requests  []string

	// Default response
	DefaultResponse []byte
	DefaultErr      error
}

// NewMockHTTPClient creates a new mock HTTP client.
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses: make(map[string][]byte),
		errors:    make(map[string]error),
	}
}

// Get records a GET request and returns configured response.
func (m *MockHTTPClient) Get(url string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, url)

	if err, ok := m.errors[url]; ok {
		return nil, err
	}
	if resp, ok := m.responses[url]; ok {
		return resp, nil
	}
	if m.DefaultErr != nil {
		return nil, m.DefaultErr
	}
	return m.DefaultResponse, nil
}

// SetResponse sets the response for a specific URL.
func (m *MockHTTPClient) SetResponse(url string, response []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[url] = response
}

// SetError sets an error for a specific URL.
func (m *MockHTTPClient) SetError(url string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[url] = err
}

// Requests returns all recorded requests.
func (m *MockHTTPClient) Requests() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.requests))
	copy(result, m.requests)
	return result
}

// MockProgressReporter is a mock progress reporter for testing.
type MockProgressReporter struct {
	mu       sync.Mutex
	updates  []int64
	messages []string
}

// Update records a progress update.
func (m *MockProgressReporter) Update(progress int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updates = append(m.updates, progress)
}

// SetMessage records a message update.
func (m *MockProgressReporter) SetMessage(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

// Updates returns all recorded progress updates.
func (m *MockProgressReporter) Updates() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]int64, len(m.updates))
	copy(result, m.updates)
	return result
}

// Messages returns all recorded messages.
func (m *MockProgressReporter) Messages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.messages))
	copy(result, m.messages)
	return result
}

// MockContextWithCancel returns a context with cancel and the cancel function.
func MockContextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}
