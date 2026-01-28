package regexes

import "embed"

// FS contains the runtime recognition regex databases (and only those).
//
// NOTE: Test fixtures are intentionally not embedded. They live under go/pkg/**/fixtures
// and php/Tests/** and are loaded from disk by tests/report tooling.
//
//go:embed bots.yml oss.yml vendorfragments.yml client/*.yml client/hints/*.yml device/*.yml
var FS embed.FS

