package shelltv

// Parser parses a single user agent for Shell TV information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory   *ParserFactory
	userAgent string
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// IsShellTv returns true if the UA is identified as a Shell TV device.
// Mirrors PHP: DeviceDetector\Parser\Device\ShellTv::isShellTv(): bool
func (p *Parser) IsShellTv() bool {
	if p.factory == nil || p.factory.isShellTvRegex == nil || p.userAgent == "" {
		return false
	}
	ok, err := p.factory.isShellTvRegex.MatchString(p.userAgent)
	return err == nil && ok
}

// Parse mirrors the PHP parser's parse() semantics at a high level:
// - only parse user agents containing Shell TV fragments
// - otherwise return nil
//
// Note: For now this focuses on the behavior needed by the PHP unit test (isShellTv).
// Device brand/model parsing from shell_tv.yml can be added on top.
func (p *Parser) Parse() *Match {
	if !p.IsShellTv() {
		return nil
	}
	return &Match{IsShellTv: true}
}
