package common

import (
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"
)

// UniversalRegex is a small abstraction over the two regex engines we support.
// It lets the rest of the code treat patterns uniformly without caring about
// which underlying engine is used.
type UniversalRegex interface {
	MatchString(s string) (bool, error)
}

// UniversalRegexSubmatch is a UniversalRegex that can also return capture groups.
// The returned slice matches Go's regexp.FindStringSubmatch behavior:
// element 0 is the full match, element 1..n are capture groups.
// If there is no match, it returns (nil, nil).
type UniversalRegexSubmatch interface {
	UniversalRegex
	FindStringSubmatch(s string) ([]string, error)
}

type re2Regex struct {
	re *regexp.Regexp
}

func (r re2Regex) MatchString(s string) (bool, error) {
	return r.re.MatchString(s), nil
}

func (r re2Regex) FindStringSubmatch(s string) ([]string, error) {
	return r.re.FindStringSubmatch(s), nil
}

type regexp2Regex struct {
	re *regexp2.Regexp
}

func (r regexp2Regex) MatchString(s string) (bool, error) {
	return r.re.MatchString(s)
}

func (r regexp2Regex) FindStringSubmatch(s string) ([]string, error) {
	m, err := r.re.FindStringMatch(s)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	groups := m.Groups()
	out := make([]string, len(groups))
	for i, g := range groups {
		out[i] = g.String()
	}
	return out, nil
}

// NewRE2Regex creates a UniversalRegex wrapper around a standard Go regexp.Regexp.
func NewRE2Regex(re *regexp.Regexp) UniversalRegex {
	return re2Regex{re: re}
}

// NewRegexp2Regex creates a UniversalRegex wrapper around a regexp2.Regexp.
func NewRegexp2Regex(re *regexp2.Regexp) UniversalRegex {
	return regexp2Regex{re: re}
}

// CompileRegex attempts to compile a pattern with RE2 first (fast path),
// and falls back to regexp2 if RE2 rejects it (e.g., lookarounds, backreferences).
// The wrapped pattern should already include the PHP-style prefix matching wrapper.
func CompileRegex(wrappedPattern string) (UniversalRegex, error) {
	// Try RE2 first (faster)
	if re, err := regexp.Compile("(?i)" + wrappedPattern); err == nil {
		return NewRE2Regex(re), nil
	}

	// Fall back to regexp2 for PCRE-like features
	re2x, err := regexp2.Compile(wrappedPattern, regexp2.IgnoreCase)
	if err != nil {
		return nil, err
	}
	return NewRegexp2Regex(re2x), nil
}

// CompileRegexSubmatch is like CompileRegex, but returns a value that can also
// provide capture groups.
func CompileRegexSubmatch(wrappedPattern string) (UniversalRegexSubmatch, error) {
	// Try RE2 first (faster)
	if re, err := regexp.Compile("(?i)" + wrappedPattern); err == nil {
		return re2Regex{re: re}, nil
	}

	// Fall back to regexp2 for PCRE-like features
	re2x, err := regexp2.Compile(wrappedPattern, regexp2.IgnoreCase)
	if err != nil {
		return nil, err
	}
	return regexp2Regex{re: re2x}, nil
}

// WrapDeviceDetectorPattern wraps a raw YAML regex with the same boundary logic used by
// PHP device-detector's AbstractParser::matchUserAgent().
//
// PHP: '/(?:^|[^A-Z0-9_-]|[^A-Z0-9-]_|sprd-|MZ-)(?:' . $regex . ')/i'
func WrapDeviceDetectorPattern(pattern string) string {
	// Escape forward slashes in the pattern (PHP does this)
	pattern = strings.ReplaceAll(pattern, "/", `\/`)
	return `(?:^|[^A-Z0-9_-]|[^A-Z0-9-]_|sprd-|MZ-)(?:` + pattern + `)`
}
