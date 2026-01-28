// Package common provides reusable components for device detection parsers.
package common

import (
	"regexp"
	"strings"
	"unicode"
)

// ExtractKeywords extracts indexable keywords from a regex pattern.
// These are literal strings that MUST appear in any matching input.
func ExtractKeywords(pattern string) []string {
	seen := make(map[string]bool)
	var keywords []string

	addKeyword := func(kw string, minLen int) {
		kw = strings.TrimSpace(kw)
		if len(kw) >= minLen && IsUsefulKeyword(kw) && !seen[kw] {
			seen[kw] = true
			keywords = append(keywords, kw)
		}
	}

	// Strategy 1: Extract keywords using the advanced extractor that handles
	// prefix+group patterns like S(?:CH|GH|M)- → SCH-, SGH-, SM-
	for _, kw := range extractKeywordsAdvanced(pattern) {
		addKeyword(kw, 3)
	}

	// Strategy 2: Handle top-level alternation - extract from each branch
	if strings.Contains(pattern, "|") {
		for _, branch := range splitAlternation(pattern) {
			for _, kw := range extractKeywordsAdvanced(branch) {
				addKeyword(kw, 3)
			}
		}
	}

	// Strategy 3: Add keyword aliases for better coverage
	for _, kw := range keywords {
		for _, alias := range getKeywordAliases(kw) {
			addKeyword(alias, 3)
		}
	}

	// Strategy 4: Handle short brand prefixes (2-char prefixes like LG-, BQ-, WH)
	// These are intentionally allowed to be shorter (min 2 chars) since they're
	// pre-vetted as distinctive prefixes
	for _, kw := range getShortPrefixKeywords(pattern) {
		addKeyword(kw, 2)
	}

	return keywords
}

// extractKeywordsAdvanced extracts keywords with special handling for patterns like:
// - S(?:CH|GH|M)- → SCH-, SGH-, SM-
// - LG(?!...) → LG (literal before lookahead)
// - (?:prefix)?suffix → suffix (optional prefix with required suffix)
func extractKeywordsAdvanced(pattern string) []string {
	var results []string

	// First, expand simple prefix+alternation patterns: X(?:A|B|C)Y → XAY, XBY, XCY
	expanded := expandPrefixAlternations(pattern)
	for _, exp := range expanded {
		for _, lit := range extractLiterals(exp) {
			results = append(results, lit)
		}
	}

	// Also extract literals that appear before lookahead assertions
	results = append(results, extractBeforeLookahead(pattern)...)

	return results
}

// expandPrefixAlternations expands patterns like X(?:A|B|C)Y into [XAY, XBY, XCY].
// Only expands when alternatives are short (<=4 chars each) to avoid explosion.
var prefixAltRegex = regexp.MustCompile(`([A-Za-z0-9_-]{1,4})\(\?:([A-Za-z0-9_|/-]{1,30})\)([A-Za-z0-9_/-]{0,4})`)

func expandPrefixAlternations(pattern string) []string {
	matches := prefixAltRegex.FindAllStringSubmatch(pattern, -1)
	if len(matches) == 0 {
		return []string{pattern}
	}

	var expanded []string
	for _, m := range matches {
		prefix := m[1]
		alts := strings.Split(m[2], "|")
		suffix := m[3]

		// Only expand if we have reasonable number of alternatives
		if len(alts) <= 10 {
			for _, alt := range alts {
				// Skip if alternative contains regex metacharacters
				if strings.ContainsAny(alt, "[]().*+?^${}\\") {
					continue
				}
				combined := prefix + alt + suffix
				if len(combined) >= 3 {
					expanded = append(expanded, combined)
				}
			}
		}
	}

	// Also return original pattern for standard extraction
	expanded = append(expanded, pattern)
	return expanded
}

// extractBeforeLookahead extracts literals that appear immediately before lookahead.
// Pattern: LITERAL(?!...) or LITERAL(?=...) → LITERAL
var lookaheadRegex = regexp.MustCompile(`([A-Za-z0-9_/-]{2,})\(\?[!=]`)

func extractBeforeLookahead(pattern string) []string {
	matches := lookaheadRegex.FindAllStringSubmatch(pattern, -1)
	var results []string
	for _, m := range matches {
		if len(m) > 1 && len(m[1]) >= 2 {
			results = append(results, m[1])
		}
	}
	return results
}

