package common

import (
	"strconv"
	"strings"
)

// BuildModel performs template substitution and cleanup for device models.
// This mirrors PHP AbstractDeviceParser::buildModel().
//
// Template substitution: $1, $2, ... are replaced with regex capture groups.
// Cleanup: underscores to spaces, trailing " TD" removed, "Build" rejected.
//
// Used by Console, Camera, CarBrowser, Notebook, and Mobile parsers.
func BuildModel(template string, matches []string) string {
	s := BuildByMatch(template, matches)
	s = strings.ReplaceAll(s, "_", " ")

	// Remove trailing " TD" (case-insensitive) - PHP quirk for technical debt markers
	if len(s) >= 3 && strings.EqualFold(s[len(s)-3:], " TD") {
		s = s[:len(s)-3]
	}

	s = strings.TrimSpace(s)

	// "Build" is Android's default build string - not a valid model name
	if s == "" || s == "Build" {
		return ""
	}
	return s
}

// BuildByMatch substitutes $1..$n with corresponding regex capture groups.
// This matches device-detector's template style used across YAML DBs.
//
// Replacement is done from high to low index to avoid $10 being partially
// replaced as $1 + "0".
func BuildByMatch(template string, matches []string) string {
	if template == "" || len(matches) == 0 {
		return template
	}

	out := template
	// Replace from high to low to avoid $10 being partially replaced as $1 + "0"
	for i := len(matches) - 1; i >= 1; i-- {
		out = strings.ReplaceAll(out, "$"+strconv.Itoa(i), matches[i])
	}
	return out
}
