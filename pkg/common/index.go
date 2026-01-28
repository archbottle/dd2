package common

import (
	"sort"
	"strings"

	"github.com/cloudflare/ahocorasick"
)

// Pattern is the interface that pattern types must implement.
type Pattern interface {
	GetRegex() string
}

// OrderedPattern extends Pattern with position tracking for order-sensitive matching.
// Patterns implementing this interface will be sorted by position in FindCandidates.
type OrderedPattern interface {
	Pattern
	GetPosition() int
	SetPosition(int)
}

// PatternIndex provides fast keyword-based pattern lookup using Aho-Corasick.
// T must be a pointer type that implements the Pattern interface.
type PatternIndex[T Pattern] struct {
	matcher      *ahocorasick.Matcher
	keywords     []string    // keyword at index i corresponds to matcher result i
	kwToPatterns map[int][]T // keyword index -> patterns that contain this keyword
	noKeywords   []T         // patterns without extractable keywords (always included)
	allPatterns  []T         // all patterns for fallback
}

// IndexStats provides statistics about the index.
type IndexStats struct {
	TotalPatterns         int
	IndexedPatterns       int
	NoKeywordCount        int
	UniqueKeywords        int
	AvgKeywordsPerPattern float64
}

// NewPatternIndex builds an index from patterns.
// Patterns are analyzed to extract keywords, which are then indexed
// using the Aho-Corasick algorithm for O(n) multi-pattern matching.
// If patterns implement OrderedPattern, their positions are set automatically.
func NewPatternIndex[T Pattern](patterns []T) *PatternIndex[T] {
	idx := &PatternIndex[T]{
		kwToPatterns: make(map[int][]T),
		allPatterns:  patterns,
	}

	// Extract keywords from each pattern
	keywordSet := make(map[string]int) // normalized keyword -> index in keywords slice

	for i, p := range patterns {
		// Set position if pattern supports ordering
		if op, ok := any(p).(OrderedPattern); ok {
			op.SetPosition(i)
		}

		keywords := ExtractKeywords(p.GetRegex())

		if len(keywords) == 0 {
			// No keywords extractable - must always be checked
			idx.noKeywords = append(idx.noKeywords, p)
			continue
		}

		// Add pattern to each keyword's list
		for _, kw := range keywords {
			norm := strings.ToLower(kw)
			kwIdx, exists := keywordSet[norm]
			if !exists {
				kwIdx = len(idx.keywords)
				keywordSet[norm] = kwIdx
				// We store normalized keywords so the matcher can be case-insensitive.
				idx.keywords = append(idx.keywords, norm)
			}
			idx.kwToPatterns[kwIdx] = append(idx.kwToPatterns[kwIdx], p)
		}
	}

	// Build Aho-Corasick matcher if we have keywords
	if len(idx.keywords) > 0 {
		idx.matcher = ahocorasick.NewStringMatcher(idx.keywords)
	}

	return idx
}

// FindCandidates returns patterns that might match the given text.
// Uses Aho-Corasick for O(n) keyword matching where n is text length.
// Returns patterns whose keywords were found, plus all patterns without keywords.
// If patterns implement OrderedPattern, results are sorted by original position.
func (idx *PatternIndex[T]) FindCandidates(text string) []T {
	if idx.matcher == nil {
		// No keywords indexed, return all patterns
		return idx.allPatterns
	}

	// Find all keywords present in the text - O(n) where n = len(text)
	matchedIndices := idx.matcher.Match([]byte(strings.ToLower(text)))

	if len(matchedIndices) == 0 {
		// No keywords matched, only return patterns without keywords
		return idx.noKeywords
	}

	// Collect unique patterns from matched keywords
	seen := make(map[any]bool)
	var candidates []T

	for _, kwIdx := range matchedIndices {
		for _, p := range idx.kwToPatterns[kwIdx] {
			// Use pattern pointer as key for deduplication
			if !seen[any(p)] {
				seen[any(p)] = true
				candidates = append(candidates, p)
			}
		}
	}

	// Always include patterns without keywords
	candidates = append(candidates, idx.noKeywords...)

	// Sort by original position if patterns support ordering
	// This ensures first-match-wins semantics like PHP
	if len(candidates) > 0 {
		if _, ok := any(candidates[0]).(OrderedPattern); ok {
			sort.Slice(candidates, func(i, j int) bool {
				pi := any(candidates[i]).(OrderedPattern)
				pj := any(candidates[j]).(OrderedPattern)
				return pi.GetPosition() < pj.GetPosition()
			})
		}
	}

	return candidates
}

// Stats returns statistics about the index.
func (idx *PatternIndex[T]) Stats() IndexStats {
	indexed := len(idx.allPatterns) - len(idx.noKeywords)
	avgKw := 0.0
	if indexed > 0 {
		totalKwRefs := 0
		for _, patterns := range idx.kwToPatterns {
			totalKwRefs += len(patterns)
		}
		avgKw = float64(totalKwRefs) / float64(indexed)
	}

	return IndexStats{
		TotalPatterns:         len(idx.allPatterns),
		IndexedPatterns:       indexed,
		NoKeywordCount:        len(idx.noKeywords),
		UniqueKeywords:        len(idx.keywords),
		AvgKeywordsPerPattern: avgKw,
	}
}

// AllPatterns returns all patterns (for fallback scenarios).
func (idx *PatternIndex[T]) AllPatterns() []T {
	return idx.allPatterns
}

// NoKeywordPatterns returns patterns that couldn't be indexed.
func (idx *PatternIndex[T]) NoKeywordPatterns() []T {
	return idx.noKeywords
}
