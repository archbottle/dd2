package camera

import (
	"github.com/archbottle/dd2/pkg/common"
)

// Parser parses a single user agent for Camera device information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory   *ParserFactory
	userAgent string
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// Parse matches the UA against camera patterns and returns the detected device match.
// Mirrors PHP: DeviceDetector\Parser\Device\Camera::parse(): ?array
func (p *Parser) Parse() *Match {
	if p.factory == nil || p.userAgent == "" {
		return nil
	}

	candidates := p.factory.patterns
	usedIndex := false
	if p.factory.index != nil {
		usedIndex = true
		candidates = p.factory.candidatesFor(p.userAgent)
	}

	for _, brand := range candidates {
		if brand == nil || brand.compiled == nil {
			continue
		}

		matches, err := brand.compiled.FindStringSubmatch(p.userAgent)
		if err != nil || matches == nil {
			continue
		}

		out := &Match{
			Type:  brand.Device,
			Brand: brand.Brand,
			Model: "",
		}

		if brand.Model != "" {
			out.Model = common.BuildModel(brand.Model, matches)
		}

		// Optional per-brand model overrides (checked in order, like PHP).
		if len(brand.Models) != 0 {
			for i := range brand.Models {
				mp := &brand.Models[i]
				if mp.compiled == nil {
					continue
				}
				modelMatches, err := mp.compiled.FindStringSubmatch(p.userAgent)
				if err != nil || modelMatches == nil {
					continue
				}

				out.Model = common.BuildModel(mp.Model, modelMatches)
				if mp.Brand != "" {
					out.Brand = mp.Brand
				}
				if mp.Device != "" {
					out.Type = mp.Device
				}
				return out
			}

			// Brand matched, but no model override matched.
			return out
		}

		return out
	}

	// If we used the index but found no match, fall back to the full scan to preserve correctness.
	if usedIndex {
		for _, brand := range p.factory.patterns {
			if brand == nil || brand.compiled == nil {
				continue
			}

			matches, err := brand.compiled.FindStringSubmatch(p.userAgent)
			if err != nil || matches == nil {
				continue
			}

			out := &Match{
				Type:  brand.Device,
				Brand: brand.Brand,
				Model: "",
			}

			if brand.Model != "" {
				out.Model = common.BuildModel(brand.Model, matches)
			}

			if len(brand.Models) != 0 {
				for i := range brand.Models {
					mp := &brand.Models[i]
					if mp.compiled == nil {
						continue
					}
					modelMatches, err := mp.compiled.FindStringSubmatch(p.userAgent)
					if err != nil || modelMatches == nil {
						continue
					}

					out.Model = common.BuildModel(mp.Model, modelMatches)
					if mp.Brand != "" {
						out.Brand = mp.Brand
					}
					if mp.Device != "" {
						out.Type = mp.Device
					}
					return out
				}
				return out
			}

			return out
		}
	}

	return nil
}
