package database

import (
	"strings"
	"testing"
)

func TestEscapeFTSQuery(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", `""`},
		{"whitespace only", "   ", `""`},
		{"single word", "illumina", `"illumina"`},
		{"accession", "SRP123456", `"SRP123456"`},
		// Multi-word queries become a conjunction of quoted terms (implicit AND),
		// not one rigid phrase, so recall is preserved.
		{"two words", "homo sapiens", `"homo" "sapiens"`},
		// Hyphens and other FTS5 operators are kept literal via quoting.
		{"hyphenated", "RNA-Seq", `"RNA-Seq"`},
		{"operators", "rna-seq illumina", `"rna-seq" "illumina"`},
		// Embedded double quotes are doubled, the only FTS5 escape mechanism.
		{"embedded quote", `say "hi"`, `"say" """hi"""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeFTSQuery(tt.input); got != tt.want {
				t.Errorf("escapeFTSQuery(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSearchExperimentsByFilterRejectsUnknownField(t *testing.T) {
	mgr := &FTS5Manager{}
	_, err := mgr.SearchExperimentsByFilter(map[string]string{"unknown_col": "x"}, 10)
	if err == nil {
		t.Fatal("expected error for unsupported filter field, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported filter field") {
		t.Errorf("error = %q, want it to mention unsupported filter field", err.Error())
	}
}
