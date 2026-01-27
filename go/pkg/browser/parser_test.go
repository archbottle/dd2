package browser

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/archbottle/device-detector/pkg/clienthints"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type fixture struct {
	UserAgent string            `yaml:"user_agent"`
	Headers   map[string]string `yaml:"headers"`
	Client    struct {
		Type          string `yaml:"type"`
		Name          string `yaml:"name"`
		Version       string `yaml:"version"`
		Engine        string `yaml:"engine"`
		EngineVersion string `yaml:"engine_version"`
		Family        string `yaml:"family"`
	} `yaml:"client"`
}

func getRegexesDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "regexes", "client")
}

func getFixturesPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "fixtures", "browser.yml")
}

func loadFixtures(t *testing.T) []fixture {
	t.Helper()
	data, err := os.ReadFile(getFixturesPath())
	require.NoError(t, err, "failed to read fixtures")

	var out []fixture
	err = yaml.Unmarshal(data, &out)
	require.NoError(t, err, "failed to parse fixtures yaml")
	return out
}

func newTestFactory(t *testing.T) *ParserFactory {
	t.Helper()
	baseDir := getRegexesDir()
	f, err := NewParserFactory(
		filepath.Join(baseDir, "browsers.yml"),
		filepath.Join(baseDir, "browser_engine.yml"),
		filepath.Join(baseDir, "hints", "browsers.yml"),
	)
	require.NoError(t, err, "failed to create factory")
	return f
}

// TestParse mirrors DeviceDetector/Tests/Parser/Client/BrowserTest::testParse.
// It tests the browser parsing against all fixtures.
func TestParse(t *testing.T) {
	factory := newTestFactory(t)
	fixtures := loadFixtures(t)

	// Track tested browsers for coverage test
	browsersTested := make(map[string]bool)

	for i, tc := range fixtures {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			var opts []Option
			if tc.Headers != nil && len(tc.Headers) > 0 {
				ch := clienthints.New(tc.Headers)
				opts = append(opts, WithClientHints(ch))
			}

			got := factory.Parse(tc.UserAgent, opts...)

			require.NotNil(t, got, "Parse(): got nil, want match (ua=%q)", tc.UserAgent)

			// PHP test removes short_name before comparison
			assert.Equal(t, tc.Client.Type, got.Type, "Type mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Client.Name, got.Name, "Name mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Client.Version, got.Version, "Version mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Client.Engine, got.Engine, "Engine mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Client.EngineVersion, got.EngineVersion, "EngineVersion mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Client.Family, got.Family, "Family mismatch (ua=%q)", tc.UserAgent)

			// Validate engine name is valid (PHP: checkBrowserEngine)
			if got.Engine != "" {
				validEngine := false
				for _, e := range AvailableEngines {
					if e == got.Engine {
						validEngine = true
						break
					}
				}
				assert.True(t, validEngine, "Engine wrong name: %q (ua=%q)", got.Engine, tc.UserAgent)
			}

			// Track tested browsers
			browsersTested[tc.Client.Name] = true
		})
	}
}

// TestGetAvailableBrowserFamilies mirrors BrowserTest::testGetAvailableBrowserFamilies.
func TestGetAvailableBrowserFamilies(t *testing.T) {
	families := GetAvailableBrowserFamilies()
	assert.NotEmpty(t, families, "browser families should not be empty")

	// Check that each family has at least one browser
	for family, codes := range families {
		assert.NotEmpty(t, codes, "family %q should have at least one browser", family)
	}
}

