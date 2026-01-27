package common

// FactoryConfig controls shared behavior across parser factories.
type FactoryConfig struct {
	CandidateMode CandidateMode
}

// FactoryOption configures a parser factory at construction time.
type FactoryOption func(*FactoryConfig)

func defaultFactoryConfig() FactoryConfig {
	return FactoryConfig{CandidateMode: Compatibility}
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
