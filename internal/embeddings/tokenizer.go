package embeddings

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"unicode"
)

// Tokenizer handles text tokenization for BERT models
type Tokenizer struct {
	vocab         map[string]int64
	invVocab      map[int64]string
	specialTokens map[string]int64
	maxLength     int
	doLowerCase   bool
	unkToken      string
	sepToken      string
	padToken      string
	clsToken      string
	maskToken     string
}

// TokenizerConfig represents tokenizer configuration
type TokenizerConfig struct {
	DoLowerCase   bool             `json:"do_lower_case"`
	MaxLength     int              `json:"model_max_length"`
	PadToken      string           `json:"pad_token"`
	SepToken      string           `json:"sep_token"`
	ClsToken      string           `json:"cls_token"`
	UnkToken      string           `json:"unk_token"`
	MaskToken     string           `json:"mask_token"`
	SpecialTokens map[string]int64 `json:"special_tokens_map"`
}

// Encoding represents tokenized text
type Encoding struct {
	InputIDs      []int64  `json:"input_ids"`
	AttentionMask []int64  `json:"attention_mask"`
	TokenTypeIDs  []int64  `json:"token_type_ids"`
	Tokens        []string `json:"tokens"`
}

// LoadTokenizer loads a tokenizer from model directory
func LoadTokenizer(modelPath string) (*Tokenizer, error) {
	tokenizer := &Tokenizer{
		vocab:         make(map[string]int64),
		invVocab:      make(map[int64]string),
		specialTokens: make(map[string]int64),
		maxLength:     512,
		doLowerCase:   true,
		unkToken:      "[UNK]",
		sepToken:      "[SEP]",
		padToken:      "[PAD]",
		clsToken:      "[CLS]",
		maskToken:     "[MASK]",
	}

	// Try to load tokenizer_config.json
	configPath := filepath.Join(modelPath, "tokenizer_config.json")
	if configData, err := ioutil.ReadFile(configPath); err == nil {
		var config TokenizerConfig
		if err := json.Unmarshal(configData, &config); err == nil {
			tokenizer.doLowerCase = config.DoLowerCase
			if config.MaxLength > 0 {
				tokenizer.maxLength = config.MaxLength
			}
			if config.PadToken != "" {
				tokenizer.padToken = config.PadToken
			}
			if config.SepToken != "" {
				tokenizer.sepToken = config.SepToken
			}
			if config.ClsToken != "" {
				tokenizer.clsToken = config.ClsToken
			}
			if config.UnkToken != "" {
				tokenizer.unkToken = config.UnkToken
			}
			if config.MaskToken != "" {
				tokenizer.maskToken = config.MaskToken
			}
		}
	}

	// Try to load vocab.txt
	vocabPath := filepath.Join(modelPath, "vocab.txt")
	if vocabData, err := ioutil.ReadFile(vocabPath); err == nil {
		lines := strings.Split(string(vocabData), "\n")
		for i, line := range lines {
			token := strings.TrimSpace(line)
			if token != "" {
				tokenizer.vocab[token] = int64(i)
				tokenizer.invVocab[int64(i)] = token
			}
		}
	} else {
		// Try to load tokenizer.json (HuggingFace format)
		tokenizerPath := filepath.Join(modelPath, "tokenizer.json")
		if err := tokenizer.loadHuggingFaceTokenizer(tokenizerPath); err != nil {
			// Fallback to basic tokenizer
			tokenizer.initBasicVocab()
		}
	}

	// Set special token IDs
	tokenizer.specialTokens[tokenizer.padToken] = tokenizer.vocab[tokenizer.padToken]
	tokenizer.specialTokens[tokenizer.clsToken] = tokenizer.vocab[tokenizer.clsToken]
	tokenizer.specialTokens[tokenizer.sepToken] = tokenizer.vocab[tokenizer.sepToken]
	tokenizer.specialTokens[tokenizer.unkToken] = tokenizer.vocab[tokenizer.unkToken]
	tokenizer.specialTokens[tokenizer.maskToken] = tokenizer.vocab[tokenizer.maskToken]

	return tokenizer, nil
}

