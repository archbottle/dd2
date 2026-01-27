package bots

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
			if p.matchPattern(entry.Regex) {
				return &BotMatch{Name: "true"} // Indicates "is bot" without details
			}
		}
		return nil
	}

	// Find candidates via keyword index (Aho-Corasick O(n) search)
	candidates := p.factory.index.FindCandidates(p.userAgent)

	// Regex match only on candidates (typically 1-10 patterns)
	for _, entry := range candidates {
		if p.matchPattern(entry.Regex) {
			return &BotMatch{
				Name:     entry.Name,
				Category: entry.Category,
				URL:      entry.URL,
				Producer: entry.Producer,
			}
		}
	}

	return nil
}

// matchPattern checks if the user agent matches a pattern.
func (p *Parser) matchPattern(pattern string) bool {
	re, ok := p.factory.compiled[pattern]
	if !ok || re == nil {
		return false
	}
	match, err := re.MatchString(p.userAgent)
	return err == nil && match
}

// IsBot returns true if the user agent is detected as a bot.
func (p *Parser) IsBot() bool {
	return p.Parse() != nil
}
