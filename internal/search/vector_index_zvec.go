//go:build zvec
// +build zvec

package search

import (
	"fmt"
	"sync"

	zvec "github.com/zvec-ai/zvec-go"
)

// VectorIndexSupported reports whether this build includes a nearest-neighbor
// vector index. True here (built with the "zvec" tag).
const VectorIndexSupported = true

const zvecVectorField = "embedding"

// zvecOnce guards process-wide zvec library initialization.
var zvecOnce sync.Once
var zvecInitErr error

func zvecInit() error {
	zvecOnce.Do(func() {
		zvecInitErr = zvec.Initialize(nil)
	})
	return zvecInitErr
}

// zvecIndex is a VectorIndex backed by a zvec HNSW/cosine collection.
type zvecIndex struct {
	mu  sync.Mutex
	col *zvec.Collection
}

// newVectorIndex opens (or creates) a zvec collection at path holding dim-wide
// FP32 vectors indexed with HNSW + cosine similarity.
func newVectorIndex(path string, dim int) (VectorIndex, error) {
	if err := zvecInit(); err != nil {
		return nil, fmt.Errorf("zvec initialize: %w", err)
	}

	schema := zvec.NewCollectionSchema("srake_studies")
	field := zvec.NewFieldSchema(zvecVectorField, zvec.DataTypeVectorFP32, false, uint32(dim))
	params, err := zvec.NewHNSWIndexParams(zvec.MetricTypeCosine, 16, 200)
	if err != nil {
		return nil, fmt.Errorf("zvec hnsw params: %w", err)
	}
	field.SetIndexParams(params)
	schema.AddField(field)

	col, err := zvec.CreateAndOpen(path, schema, nil)
	if err != nil {
		return nil, fmt.Errorf("zvec open collection at %s: %w", path, err)
	}
	return &zvecIndex{col: col}, nil
}

func (z *zvecIndex) Add(pk string, vector []float32) error {
	z.mu.Lock()
	defer z.mu.Unlock()

	doc := zvec.NewDoc()
	defer doc.Destroy()
	doc.SetPK(pk)
	doc.AddVectorFP32Field(zvecVectorField, vector)
	if _, err := z.col.Insert([]*zvec.Doc{doc}); err != nil {
		return fmt.Errorf("zvec insert %s: %w", pk, err)
	}
	return nil
}

func (z *zvecIndex) Search(vector []float32, topK int) ([]VectorHit, error) {
	z.mu.Lock()
	defer z.mu.Unlock()

	q := zvec.NewSearchQuery()
	defer q.Destroy()
	if err := q.SetFieldName(zvecVectorField); err != nil {
		return nil, err
	}
	if err := q.SetQueryVector(vector); err != nil {
		return nil, err
	}
	if err := q.SetTopK(topK); err != nil {
		return nil, err
	}

	docs, err := z.col.Query(q)
	if err != nil {
		return nil, fmt.Errorf("zvec query: %w", err)
	}
	defer zvec.FreeDocs(docs)

	hits := make([]VectorHit, 0, len(docs))
	for _, d := range docs {
		hits = append(hits, VectorHit{PK: d.GetPK(), Score: d.GetScore()})
	}
	return hits, nil
}

func (z *zvecIndex) Flush() error {
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.col.Flush()
}

func (z *zvecIndex) Close() error {
	z.mu.Lock()
	defer z.mu.Unlock()
	if z.col != nil {
		err := z.col.Close()
		z.col = nil
		return err
	}
	return nil
}