// loadHuggingFaceTokenizer loads a HuggingFace tokenizer.json file
func (t *Tokenizer) loadHuggingFaceTokenizer(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var tokenizerData map[string]interface{}
	if err := json.Unmarshal(data, &tokenizerData); err != nil {
		return err
	}

	// Extract vocab from model section
	if model, ok := tokenizerData["model"].(map[string]interface{}); ok {
		if vocab, ok := model["vocab"].(map[string]interface{}); ok {
			for token, idx := range vocab {
				if idxFloat, ok := idx.(float64); ok {
					t.vocab[token] = int64(idxFloat)
					t.invVocab[int64(idxFloat)] = token
				}
			}
		}
	}

	return nil
}

// initBasicVocab initializes a basic vocabulary for testing
func (t *Tokenizer) initBasicVocab() {
	// Initialize with special tokens and basic vocabulary
	basicTokens := []string{
		"[PAD]", "[UNK]", "[CLS]", "[SEP]", "[MASK]",
	}

	for i, token := range basicTokens {
		t.vocab[token] = int64(i)
		t.invVocab[int64(i)] = token
	}

	// Add some common words for testing
	commonWords := []string{
		"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for",
		"of", "with", "by", "from", "as", "is", "was", "are", "were", "been",
		"human", "mouse", "cell", "gene", "protein", "rna", "dna", "seq",
		"experiment", "sample", "study", "analysis", "sequencing",
	}

	for i, word := range commonWords {
		idx := int64(len(basicTokens) + i)
		t.vocab[word] = idx
		t.invVocab[idx] = word
		// Also add uppercase variant
		upperWord := strings.ToUpper(word)
		t.vocab[upperWord] = idx + int64(len(commonWords))
		t.invVocab[idx+int64(len(commonWords))] = upperWord
	}
}

// Encode tokenizes text and returns token IDs
func (t *Tokenizer) Encode(text string, maxLength int) (*Encoding, error) {
	if maxLength <= 0 {
		maxLength = t.maxLength
	}

	// Preprocess text
	if t.doLowerCase {
		text = strings.ToLower(text)
	}

	// Simple whitespace tokenization (should use WordPiece for real BERT)
	tokens := t.tokenize(text)

	// Add special tokens
	tokens = append([]string{t.clsToken}, tokens...)
	tokens = append(tokens, t.sepToken)

	// Truncate if needed (leaving room for special tokens)
	if len(tokens) > maxLength {
		tokens = tokens[:maxLength-1]
		tokens = append(tokens, t.sepToken)
	}

	// Convert tokens to IDs
	inputIDs := make([]int64, len(tokens))
	for i, token := range tokens {
		if id, ok := t.vocab[token]; ok {
			inputIDs[i] = id
		} else {
			inputIDs[i] = t.vocab[t.unkToken]
		}
	}

	// Create attention mask (1 for real tokens, 0 for padding)
	attentionMask := make([]int64, len(inputIDs))
	for i := range attentionMask {
		attentionMask[i] = 1
	}

	// Pad to maxLength if needed
	for len(inputIDs) < maxLength {
		inputIDs = append(inputIDs, t.vocab[t.padToken])
		attentionMask = append(attentionMask, 0)
	}

	// Token type IDs (0 for single sequence)
	tokenTypeIDs := make([]int64, len(inputIDs))

	return &Encoding{
		InputIDs:      inputIDs,
		AttentionMask: attentionMask,
		TokenTypeIDs:  tokenTypeIDs,
		Tokens:        tokens,
	}, nil
}

// tokenize performs basic tokenization
func (t *Tokenizer) tokenize(text string) []string {
	var tokens []string
	words := strings.Fields(text)

	for _, word := range words {
		// Split on punctuation
		subTokens := t.splitOnPunctuation(word)
		tokens = append(tokens, subTokens...)
	}

	return tokens
}

// splitOnPunctuation splits a word on punctuation
func (t *Tokenizer) splitOnPunctuation(word string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range word {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(r))
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// Decode converts token IDs back to text
func (t *Tokenizer) Decode(inputIDs []int64) string {
	var tokens []string

	for _, id := range inputIDs {
		if token, ok := t.invVocab[id]; ok {
			// Skip special tokens
			if token == t.padToken || token == t.clsToken || token == t.sepToken {
				continue
			}
			tokens = append(tokens, token)
		}
	}

	return strings.Join(tokens, " ")
}

// GetVocabSize returns the size of the vocabulary
func (t *Tokenizer) GetVocabSize() int {
	return len(t.vocab)
}

// GetSpecialTokens returns the special tokens
func (t *Tokenizer) GetSpecialTokens() map[string]int64 {
	return t.specialTokens
}
