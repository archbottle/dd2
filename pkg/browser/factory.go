package browser

import (
	"fmt"

	"github.com/archbottle/dd2/pkg/clienthints"
	"github.com/archbottle/dd2/pkg/common"
	"github.com/archbottle/dd2/regexes"
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

	// Engine parser for detecting browser engines.
	engineParser *EngineParser

	// Browser hints for app ID to browser name mapping.
	browserHints *BrowserHints
}

// NewParserFactory creates a factory by loading and compiling regexes from the embedded YAML DB.
func NewParserFactory(opts ...common.FactoryOption) (*ParserFactory, error) {
	// Load browser entries
	data, err := regexes.FS.ReadFile("client/browsers.yml")
	if err != nil {
		return nil, fmt.Errorf("reading browsers regexes file: %w", err)
	}

	var entries []Entry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing browsers regexes YAML: %w", err)
	}

	for i := range entries {
		entries[i].orderIdx = i
	}

	f := &ParserFactory{entries: entries}
	f.patterns = make([]*Entry, len(f.entries))
	for i := range f.entries {
		f.patterns[i] = &f.entries[i]
	}

	db, err := common.NewYAMLListDB(f.patterns, func(e *Entry, compiler *common.RegexCompiler) error {
		if e == nil || e.Regex == "" {
			return nil
		}
		wrapped := common.WrapDeviceDetectorPattern(e.Regex)
		re, compileErr := compiler.CompileSubmatch(wrapped)
		if compileErr != nil {
			return fmt.Errorf("compiling browser pattern (%s): %w", e.Name, compileErr)
		}
		e.compiled = re
		return nil
	}, opts...)
	if err != nil {
		return nil, err
	}
	f.db = db

	// Load engine parser
	engineParser, err := NewEngineParser(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating engine parser: %w", err)
	}
	f.engineParser = engineParser

	// Load browser hints
	browserHints, err := NewBrowserHints()
	if err != nil {
		return nil, fmt.Errorf("creating browser hints: %w", err)
	}
	f.browserHints = browserHints

	return f, nil
}

// NewDefaultParserFactory is an alias for NewParserFactory kept for compatibility.
func NewDefaultParserFactory(opts ...common.FactoryOption) (*ParserFactory, error) {
	return NewParserFactory(opts...)
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

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// WithClientHints sets client hints for the parser.
func WithClientHints(ch *clienthints.ClientHints) Option {
	return func(p *Parser) {
		p.clientHints = ch
	}
}
