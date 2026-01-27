package analyzer

import (
	"path/filepath"
	"runtime"
	"testing"
)

func getRegexesDir() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "..", "..", "device-detector", "regexes")
}

func TestAnalyzer_RealYAMLFiles(t *testing.T) {
	regexesDir := getRegexesDir()

	analyzer := NewAnalyzer(regexesDir)
	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Basic sanity checks
	t.Logf("Total patterns analyzed: %d", result.Summary.TotalPatterns)
	t.Logf("RE2-safe patterns: %d (%.1f%%)",
		result.Summary.RE2SafePatterns,
		float64(result.Summary.RE2SafePatterns)/float64(result.Summary.TotalPatterns)*100)
	t.Logf("Patterns needing regexp2: %d (%.1f%%)",
		result.Summary.Regexp2Patterns,
		float64(result.Summary.Regexp2Patterns)/float64(result.Summary.TotalPatterns)*100)
	t.Logf("Indexed keywords: %d", result.Summary.IndexedKeywords)

	// Should have found patterns
	if result.Summary.TotalPatterns == 0 {
		t.Error("No patterns found")
	}

	// Should have keywords
	if result.Summary.IndexedKeywords == 0 {
		t.Error("No keywords indexed")
	}

	// Most patterns should be RE2 safe
	re2Percent := float64(result.Summary.RE2SafePatterns) / float64(result.Summary.TotalPatterns) * 100
	if re2Percent < 80 {
		t.Errorf("Expected >80%% RE2-safe patterns, got %.1f%%", re2Percent)
	}
}

func TestAnalyzer_KeywordIndexQuality(t *testing.T) {
	regexesDir := getRegexesDir()

	analyzer := NewAnalyzer(regexesDir)
	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Check browser index has expected keywords
	browserKeywords := []string{"Chrome/", "Firefox/", "OPR/"}
	for _, kw := range browserKeywords {
		if patterns, ok := result.BrowserIndex.Keywords[kw]; !ok || len(patterns) == 0 {
			t.Errorf("Browser index missing keyword %q", kw)
		} else {
			t.Logf("Keyword %q maps to %d browser patterns: %v", kw, len(patterns), patterns[:min(3, len(patterns))])
		}
	}

	// Check device index has expected keywords
	deviceKeywords := []string{"SAMSUNG", "HUAWEI", "iPhone"}
	for _, kw := range deviceKeywords {
		if patterns, ok := result.DeviceIndex.Keywords[kw]; !ok || len(patterns) == 0 {
			t.Errorf("Device index missing keyword %q", kw)
		} else {
			t.Logf("Keyword %q maps to %d device patterns", kw, len(patterns))
		}
	}

	// Check OS index
	osKeywords := []string{"Android", "Windows", "iPhone"}
	for _, kw := range osKeywords {
		if patterns, ok := result.OSIndex.Keywords[kw]; ok && len(patterns) > 0 {
			t.Logf("OS Keyword %q maps to %d patterns", kw, len(patterns))
		}
	}
}

func TestAnalyzer_Regexp2Patterns(t *testing.T) {
	regexesDir := getRegexesDir()

	analyzer := NewAnalyzer(regexesDir)
	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// List patterns that need regexp2
	t.Log("Patterns requiring regexp2 (has lookaround):")
	count := 0
	for _, p := range result.Patterns {
		if p.HasLookaround {
			count++
			if count <= 10 { // Show first 10
				t.Logf("  [%s] %s: %s", p.Category, p.Name, p.LookaroundType)
				t.Logf("    Regex: %.80s...", p.OriginalRegex)
			}
		}
	}
	t.Logf("Total patterns with lookaround: %d", count)
}

// Demonstrate how the index would be used
func TestDemoKeywordLookup(t *testing.T) {
	regexesDir := getRegexesDir()

	analyzer := NewAnalyzer(regexesDir)
	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Simulate looking up a User-Agent
	testUA := "Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.36 Chrome/91.0.4472.120"

	t.Logf("Test UA: %s", testUA)
	t.Log("")
	t.Log("Keywords found in UA -> Candidate patterns:")

	// Check which keywords match
	candidateBrowsers := make(map[string]bool)
	candidateDevices := make(map[string]bool)

	for kw, patterns := range result.BrowserIndex.Keywords {
		if containsKeyword(testUA, kw) {
			t.Logf("  Browser keyword %q -> %v", kw, patterns)
			for _, p := range patterns {
				candidateBrowsers[p] = true
			}
		}
	}

	for kw, patterns := range result.DeviceIndex.Keywords {
		if containsKeyword(testUA, kw) {
			t.Logf("  Device keyword %q -> %v", kw, patterns)
			for _, p := range patterns {
				candidateDevices[p] = true
			}
		}
	}

	t.Log("")
	t.Logf("Candidate browsers to test: %d (instead of all browser patterns)", len(candidateBrowsers))
	t.Logf("Candidate devices to test: %d (instead of all device patterns)", len(candidateDevices))
}

func containsKeyword(ua, keyword string) bool {
	// Simple contains check - in real implementation might use more sophisticated matching
	for i := 0; i <= len(ua)-len(keyword); i++ {
		if ua[i:i+len(keyword)] == keyword {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
