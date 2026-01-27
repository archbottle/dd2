// Package common provides reusable components for device detection parsers.
package common

import (
	"strings"
	"unicode"
)

// ExtractKeywords extracts indexable keywords from a regex pattern.
// These are literal strings that MUST appear in any matching input.
func ExtractKeywords(pattern string) []string {
	keywords := []string{}

	normalizeKeyword := func(s string) string {
		// Keywords are only used for candidate indexing, so we can safely normalize
		// away leading/trailing whitespace that commonly appears in regex literals.
		// This fixes cases like `Arch ?Linux` where the extracted literal becomes `Arch `
		// (trailing space) and would otherwise never match `ArchLinux` in the UA.
		s = strings.TrimSpace(s)
		return s
	}

	// Strategy 1: Find literal strings (not inside character classes or groups)
	literals := extractLiterals(pattern)
	for _, lit := range literals {
		lit = normalizeKeyword(lit)
		// Only use keywords that are at least 3 chars and meaningful
		if len(lit) >= 3 && IsUsefulKeyword(lit) {
			keywords = append(keywords, lit)
		}
	}

	// Strategy 2: Handle alternation - extract from each branch
	if strings.Contains(pattern, "|") {
		branches := splitAlternation(pattern)
		for _, branch := range branches {
			branchKeywords := extractLiterals(branch)
			for _, kw := range branchKeywords {
				kw = normalizeKeyword(kw)
				if len(kw) >= 3 && IsUsefulKeyword(kw) {
					// Only add if not already present
					found := false
					for _, existing := range keywords {
						if existing == kw {
							found = true
							break
						}
					}
					if !found {
						keywords = append(keywords, kw)
					}
				}
			}
		}
	}

	return keywords
}

// extractLiterals finds literal string sequences in a regex.
func extractLiterals(pattern string) []string {
	var literals []string
	var current strings.Builder

	inCharClass := false
	inGroup := 0
	escaped := false

	for i := 0; i < len(pattern); i++ {
		c := pattern[i]

		if escaped {
			// Handle common escaped literals
			switch c {
			case 'd', 'w', 's', 'D', 'W', 'S', 'b', 'B':
				// These are metacharacters, flush current literal
				if current.Len() > 0 {
					literals = append(literals, current.String())
					current.Reset()
				}
			case '.', '/', '-', '_', ' ':
				// These are escaped literals, keep them
				current.WriteByte(c)
			default:
				// Other escaped chars - flush to be safe
				if current.Len() > 0 {
					literals = append(literals, current.String())
					current.Reset()
				}
			}
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if c == '[' && !inCharClass {
			inCharClass = true
			if current.Len() > 0 {
				literals = append(literals, current.String())
				current.Reset()
			}
			continue
		}

		if c == ']' && inCharClass {
			inCharClass = false
			continue
		}

		if inCharClass {
			continue // Skip character class contents
		}

		if c == '(' {
			// Groups break literal sequences.
			if current.Len() > 0 {
				literals = append(literals, current.String())
				current.Reset()
			}

			inGroup++

			// Skip group prefix tokens like (?:, (?=, (?!, (?<=, (?<!
			// We still want to consider literals inside the group; we just don't want
			// the prefix syntax itself to become a "keyword" (e.g. ":Tesla/").
			if i+1 < len(pattern) && pattern[i+1] == '?' {
				i++ // skip '?'
				// Optional "<" for lookbehind: (?<= or (?<!
				if i+1 < len(pattern) && pattern[i+1] == '<' {
					i++ // skip '<'
				}
				// Optional one-char modifier: ':', '=', '!', 'i', etc.
				// We only need to handle the common ones; unknown modifiers should still
				// not become literals, so we skip a single char if present.
				if i+1 < len(pattern) {
					switch pattern[i+1] {
					case ':', '=', '!':
						i++ // skip modifier
					}
				}
			}
			continue
		}

		if c == ')' {
			// Groups break literal sequences.
			if current.Len() > 0 {
				literals = append(literals, current.String())
				current.Reset()
			}
			inGroup--
			continue
		}

		// Metacharacters that break literal sequences
		switch c {
		case '.', '*', '+', '?', '^', '$', '{', '}', '|':
			if current.Len() > 0 {
				literals = append(literals, current.String())
				current.Reset()
			}
		default:
			// Regular character - add to current literal
			if c >= 32 && c < 127 { // Printable ASCII
				current.WriteByte(c)
			}
		}
	}

	// Don't forget the last literal
	if current.Len() > 0 {
		literals = append(literals, current.String())
	}

	return literals
}

// splitAlternation splits a regex by top-level alternation (|).
func splitAlternation(pattern string) []string {
	var branches []string
	var current strings.Builder

	depth := 0
	inCharClass := false
	escaped := false

	for i := 0; i < len(pattern); i++ {
		c := pattern[i]

		if escaped {
			current.WriteByte('\\')
			current.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if c == '[' && !inCharClass {
			inCharClass = true
			current.WriteByte(c)
			continue
		}

		if c == ']' && inCharClass {
			inCharClass = false
			current.WriteByte(c)
			continue
		}

		if inCharClass {
			current.WriteByte(c)
			continue
		}

		if c == '(' {
			depth++
			current.WriteByte(c)
			continue
		}

		if c == ')' {
			depth--
			current.WriteByte(c)
			continue
		}

		if c == '|' && depth == 0 {
			branches = append(branches, current.String())
			current.Reset()
			continue
		}

		current.WriteByte(c)
	}

	if current.Len() > 0 {
		branches = append(branches, current.String())
	}

	return branches
}

// IsUsefulKeyword checks if a keyword is useful for indexing.
// Filters out very common strings that appear in almost all UAs.
func IsUsefulKeyword(kw string) bool {
	// Skip very common strings that appear in almost all UAs
	commonStrings := map[string]bool{
		"Mozilla":     true,
		"AppleWebKit": true,
		"KHTML":       true,
		"like":        true,
		"Gecko":       true,
		"Safari":      true,
		"Build":       true,
		"Linux":       true,
		"Windows":     true,
		"compatible":  true,
		"MSIE":        true,
	}

	if commonStrings[kw] {
		return false
	}

	// Must have at least one letter
	hasLetter := false
	for _, r := range kw {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}

	return hasLetter
}
