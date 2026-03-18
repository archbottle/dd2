package common

import (
	"sort"
	"strings"
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

// PatternIndex provides fast keyword-based pattern lookup using map-based substring search.
// T must be a pointer type that implements the Pattern interface.
type PatternIndex[T Pattern] struct {
	// Map-based substring search (replaces Aho-Corasick)
	keywordMap  map[string]int // normalized keyword -> index in keywords slice
	keywordLens []int          // unique keyword lengths (sorted)

	keywords     []string    // keyword at index i
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
// using map-based substring search for memory-efficient multi-pattern matching.
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

	// Build map-based index
	if len(idx.keywords) > 0 {
		idx.keywordMap = make(map[string]int, len(idx.keywords))
		lenSet := make(map[int]bool)
		for i, kw := range idx.keywords {
			idx.keywordMap[kw] = i
			lenSet[len(kw)] = true
		}
		// Store unique lengths sorted
		for l := range lenSet {
			idx.keywordLens = append(idx.keywordLens, l)
		}
		sort.Ints(idx.keywordLens)
	}

	return idx
}

// FindCandidates returns patterns that might match the given text.
// Uses sliding-window map lookup per unique keyword length for memory-efficient matching.
// Returns patterns whose keywords were found, plus all patterns without keywords.
// If patterns implement OrderedPattern, results are sorted by original position.
func (idx *PatternIndex[T]) FindCandidates(text string) []T {
	if len(idx.keywordMap) == 0 {
		return idx.allPatterns
	}

	lower := strings.ToLower(text)
	tlen := len(lower)

	// Find all keywords present using sliding window per unique length
	seen := make(map[any]bool)
	var candidates []T

	for _, klen := range idx.keywordLens {
		if klen > tlen {
			break
		}
		for i := 0; i <= tlen-klen; i++ {
			sub := lower[i : i+klen]
			if kwIdx, ok := idx.keywordMap[sub]; ok {
				for _, p := range idx.kwToPatterns[kwIdx] {
					if !seen[any(p)] {
						seen[any(p)] = true
						candidates = append(candidates, p)
					}
				}
			}
		}
	}

	if len(candidates) == 0 {
		return idx.noKeywords
	}

	candidates = append(candidates, idx.noKeywords...)

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
