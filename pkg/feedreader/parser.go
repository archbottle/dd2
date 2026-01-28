package feedreader

import (
	"strconv"
	"strings"
)

// Parser parses a single user agent for feed reader client information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory   *ParserFactory
	userAgent string
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// Parse detects feed readers and returns a Match, or nil if not detected.
// Mirrors PHP: DeviceDetector\Parser\Client\FeedReader::parse(): ?array
func (p *Parser) Parse() *Match {
	if p.factory == nil || p.userAgent == "" {
		return nil
	}

	candidates := []*Entry(nil)
	usedIndex := false
	if p.factory.index != nil {
		usedIndex = true
		candidates = p.factory.index.FindCandidates(p.userAgent)
		// If indexing produced no candidates, fall back to full scan for correctness.
		if len(candidates) == 0 {
			usedIndex = false
		}
	}
	if !usedIndex {
		candidates = make([]*Entry, len(p.factory.entries))
		for i := range p.factory.entries {
			candidates[i] = &p.factory.entries[i]
		}
	}

	for _, e := range candidates {
		if e == nil || e.compiled == nil {
			continue
		}
		matches, err := e.compiled.FindStringSubmatch(p.userAgent)
		if err != nil || len(matches) == 0 {
			continue
		}

		version := strings.TrimSpace(buildByMatch(e.Version, matches))
		return &Match{
			Type:    "feed reader",
			Name:    e.Name,
			Version: version,
		}
	}

	return nil
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
