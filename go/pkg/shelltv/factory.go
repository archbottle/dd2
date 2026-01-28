package shelltv

import (
	"fmt"

	"github.com/archbottle/device-detector/pkg/common"
)

// ParserFactory holds pre-compiled matchers and creates Parser instances.
// Thread-safe for concurrent use - create once, use from multiple goroutines.
type ParserFactory struct {
	isShellTvRegex common.UniversalRegex
}

// NewParserFactory creates a factory with all regexes compiled once.
func NewParserFactory(opts ...common.FactoryOption) (*ParserFactory, error) {
	cfg := common.ApplyFactoryOptions(opts)
	compiler := common.NewRegexCompiler(cfg.RegexMode)

	// PHP: '[a-z]+[ _]Shell[ _]\w{6}|tclwebkit(\d+[.\d]*)'
	raw := `[a-z]+[ _]Shell[ _]\w{6}|tclwebkit(\d+[.\d]*)`
	wrapped := common.WrapDeviceDetectorPattern(raw)

	re, err := compiler.Compile(wrapped)
	if err != nil {
		// In Re2Only mode, return error if can't compile
		if cfg.RegexMode == common.Re2Only {
			return nil, fmt.Errorf("compiling isShellTv regex (RE2-only mode): %w", err)
		}
		return nil, fmt.Errorf("compiling isShellTv regex: %w", err)
	}

	return &ParserFactory{isShellTvRegex: re}, nil
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

// IsShellTv is a convenience method mirroring PHP's $parser->isShellTv().
func (f *ParserFactory) IsShellTv(ua string) bool {
	return f.NewParser(ua).IsShellTv()
}
