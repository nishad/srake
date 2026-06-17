package embeddings

import (
	"strings"
	"testing"

	"github.com/nishad/srake/internal/paths"
)

// TestEmbedTruncatesLongInput guards against the regression where inputs longer
// than the model's positional-embedding limit (maxSequenceLength) were passed to
// ONNX inference unchanged, causing a "broadcast an axis ... 512 by N" error and,
// for some inputs, a slice-bounds panic in the tokenizer.
//
// The test needs the SapBERT model and the onnxruntime shared library to be
// present; when they are not (e.g. CI without the model), the embedder reports
// itself disabled and the test skips.
func TestEmbedTruncatesLongInput(t *testing.T) {
	emb, err := NewONNXEmbedder("Xenova/SapBERT-from-PubMedBERT-fulltext", paths.GetModelsPath())
	if err != nil {
		t.Skipf("embedder unavailable: %v", err)
	}
	if !emb.IsEnabled() {
		t.Skip("ONNX embedder not enabled (model or onnxruntime missing)")
	}
	defer emb.Close()

	cases := []struct {
		name string
		text string
	}{
		{"short", "Lung adenocarcinoma RNA-Seq tumor microenvironment"},
		// Well over 512 tokens: previously triggered the broadcast error.
		{"long", strings.Repeat("whole transcriptome sequencing of human lung tumor ", 80)},
		// Many repeated tokens near the character-truncation boundary.
		{"pathological", strings.Repeat("RNA-Seq ", 400)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vec, err := emb.Embed(tc.text)
			if err != nil {
				t.Fatalf("Embed returned error for %q-length input: %v", tc.name, err)
			}
			if len(vec) == 0 {
				t.Fatalf("Embed returned empty vector for %s input", tc.name)
			}
			nonzero := false
			for _, v := range vec {
				if v != 0 {
					nonzero = true
					break
				}
			}
			if !nonzero {
				t.Errorf("Embed returned all-zero vector for %s input", tc.name)
			}
		})
	}
}