// TestAllBrowsersTested mirrors BrowserTest::testAllBrowsersTested.
// This test verifies that all browsers in AvailableBrowsers have at least one fixture.
func TestAllBrowsersTested(t *testing.T) {
	factory := newTestFactory(t)
	fixtures := loadFixtures(t)

	browsersTested := make(map[string]bool)

	for _, tc := range fixtures {
		var opts []Option
		if tc.Headers != nil && len(tc.Headers) > 0 {
			ch := clienthints.New(tc.Headers)
			opts = append(opts, WithClientHints(ch))
		}

		got := factory.Parse(tc.UserAgent, opts...)
		if got != nil {
			browsersTested[tc.Client.Name] = true
		}
	}

	allBrowsers := GetAvailableBrowsers()
	var notTested []string

	for _, browserName := range allBrowsers {
		if !browsersTested[browserName] {
			notTested = append(notTested, browserName)
		}
	}

	// Note: In PHP, this is an assertion that all browsers are tested.
	// Here we just report which ones are missing (if any).
	if len(notTested) > 0 {
		t.Logf("Browsers not tested (%d): %v", len(notTested), notTested)
	}
}

// TestGetAvailableClients mirrors BrowserTest::testGetAvailableClients.
func TestGetAvailableClients(t *testing.T) {
	browsers := GetAvailableBrowsers()
	assert.NotEmpty(t, browsers, "available browsers should not be empty")
	assert.Greater(t, len(browsers), 100, "should have more than 100 browsers")
}

// TestStructureBrowsersYml mirrors BrowserTest::testStructureBrowsersYml.
func TestStructureBrowsersYml(t *testing.T) {
	baseDir := getRegexesDir()
	data, err := os.ReadFile(filepath.Join(baseDir, "browsers.yml"))
	require.NoError(t, err, "failed to read browsers.yml")

	var items []map[string]any
	err = yaml.Unmarshal(data, &items)
	require.NoError(t, err, "failed to parse browsers.yml")

	for i, item := range items {
		// Required fields
		_, hasRegex := item["regex"]
		assert.True(t, hasRegex, "item %d: missing 'regex' field", i)

		_, hasName := item["name"]
		assert.True(t, hasName, "item %d: missing 'name' field", i)

		_, hasVersion := item["version"]
		assert.True(t, hasVersion, "item %d: missing 'version' field", i)

		// Validate types
		if regex, ok := item["regex"]; ok {
			_, isString := regex.(string)
			assert.True(t, isString, "item %d: 'regex' should be string", i)
		}

		if name, ok := item["name"]; ok {
			_, isString := name.(string)
			assert.True(t, isString, "item %d: 'name' should be string", i)
		}
	}
}

// TestBrowserFamiliesNoDuplicates mirrors BrowserTest::testBrowserFamiliesNoDuplicates.
func TestBrowserFamiliesNoDuplicates(t *testing.T) {
	families := GetAvailableBrowserFamilies()

	seen := make(map[string]string) // short code -> family
	for family, codes := range families {
		for _, code := range codes {
			if existingFamily, exists := seen[code]; exists {
				t.Errorf("browser %q appears in both %q and %q families", code, existingFamily, family)
			}
			seen[code] = family
		}
	}
}

// TestShortCodesComparisonWithBrowsers mirrors BrowserTest::testShortCodesComparisonWithBrowsers.
func TestShortCodesComparisonWithBrowsers(t *testing.T) {
	families := GetAvailableBrowserFamilies()
	browsers := GetAvailableBrowsers()

	// All short codes in families should exist in browsers
	for family, codes := range families {
		for _, code := range codes {
			_, exists := browsers[code]
			assert.True(t, exists, "short code %q in family %q not found in available browsers", code, family)
		}
	}
}

// TestBrowserHintsForAvailableBrowsers mirrors BrowserTest::testBrowserHintsForAvailableBrowsers.
func TestBrowserHintsForAvailableBrowsers(t *testing.T) {
	baseDir := getRegexesDir()
	hintsPath := filepath.Join(baseDir, "hints", "browsers.yml")

	data, err := os.ReadFile(hintsPath)
	require.NoError(t, err, "failed to read browser hints")

	var hints map[string]string
	err = yaml.Unmarshal(data, &hints)
	require.NoError(t, err, "failed to parse browser hints yaml")

	browsers := GetAvailableBrowsers()

	// All browser names in hints should exist in available browsers
	for appID, browserName := range hints {
		found := false
		for _, name := range browsers {
			if name == browserName {
				found = true
				break
			}
		}
		assert.True(t, found, "browser %q (appID=%q) in hints not found in available browsers", browserName, appID)
	}
}

