package pim

import (
	"strconv"
	"strings"

	"github.com/archbottle/device-detector/pkg/common"
)

// Parser parses a single user agent for PIM information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory   *ParserFactory
	userAgent string
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// Parse detects PIM clients and returns a Match, or nil if not detected.
// Mirrors PHP: DeviceDetector\Parser\Client\PIM::parse(): ?array
func (p *Parser) Parse() *Match {
	if p.factory == nil || p.userAgent == "" {
		return nil
	}

	matchFrom := func(candidates []*Entry) *Match {
		for _, e := range candidates {
			if e == nil || e.compiled == nil {
				continue
			}

			matches, err := e.compiled.FindStringSubmatch(p.userAgent)
			if err != nil || len(matches) == 0 {
				continue
			}

			name := strings.TrimSpace(buildByMatch(e.Name, matches))
			version := buildVersion(e.Version, matches)

			return &Match{
				Type:    "pim",
				Name:    name,
				Version: version,
			}
		}
		return nil
	}

	candidates := []*Entry(nil)
	if p.factory.db != nil {
		candidates = p.factory.db.Candidates(p.userAgent)
	} else {
		candidates = p.factory.patterns
	}
	if m := matchFrom(candidates); m != nil {
		return m
	}

	// Ensure index is only an optimization: fall back to a full scan if no match.
	if p.factory.db != nil && p.factory.db.Index != nil && p.factory.db.Mode == common.Compatibility {
		return matchFrom(p.factory.patterns)
	}

	return nil
}

func buildVersion(template string, matches []string) string {
	v := strings.TrimSpace(buildByMatch(template, matches))
	if v == "" {
		return ""
	}
	v = strings.ReplaceAll(v, "_", ".")
	v = strings.Trim(v, " .")
	return v
}

// buildByMatch substitutes $1..$n with corresponding capture groups.
// PHP replaces missing capture groups with "" when placeholders exist.
func buildByMatch(template string, matches []string) string {
	if template == "" {
		return template
	}

	const maxGroups = 30
	out := template
	for i := 1; i <= maxGroups; i++ {
		repl := ""
		if i < len(matches) {
			repl = matches[i]
		}
		out = strings.ReplaceAll(out, "$"+strconv.Itoa(i), repl)
	}
	return out
}
