package browser

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/archbottle/dd2/pkg/common"
	"github.com/archbottle/dd2/regexes"
	"gopkg.in/yaml.v3"
)

// Pre-compiled regexes for engine version parsing (initialized once)
var (
	geckoVersionPattern     *regexp.Regexp
	geckoVersionPatternOnce sync.Once

	// Cache for dynamically compiled engine version patterns
	engineVersionPatterns     = make(map[string]*engineVersionRegexes)
	engineVersionPatternsLock sync.RWMutex
)

type engineVersionRegexes struct {
	withDot *regexp.Regexp
	noDot   *regexp.Regexp
}

// EngineParser handles browser engine detection.
type EngineParser struct {
	entries  []EngineEntry
	patterns []*EngineEntry
	db       *common.YAMLListDB[*EngineEntry]
}

// NewEngineParser creates an engine parser from the embedded YAML DB.
func NewEngineParser(opts ...common.FactoryOption) (*EngineParser, error) {
	data, err := regexes.FS.ReadFile("client/browser_engine.yml")
	if err != nil {
		return nil, fmt.Errorf("reading engine regexes file: %w", err)
	}

	var entries []EngineEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing engine regexes YAML: %w", err)
	}

	for i := range entries {
		entries[i].orderIdx = i
	}

	p := &EngineParser{entries: entries}
	p.patterns = make([]*EngineEntry, len(p.entries))
	for i := range p.entries {
		p.patterns[i] = &p.entries[i]
	}

	db, err := common.NewYAMLListDB(p.patterns, func(e *EngineEntry, compiler *common.RegexCompiler) error {
		if e == nil || e.Regex == "" {
			return nil
		}
		wrapped := common.WrapDeviceDetectorPattern(e.Regex)
		re, compileErr := compiler.CompileSubmatch(wrapped)
		if compileErr != nil {
			return fmt.Errorf("compiling engine pattern (%s): %w", e.Name, compileErr)
		}
		e.compiled = re
		return nil
	}, opts...)
	if err != nil {
		return nil, err
	}
	p.db = db

	return p, nil
}

// NewDefaultEngineParser is an alias for NewEngineParser kept for compatibility.
func NewDefaultEngineParser() (*EngineParser, error) { return NewEngineParser() }

// Parse detects the browser engine from the user agent.
func (p *EngineParser) Parse(ua string) string {
	if p == nil || ua == "" {
		return ""
	}

	candidates := p.db.Candidates(ua)
	for _, e := range candidates {
		if e == nil || e.compiled == nil {
			continue
		}

		matches, err := e.compiled.FindStringSubmatch(ua)
		if err != nil || len(matches) == 0 {
			continue
		}

		// Validate engine name against available engines
		name := buildByMatch(e.Name, matches)
		for _, validEngine := range AvailableEngines {
			if strings.EqualFold(name, validEngine) {
				return validEngine
			}
		}
	}

	// Fallback scan in compatibility mode
	if p.db != nil && p.db.Index != nil && p.db.Mode == common.Compatibility {
		for _, e := range p.patterns {
			if e == nil || e.compiled == nil {
				continue
			}

			matches, err := e.compiled.FindStringSubmatch(ua)
			if err != nil || len(matches) == 0 {
				continue
			}

			name := buildByMatch(e.Name, matches)
			for _, validEngine := range AvailableEngines {
				if strings.EqualFold(name, validEngine) {
					return validEngine
				}
			}
		}
	}

	return ""
}

// ParseEngineVersion extracts the version for a given engine from the user agent.
// This mirrors PHP's Browser\Engine\Version::parse().
func ParseEngineVersion(ua, engine string) string {
	if engine == "" || ua == "" {
		return ""
	}

	// Handle Gecko/Clecko specially - requires rv: pattern with 8-10 digit date
	if engine == "Gecko" || engine == "Clecko" {
		// Initialize Gecko pattern once
		geckoVersionPatternOnce.Do(func() {
			geckoVersionPattern = regexp.MustCompile(`(?i)[ ](?:rv[: ]([0-9.]+)).*(?:g|cl)ecko/[0-9]{8,10}`)
		})
		if matches := geckoVersionPattern.FindStringSubmatch(ua); len(matches) > 1 {
			return matches[1]
		}
		// If no rv: pattern found with date format, fall through to general pattern
	}

	// Get or create cached patterns for this engine
	patterns := getEngineVersionPatterns(engine)
	if patterns == nil {
		return ""
	}

	// First try to match version with dots (e.g., 91.0.4472.124)
	if matches := patterns.withDot.FindStringSubmatch(ua); len(matches) > 1 {
		return matches[1]
	}

	// Fallback: match up to 7 digits (no dot), but only if followed by non-digit or end
	if matches := patterns.noDot.FindStringSubmatch(ua); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// getEngineVersionPatterns returns cached compiled patterns for an engine.
func getEngineVersionPatterns(engine string) *engineVersionRegexes {
	// Fast path: check cache with read lock
	engineVersionPatternsLock.RLock()
	patterns, ok := engineVersionPatterns[engine]
	engineVersionPatternsLock.RUnlock()
	if ok {
		return patterns
	}

	// Slow path: compile and cache
	engineToken := engine
	switch engine {
	case "Blink":
		engineToken = `Chr[o0]me|Chromium|Cronet`
	case "Arachne":
		engineToken = `Arachne\/5\.`
	case "LibWeb":
		engineToken = `LibWeb\+LibJs`
	}

	patterns = &engineVersionRegexes{
		withDot: regexp.MustCompile(`(?i)(?:` + engineToken + `)\s*[/_]?\s*(\d+\.\d+(?:\.\d+)*)`),
		noDot:   regexp.MustCompile(`(?i)(?:` + engineToken + `)\s*[/_]?\s*(\d{1,7})(?:\D|$)`),
	}

	engineVersionPatternsLock.Lock()
	engineVersionPatterns[engine] = patterns
	engineVersionPatternsLock.Unlock()

	return patterns
}
