package common

// YAMLListDB is a small helper for parsers whose DB is a YAML sequence (list) of entries.
// It centralizes: precompiled candidate index, candidate mode, and compilation-at-init.
//
// T is typically a pointer to an entry struct (e.g. *library.Entry) that implements Pattern.
type YAMLListDB[T Pattern] struct {
	Patterns []T
	Index    *PatternIndex[T]
	Mode     CandidateMode
}

// NewYAMLListDB builds an index and optionally compiles patterns once at init.
// The compile callback may set compiled regex fields on the entry.
func NewYAMLListDB[T Pattern](patterns []T, compile func(T) error, opts ...FactoryOption) (*YAMLListDB[T], error) {
	cfg := ApplyFactoryOptions(opts)
	db := &YAMLListDB[T]{
		Patterns: patterns,
		Mode:     cfg.CandidateMode,
	}
	if len(patterns) > 0 {
		db.Index = NewPatternIndex(patterns)
	}
	if compile != nil {
		for _, p := range patterns {
			if err := compile(p); err != nil {
				return nil, err
			}
		}
	}
	return db, nil
}

// Candidates returns ordered candidates for a UA, following the configured mode.
func (db *YAMLListDB[T]) Candidates(ua string) []T {
	if db == nil {
		return nil
	}
	return SelectCandidates(db.Patterns, db.Index, ua, db.Mode)
}
