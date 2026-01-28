package library

import (
	"fmt"

	"github.com/archbottle/device-detector/pkg/common"
	"github.com/archbottle/device-detector/regexes"
	"gopkg.in/yaml.v3"
)

// ParserFactory holds pre-compiled regexes and creates Parser instances.
// Thread-safe for concurrent use - create once, use from multiple goroutines.
type ParserFactory struct {
	entries []Entry

	// Pointer view used by the keyword index and candidate selection.
	patterns []*Entry

	// Keyword index for fast candidate lookup.
	index *common.PatternIndex[*Entry]

	// Candidate selection behavior (Compatibility vs StrictIndex).
	mode common.CandidateMode
}

// NewParserFactory creates a factory by loading and compiling regexes from the embedded YAML DB.
func NewParserFactory(opts ...common.FactoryOption) (*ParserFactory, error) {
	data, err := regexes.FS.ReadFile("client/libraries.yml")
	if err != nil {
		return nil, fmt.Errorf("reading regexes file: %w", err)
	}

	var entries []Entry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing regexes YAML: %w", err)
	}
	for i := range entries {
		entries[i].orderIdx = i
	}

	cfg := common.ApplyFactoryOptions(opts)
	compiler := common.NewRegexCompiler(cfg.RegexMode)
	f := &ParserFactory{
		entries: entries,
		mode:    cfg.CandidateMode,
	}
	f.patterns = make([]*Entry, len(f.entries))
	for i := range f.entries {
		f.patterns[i] = &f.entries[i]
	}
	f.buildKeywordIndex()

	if err := f.compileAll(compiler, cfg.RegexMode); err != nil {
		return nil, err
	}

	return f, nil
}

// NewDefaultParserFactory is an alias for NewParserFactory kept for compatibility.
func NewDefaultParserFactory() (*ParserFactory, error) { return NewParserFactory() }

func (f *ParserFactory) buildKeywordIndex() {
	if len(f.patterns) == 0 {
		return
	}
	f.index = common.NewPatternIndex(f.patterns)
}

func (f *ParserFactory) compileAll(compiler *common.RegexCompiler, regexMode common.RegexMode) error {
	for i := range f.entries {
		e := &f.entries[i]
		if e.Regex == "" {
			continue
		}

		wrapped := common.WrapDeviceDetectorPattern(e.Regex)
		re, err := compiler.CompileSubmatch(wrapped)
		if err != nil {
			// In Re2Only mode, skip patterns that can't compile
			if regexMode == common.Re2Only {
				continue
			}
			return fmt.Errorf("compiling library pattern (%s): %w", e.Name, err)
		}
		e.compiled = re
	}
	return nil
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
