package console

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/archbottle/device-detector/pkg/common"
	"gopkg.in/yaml.v3"
)

// ParserFactory holds pre-compiled matchers and creates Parser instances.
// Thread-safe for concurrent use - create once, use from multiple goroutines.
type ParserFactory struct {
	entries []*Entry

	// Keyword index for fast candidate lookup.
	index *common.PatternIndex[*Entry]

	// Candidate selection behavior (Compatibility vs StrictIndex).
	mode common.CandidateMode
}

// NewParserFactory loads and compiles the console regex DB from YAML.
func NewParserFactory(regexesPath string, opts ...common.FactoryOption) (*ParserFactory, error) {
	// #nosec G304 -- regexesPath is provided by the library caller.
	data, err := os.ReadFile(regexesPath)
	if err != nil {
		return nil, fmt.Errorf("reading regexes file: %w", err)
	}

	entries, err := parseConsolesYAMLOrdered(data)
	if err != nil {
		return nil, fmt.Errorf("parsing consoles YAML: %w", err)
	}

	cfg := common.ApplyFactoryOptions(opts)
	f := &ParserFactory{
		entries: entries,
		mode:    cfg.CandidateMode,
	}
	f.buildKeywordIndex()

	if err := f.compileAll(); err != nil {
		return nil, err
	}

	return f, nil
}

// NewDefaultParserFactory loads regexes from the canonical repo path.
// Helpful for CLI/tools; tests should generally pass an explicit path.
func NewDefaultParserFactory() (*ParserFactory, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to get caller info")
	}
	regexesPath := filepath.Join(filepath.Dir(filename), "..", "..", "regexes", "device", "consoles.yml")
	return NewParserFactory(regexesPath)
}

// NewParser creates a new Parser instance for parsing a single user agent.
func (f *ParserFactory) NewParser(ua string, opts ...Option) *Parser {
	p := &Parser{
		factory:   f,
		userAgent: ua,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (f *ParserFactory) buildKeywordIndex() {
	f.index = common.NewPatternIndex(f.entries)
}

func (f *ParserFactory) compileAll() error {
	for _, e := range f.entries {
		if e == nil {
			continue
		}

		wrapped := common.WrapDeviceDetectorPattern(e.Regex)
		re, err := common.CompileRegexSubmatch(wrapped)
		if err != nil {
			return fmt.Errorf("compiling brand regex (%s / %q): %w", e.Brand, e.Regex, err)
		}
		e.compiledBrand = re

		e.compiledModels = nil
		if len(e.Models) > 0 {
			e.compiledModels = make([]common.UniversalRegexSubmatch, len(e.Models))
			for i := range e.Models {
				m := e.Models[i]
				w := common.WrapDeviceDetectorPattern(m.Regex)
				mre, err := common.CompileRegexSubmatch(w)
				if err != nil {
					return fmt.Errorf("compiling model regex (%s / %q): %w", e.Brand, m.Regex, err)
				}
				e.compiledModels[i] = mre
			}
		}
	}
	return nil
}

func parseConsolesYAMLOrdered(data []byte) ([]*Entry, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected YAML mapping at document root")
	}

	m := root.Content[0]
	out := make([]*Entry, 0, len(m.Content)/2)

	type rawEntry struct {
		Regex  string  `yaml:"regex"`
		Device string  `yaml:"device"`
		Model  string  `yaml:"model,omitempty"`
		Models []Model `yaml:"models,omitempty"`
	}

	for i := 0; i < len(m.Content); i += 2 {
		k := m.Content[i]
		v := m.Content[i+1]

		brand := k.Value
		var tmp rawEntry
		if err := v.Decode(&tmp); err != nil {
			return nil, fmt.Errorf("decoding brand %q: %w", brand, err)
		}
		if tmp.Regex == "" {
			return nil, fmt.Errorf("missing regex for brand %q", brand)
		}

		out = append(out, &Entry{
			Brand:    brand,
			Regex:    tmp.Regex,
			Device:   tmp.Device,
			Model:    tmp.Model,
			Models:   tmp.Models,
			orderIdx: len(out),
		})
	}

	return out, nil
}
