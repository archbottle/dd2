package mobile

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

// NewParserFactory loads and compiles the mobile regex DB from YAML.
// This is a large file (~45K lines, ~2000 brands) - loading takes a few seconds.
func NewParserFactory(regexesPath string, opts ...common.FactoryOption) (*ParserFactory, error) {
	// #nosec G304 -- regexesPath is provided by the library caller.
	data, err := os.ReadFile(regexesPath)
	if err != nil {
		return nil, fmt.Errorf("reading regexes file: %w", err)
	}

	entries, err := parseMobilesYAMLOrdered(data)
	if err != nil {
		return nil, fmt.Errorf("parsing mobiles YAML: %w", err)
	}

	cfg := common.ApplyFactoryOptions(opts)
	compiler := common.NewRegexCompiler(cfg.RegexMode)
	f := &ParserFactory{
		entries: entries,
		mode:    cfg.CandidateMode,
	}
	f.buildKeywordIndex()

	if err := f.compileAll(compiler, cfg.RegexMode); err != nil {
		return nil, err
	}

	return f, nil
}

// NewDefaultParserFactory loads regexes from the canonical repo path.
func NewDefaultParserFactory() (*ParserFactory, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to get caller info")
	}
	regexesPath := filepath.Join(filepath.Dir(filename), "..", "..", "regexes", "device", "mobiles.yml")
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

// Parse is a convenience method that creates a parser and immediately parses.
func (f *ParserFactory) Parse(ua string, opts ...Option) *Match {
	return f.NewParser(ua, opts...).Parse()
}

// Stats returns index statistics for debugging/monitoring.
func (f *ParserFactory) Stats() common.IndexStats {
	if f.index == nil {
		return common.IndexStats{TotalPatterns: len(f.entries)}
	}
	return f.index.Stats()
}

func (f *ParserFactory) buildKeywordIndex() {
	f.index = common.NewPatternIndex(f.entries)
}

func (f *ParserFactory) compileAll(compiler *common.RegexCompiler, regexMode common.RegexMode) error {
	for _, e := range f.entries {
		if e == nil {
			continue
		}

		wrapped := common.WrapDeviceDetectorPattern(e.Regex)
		re, err := compiler.CompileSubmatch(wrapped)
		if err != nil {
			// In Re2Only mode, skip patterns that can't compile
			if regexMode == common.Re2Only {
				continue
			}
			return fmt.Errorf("compiling brand regex (%s / %q): %w", e.Brand, e.Regex, err)
		}
		e.compiledBrand = re

		e.compiledModels = nil
		if len(e.Models) > 0 {
			e.compiledModels = make([]common.UniversalRegexSubmatch, len(e.Models))
			for i := range e.Models {
				m := e.Models[i]
				w := common.WrapDeviceDetectorPattern(m.Regex)
				mre, err := compiler.CompileSubmatch(w)
				if err != nil {
					// In Re2Only mode, skip model patterns that can't compile
					if regexMode == common.Re2Only {
						continue
					}
					return fmt.Errorf("compiling model regex (%s / %q): %w", e.Brand, m.Regex, err)
				}
				e.compiledModels[i] = mre
			}
		}
	}
	return nil
}

// parseMobilesYAMLOrdered parses the mobiles.yml file preserving key order.
// The YAML structure is a mapping of brand names to their patterns.
func parseMobilesYAMLOrdered(data []byte) ([]*Entry, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected YAML mapping at document root")
	}

	m := root.Content[0]
	out := make([]*Entry, 0, len(m.Content)/2)

	type rawModel struct {
		Regex  string `yaml:"regex"`
		Model  string `yaml:"model"`
		Brand  string `yaml:"brand,omitempty"`
		Device string `yaml:"device,omitempty"`
	}

	type rawEntry struct {
		Regex  string     `yaml:"regex"`
		Device string     `yaml:"device"`
		Model  string     `yaml:"model,omitempty"`
		Models []rawModel `yaml:"models,omitempty"`
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

		entry := &Entry{
			Brand:    brand,
			Regex:    tmp.Regex,
			Device:   tmp.Device,
			Model:    tmp.Model,
			orderIdx: len(out),
		}

		if len(tmp.Models) > 0 {
			entry.Models = make([]Model, len(tmp.Models))
			for j, rm := range tmp.Models {
				entry.Models[j] = Model{
					Regex:  rm.Regex,
					Model:  rm.Model,
					Brand:  rm.Brand,
					Device: rm.Device,
				}
			}
		}

		out = append(out, entry)
	}

	return out, nil
}
