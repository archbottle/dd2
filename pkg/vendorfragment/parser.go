package vendorfragment

// Parser parses a single user agent for vendor fragment information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory      *ParserFactory
	userAgent    string
	matchedRegex string
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// Parse matches the UA against vendor fragments and returns the detected brand.
// Mirrors PHP: VendorFragment::parse(): ?array returning ['brand' => $brand]
func (p *Parser) Parse() map[string]string {
	p.matchedRegex = ""

	if p.factory == nil || p.userAgent == "" {
		return nil
	}

	for _, g := range p.factory.groups {
		for _, raw := range g.Regexes {
			re := p.factory.compiled[raw]
			if re == nil {
				continue
			}
			ok, err := re.MatchString(p.userAgent)
			if err == nil && ok {
				p.matchedRegex = raw
				return map[string]string{"brand": g.Brand}
			}
		}
	}

	return nil
}

// MatchedRegex returns the fragment (regex) that matched during the last Parse().
// Mirrors PHP: VendorFragment::getMatchedRegex(): ?string
func (p *Parser) MatchedRegex() string {
	return p.matchedRegex
}
