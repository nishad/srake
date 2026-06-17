package embeddings

import (
	"math"
	"testing"

	"github.com/nishad/srake/internal/paths"
)

func cosineSim(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return -1
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// TestEmbedderMatchesONNXEmbedder guards the fix that routed Embedder (the index
// and sync path) through the real ONNXEmbedder instead of a random-vector stub.
// Index-time embeddings (Embedder) must match query-time embeddings
// (ONNXEmbedder) for the same text, and constructing both in one process must
// work (the onnxruntime environment is a singleton guarded by sync.Once).
//
// Skips when onnxruntime / the model is unavailable.
func TestEmbedderMatchesONNXEmbedder(t *testing.T) {
	const model = "Xenova/SapBERT-from-PubMedBERT-fulltext"
	text := "single cell RNA sequencing of human lung adenocarcinoma tumor microenvironment"

	onnx, err := NewONNXEmbedder(model, paths.GetModelsPath())
	if err != nil || !onnx.IsEnabled() {
		t.Skipf("ONNX embedder unavailable: %v", err)
	}
	defer onnx.Close()
	queryVec, err := onnx.Embed(text)
	if err != nil {
		t.Fatalf("query embed: %v", err)
	}

	// Second embedder in the same process — must not fail on ORT re-init.
	cfg := DefaultEmbedderConfig()
	cfg.ModelsDir = paths.GetModelsPath()
	emb, err := NewEmbedder(cfg)
	if err != nil {
		t.Fatalf("NewEmbedder: %v", err)
	}
	defer emb.Close()
	if err := emb.LoadDefaultModel(); err != nil {
		t.Fatalf("LoadDefaultModel: %v", err)
	}
	if !emb.IsEnabled() {
		t.Fatal("Embedder not enabled after LoadDefaultModel (second ORT init failed?)")
	}
	indexVec, err := emb.EmbedText(text)
	if err != nil {
		t.Fatalf("index embed: %v", err)
	}

	if cos := cosineSim(indexVec, queryVec); cos < 0.9999 {
		t.Errorf("index embedding does not match query embedding: cos=%.6f (want ~1.0); "+
			"the index path may still be producing stub/random vectors", cos)
	}
}
