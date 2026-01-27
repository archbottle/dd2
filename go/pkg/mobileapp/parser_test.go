package mobileapp

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type fixture struct {
	UserAgent string `yaml:"user_agent"`
	Client    struct {
		Type    string `yaml:"type"`
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"client"`
}

func getRegexesPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "regexes", "client", "mobile_apps.yml")
}

func getFixturesPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "fixtures", "mobile_app.yml")
}

func loadFixtures(t *testing.T) []fixture {
	t.Helper()
	data, err := os.ReadFile(getFixturesPath())
	if err != nil {
		t.Fatalf("failed to read fixtures: %v", err)
	}
	var out []fixture
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

// TestParse mirrors DeviceDetector/Tests/Parser/Client/MobileAppTest.php.
func TestParse(t *testing.T) {
	factory := newTestFactory(t)
	fixtures := loadFixtures(t)

	for i, tc := range fixtures {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			got := factory.Parse(tc.UserAgent)
			assert.NotNil(t, got, "Parse(): got nil, want match (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Client.Type, got.Type, "Type mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Client.Name, got.Name, "Name mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Client.Version, got.Version, "Version mismatch (ua=%q)", tc.UserAgent)
		})
	}
}

// TestStructureMobileAppYml mirrors MobileAppTest::testStructureMobileAppYml.
func TestStructureMobileAppYml(t *testing.T) {
	data, err := os.ReadFile(getRegexesPath())
	assert.NoError(t, err, "failed to read regexes")

	var items []map[string]any
	err = yaml.Unmarshal(data, &items)
	assert.NoError(t, err, "failed to parse regexes yaml")

	for _, item := range items {
		for _, k := range []string{"regex", "name", "version"} {
			v, ok := item[k]
			assert.True(t, ok, "key %q not exist", k)
			_, ok = v.(string)
			assert.True(t, ok, "key %q: got %T, want string", k, v)
		}
	}
}

