package bots

import (
	"fmt"
	"strings"

	"github.com/archbottle/dd2/pkg/common"
	"github.com/archbottle/dd2/regexes"
	"gopkg.in/yaml.v3"
)

// ParserFactory holds pre-compiled regexes and creates Parser instances.
// Thread-safe for concurrent use - create once, use from multiple goroutines.
type ParserFactory struct {
	regexes []BotEntry

	// Keyword index for fast candidate lookup
	index *common.PatternIndex[*BotEntry]

	// Pre-compiled regexes (RE2 or regexp2) hidden behind a single interface.
	// Bots require capture groups for name templating (e.g. "$1"), so we store a
	// submatch-capable regex.
	compiled map[string]common.UniversalRegexSubmatch
}

// NewParserFactory creates a factory by loading and compiling regexes from the embedded YAML DB.
func NewParserFactory(opts ...common.FactoryOption) (*ParserFactory, error) {
	data, err := regexes.FS.ReadFile("bots.yml")
	if err != nil {
		return nil, fmt.Errorf("reading regexes file: %w", err)
	}

	var entries []BotEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing regexes YAML: %w", err)
	}

	cfg := common.ApplyFactoryOptions(opts)
	compiler := common.NewRegexCompiler(cfg.RegexMode)

	f := &ParserFactory{
		regexes:  entries,
		compiled: make(map[string]common.UniversalRegexSubmatch),
	}

	// Build keyword index for fast candidate lookup
	f.buildKeywordIndex()

	// Pre-compile all regexes
	if err := f.compileAll(compiler, cfg.RegexMode); err != nil {
		return nil, err
	}

	return f, nil
}

// buildKeywordIndex creates an Aho-Corasick based index for fast keyword lookup.
func (f *ParserFactory) buildKeywordIndex() {
	// Convert to pointer slice for the index
	patterns := make([]*BotEntry, len(f.regexes))
	for i := range f.regexes {
		patterns[i] = &f.regexes[i]
	}
	f.index = common.NewPatternIndex(patterns)
}

// compileAll pre-compiles all regex patterns.
func (f *ParserFactory) compileAll(compiler *common.RegexCompiler, regexMode common.RegexMode) error {
	// Pre-compile individual patterns
	for _, entry := range f.regexes {
		if err := f.compilePattern(entry.Regex, compiler); err != nil {
			// In Re2Only mode, skip patterns that can't compile
			if regexMode == common.Re2Only {
				continue
			}
			return fmt.Errorf("compiling pattern %q: %w", entry.Regex, err)
		}
	}

	return nil
}

// compilePattern compiles a single pattern using the appropriate engine.
func (f *ParserFactory) compilePattern(pattern string, compiler *common.RegexCompiler) error {
	wrapped := wrapPattern(pattern)

	re, err := compiler.CompileSubmatch(wrapped)
	if err != nil {
		return err
	}
	f.compiled[pattern] = re

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
func (f *ParserFactory) Parse(ua string, opts ...Option) *BotMatch {
	return f.NewParser(ua, opts...).Parse()
}

// IsBot is a convenience method that checks if a user agent is a bot.
func (f *ParserFactory) IsBot(ua string) bool {
	return f.Parse(ua) != nil
}

// IndexStats returns statistics about the keyword index.
func (f *ParserFactory) IndexStats() common.IndexStats {
	return f.index.Stats()
}

// wrapPattern wraps a regex pattern with the prefix matching logic from PHP.
// PHP: '/(?:^|[^A-Z0-9_-]|[^A-Z0-9-]_|sprd-|MZ-)(?:' . $regex . ')/i'
func wrapPattern(pattern string) string {
	// Escape forward slashes in the pattern (PHP does this)
	pattern = strings.ReplaceAll(pattern, "/", `\/`)
	return `(?:^|[^A-Z0-9_-]|[^A-Z0-9-]_|sprd-|MZ-)(?:` + pattern + `)`
}

// NOTE: We intentionally do not implement a heuristic like hasLookaround() here.
// Instead, we optimistically try compiling with RE2 and fall back to regexp2 if RE2 rejects it.
