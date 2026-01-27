package camera

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
	return filepath.Join(filepath.Dir(filename), "..", "..", "regexes", "device", "cameras.yml")
}

func getFixturesPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "fixtures", "camera.yml")
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

// TestParse mirrors DeviceDetector/Tests/Parser/Device/CameraTest.php.
func TestParse(t *testing.T) {
	factory := newTestFactory(t)
	fixtures := loadFixtures(t)

	for i, tc := range fixtures {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			got := factory.Parse(tc.UserAgent)
			assert.NotNil(t, got, "Parse(): got nil, want match (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Device.Type, got.Type, "Type mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Device.Brand, got.Brand, "Brand mismatch (ua=%q)", tc.UserAgent)
			assert.Equal(t, tc.Device.Model, got.Model, "Model mismatch (ua=%q)", tc.UserAgent)
		})
	}
}
