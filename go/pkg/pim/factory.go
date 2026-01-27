package pim

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

	// Pointer view used by shared list DB/index.
	patterns []*Entry

	// Shared list DB (index + mode).
	db *common.YAMLListDB[*Entry]
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

	f := &ParserFactory{entries: entries}
	f.patterns = make([]*Entry, len(f.entries))
	for i := range f.entries {
		f.patterns[i] = &f.entries[i]
	}

	db, err := common.NewYAMLListDB(f.patterns, func(e *Entry) error {
		if e == nil || e.Regex == "" {
			return nil
		}
		wrapped := common.WrapDeviceDetectorPattern(e.Regex)
		re, err := common.CompileRegexSubmatch(wrapped)
		if err != nil {
			return fmt.Errorf("compiling pim pattern (%s): %w", e.Name, err)
		}
		e.compiled = re
		return nil
	}, opts...)
	if err != nil {
		return nil, err
	}
	f.db = db

	return f, nil
}

// NewDefaultParserFactory creates a factory using the repo-local pim.yml path.
func NewDefaultParserFactory() (*ParserFactory, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to get caller info")
	}
	regexesPath := filepath.Join(filepath.Dir(filename), "..", "..", "regexes", "client", "pim.yml")
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
