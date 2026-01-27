package bots

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// getRegexesPath returns the path to the regexes directory.
func getRegexesPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "regexes", "bots.yml")
}

// getFixturesPath returns the path to the fixtures file.
func getFixturesPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "fixtures", "bots.yml")
}

// newTestFactory creates a factory for testing.
func newTestFactory(t *testing.T) *ParserFactory {
	t.Helper()
	factory, err := NewParserFactory(getRegexesPath())
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}
	return factory
}

// botFixture matches the PHP bots.yml fixture format
type botFixture struct {
	UserAgent string `yaml:"user_agent"`
	Bot       struct {
		Name     string `yaml:"name"`
		Category string `yaml:"category,omitempty"`
		URL      string `yaml:"url,omitempty"`
		Producer *struct {
			Name string `yaml:"name"`
			URL  string `yaml:"url"`
		} `yaml:"producer,omitempty"`
	} `yaml:"bot"`
}

func loadBotFixtures(t *testing.T) []botFixture {
	t.Helper()
	data, err := os.ReadFile(getFixturesPath())
	require.NoError(t, err, "failed to read fixtures")

	var out []botFixture
	err = yaml.Unmarshal(data, &out)
	require.NoError(t, err, "failed to parse fixtures yaml")
	return out
}

// Valid bot categories from PHP
var validBotCategories = []string{
	"Benchmark", "Crawler", "Feed Fetcher", "Feed Parser", "Feed Reader",
	"Network Monitor", "Read-it-later Service", "Search bot", "Search tools",
	"Security Checker", "Security search bot", "Service Agent", "Service bot",
	"Site Monitor", "Social Media Agent", "Validator", "AI Agent",
	"AI Assistant", "AI Data Scraper", "AI Search Crawler",
}

// TestParseBots mirrors DeviceDetectorTest::testParseBots
// Tests bot parsing against all fixtures from bots.yml
func TestParseBots(t *testing.T) {
	factory := newTestFactory(t)
	fixtures := loadBotFixtures(t)

	for i, tc := range fixtures {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			got := factory.Parse(tc.UserAgent)

			// Must be detected as bot
			require.NotNil(t, got, "expected bot to be detected (ua=%q)", tc.UserAgent)

			// Check name
			assert.Equal(t, tc.Bot.Name, got.Name, "Name mismatch (ua=%q)", tc.UserAgent)

			// Check category
			assert.Equal(t, tc.Bot.Category, got.Category, "Category mismatch (ua=%q)", tc.UserAgent)

			// Check URL
			assert.Equal(t, tc.Bot.URL, got.URL, "URL mismatch (ua=%q)", tc.UserAgent)

			// Check producer
			if tc.Bot.Producer != nil {
				require.NotNil(t, got.Producer, "Producer should not be nil (ua=%q)", tc.UserAgent)
				assert.Equal(t, tc.Bot.Producer.Name, got.Producer.Name, "Producer.Name mismatch (ua=%q)", tc.UserAgent)
				assert.Equal(t, tc.Bot.Producer.URL, got.Producer.URL, "Producer.URL mismatch (ua=%q)", tc.UserAgent)
			} else {
				assert.Nil(t, got.Producer, "Producer should be nil (ua=%q)", tc.UserAgent)
			}

			// Validate category if present (PHP test does this)
			if got.Category != "" {
				valid := false
				for _, c := range validBotCategories {
					if c == got.Category {
						valid = true
						break
					}
				}
				assert.True(t, valid, "Unknown category: %q (ua=%q)", got.Category, tc.UserAgent)
			}
		})
	}
}

// TestGetInfoFromUABot matches PHP BotTest::testGetInfoFromUABot
// Tests that a known bot UA returns full bot information.
func TestGetInfoFromUABot(t *testing.T) {
	factory := newTestFactory(t)

	result := factory.Parse("Googlebot/2.1 (http://www.googlebot.com/bot.html)")

	assert.NotNil(t, result, "expected bot to be detected, got nil")

	// Expected values from PHP test
	expected := BotMatch{
		Name:     "Googlebot",
		Category: "Search bot",
		URL:      "https://developers.google.com/search/docs/crawling-indexing/overview-google-crawlers",
		Producer: &Producer{
			Name: "Google Inc.",
			URL:  "https://www.google.com/",
		},
	}

	assert.Equal(t, expected.Name, result.Name, "Name")
	assert.Equal(t, expected.Category, result.Category, "Category")
	assert.Equal(t, expected.URL, result.URL, "URL")
	assert.NotNil(t, result.Producer, "Producer")
	assert.Equal(t, expected.Producer.Name, result.Producer.Name, "Producer.Name")
	assert.Equal(t, expected.Producer.URL, result.Producer.URL, "Producer.URL")
}

