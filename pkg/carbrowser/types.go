package carbrowser

import "github.com/archbottle/device-detector/pkg/common"

// Match is the parsed device result for car browser detection.
// Mirrors the PHP test fixtures expectation: {type, brand, model}.
type Match struct {
	Type  string `json:"type"`
	Brand string `json:"brand"`
	Model string `json:"model"`
}

// Entry represents a single brand entry from car_browsers.yml.
type Entry struct {
	Brand  string
	Regex  string  `yaml:"regex"`
	Device string  `yaml:"device"`
	Model  string  `yaml:"model,omitempty"`
	Models []Model `yaml:"models,omitempty"`

	orderIdx int

	// Compiled matchers (built once in the factory).
	compiledBrand  common.UniversalRegexSubmatch
	compiledModels []common.UniversalRegexSubmatch
}

// GetRegex implements common.Pattern for keyword indexing.
func (e *Entry) GetRegex() string {
	return e.Regex
}

// Order preserves YAML order for deterministic "first match wins" selection.
func (e *Entry) Order() int { return e.orderIdx }

// Model represents a model-specific rule inside an entry.
type Model struct {
	Regex  string `yaml:"regex"`
	Model  string `yaml:"model"`
	Brand  string `yaml:"brand,omitempty"`
	Device string `yaml:"device,omitempty"`
}
