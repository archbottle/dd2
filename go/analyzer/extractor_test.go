package analyzer

import (
	"testing"
)

func TestCheckRE2Compatibility(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		expectRE2Safe bool
		expectType    string
	}{
		{
			name:          "simple literal",
			pattern:       "Chrome/([0-9.]+)",
			expectRE2Safe: true,
			expectType:    "",
		},
		{
			name:          "negative lookbehind",
			pattern:       `(?<!like )Android`,
			expectRE2Safe: false,
			expectType:    "negative_lookbehind",
		},
		{
			name:          "positive lookbehind",
			pattern:       `(?<=Version/)(\d+)`,
			expectRE2Safe: false,
			expectType:    "positive_lookbehind",
		},
		{
			name:          "negative lookahead",
			pattern:       `Chrome(?!Frame)`,
			expectRE2Safe: false,
			expectType:    "negative_lookahead",
		},
		{
			name:          "positive lookahead",
			pattern:       `Safari(?=/)`,
			expectRE2Safe: false,
			expectType:    "positive_lookahead",
		},
		{
			name:          "non-capturing group (ok)",
			pattern:       `(?:Chrome|CriOS)/(\d+)`,
			expectRE2Safe: true,
			expectType:    "",
		},
		{
			name:          "complex but RE2 safe",
			pattern:       `SAMSUNG|Galaxy|SM-[A-Z][0-9]+`,
			expectRE2Safe: true,
			expectType:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isRE2Safe, lookaroundType := CheckRE2Compatibility(tc.pattern)
			if isRE2Safe != tc.expectRE2Safe {
				t.Errorf("RE2 safety: got %v, want %v", isRE2Safe, tc.expectRE2Safe)
			}
			if lookaroundType != tc.expectType {
				t.Errorf("lookaround type: got %q, want %q", lookaroundType, tc.expectType)
			}
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "simple browser pattern",
			pattern:  `Chrome/(\d+[\.\d]+)`,
			expected: []string{"Chrome/"},
		},
		{
			name:     "alternation pattern",
			pattern:  `SAMSUNG|Galaxy|SM-[A-Z]`,
			expected: []string{"SAMSUNG", "Galaxy"},
		},
		{
			name:     "complex device pattern",
			pattern:  `iPhone.*OS\s+(\d+)`,
			expected: []string{"iPhone"},
		},
		{
			name:     "pattern with escaped chars",
			pattern:  `Firefox\/(\d+\.\d+)`,
			expected: []string{"Firefox/"},
		},
		{
			name:     "Huawei pattern",
			pattern:  `HUAWEI|Honor|HW-`,
			expected: []string{"HUAWEI", "Honor"},
		},
		{
			name:     "Opera with version",
			pattern:  `OPR/(\d+[\.\d]+)`,
			expected: []string{"OPR/"},
		},
		{
			name:     "common prefix filtered out",
			pattern:  `Mozilla/5\.0.*Firefox`,
			expected: []string{"Firefox"}, // Mozilla should be filtered as too common
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keywords := ExtractKeywords(tc.pattern)

			// Check that expected keywords are present
			for _, exp := range tc.expected {
				found := false
				for _, kw := range keywords {
					if kw == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected keyword %q not found in %v", exp, keywords)
				}
			}
		})
	}
}

func TestExtractLiterals(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "simple literal",
			pattern:  "Chrome",
			expected: []string{"Chrome"},
		},
		{
			name:     "literal with quantifier",
			pattern:  "Chrome+",
			expected: []string{"Chrome"}, // whole word extracted (quantifier handling is basic)
		},
		{
			name:     "character class skipped",
			pattern:  "SM-[A-Z]123",
			expected: []string{"SM-", "123"},
		},
		{
			name:     "escaped slash",
			pattern:  `Chrome\/`,
			expected: []string{"Chrome/"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			literals := extractLiterals(tc.pattern)

			for _, exp := range tc.expected {
				found := false
				for _, lit := range literals {
					if lit == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected literal %q not found in %v", exp, literals)
				}
			}
		})
	}
}

func TestSplitAlternation(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "simple alternation",
			pattern:  "foo|bar|baz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "nested groups preserved",
			pattern:  "(a|b)|c",
			expected: []string{"(a|b)", "c"},
		},
		{
			name:     "character class preserved",
			pattern:  "[a|b]|c",
			expected: []string{"[a|b]", "c"},
		},
		{
			name:     "no alternation",
			pattern:  "foobar",
			expected: []string{"foobar"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			branches := splitAlternation(tc.pattern)

			if len(branches) != len(tc.expected) {
				t.Errorf("got %d branches, want %d: %v", len(branches), len(tc.expected), branches)
				return
			}

			for i, exp := range tc.expected {
				if branches[i] != exp {
					t.Errorf("branch %d: got %q, want %q", i, branches[i], exp)
				}
			}
		})
	}
}
