package console

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/archbottle/device-detector/pkg/common"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type fixtureCase struct {
	UserAgent string `yaml:"user_agent"`
	Device    struct {
		Type  string `yaml:"type"`
		Brand string `yaml:"brand"`
		Model string `yaml:"model"`
	} `yaml:"device"`
}

func getRegexesPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "regexes", "device", "consoles.yml")
}

func getFixturesPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "fixtures", "console.yml")
}

func loadFixtures(t *testing.T) []fixtureCase {
	t.Helper()
	data, err := os.ReadFile(getFixturesPath())
	if err != nil {
		t.Fatalf("failed to read fixtures: %v", err)
	}
	var out []fixtureCase
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatalf("failed to parse fixtures yaml: %v", err)
	}
	return out
}

func newTestFactory(t *testing.T) *ParserFactory {
	t.Helper()
	f, err := NewParserFactory(getRegexesPath())
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}
	return f
}

// TestParse mirrors the PHP ConsoleTest fixture assertions.
func TestParse(t *testing.T) {
	factory := newTestFactory(t)
	fixtures := loadFixtures(t)

	for i, tc := range fixtures {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			p := factory.NewParser(tc.UserAgent)
			got := p.Parse()
			if got == nil {
				candidates := common.SelectCandidates(factory.entries, factory.index, tc.UserAgent, factory.mode)
				brands := make([]string, 0, len(candidates))
				for _, c := range candidates {
					if c != nil {
						brands = append(brands, c.Brand)
					}
				}
				t.Fatalf("Parse(): got nil, want match (mode=%v candidates=%v ua=%q)", factory.mode, brands, tc.UserAgent)
			}

			assert.Equal(t, tc.Device.Type, got.Type, "Type mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Device.Brand, got.Brand, "Brand mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Device.Model, got.Model, "Model mismatch (ua=%q)", tc.UserAgent)
		})
	}
}
