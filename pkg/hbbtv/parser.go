package hbbtv

// Parser parses a single user agent for HbbTV information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory   *ParserFactory
	userAgent string
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// IsHbbTv returns the parsed HbbTV version if detected, otherwise "".
// Mirrors PHP: DeviceDetector\Parser\Device\HbbTv::isHbbTv(): ?string
func (p *Parser) IsHbbTv() string {
	if p.factory == nil || p.factory.isHbbTvRegex == nil || p.userAgent == "" {
		return ""
	}

	matches, err := p.factory.isHbbTvRegex.FindStringSubmatch(p.userAgent)
	if err != nil || len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// Parse mirrors the PHP parser's parse() semantics at a high level:
// - only parse user agents containing HbbTV/SmartTvA
// - otherwise return nil
//
// Note: For now this focuses on the behavior needed by the PHP unit test
// (isHbbTv). TV brand/model parsing from televisions.yml can be added on top.
func (p *Parser) Parse() *Match {
	v := p.IsHbbTv()
	if v == "" {
		return nil
	}
	return &Match{Version: v}
}
