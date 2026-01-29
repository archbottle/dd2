// Package feedreader implements the FeedReader client parser.
package feedreader

import "github.com/archbottle/dd2/pkg/common"

// Entry represents a single feed reader definition from the YAML regex file.
type Entry struct {
	Regex   string `yaml:"regex"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	URL     string `yaml:"url"`

	compiled common.UniversalRegexSubmatch
}

// GetRegex is used for keyword extraction / indexing.
func (e *Entry) GetRegex() string { return e.Regex }

// Match represents the result of a successful feed reader detection.
// This matches the PHP return structure (type/name/version).
type Match struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
}
