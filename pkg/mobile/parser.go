package mobile

import (
	"github.com/archbottle/device-detector/pkg/clienthints"
	"github.com/archbottle/device-detector/pkg/common"
)

// Parser parses a single user agent for mobile device information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory     *ParserFactory
	userAgent   string
	clientHints *clienthints.ClientHints
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// WithClientHints enables client hints for device detection.
func WithClientHints(ch *clienthints.ClientHints) Option {
	return func(p *Parser) {
		p.clientHints = ch
	}
}

// Parse detects mobile devices and returns a Match, or nil if not detected.
// Mirrors PHP: DeviceDetector\Parser\Device\Mobile::parse(): ?array
func (p *Parser) Parse() *Match {
	if p.factory == nil || p.userAgent == "" {
		return nil
	}

	// Get effective user agent (potentially restored from client hints)
	ua := p.getEffectiveUserAgent()

	candidates := common.SelectCandidates(p.factory.entries, p.factory.index, ua, p.factory.mode)

	// Phase 1: try indexed candidates (fast path).
	if m := p.matchFrom(candidates, ua); m != nil {
		return m
	}

	// Phase 2: if index was used but produced no match, fall back to full scan.
	if p.factory.index != nil && p.factory.mode == common.Compatibility {
		return p.matchFrom(p.factory.entries, ua)
	}

	return nil
}

// getEffectiveUserAgent returns the user agent to use for matching.
// If client hints are present and the UA has the client hints fragment,
// we restore the device model into the UA for regex matching.
func (p *Parser) getEffectiveUserAgent() string {
	if p.clientHints == nil {
		return p.userAgent
	}

	// Check if this UA uses client hints format (Android 10+ with "; K")
	if !HasClientHintsFragment(p.userAgent) {
		return p.userAgent
	}

	// Restore device model from client hints into the UA
	return RestoreUserAgent(p.userAgent, p.clientHints)
}

func (p *Parser) matchFrom(candidates []*Entry, ua string) *Match {
	for _, e := range candidates {
		if m := p.matchEntry(e, ua); m != nil {
			return m
		}
	}
	return nil
}

func (p *Parser) matchEntry(e *Entry, ua string) *Match {
	if e == nil || e.compiledBrand == nil {
		return nil
	}

	brandMatches, err := e.compiledBrand.FindStringSubmatch(ua)
	if err != nil || len(brandMatches) == 0 {
		return nil
	}

	brand := e.Brand
	deviceType := e.Device
	if deviceType == "" {
		deviceType = "smartphone" // Default device type for mobile
	}

	// Brand-level model (when no models array exists).
	if e.Model != "" && len(e.Models) == 0 {
		return &Match{
			Type:  deviceType,
			Brand: brand,
			Model: common.BuildModel(e.Model, brandMatches),
		}
	}

	// Model list: first matching model wins.
	if len(e.Models) == 0 || len(e.compiledModels) == 0 {
		// Brand matched but no model template - return with empty model
		model := ""
		if e.Model != "" {
			model = common.BuildModel(e.Model, brandMatches)
		}
		return &Match{Type: deviceType, Brand: brand, Model: model}
	}

	for i := range e.Models {
		m := e.Models[i]
		re := e.compiledModels[i]
		if re == nil {
			continue
		}

		modelMatches, err := re.FindStringSubmatch(ua)
		if err != nil || len(modelMatches) == 0 {
			continue
		}

		// Model rules can override brand/device (for parity with PHP AbstractDeviceParser).
		if m.Brand != "" {
			brand = m.Brand
		}
		if m.Device != "" {
			deviceType = m.Device
		}

		return &Match{
			Type:  deviceType,
			Brand: brand,
			Model: common.BuildModel(m.Model, modelMatches),
		}
	}

	// Brand matched but no model matched => still a detection, use brand-level model if any.
	model := ""
	if e.Model != "" {
		model = common.BuildModel(e.Model, brandMatches)
	}
	return &Match{Type: deviceType, Brand: brand, Model: model}
}
