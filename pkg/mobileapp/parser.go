package mobileapp

import (
	"strconv"
	"strings"

	"github.com/archbottle/device-detector/pkg/common"
)

// Parser parses a single user agent for mobile app client information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory   *ParserFactory
	userAgent string
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// Parse detects mobile apps and returns a Match, or nil if not detected.
// Mirrors PHP: DeviceDetector\Parser\Client\MobileApp::parse(): ?array
func (p *Parser) Parse() *Match {
	if p.factory == nil || p.userAgent == "" {
		return nil
	}

	candidates := []*Entry(nil)
	if p.factory.db != nil {
		candidates = p.factory.db.Candidates(p.userAgent)
	} else {
		candidates = p.factory.patterns
	}
	for _, e := range candidates {
		if e == nil || e.compiled == nil {
			continue
		}

		matches, err := e.compiled.FindStringSubmatch(p.userAgent)
		if err != nil || len(matches) == 0 {
			continue
		}

		name := strings.TrimSpace(buildByMatch(e.Name, matches))
		if name == "" {
			// PHP MobileApp::parse(): return null if name is empty.
			return nil
		}

		version := buildVersion(e.Version, matches)

		return &Match{
			Type:    "mobile app",
			Name:    name,
			Version: version,
		}
	}

	// Ensure index is only an optimization: fall back to a full scan if no match.
	if p.factory.db != nil && p.factory.db.Index != nil && p.factory.db.Mode == common.Compatibility {
		for _, e := range p.factory.patterns {
			if e == nil || e.compiled == nil {
				continue
			}
			matches, err := e.compiled.FindStringSubmatch(p.userAgent)
			if err != nil || len(matches) == 0 {
				continue
			}

			name := strings.TrimSpace(buildByMatch(e.Name, matches))
			if name == "" {
				return nil
			}

			version := buildVersion(e.Version, matches)
			return &Match{
				Type:    "mobile app",
				Name:    name,
				Version: version,
			}
		}
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
// This matches device-detector's template style used across YAML DBs.
func buildByMatch(template string, matches []string) string {
	if template == "" || len(matches) == 0 {
		return template
	}
	out := template
	// Replace from high to low to avoid $10 being partially replaced as $1 + "0".
	for i := len(matches) - 1; i >= 1; i-- {
		out = strings.ReplaceAll(out, "$"+strconv.Itoa(i), matches[i])
	}
	return out
}
