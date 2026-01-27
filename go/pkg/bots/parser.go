package bots

import (
	"strconv"
	"strings"
)

// Parser parses a single user agent for bot information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory        *ParserFactory
	userAgent      string
	discardDetails bool
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// WithDiscardDetails enables discarding detailed bot info.
// When enabled, Parse() returns a minimal result instead of full bot info.
func WithDiscardDetails() Option {
	return func(p *Parser) {
		p.discardDetails = true
	}
}

// Parse parses the user agent and checks for bot information.
//
// Returns:
//   - *BotMatch with full bot info if a bot is detected
//   - nil if no bot is detected
//
// If WithDiscardDetails() was used, returns a minimal BotMatch with only
// a placeholder name "true" instead of full info (matching PHP's [true] return).
func (p *Parser) Parse() *BotMatch {
	if p.userAgent == "" || p.factory == nil {
		return nil
	}

	// If discarding details, return early with minimal result
	if p.discardDetails {
		candidates := p.factory.index.FindCandidates(p.userAgent)
		for _, entry := range candidates {
			if _, ok := p.matchPattern(entry.Regex); ok {
				return &BotMatch{Name: "true"} // Indicates "is bot" without details
			}
		}
		return nil
	}

	// Find candidates via keyword index (Aho-Corasick O(n) search)
	candidates := p.factory.index.FindCandidates(p.userAgent)

	// Regex match only on candidates (typically 1-10 patterns)
	for _, entry := range candidates {
		matches, ok := p.matchPattern(entry.Regex)
		if ok {
			name := entry.Name
			if name != "" && matches != nil {
				name = buildByMatch(name, matches)
			}
			return &BotMatch{
				Name:     name,
				Category: entry.Category,
				URL:      entry.URL,
				Producer: entry.Producer,
			}
		}
	}

	return nil
}

// matchPattern checks if the user agent matches a pattern and returns submatches.
// If there is a match, matches[0] is the full match and matches[1..] are capture groups.
func (p *Parser) matchPattern(pattern string) ([]string, bool) {
	re, ok := p.factory.compiled[pattern]
	if !ok || re == nil {
		return nil, false
	}
	matches, err := re.FindStringSubmatch(p.userAgent)
	return matches, err == nil && matches != nil
}

// IsBot returns true if the user agent is detected as a bot.
func (p *Parser) IsBot() bool {
	return p.Parse() != nil
}

// buildByMatch replaces $1..$n placeholders with capture group values.
// Matches follow Go's FindStringSubmatch convention: [0]=full match, [1..]=groups.
func buildByMatch(template string, matches []string) string {
	if template == "" || !strings.Contains(template, "$") || len(matches) <= 1 {
		return strings.TrimSpace(template)
	}

	args := make([]string, 0, (len(matches)-1)*2)
	for i := 1; i < len(matches); i++ {
		args = append(args, "$"+strconv.Itoa(i), matches[i])
	}
	return strings.TrimSpace(strings.NewReplacer(args...).Replace(template))
}
