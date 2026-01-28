// Package browser implements the Browser client parser.
package browser

import "github.com/archbottle/device-detector/pkg/common"

// Entry represents a single browser definition from the YAML regex file.
type Entry struct {
	Regex   string      `yaml:"regex"`
	Name    string      `yaml:"name"`
	Version string      `yaml:"version"`
	Engine  *EngineSpec `yaml:"engine,omitempty"`

	orderIdx int
	compiled common.UniversalRegexSubmatch
}

// EngineSpec represents the engine specification in a browser entry.
type EngineSpec struct {
	Default  string            `yaml:"default,omitempty"`
	Versions map[string]string `yaml:"versions,omitempty"`
}

// GetRegex is used for keyword extraction / indexing.
func (e *Entry) GetRegex() string { return e.Regex }

// Order preserves YAML order for deterministic "first match wins" selection.
func (e *Entry) Order() int { return e.orderIdx }

// EngineEntry represents a single engine definition from browser_engine.yml.
type EngineEntry struct {
	Regex string `yaml:"regex"`
	Name  string `yaml:"name"`

	orderIdx int
	compiled common.UniversalRegexSubmatch
}

// GetRegex is used for keyword extraction / indexing.
func (e *EngineEntry) GetRegex() string { return e.Regex }

// Order preserves YAML order for deterministic "first match wins" selection.
func (e *EngineEntry) Order() int { return e.orderIdx }

// Match represents the result of a successful browser detection.
// This matches the PHP return structure.
type Match struct {
	Type          string `json:"type" yaml:"type"`
	Name          string `json:"name" yaml:"name"`
	ShortName     string `json:"short_name" yaml:"short_name"`
	Version       string `json:"version" yaml:"version"`
	Engine        string `json:"engine" yaml:"engine"`
	EngineVersion string `json:"engine_version" yaml:"engine_version"`
	Family        string `json:"family" yaml:"family"`
}
