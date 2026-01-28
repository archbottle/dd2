// Package vendorfragment implements the VendorFragment parser.
package vendorfragment

import (
	"fmt"

	"github.com/archbottle/device-detector/pkg/common"
	"github.com/archbottle/device-detector/regexes"
	"gopkg.in/yaml.v3"
)

type vendorGroup struct {
	Brand   string
	Regexes []string
}

// ParserFactory holds pre-compiled regexes and creates Parser instances.
// Thread-safe for concurrent use - create once, use from multiple goroutines.
type ParserFactory struct {
	groups   []vendorGroup
	compiled map[string]common.UniversalRegex // raw regex fragment -> compiled matcher
}

// NewParserFactory creates a factory by loading and compiling vendor fragment regexes from the embedded YAML DB.
func NewParserFactory(opts ...common.FactoryOption) (*ParserFactory, error) {
	data, err := regexes.FS.ReadFile("vendorfragments.yml")
	if err != nil {
		return nil, fmt.Errorf("reading regexes file: %w", err)
	}

	groups, err := parseVendorFragmentsYAMLOrdered(data)
	if err != nil {
		return nil, fmt.Errorf("parsing vendorfragments YAML: %w", err)
	}

	cfg := common.ApplyFactoryOptions(opts)
	compiler := common.NewRegexCompiler(cfg.RegexMode)
	f := &ParserFactory{
		groups:   groups,
		compiled: make(map[string]common.UniversalRegex),
	}

	if err := f.compileAll(compiler, cfg.RegexMode); err != nil {
		return nil, err
	}

	return f, nil
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

// compileAll pre-compiles all vendor fragment patterns.
func (f *ParserFactory) compileAll(compiler *common.RegexCompiler, regexMode common.RegexMode) error {
	for _, g := range f.groups {
		for _, raw := range g.Regexes {
			if _, ok := f.compiled[raw]; ok {
				continue
			}

			// PHP: matchUserAgent($regex . '[^a-z0-9]+')
			// i.e. require the fragment followed by a non-alphanumeric separator.
			// Then wrap with AbstractParser boundary logic.
			wrapped := common.WrapDeviceDetectorPattern(raw + `[^a-z0-9]+`)
			re, err := compiler.Compile(wrapped)
			if err != nil {
				// In Re2Only mode, skip patterns that can't compile
				if regexMode == common.Re2Only {
					continue
				}
				return fmt.Errorf("compiling vendor fragment %q: %w", raw, err)
			}
			f.compiled[raw] = re
		}
	}
	return nil
}

func parseVendorFragmentsYAMLOrdered(data []byte) ([]vendorGroup, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected YAML mapping at document root")
	}

	m := root.Content[0]
	groups := make([]vendorGroup, 0, len(m.Content)/2)
	for i := 0; i < len(m.Content); i += 2 {
		k := m.Content[i]
		v := m.Content[i+1]

		brand := k.Value
		if v.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("expected sequence for vendor %q", brand)
		}

		regexes := make([]string, 0, len(v.Content))
		for _, item := range v.Content {
			regexes = append(regexes, item.Value)
		}

		groups = append(groups, vendorGroup{
			Brand:   brand,
			Regexes: regexes,
		})
	}

	return groups, nil
}
