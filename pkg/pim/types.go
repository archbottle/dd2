// Package pim implements the PIM client parser.
package pim

import "github.com/archbottle/device-detector/pkg/common"

// Entry represents a single PIM definition from the YAML regex file.
type Entry struct {
	Regex   string `yaml:"regex"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`

	orderIdx int
	compiled common.UniversalRegexSubmatch
}

// GetRegex is used for keyword extraction / indexing.
func (e *Entry) GetRegex() string { return e.Regex }

// Order preserves YAML order for deterministic "first match wins" selection.
func (e *Entry) Order() int { return e.orderIdx }

// Match represents the result of a successful PIM detection.
// This matches the PHP return structure (type/name/version).
type Match struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
}
