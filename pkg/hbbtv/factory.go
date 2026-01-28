package hbbtv

import (
	"fmt"

	"github.com/archbottle/device-detector/pkg/common"
)

// ParserFactory holds pre-compiled matchers and creates Parser instances.
// Thread-safe for concurrent use - create once, use from multiple goroutines.
type ParserFactory struct {
	// isHbbTvRegex matches (?:HbbTV|SmartTvA)/version and captures the version.
	// This mirrors DeviceDetector\Parser\Device\HbbTv::isHbbTv().
	isHbbTvRegex common.UniversalRegexSubmatch
}

// NewParserFactory creates a factory with all regexes compiled once.
func NewParserFactory(opts ...common.FactoryOption) (*ParserFactory, error) {
	cfg := common.ApplyFactoryOptions(opts)
	compiler := common.NewRegexCompiler(cfg.RegexMode)

	// PHP: (?:HbbTV|SmartTvA)/([1-9]{1}(?:\.[0-9]{1}){1,2})
	raw := `(?:HbbTV|SmartTvA)/([1-9]{1}(?:\.[0-9]{1}){1,2})`
	wrapped := common.WrapDeviceDetectorPattern(raw)

	re, err := compiler.CompileSubmatch(wrapped)
	if err != nil {
		// In Re2Only mode, return error if can't compile
		if cfg.RegexMode == common.Re2Only {
			return nil, fmt.Errorf("compiling isHbbTv regex (RE2-only mode): %w", err)
		}
		return nil, fmt.Errorf("compiling isHbbTv regex: %w", err)
	}

	return &ParserFactory{isHbbTvRegex: re}, nil
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

// IsHbbTv is a convenience method mirroring PHP's $parser->isHbbTv().
// It returns the HbbTV version string if detected, otherwise "".
func (f *ParserFactory) IsHbbTv(ua string) string {
	return f.NewParser(ua).IsHbbTv()
}
