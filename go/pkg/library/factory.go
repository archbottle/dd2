package library

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/archbottle/device-detector/pkg/common"
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

// NewParserFactory creates a factory by loading and compiling regexes from a YAML file.
func NewParserFactory(regexesPath string, opts ...common.FactoryOption) (*ParserFactory, error) {
	// #nosec G304 -- regexesPath is provided by the library caller.
	data, err := os.ReadFile(regexesPath)
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
	f := &ParserFactory{
		entries: entries,
		mode:    cfg.CandidateMode,
	}
	f.patterns = make([]*Entry, len(f.entries))
	for i := range f.entries {
		f.patterns[i] = &f.entries[i]
	}
	f.buildKeywordIndex()

	if err := f.compileAll(); err != nil {
		return nil, err
	}

	return f, nil
}

// NewDefaultParserFactory creates a factory using the repo-local libraries.yml path.
func NewDefaultParserFactory() (*ParserFactory, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to get caller info")
	}
	regexesPath := filepath.Join(filepath.Dir(filename), "..", "..", "regexes", "client", "libraries.yml")
	return NewParserFactory(regexesPath)
}

func (f *ParserFactory) buildKeywordIndex() {
	if len(f.patterns) == 0 {
		return
	}
	f.index = common.NewPatternIndex(f.patterns)
}

func (f *ParserFactory) compileAll() error {
	for i := range f.entries {
		e := &f.entries[i]
		if e.Regex == "" {
			continue
		}

		wrapped := common.WrapDeviceDetectorPattern(e.Regex)
		re, err := common.CompileRegexSubmatch(wrapped)
		if err != nil {
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
