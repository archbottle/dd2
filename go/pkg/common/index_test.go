package common

import (
	"strings"
	"testing"
)

// testPattern is a simple pattern implementation for testing.
type testPattern struct {
	name  string
	regex string
}

func (p *testPattern) GetRegex() string {
	return p.regex
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "simple literal",
			pattern:  "Googlebot",
			expected: []string{"Googlebot"},
		},
		{
			name:     "literal with version",
			pattern:  `Chrome/(\d+[\.\d]+)`,
			expected: []string{"Chrome/"},
		},
		{
			name:     "alternation",
			pattern:  "SAMSUNG|Galaxy|GT-",
			expected: []string{"SAMSUNG", "Galaxy", "GT-"},
		},
		{
			name:     "complex pattern",
			pattern:  `SM-G998([BNU])`,
			expected: []string{"SM-G998"},
		},
		{
			name:     "no useful keywords",
			pattern:  `\d+\.\d+`,
			expected: []string{},
		},
		{
			name:     "pattern with common prefix",
			pattern:  "Mozilla/5.0",
			expected: []string{"Mozilla/5"}, // extracts the literal part
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractKeywords(tt.pattern)
			if len(got) != len(tt.expected) {
				t.Errorf("ExtractKeywords(%q) = %v, want %v", tt.pattern, got, tt.expected)
				return
			}
			for i, kw := range got {
				if kw != tt.expected[i] {
					t.Errorf("ExtractKeywords(%q)[%d] = %q, want %q", tt.pattern, i, kw, tt.expected[i])
				}
			}
		})
	}
}

func TestExtractKeywords_SkipsGroupPrefixTokens(t *testing.T) {
	// Regression: patterns with (?:...) should not produce bogus literals like ":Tesla/".
	pattern := `(?:Tesla/(?:(?:develop|feature|terminal-das-fsd-eap)-)?[0-9.]+|QtCarBrowser)`
	got := ExtractKeywords(pattern)

	hasTesla := false
	hasQt := false
	hasBogus := false
	for _, kw := range got {
		if kw == "Tesla/" {
			hasTesla = true
		}
		if kw == "QtCarBrowser" {
			hasQt = true
		}
		if strings.Contains(kw, ":Tesla/") || strings.HasPrefix(kw, ":") {
			hasBogus = true
		}
	}

	if hasBogus {
		t.Fatalf("ExtractKeywords() produced bogus group-prefix keyword(s): %v", got)
	}
	if !hasTesla && !hasQt {
		t.Fatalf("ExtractKeywords() did not extract expected literals from pattern; got: %v", got)
	}
}

func TestPatternIndex_FindCandidates(t *testing.T) {
	patterns := []*testPattern{
		{name: "Googlebot", regex: "Googlebot"},
		{name: "AhrefsBot", regex: "AhrefsBot"},
		{name: "Chrome", regex: `Chrome/(\d+)`},
		{name: "Samsung", regex: "SM-G998"},
		{name: "Generic", regex: `\d+\.\d+`}, // No keywords - always included
	}

	idx := NewPatternIndex(patterns)

	tests := []struct {
		name         string
		text         string
		wantNames    []string
		wantMinCount int
		wantMaxCount int
	}{
		{
			name:         "Googlebot UA",
			text:         "Googlebot/2.1 (http://www.googlebot.com/bot.html)",
			wantNames:    []string{"Googlebot", "Generic"},
			wantMinCount: 2,
			wantMaxCount: 2,
		},
		{
			name:         "AhrefsBot UA",
			text:         "Mozilla/5.0 (compatible; AhrefsBot/7.0; +http://ahrefs.com/robot/)",
			wantNames:    []string{"AhrefsBot", "Generic"},
			wantMinCount: 2,
			wantMaxCount: 2,
		},
		{
			name:         "Chrome browser",
			text:         "Mozilla/5.0 Chrome/91.0.4472.124",
			wantNames:    []string{"Chrome", "Generic"},
			wantMinCount: 2,
			wantMaxCount: 2,
		},
		{
			name:         "Samsung device",
			text:         "Mozilla/5.0 (Linux; Android 10; SM-G998B)",
			wantNames:    []string{"Samsung", "Generic"},
			wantMinCount: 2,
			wantMaxCount: 2,
		},
		{
			name:         "No match - only generic",
			text:         "Some random user agent",
			wantNames:    []string{"Generic"},
			wantMinCount: 1,
			wantMaxCount: 1,
		},
		{
			name:         "Multiple matches",
			text:         "Googlebot Chrome/91.0",
			wantNames:    []string{"Googlebot", "Chrome", "Generic"},
			wantMinCount: 3,
			wantMaxCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := idx.FindCandidates(tt.text)

			if len(candidates) < tt.wantMinCount || len(candidates) > tt.wantMaxCount {
				t.Errorf("FindCandidates(%q) returned %d candidates, want %d-%d",
					tt.text, len(candidates), tt.wantMinCount, tt.wantMaxCount)
			}

			// Check that expected patterns are present
			foundNames := make(map[string]bool)
			for _, p := range candidates {
				foundNames[p.name] = true
			}

			for _, name := range tt.wantNames {
				if !foundNames[name] {
					t.Errorf("FindCandidates(%q) missing pattern %q", tt.text, name)
				}
			}
		})
	}
}

