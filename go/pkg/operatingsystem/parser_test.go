package operatingsystem

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
	OS        struct {
		Name      string `yaml:"name"`
		ShortName string `yaml:"short_name"`
		Version   string `yaml:"version"`
		Platform  string `yaml:"platform"`
		Family    string `yaml:"family"`
	} `yaml:"os"`
}

func getRegexesDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "regexes")
}

func getFixturesPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "fixtures", "oss.yml")
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
	f, err := NewParserFactory(filepath.Join(baseDir, "oss.yml"))
	require.NoError(t, err, "failed to create factory")
	return f
}

// osTested tracks which OS names have been tested
var osTested = make(map[string]bool)

// TestParse mirrors DeviceDetector/Tests/Parser/OperatingSystemTest::testParse.
// It tests the OS parsing against all fixtures.
func TestParse(t *testing.T) {
	factory := newTestFactory(t)
	fixtures := loadFixtures(t)

	for i, tc := range fixtures {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			var opts []Option
			if tc.Headers != nil && len(tc.Headers) > 0 {
				ch := clienthints.New(tc.Headers)
				opts = append(opts, WithClientHints(ch))
			}

			got := factory.Parse(tc.UserAgent, opts...)

			// Handle empty OS case
			if tc.OS.Name == "" {
				assert.Nil(t, got, "Parse(): expected nil for empty OS (ua=%q)", tc.UserAgent)
				return
			}

			require.NotNil(t, got, "Parse(): got nil, want match (ua=%q)", tc.UserAgent)

			assert.Equal(t, tc.OS.Name, got.Name, "Name mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.OS.ShortName, got.ShortName, "ShortName mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.OS.Version, got.Version, "Version mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.OS.Platform, got.Platform, "Platform mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.OS.Family, got.Family, "Family mismatch (ua=%q)", tc.UserAgent)

			// Track tested OS
			osTested[tc.OS.Name] = true
		})
	}
}

// TestOSInGroup mirrors OperatingSystemTest::testOSInGroup.
// Verifies that every OS belongs to a family group.
func TestOSInGroup(t *testing.T) {
	// Flatten all family members into a single list
	var familyOs []string
	for _, codes := range OSFamilies {
		familyOs = append(familyOs, codes...)
	}

	// Every OS short code should be in some family
	for osShort := range OperatingSystems {
		found := false
		for _, code := range familyOs {
			if code == osShort {
				found = true
				break
			}
		}
		assert.True(t, found, "OS %q (short: %q) not in any family", OperatingSystems[osShort], osShort)
	}
}

// TestFamilyOSExists mirrors OperatingSystemTest::testFamilyOSExists.
// Verifies that every family member exists in the OS list.
func TestFamilyOSExists(t *testing.T) {
	// Flatten all family members into a single list
	var familyOs []string
	for _, codes := range OSFamilies {
		familyOs = append(familyOs, codes...)
	}

	// Every family member should exist in OperatingSystems
	for _, osShort := range familyOs {
		_, exists := OperatingSystems[osShort]
		assert.True(t, exists, "Family member %q not found in OperatingSystems", osShort)
	}
}

// TestGetAvailableOperatingSystems mirrors OperatingSystemTest::testGetAvailableOperatingSystems.
func TestGetAvailableOperatingSystems(t *testing.T) {
	oss := GetAvailableOperatingSystems()
	assert.Greater(t, len(oss), 70, "should have more than 70 operating systems")
}

// TestGetNameFromId mirrors OperatingSystemTest::testGetNameFromId.
func TestGetNameFromId(t *testing.T) {
	tests := []struct {
		os       string
		version  string
		expected string
	}{
		{"DEB", "4.5", "Debian 4.5"},
		{"WRT", "", "Windows RT"},
		{"WIN", "98", "Windows 98"},
		{"XXX", "4.5", ""},
	}

	for _, tc := range tests {
		name := tc.os + "_" + tc.version
		t.Run(name, func(t *testing.T) {
			got := GetNameFromId(tc.os, tc.version)
			if tc.expected == "" {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tc.expected, got)
			}
		})
	}
}

// TestAllOperatingSystemsTested mirrors OperatingSystemTest::testAllOperatingSystemsTested.
// This test should run after TestParse to verify all OS have fixtures.
func TestAllOperatingSystemsTested(t *testing.T) {
	// Need to run TestParse first to populate osTested
	if len(osTested) == 0 {
		factory := newTestFactory(t)
		fixtures := loadFixtures(t)

		for _, tc := range fixtures {
			var opts []Option
			if tc.Headers != nil && len(tc.Headers) > 0 {
				ch := clienthints.New(tc.Headers)
				opts = append(opts, WithClientHints(ch))
			}

			got := factory.Parse(tc.UserAgent, opts...)
			if got != nil {
				osTested[tc.OS.Name] = true
			}
		}
	}

	allOS := GetAvailableOperatingSystems()
	var notTested []string

	for _, osName := range allOS {
		if !osTested[osName] {
			notTested = append(notTested, osName)
		}
	}

	assert.Empty(t, notTested, "Following OSs are not tested: %v", notTested)
}

