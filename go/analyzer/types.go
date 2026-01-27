package analyzer

// BrowserEntry represents a browser pattern from browsers.yml
type BrowserEntry struct {
	Regex   string         `yaml:"regex"`
	Name    string         `yaml:"name"`
	Version string         `yaml:"version"`
	Engine  map[string]any `yaml:"engine,omitempty"`
}

// DeviceBrand represents a device brand with its patterns from mobiles.yml
type DeviceBrand struct {
	Regex  string        `yaml:"regex"`
	Device string        `yaml:"device"`
	Model  string        `yaml:"model,omitempty"`
	Models []DeviceModel `yaml:"models,omitempty"`
}

// DeviceModel represents a specific device model pattern
type DeviceModel struct {
	Regex  string `yaml:"regex"`
	Model  string `yaml:"model"`
	Brand  string `yaml:"brand,omitempty"`
	Device string `yaml:"device,omitempty"`
}

// OSEntry represents an OS pattern from oss.yml
type OSEntry struct {
	Regex    string      `yaml:"regex"`
	Name     string      `yaml:"name"`
	Version  string      `yaml:"version"`
	Versions []OSVersion `yaml:"versions,omitempty"`
}

// OSVersion represents version-specific OS patterns
type OSVersion struct {
	Regex   string `yaml:"regex"`
	Name    string `yaml:"name,omitempty"`
	Version string `yaml:"version,omitempty"`
}

// PatternInfo holds analyzed information about a regex pattern
type PatternInfo struct {
	Category       string   `yaml:"category"` // "browser", "os", "device"
	Name           string   `yaml:"name"`     // Browser/OS/Brand name
	OriginalRegex  string   `yaml:"original_regex"`
	Keywords       []string `yaml:"keywords"`                  // Extracted keywords for indexing
	HasLookaround  bool     `yaml:"has_lookaround"`            // Needs regexp2?
	LookaroundType string   `yaml:"lookaround_type,omitempty"` // Which type
	IsRE2Safe      bool     `yaml:"is_re2_safe"`               // Can use Go's regexp?
}

// KeywordIndex maps keywords to pattern names
type KeywordIndex struct {
	Keywords map[string][]string `yaml:"keywords"` // keyword -> [pattern names]
}

// AnalysisResult contains the full analysis output
type AnalysisResult struct {
	Summary struct {
		TotalPatterns   int `yaml:"total_patterns"`
		RE2SafePatterns int `yaml:"re2_safe_patterns"`
		Regexp2Patterns int `yaml:"regexp2_patterns"`
		IndexedKeywords int `yaml:"indexed_keywords"`
	} `yaml:"summary"`

	BrowserIndex KeywordIndex `yaml:"browser_index"`
	OSIndex      KeywordIndex `yaml:"os_index"`
	DeviceIndex  KeywordIndex `yaml:"device_index"`

	Patterns []PatternInfo `yaml:"patterns"`
}
