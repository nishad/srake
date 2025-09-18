package search

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

// QueryParser handles advanced query syntax parsing
type QueryParser struct {
	// Field mappings for shorthand notation
	fieldAliases map[string]string
}

// NewQueryParser creates a new query parser
func NewQueryParser() *QueryParser {
	return &QueryParser{
		fieldAliases: map[string]string{
			"org":      "organism",
			"plat":     "platform",
			"lib":      "library_strategy",
			"strat":    "library_strategy",
			"study":    "study_type",
			"inst":     "instrument_model",
			"acc":      "accession",
			"title":    "title",
			"abstract": "study_abstract",
			"spots":    "spots",
			"bases":    "bases",
			"layout":   "library_layout",
			"source":   "library_source",
			"select":   "library_selection",
		},
	}
}

// ParseAdvancedQuery parses an advanced query string into a Bleve query
func (p *QueryParser) ParseAdvancedQuery(queryStr string) (query.Query, error) {
	// Handle empty query
	if queryStr == "" {
		return bleve.NewMatchAllQuery(), nil
	}

	// Check for field-specific queries (field:value)
	if strings.Contains(queryStr, ":") {
		return p.parseFieldQuery(queryStr)
	}

	// Check for boolean operators
	if containsBooleanOp(queryStr) {
		return p.parseBooleanQuery(queryStr)
	}

	// Check for wildcards
	if strings.Contains(queryStr, "*") || strings.Contains(queryStr, "?") {
		return p.parseWildcardQuery(queryStr)
	}

	// Check for phrase queries (quoted strings)
	if strings.Contains(queryStr, "\"") {
		return p.parsePhraseQuery(queryStr)
	}

	// Default to simple query string
	return bleve.NewQueryStringQuery(queryStr), nil
}

// parseFieldQuery parses field-specific queries like "organism:human"
func (p *QueryParser) parseFieldQuery(queryStr string) (query.Query, error) {
	// Regular expression to match field:value patterns
	fieldPattern := regexp.MustCompile(`(\w+):("[^"]+"|[^\s]+)`)
	matches := fieldPattern.FindAllStringSubmatch(queryStr, -1)

	if len(matches) == 0 {
		return bleve.NewQueryStringQuery(queryStr), nil
	}

	queries := []query.Query{}

	for _, match := range matches {
		field := match[1]
		value := strings.Trim(match[2], "\"")

		// Resolve field alias
		if alias, ok := p.fieldAliases[field]; ok {
			field = alias
		}

		// Check for range queries
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			rangeQuery, err := p.parseRangeQuery(field, value)
			if err != nil {
				return nil, err
			}
			queries = append(queries, rangeQuery)
		} else {
			// Create appropriate query based on field type
			fieldQuery := p.createFieldQuery(field, value)
			queries = append(queries, fieldQuery)
		}
	}

	// Handle remaining text not captured by field patterns
	remaining := fieldPattern.ReplaceAllString(queryStr, "")
	remaining = strings.TrimSpace(remaining)
	if remaining != "" {
		queries = append(queries, bleve.NewQueryStringQuery(remaining))
	}

	// Combine queries
	if len(queries) == 1 {
		return queries[0], nil
	}
	return bleve.NewConjunctionQuery(queries...), nil
}

// parseBooleanQuery parses queries with AND, OR, NOT operators
func (p *QueryParser) parseBooleanQuery(queryStr string) (query.Query, error) {
	// Split by OR first (lower precedence)
	orParts := splitByOperator(queryStr, " OR ")
	if len(orParts) > 1 {
		orQueries := []query.Query{}
		for _, part := range orParts {
			q, err := p.parseAndQuery(part)
			if err != nil {
				return nil, err
			}
			orQueries = append(orQueries, q)
		}
		return bleve.NewDisjunctionQuery(orQueries...), nil
	}

	// No OR, try AND
	return p.parseAndQuery(queryStr)
}

// parseAndQuery handles AND and NOT operators
func (p *QueryParser) parseAndQuery(queryStr string) (query.Query, error) {
	// Split by AND
	andParts := splitByOperator(queryStr, " AND ")

	mustQueries := []query.Query{}
	mustNotQueries := []query.Query{}

	for _, part := range andParts {
		part = strings.TrimSpace(part)

		// Handle NOT operator
		if strings.HasPrefix(part, "NOT ") {
			notPart := strings.TrimPrefix(part, "NOT ")
			q, err := p.ParseAdvancedQuery(notPart)
			if err != nil {
				return nil, err
			}
			mustNotQueries = append(mustNotQueries, q)
		} else {
			q, err := p.ParseAdvancedQuery(part)
			if err != nil {
				return nil, err
			}
			mustQueries = append(mustQueries, q)
		}
	}

	// Build boolean query
	if len(mustNotQueries) > 0 {
		boolQuery := bleve.NewBooleanQuery()
		for _, q := range mustQueries {
			boolQuery.AddMust(q)
		}
		for _, q := range mustNotQueries {
			boolQuery.AddMustNot(q)
		}
		return boolQuery, nil
	}

	if len(mustQueries) == 1 {
		return mustQueries[0], nil
	}
	return bleve.NewConjunctionQuery(mustQueries...), nil
}