func TestPatternIndex_CaseInsensitive(t *testing.T) {
	patterns := []*testPattern{
		{name: "AhrefsBot", regex: "AhrefsBot"},
		{name: "Generic", regex: `\d+\.\d+`}, // noKeywords
	}

	idx := NewPatternIndex(patterns)
	candidates := idx.FindCandidates("mozilla/5.0 (compatible; ahrefsbot/7.0; +http://ahrefs.com/robot/)")

	found := map[string]bool{}
	for _, p := range candidates {
		found[p.name] = true
	}

	if !found["AhrefsBot"] {
		t.Fatalf("expected AhrefsBot to be in candidates for lowercase UA; got names=%v", found)
	}
	if !found["Generic"] {
		t.Fatalf("expected Generic (noKeywords) to always be included; got names=%v", found)
	}
}

func TestPatternIndex_Stats(t *testing.T) {
	patterns := []*testPattern{
		{name: "Googlebot", regex: "Googlebot"},
		{name: "AhrefsBot", regex: "AhrefsBot"},
		{name: "Generic", regex: `\d+`}, // No keywords
	}

	idx := NewPatternIndex(patterns)
	stats := idx.Stats()

	if stats.TotalPatterns != 3 {
		t.Errorf("TotalPatterns = %d, want 3", stats.TotalPatterns)
	}

	if stats.IndexedPatterns != 2 {
		t.Errorf("IndexedPatterns = %d, want 2", stats.IndexedPatterns)
	}

	if stats.NoKeywordCount != 1 {
		t.Errorf("NoKeywordCount = %d, want 1", stats.NoKeywordCount)
	}

	if stats.UniqueKeywords != 2 {
		t.Errorf("UniqueKeywords = %d, want 2", stats.UniqueKeywords)
	}
}

func TestPatternIndex_EmptyPatterns(t *testing.T) {
	var patterns []*testPattern
	idx := NewPatternIndex(patterns)

	candidates := idx.FindCandidates("any text")
	if len(candidates) != 0 {
		t.Errorf("Expected 0 candidates for empty index, got %d", len(candidates))
	}
}

func BenchmarkFindCandidates(b *testing.B) {
	// Create a realistic set of patterns
	patterns := make([]*testPattern, 800)
	for i := 0; i < 800; i++ {
		patterns[i] = &testPattern{
			name:  "Bot" + string(rune('A'+i%26)),
			regex: "Bot" + string(rune('A'+i%26)) + `bot/\d+`,
		}
	}

	idx := NewPatternIndex(patterns)
	ua := "Mozilla/5.0 (compatible; BotAbot/7.0; +http://example.com/)"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = idx.FindCandidates(ua)
	}
}