// TestGetOsFamily tests the GetOsFamily function.
func TestGetOsFamily(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AND", "Android"},
		{"IOS", "iOS"},
		{"WIN", "Windows"},
		{"MAC", "Mac"},
		{"LIN", "GNU/Linux"},
		{"Android", "Android"},
		{"iOS", "iOS"},
		{"Windows", "Windows"},
		{"NonExistent", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := GetOsFamily(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestIsDesktopOs tests the IsDesktopOs function.
func TestIsDesktopOs(t *testing.T) {
	tests := []struct {
		os       string
		expected bool
	}{
		{"WIN", true},  // Windows
		{"MAC", true},  // Mac
		{"LIN", true},  // GNU/Linux
		{"AND", false}, // Android
		{"IOS", false}, // iOS
		{"FOS", false}, // Firefox OS
		{"Windows", true},
		{"Mac", true},
		{"Android", false},
	}

	for _, tc := range tests {
		t.Run(tc.os, func(t *testing.T) {
			got := IsDesktopOs(tc.os)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestOSFamiliesNoDuplicates verifies no OS appears in multiple families.
func TestOSFamiliesNoDuplicates(t *testing.T) {
	seen := make(map[string]string) // short code -> family
	for family, codes := range OSFamilies {
		for _, code := range codes {
			if existingFamily, exists := seen[code]; exists {
				t.Errorf("OS %q appears in both %q and %q families", code, existingFamily, family)
			}
			seen[code] = family
		}
	}
}

// TestParsePlatform tests platform detection from user agents.
func TestParsePlatform(t *testing.T) {
	factory := newTestFactory(t)

	tests := []struct {
		name     string
		ua       string
		expected string
	}{
		{
			"x64 from Win64",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"x64",
		},
		{
			"x86 from i686",
			"Mozilla/5.0 (X11; Linux i686) AppleWebKit/537.36",
			"x86",
		},
		{
			"x64 from x86_64",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
			"x64",
		},
		{
			"ARM from arm",
			"Mozilla/5.0 (Linux; Android 10; arm) AppleWebKit/537.36",
			"ARM",
		},
		{
			"ARM from aarch64",
			"Mozilla/5.0 (Linux; Android 10; aarch64) AppleWebKit/537.36",
			"ARM",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := factory.NewParser(tc.ua)
			got := p.parsePlatform()
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestClientHintOSDetection tests OS detection from client hints.
func TestClientHintOSDetection(t *testing.T) {
	factory := newTestFactory(t)

	tests := []struct {
		name     string
		ua       string
		headers  map[string]string
		expected string
	}{
		{
			"Windows from client hints",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			map[string]string{
				"Sec-CH-UA-Platform":         "\"Windows\"",
				"Sec-CH-UA-Platform-Version": "\"15.0.0\"",
			},
			"Windows",
		},
		{
			"Android from client hints",
			"Mozilla/5.0 (Linux; Android 10)",
			map[string]string{
				"Sec-CH-UA-Platform":         "\"Android\"",
				"Sec-CH-UA-Platform-Version": "\"10\"",
			},
			"Android",
		},
		{
			"Linux mapped to GNU/Linux",
			"Mozilla/5.0 (X11; Linux x86_64)",
			map[string]string{
				"Sec-CH-UA-Platform": "\"Linux\"",
			},
			"GNU/Linux",
		},
		{
			"MacOS mapped to Mac",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			map[string]string{
				"Sec-CH-UA-Platform": "\"macOS\"",
			},
			"Mac",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ch := clienthints.New(tc.headers)
			got := factory.Parse(tc.ua, WithClientHints(ch))

			require.NotNil(t, got, "Parse() returned nil")
			assert.Equal(t, tc.expected, got.Name)
		})
	}
}

// TestVersionBuilding tests version template processing.
func TestVersionBuilding(t *testing.T) {
	tests := []struct {
		template string
		matches  []string
		expected string
	}{
		{"$1", []string{"", "10"}, "10"},
		{"$1.$2", []string{"", "10", "15"}, "10.15"},
		{"", []string{}, ""},
		{"1.0", []string{}, "1.0"},
		{"$1_$2", []string{"", "10", "15"}, "10.15"}, // underscores become dots
	}

	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := buildVersion(tc.template, tc.matches)
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
		{"Windows", "windows", true},
		{"Mac OS", "MacOS", true},
		{"GNU/Linux", "GNU/Linux", true}, // exact match (/ is not normalized)
		{"Android", "iOS", false},
	}

	for _, tc := range tests {
		t.Run(tc.s1+"_"+tc.s2, func(t *testing.T) {
			got := fuzzyCompare(tc.s1, tc.s2)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestShortOsData tests the getShortOsData function.
func TestShortOsData(t *testing.T) {
	tests := []struct {
		name          string
		expectedName  string
		expectedShort string
	}{
		{"Android", "Android", "AND"},
		{"Windows", "Windows", "WIN"},
		{"iOS", "iOS", "IOS"},
		{"Mac", "Mac", "MAC"},
		{"GNU/Linux", "GNU/Linux", "LIN"},
		{"Unknown OS", "Unknown OS", "UNK"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getShortOsData(tc.name)
			assert.Equal(t, tc.expectedName, got.Name)
			assert.Equal(t, tc.expectedShort, got.Short)
		})
	}
}