// parseWildcardQuery handles wildcard patterns
func (p *QueryParser) parseWildcardQuery(queryStr string) (query.Query, error) {
	// Convert simple wildcards to regex
	pattern := strings.ReplaceAll(queryStr, "*", ".*")
	pattern = strings.ReplaceAll(pattern, "?", ".")

	regexQuery := bleve.NewRegexpQuery(pattern)
	return regexQuery, nil
}

// parsePhraseQuery handles quoted phrases
func (p *QueryParser) parsePhraseQuery(queryStr string) (query.Query, error) {
	// Extract quoted phrases
	phrasePattern := regexp.MustCompile(`"([^"]+)"`)
	matches := phrasePattern.FindAllStringSubmatch(queryStr, -1)

	if len(matches) == 0 {
		return bleve.NewQueryStringQuery(queryStr), nil
	}

	queries := []query.Query{}

	for _, match := range matches {
		phrase := match[1]
		phraseQuery := bleve.NewMatchPhraseQuery(phrase)
		queries = append(queries, phraseQuery)
	}

	// Handle remaining text
	remaining := phrasePattern.ReplaceAllString(queryStr, "")
	remaining = strings.TrimSpace(remaining)
	if remaining != "" {
		queries = append(queries, bleve.NewQueryStringQuery(remaining))
	}

	if len(queries) == 1 {
		return queries[0], nil
	}
	return bleve.NewConjunctionQuery(queries...), nil
}

// parseRangeQuery parses range queries like "[100 TO 1000]"
func (p *QueryParser) parseRangeQuery(field, value string) (query.Query, error) {
	// Remove brackets
	value = strings.Trim(value, "[]")

	// Split by TO
	parts := strings.Split(value, " TO ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range syntax: %s", value)
	}

	min := strings.TrimSpace(parts[0])
	max := strings.TrimSpace(parts[1])

	// Check if numeric
	if isNumeric(min) && isNumeric(max) {
		minVal, _ := strconv.ParseFloat(min, 64)
		maxVal, _ := strconv.ParseFloat(max, 64)

		rangeQuery := bleve.NewNumericRangeQuery(&minVal, &maxVal)
		rangeQuery.SetField(field)
		return rangeQuery, nil
	}

	// For non-numeric, use term range
	rangeQuery := bleve.NewTermRangeQuery(min, max)
	rangeQuery.SetField(field)
	return rangeQuery, nil
}

// createFieldQuery creates appropriate query type based on field
func (p *QueryParser) createFieldQuery(field, value string) query.Query {
	// Numeric fields
	if isNumericField(field) {
		if num, err := strconv.ParseFloat(value, 64); err == nil {
			numQuery := bleve.NewNumericRangeQuery(&num, &num)
			numQuery.SetField(field)
			return numQuery
		}
	}

	// Keyword fields (exact match)
	if isKeywordField(field) {
		termQuery := bleve.NewTermQuery(value)
		termQuery.SetField(field)
		return termQuery
	}

	// Text fields (analyzed)
	matchQuery := bleve.NewMatchQuery(value)
	matchQuery.SetField(field)
	return matchQuery
}

// Helper functions
func containsBooleanOp(s string) bool {
	return strings.Contains(s, " AND ") ||
	       strings.Contains(s, " OR ") ||
	       strings.Contains(s, "NOT ")
}

func splitByOperator(s, op string) []string {
	// Split while respecting quoted strings
	var parts []string
	var current strings.Builder
	inQuotes := false

	words := strings.Split(s, " ")
	for i, word := range words {
		if strings.Count(word, "\"")%2 == 1 {
			inQuotes = !inQuotes
		}

		if !inQuotes && i > 0 && word == strings.TrimSpace(op) {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			if current.Len() > 0 {
				current.WriteString(" ")
			}
			current.WriteString(word)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isNumericField(field string) bool {
	numericFields := []string{"spots", "bases", "size", "count"}
	for _, nf := range numericFields {
		if field == nf {
			return true
		}
	}
	return false
}

func isKeywordField(field string) bool {
	keywordFields := []string{
		"platform", "instrument_model", "study_type",
		"library_layout", "accession", "*_accession",
	}
	for _, kf := range keywordFields {
		if field == kf || strings.HasSuffix(field, "_accession") {
			return true
		}
	}
	return false
}

// ParseFilters converts filter flags to field queries
func (p *QueryParser) ParseFilters(filters map[string]string) []query.Query {
	queries := []query.Query{}

	for field, value := range filters {
		if value == "" {
			continue
		}

		// Resolve field alias
		if alias, ok := p.fieldAliases[field]; ok {
			field = alias
		}

		fieldQuery := p.createFieldQuery(field, value)
		queries = append(queries, fieldQuery)
	}

	return queries
}