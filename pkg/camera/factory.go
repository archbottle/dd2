package camera

import (
	"fmt"
	"sort"

	"github.com/archbottle/dd2/pkg/common"
	"github.com/archbottle/dd2/regexes"
	"gopkg.in/yaml.v3"
)

type modelPatternYAML struct {
	Regex  string `yaml:"regex"`
	Model  string `yaml:"model"`
	Brand  string `yaml:"brand,omitempty"`
	Device string `yaml:"device,omitempty"`
}

type brandPatternYAML struct {
	Regex  string             `yaml:"regex"`
	Device string             `yaml:"device,omitempty"`
	Model  string             `yaml:"model,omitempty"`
	Models []modelPatternYAML `yaml:"models,omitempty"`
}

type modelPattern struct {
	Regex  string
	Model  string
	Brand  string
	Device string

	compiled common.UniversalRegexSubmatch
}

type brandPattern struct {
	Brand  string
	Regex  string
	Device string
	Model  string
	Models []modelPattern

	orderIdx int
	compiled common.UniversalRegexSubmatch
}

// GetRegex is used only for keyword extraction / indexing.
// The index itself is case-insensitive, so we return the raw regex here.
func (p *brandPattern) GetRegex() string { return p.Regex }

// ParserFactory holds pre-compiled camera matchers and creates Parser instances.
// Thread-safe for concurrent use - create once, use from multiple goroutines.
type ParserFactory struct {
	patterns []*brandPattern

	// Keyword index for fast candidate lookup.
	index *common.PatternIndex[*brandPattern]
}

// NewParserFactory creates a factory by loading and compiling camera regexes from the embedded YAML DB.
func NewParserFactory(opts ...common.FactoryOption) (*ParserFactory, error) {
	data, err := regexes.FS.ReadFile("device/cameras.yml")
	if err != nil {
		return nil, fmt.Errorf("reading regexes file: %w", err)
	}

	patterns, err := parseOrderedBrandPatterns(data)
	if err != nil {
		return nil, err
	}

	cfg := common.ApplyFactoryOptions(opts)
	compiler := common.NewRegexCompiler(cfg.RegexMode)
	f := &ParserFactory{patterns: patterns}
	f.buildKeywordIndex()

	if err := f.compileAll(compiler, cfg.RegexMode); err != nil {
		return nil, err
	}

	return f, nil
}

// NewDefaultParserFactory is an alias for NewParserFactory kept for compatibility.
func NewDefaultParserFactory() (*ParserFactory, error) { return NewParserFactory() }

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

func (f *ParserFactory) buildKeywordIndex() {
	if len(f.patterns) == 0 {
		return
	}
	f.index = common.NewPatternIndex(f.patterns)
}

func (f *ParserFactory) candidatesFor(ua string) []*brandPattern {
	cands := f.index.FindCandidates(ua)
	// Preserve YAML order (important for deterministic selection on overlaps).
	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].orderIdx < cands[j].orderIdx
	})
	return cands
}

func (f *ParserFactory) compileAll(compiler *common.RegexCompiler, regexMode common.RegexMode) error {
	for _, bp := range f.patterns {
		if bp == nil {
			continue
		}

		wrapped := common.WrapDeviceDetectorPattern(bp.Regex)
		re, err := compiler.CompileSubmatch(wrapped)
		if err != nil {
			// In Re2Only mode, skip patterns that can't compile
			if regexMode == common.Re2Only {
				continue
			}
			return fmt.Errorf("compiling brand pattern (%s): %w", bp.Brand, err)
		}
		bp.compiled = re

		for i := range bp.Models {
			mp := &bp.Models[i]
			wrappedModel := common.WrapDeviceDetectorPattern(mp.Regex)
			mre, err := compiler.CompileSubmatch(wrappedModel)
			if err != nil {
				// In Re2Only mode, skip model patterns that can't compile
				if regexMode == common.Re2Only {
					continue
				}
				return fmt.Errorf("compiling model pattern (%s / %s): %w", bp.Brand, mp.Regex, err)
			}
			mp.compiled = mre
		}
	}

	return nil
}

func parseOrderedBrandPatterns(data []byte) ([]*brandPattern, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing regexes YAML: %w", err)
	}

	// Unmarshal into a node so we can preserve map order.
	n := &root
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}
	if n.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("unexpected YAML root kind %d, want mapping", n.Kind)
	}

	out := make([]*brandPattern, 0, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i]
		val := n.Content[i+1]

		brand := key.Value
		var y brandPatternYAML
		if err := val.Decode(&y); err != nil {
			return nil, fmt.Errorf("decoding brand %q: %w", brand, err)
		}

		bp := &brandPattern{
			Brand:    brand,
			Regex:    y.Regex,
			Device:   y.Device,
			Model:    y.Model,
			orderIdx: len(out),
		}

		if len(y.Models) != 0 {
			bp.Models = make([]modelPattern, len(y.Models))
			for j := range y.Models {
				bp.Models[j] = modelPattern{
					Regex:  y.Models[j].Regex,
					Model:  y.Models[j].Model,
					Brand:  y.Models[j].Brand,
					Device: y.Models[j].Device,
				}
			}
		}

		out = append(out, bp)
	}

	return out, nil
}