// getKeywordAliases returns related keywords that should also be indexed.
// This handles common cases where the same concept has multiple representations.
func getKeywordAliases(kw string) []string {
	kwLower := strings.ToLower(kw)

	// Aliases map: if we see keyword X, also index pattern for keyword Y
	aliases := map[string][]string{
		// Linux variants - if a pattern mentions "Linux", user agents with
		// various Linux-like strings should be candidates
		"linux/":     {"Linux"},
		"gnu/linux":  {"Linux"},
		"linux armv": {"Linux"},
		"linux aarc": {"Linux"},
		// Common device prefixes that might appear with or without dash
		"sm-": {"SM-G", "SM-A", "SM-J", "SM-N", "SM-T", "SM-S"},
		"lg-": {"LG-D", "LG-H", "LG-K", "LG-V", "LG-M"},
		"lm-": {"LG-D", "LG-H", "LG-K", "LG-V", "LG-M"}, // LG modern format
		// Xiaomi - MI [a-z0-9]+ patterns need to catch MI followed by anything
		"mi-one": {"MI CC", "MI A", "MI 8", "MI 9", "MI 10", "MI 11", "MI Max", "MI Note", "MI MIX", "MI PAD"},
		"xiaomi": {"MI CC", "MI A", "MI 8", "MI 9", "MI 10", "MI 11", "MI Max", "MI Note", "MI MIX", "MI PAD"},
	}

	if related, ok := aliases[kwLower]; ok {
		return related
	}
	return nil
}

// shortBrandPrefixes contains 2-character brand prefixes that commonly appear
// followed by a dash/space/slash and model identifier in user agents. These are too short
// for normal keyword extraction (min 3 chars) but are very distinctive.
var shortBrandPrefixes = []string{
	"LG", // LG-D370, LG-H850, etc.
	"BQ", // BQ-5515L, BQ-6430L, etc.
	"MI", // MI-One, MI CC 9, etc.
	"ZT", // ZTE devices
	"HM", // Xiaomi HM devices
	"WH", // Whale OS: WH1.0
}

// shortPrefixRegex matches patterns like BQ(?:S|ru)?- where a short prefix
// is followed by optional variants and a dash
var shortPrefixRegex = regexp.MustCompile(`\b([A-Z]{2})(?:\(\?:[^)]+\))?\??\s*[-_ ]`)

// getShortPrefixKeywords extracts keywords for 2-char brand prefixes.
// These are handled specially because they're below the normal 3-char minimum
// but are highly distinctive when followed by a dash.
func getShortPrefixKeywords(pattern string) []string {
	var results []string
	patternUpper := strings.ToUpper(pattern)

	for _, prefix := range shortBrandPrefixes {
		// Look for PREFIX- or PREFIX_ or PREFIX  or PREFIX/ in pattern
		// Handle patterns like: BQ(?:S|ru)?-  or  LG(?!...)  or  LG- or WH/
		dashPattern := prefix + "-"
		spacePattern := prefix + " "
		underPattern := prefix + "_"
		slashPattern := prefix + "/"
		// Also handle patterns with alternation: (?:WH|WhaleTV) or (WH|WhaleTV)
		altPattern1 := "|" + prefix + ")"
		altPattern2 := "|" + prefix + "|"
		altPattern3 := ":" + prefix + "|"

		if strings.Contains(patternUpper, dashPattern) ||
			strings.Contains(patternUpper, spacePattern) ||
			strings.Contains(patternUpper, underPattern) ||
			strings.Contains(patternUpper, slashPattern) ||
			strings.Contains(patternUpper, altPattern1) ||
			strings.Contains(patternUpper, altPattern2) ||
			strings.Contains(patternUpper, altPattern3) {
			results = append(results, prefix+"-")
			results = append(results, prefix+"/")
			results = append(results, prefix)
			continue
		}

		// Handle patterns with optional groups: BQ(?:S|ru)?-
		// These have the prefix followed by (?:...) and possibly ? then -
		optGroupPattern := prefix + "(?:"
		if strings.Contains(patternUpper, optGroupPattern) {
			results = append(results, prefix+"-")
			results = append(results, prefix+"/")
			results = append(results, prefix)
		}
	}
	return results
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
	// Skip very common standalone strings that appear in almost all UAs.
	// Note: We allow these when they're part of longer keywords (e.g., "Linux/" is useful).
	commonStrings := map[string]bool{
		"Mozilla":     true,
		"AppleWebKit": true,
		"KHTML":       true,
		"like":        true,
		"Gecko":       true,
		"Safari":      true,
		"Build":       true,
		"compatible":  true,
		"MSIE":        true,
		// Note: Linux and Windows are NOT filtered - they're needed for OS detection
		// patterns like "Linux armv" or "Linux aarch64" for GNU/Linux detection.
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
