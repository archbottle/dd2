// Package operatingsystem implements the OperatingSystem parser.
package operatingsystem

import "github.com/archbottle/device-detector/pkg/common"

// Entry represents a single OS definition from the YAML regex file.
type Entry struct {
	Regex    string    `yaml:"regex"`
	Name     string    `yaml:"name"`
	Version  string    `yaml:"version"`
	Versions []Version `yaml:"versions,omitempty"`

	orderIdx int
	compiled common.UniversalRegexSubmatch
}

// Version represents a sub-pattern for version detection within an OS entry.
type Version struct {
	Regex   string `yaml:"regex"`
	Name    string `yaml:"name,omitempty"`
	Version string `yaml:"version"`

	compiled common.UniversalRegexSubmatch
}

// GetRegex is used for keyword extraction / indexing.
func (e *Entry) GetRegex() string { return e.Regex }

// Order preserves YAML order for deterministic "first match wins" selection.
func (e *Entry) Order() int { return e.orderIdx }

// Match represents the result of a successful OS detection.
// This matches the PHP return structure.
type Match struct {
	Name      string `json:"name" yaml:"name"`
	ShortName string `json:"short_name" yaml:"short_name"`
	Version   string `json:"version" yaml:"version"`
	Platform  string `json:"platform" yaml:"platform"`
	Family    string `json:"family" yaml:"family"`
}