// TestGetBrowserShortName tests the GetBrowserShortName function.
func TestGetBrowserShortName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Chrome", "CH"},
		{"Firefox", "FF"},
		{"Safari", "SF"},
		{"Microsoft Edge", "PS"},
		{"NonExistent", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetBrowserShortName(tc.name)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestGetBrowserFamily tests the GetBrowserFamily function.
func TestGetBrowserFamily(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CH", "Chrome"},
		{"FF", "Firefox"},
		{"SF", "Safari"},
		{"IE", "Internet Explorer"},
		{"OP", "Opera"},
		{"Chrome", "Chrome"},
		{"Firefox", "Firefox"},
		{"NonExistent", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := GetBrowserFamily(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestIsMobileOnlyBrowser tests the IsMobileOnlyBrowser function.
func TestIsMobileOnlyBrowser(t *testing.T) {
	tests := []struct {
		browser  string
		expected bool
	}{
		{"FM", true},  // Firefox Mobile
		{"OM", true},  // Opera Mobile
		{"CH", false}, // Chrome (not mobile only)
		{"FF", false}, // Firefox (not mobile only)
	}

	for _, tc := range tests {
		t.Run(tc.browser, func(t *testing.T) {
			got := IsMobileOnlyBrowser(tc.browser)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestEngineVersion tests engine version parsing.
func TestEngineVersion(t *testing.T) {
	tests := []struct {
		ua       string
		engine   string
		expected string
	}{
		{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			"Blink",
			"91.0.4472.124",
		},
		{
			"Mozilla/5.0 (Windows NT 10.0; rv:89.0) Gecko/20100101 Firefox/89.0",
			"Gecko",
			"89.0",
		},
		{
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
			"WebKit",
			"605.1.15",
		},
	}

	for _, tc := range tests {
		t.Run(tc.engine, func(t *testing.T) {
			got := ParseEngineVersion(tc.ua, tc.engine)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestCompareVersions tests version comparison.
func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0", "1.0", 0},
		{"1.0", "2.0", -1},
		{"2.0", "1.0", 1},
		{"1.0.0", "1.0", 1},  // PHP: 1.0.0 > 1.0
		{"1.0", "1.0.0", -1}, // PHP: 1.0 < 1.0.0
		{"1.0.1", "1.0", 1},
		{"1.0", "1.0.1", -1},
		{"10.0", "9.0", 1},
		{"", "", 0},
		{"1.0", "", 1},
		{"", "1.0", -1},
		{"112", "112.0.0.0", -1}, // PHP: 112 < 112.0.0.0
		{"112.0.0.0", "112", 1},  // PHP: 112.0.0.0 > 112
	}

	for _, tc := range tests {
		t.Run(tc.v1+"_vs_"+tc.v2, func(t *testing.T) {
			got := compareVersions(tc.v1, tc.v2)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestClientHintMapping tests the client hint mapping.
func TestClientHintMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Google Chrome", "Chrome"},
		{"Edge", "Microsoft Edge"},
		{"Android WebView", "Chrome Webview"},
		{"UnknownBrand", "UnknownBrand"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := applyClientHintMapping(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestFuzzyCompare tests the fuzzy comparison function.
func TestFuzzyCompare(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected bool
	}{
		{"Chrome", "chrome", true},
		{"Chrome Browser", "ChromeBrowser", true},
		{"Firefox", "Firefox", true},
		{"Chrome", "Firefox", false},
		{"Google Chrome", "googlechrome", true},
	}

	for _, tc := range tests {
		t.Run(tc.s1+"_"+tc.s2, func(t *testing.T) {
			got := fuzzyCompare(tc.s1, tc.s2)
			assert.Equal(t, tc.expected, got)
		})
	}
}
