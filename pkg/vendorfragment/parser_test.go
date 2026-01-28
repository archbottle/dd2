package vendorfragment

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type fixture struct {
	UserAgent string `yaml:"useragent"`
	Vendor    string `yaml:"vendor"`
}


func getFixturesPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "fixtures", "vendorfragments.yml")
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
	f, err := NewParserFactory()
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}
	return f
}

// TestParse mirrors VendorFragmentTest::testParse (data provider).
func TestParse(t *testing.T) {
	factory := newTestFactory(t)
	fixtures := loadFixtures(t)

	for i, tc := range fixtures {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			p := factory.NewParser(tc.UserAgent)
			got := p.Parse()

			want := map[string]string{"brand": tc.Vendor}
			assert.NotNil(t, got, "Parse(): got nil (ua=%q)", tc.UserAgent)
			assert.Equal(t, want["brand"], got["brand"], "Parse(): brand mismatch (ua=%q)", tc.UserAgent)
			assert.NotEmpty(t, p.MatchedRegex(), "MatchedRegex(): got empty (ua=%q)", tc.UserAgent)
		})
	}
}

// TestAllRegexesTested mirrors VendorFragmentTest::testAllRegexesTested.
func TestAllRegexesTested(t *testing.T) {
	factory := newTestFactory(t)
	fixtures := loadFixtures(t)

	tested := map[string]bool{}
	for _, tc := range fixtures {
		p := factory.NewParser(tc.UserAgent)
		_ = p.Parse()
		tested[p.MatchedRegex()] = true
	}

	missing := []string{}
	for _, g := range factory.groups {
		for _, raw := range g.Regexes {
			if tested[raw] {
				continue
			}
			missing = append(missing, g.Brand+" / "+raw)
		}
	}

	assert.Empty(t, missing, "Following vendor fragments are not tested: %s", strings.Join(missing, ", "))
}
