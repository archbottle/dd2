package mobile

import "github.com/archbottle/device-detector/pkg/common"

// Entry represents a brand's device patterns from mobiles.yml.
// Each entry contains a brand-level regex and optional model-specific sub-patterns.
type Entry struct {
	Brand  string  // Brand name (YAML key)
	Regex  string  // Brand-level regex pattern
	Device string  // Default device type (smartphone, tablet, phablet, feature phone)
	Model  string  // Brand-level model template (optional, uses $1..$n substitution)
	Models []Model // Model-specific patterns (optional)

	// Internal fields
	orderIdx       int                           // Position in YAML for deterministic matching
	compiledBrand  common.UniversalRegexSubmatch // Compiled brand regex
	compiledModels []common.UniversalRegexSubmatch
}

// GetRegex implements common.Pattern for keyword indexing.
func (e *Entry) GetRegex() string {
	return e.Regex
}

// GetPosition implements common.OrderedPattern.
func (e *Entry) GetPosition() int {
	return e.orderIdx
}

// SetPosition implements common.OrderedPattern.
func (e *Entry) SetPosition(pos int) {
	e.orderIdx = pos
}

// Model represents a specific model pattern within a brand.
// Models can override the parent brand's device type and even the brand name.
type Model struct {
	Regex  string // Model-specific regex pattern
	Model  string // Model name template ($1..$n substitution)
	Brand  string // Override brand (optional)
	Device string // Override device type (optional)
}

// Match represents a successful device detection result.
type Match struct {
	Type  string // Device type: smartphone, tablet, phablet, feature phone
	Brand string // Device brand (manufacturer)
	Model string // Device model name
}