// TestParseNoDetails matches PHP BotTest::testParseNoDetails
// Tests that with discardDetails, Parse returns minimal result.
func TestParseNoDetails(t *testing.T) {
	factory := newTestFactory(t)

	result := factory.Parse(
		"Googlebot/2.1 (http://www.googlebot.com/bot.html)",
		WithDiscardDetails(),
	)

	assert.NotNil(t, result, "expected bot to be detected, got nil")

	// In PHP, discardDetails returns [true]
	// In Go, we return a BotMatch with Name="true" as indicator
	assert.Equal(t, "true", result.Name, "Name")

	// Other fields should be empty when details are discarded
	assert.Empty(t, result.Category, "Category")
	assert.Empty(t, result.URL, "URL")
	assert.Nil(t, result.Producer, "Producer")
}

// TestParseNoBot matches PHP BotTest::testParseNoBot
// Tests that a non-bot UA returns nil.
func TestParseNoBot(t *testing.T) {
	factory := newTestFactory(t)

	result := factory.Parse("Mozilla/4.0 (compatible; MSIE 9.0; Windows NT 6.1; SV1; SE 2.x)")

	assert.Nil(t, result, "expected nil for non-bot UA")
}

// TestIsBot tests the convenience IsBot() method.
func TestIsBot(t *testing.T) {
	factory := newTestFactory(t)

	tests := []struct {
		name      string
		userAgent string
		isBot     bool
	}{
		{
			name:      "Googlebot",
			userAgent: "Googlebot/2.1 (http://www.googlebot.com/bot.html)",
			isBot:     true,
		},
		{
			name:      "Regular browser",
			userAgent: "Mozilla/4.0 (compatible; MSIE 9.0; Windows NT 6.1; SV1; SE 2.x)",
			isBot:     false,
		},
		{
			name:      "Empty UA",
			userAgent: "",
			isBot:     false,
		},
		{
			name:      "AhrefsBot",
			userAgent: "Mozilla/5.0 (compatible; AhrefsBot/7.0; +http://ahrefs.com/robot/)",
			isBot:     true,
		},
		{
			name: "Chrome browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
				"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			isBot: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := factory.IsBot(tt.userAgent)
			assert.Equal(t, tt.isBot, got, "IsBot()")
		})
	}
}

// TestParserReuse tests that the factory can be reused for multiple parsers.
func TestParserReuse(t *testing.T) {
	factory := newTestFactory(t)

	// Create multiple parsers from the same factory
	parser1 := factory.NewParser("Googlebot/2.1 (http://www.googlebot.com/bot.html)")
	parser2 := factory.NewParser("Mozilla/5.0 Chrome/91.0")
	parser3 := factory.NewParser("AhrefsBot/7.0")

	assert.True(t, parser1.IsBot(), "parser1 should detect Googlebot")
	assert.False(t, parser2.IsBot(), "parser2 should not detect Chrome as bot")
	assert.True(t, parser3.IsBot(), "parser3 should detect AhrefsBot")
}

// TestParserWithOptions tests creating parser with options.
func TestParserWithOptions(t *testing.T) {
	factory := newTestFactory(t)

	// Parser with discardDetails option
	parser := factory.NewParser(
		"Googlebot/2.1 (http://www.googlebot.com/bot.html)",
		WithDiscardDetails(),
	)

	result := parser.Parse()
	assert.NotNil(t, result, "expected bot match")
	assert.Equal(t, "true", result.Name, "expected 'true' for discarded details")
}

// TestFactoryConvenienceMethods tests factory's convenience methods.
func TestFactoryConvenienceMethods(t *testing.T) {
	factory := newTestFactory(t)

	// Test Parse convenience method
	result := factory.Parse("Googlebot/2.1")
	assert.NotNil(t, result, "Parse() should detect Googlebot")
	assert.Equal(t, "Googlebot", result.Name, "expected Googlebot")

	// Test IsBot convenience method
	assert.True(t, factory.IsBot("Googlebot/2.1"), "IsBot() should return true for Googlebot")
	assert.False(t, factory.IsBot("Mozilla/5.0 Chrome/91.0"), "IsBot() should return false for Chrome")
}

// BenchmarkParse benchmarks parsing performance.
func BenchmarkParse(b *testing.B) {
	factory, err := NewParserFactory(getRegexesPath())
	if err != nil {
		b.Fatalf("failed to create factory: %v", err)
	}

	userAgents := []string{
		"Googlebot/2.1 (http://www.googlebot.com/bot.html)",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0.4472.124",
		"Mozilla/5.0 (compatible; AhrefsBot/7.0; +http://ahrefs.com/robot/)",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ua := userAgents[i%len(userAgents)]
		_ = factory.Parse(ua)
	}
}

// BenchmarkIsBot benchmarks IsBot performance.
func BenchmarkIsBot(b *testing.B) {
	factory, err := NewParserFactory(getRegexesPath())
	if err != nil {
		b.Fatalf("failed to create factory: %v", err)
	}

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0.4472.124"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = factory.IsBot(ua)
	}
}
