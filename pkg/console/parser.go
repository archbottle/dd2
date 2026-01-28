package console

import (
	"github.com/archbottle/device-detector/pkg/common"
)

// Parser parses a single user agent for console information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory   *ParserFactory
	userAgent string
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// Parse detects console devices and returns a Match, or nil if not detected.
// Mirrors PHP: DeviceDetector\Parser\Device\Console::parse(): ?array
func (p *Parser) Parse() *Match {
	if p.factory == nil || p.userAgent == "" {
		return nil
	}

	candidates := common.SelectCandidates(p.factory.entries, p.factory.index, p.userAgent, p.factory.mode)

	// Phase 1: try indexed candidates (fast path).
	if m := p.matchFrom(candidates); m != nil {
		return m
	}

	// Phase 2: if index was used but produced no match, fall back to full scan to preserve correctness.
	if p.factory.index != nil && p.factory.mode == common.Compatibility {
		return p.matchFrom(p.factory.entries)
	}

	return nil
}

func (p *Parser) matchFrom(candidates []*Entry) *Match {
	for _, e := range candidates {
		if m := p.matchEntry(e); m != nil {
			return m
		}
	}
	return nil
}

func (p *Parser) matchEntry(e *Entry) *Match {
	if e == nil || e.compiledBrand == nil {
		return nil
	}

	brandMatches, err := e.compiledBrand.FindStringSubmatch(p.userAgent)
	if err != nil || len(brandMatches) == 0 {
		return nil
	}

	brand := e.Brand
	deviceType := e.Device
	if deviceType == "" {
		deviceType = "console"
	}

	// Brand-level model.
	if e.Model != "" {
		return &Match{
			Type:  deviceType,
			Brand: brand,
			Model: common.BuildModel(e.Model, brandMatches),
		}
	}

	// Model list: first matching model wins.
	if len(e.Models) == 0 || len(e.compiledModels) == 0 {
		return &Match{Type: deviceType, Brand: brand, Model: ""}
	}

	for i := range e.Models {
		m := e.Models[i]
		re := e.compiledModels[i]
		if re == nil {
			continue
		}

		modelMatches, err := re.FindStringSubmatch(p.userAgent)
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

	// Brand matched but no model matched => still a detection, model empty.
	return &Match{Type: deviceType, Brand: brand, Model: ""}
}
