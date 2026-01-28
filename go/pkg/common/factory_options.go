package common

// RegexMode controls which regex engine(s) to use for pattern compilation.
type RegexMode int

const (
	// Auto tries RE2 first, then falls back to regexp2 for PCRE features.
	Auto RegexMode = iota
	// Re2Only only uses Go's regexp (RE2); patterns that can't compile are skipped.
	Re2Only
)

// FactoryConfig controls shared behavior across parser factories.
type FactoryConfig struct {
	CandidateMode CandidateMode
	RegexMode     RegexMode
}

// FactoryOption configures a parser factory at construction time.
type FactoryOption func(*FactoryConfig)

func defaultFactoryConfig() FactoryConfig {
	return FactoryConfig{
		CandidateMode: Compatibility,
		RegexMode:     Auto,
	}
}

// ApplyFactoryOptions applies options to the default config and returns the result.
func ApplyFactoryOptions(opts []FactoryOption) FactoryConfig {
	cfg := defaultFactoryConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

// WithCandidateMode sets how aggressively the parser relies on the keyword index.
func WithCandidateMode(mode CandidateMode) FactoryOption {
	return func(cfg *FactoryConfig) {
		cfg.CandidateMode = mode
	}
}

// WithRe2Only sets regex mode to RE2-only (no regexp2 fallback).
func WithRe2Only() FactoryOption {
	return func(cfg *FactoryConfig) {
		cfg.RegexMode = Re2Only
	}
}

// WithIndexOnly sets candidate mode to StrictIndex (no full scan fallback).
func WithIndexOnly() FactoryOption {
	return func(cfg *FactoryConfig) {
		cfg.CandidateMode = StrictIndex
	}
}

// WithFullCompatibility explicitly sets both modes to compatibility defaults.
func WithFullCompatibility() FactoryOption {
	return func(cfg *FactoryConfig) {
		cfg.CandidateMode = Compatibility
		cfg.RegexMode = Auto
	}
}
