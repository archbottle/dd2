package notebook

import (
	"github.com/archbottle/dd2/pkg/common"
)

// Parser parses a single user agent for notebook information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory   *ParserFactory
	userAgent string
}

// Option is a functional option for configuring Parser behavior.
type Option func(*Parser)

// Parse detects notebook devices (from FB desktop user agents) and returns a Match, or nil if not detected.
// Mirrors PHP: DeviceDetector\Parser\Device\Notebook::parse(): ?array
func (p *Parser) Parse() *Match {
	if p.factory == nil || p.userAgent == "" {
		return nil
	}

	// PHP Notebook::parse(): only parse if UA contains FBMD/ fragment (via matchUserAgent boundary wrapper).
	if p.factory.fbmd != nil {
		ok, err := p.factory.fbmd.MatchString(p.userAgent)
		if err != nil || !ok {
			return nil
		}
	}

	matchFrom := func(candidates []*Entry) *Match {
		for _, e := range candidates {
			if e == nil || e.compiledBrand == nil {
				continue
			}

			brandMatches, err := e.compiledBrand.FindStringSubmatch(p.userAgent)
			if err != nil || len(brandMatches) == 0 {
				continue
			}

			brand := e.Brand
			deviceType := e.Device
			if deviceType == "" {
				deviceType = "desktop"
			}

			if e.Model != "" {
				return &Match{
					Type:  deviceType,
					Brand: brand,
					Model: common.BuildModel(e.Model, brandMatches),
				}
			}

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

			return &Match{Type: deviceType, Brand: brand, Model: ""}
		}
		return nil
	}

	candidates := common.SelectCandidates(p.factory.entries, p.factory.index, p.userAgent, p.factory.mode)

	if m := matchFrom(candidates); m != nil {
		return m
	}
	if p.factory.index != nil && p.factory.mode == common.Compatibility {
		return matchFrom(p.factory.entries)
	}

	return nil
}
