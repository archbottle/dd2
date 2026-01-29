package library

import (
	"strconv"
	"strings"

	"github.com/archbottle/dd2/pkg/common"
)

// Parser parses a single user agent for library information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory   *ParserFactory
	userAgent string
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// Parse detects libraries and returns a Match, or nil if not detected.
// Mirrors PHP: DeviceDetector\Parser\Client\Library::parse(): ?array
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
				Type:    "library",
				Name:    name,
				Version: version,
			}
		}
		return nil
	}

	candidates := common.SelectCandidates(p.factory.patterns, p.factory.index, p.userAgent, p.factory.mode)
	if m := matchFrom(candidates); m != nil {
		return m
	}

	// Ensure the index is a pure optimization in Compatibility mode.
	if p.factory.index != nil && p.factory.mode == common.Compatibility {
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
func buildByMatch(template string, matches []string) string {
	if template == "" || len(matches) == 0 {
		return template
	}
	// PHP replaces $1..$n where n = count($matches). If a placeholder exists but the
	// capture group doesn't, PHP effectively substitutes an empty string.
	//
	// We replace a bounded range to also clear any extra placeholders like "$1"
	// when the regex had no capture groups.
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
