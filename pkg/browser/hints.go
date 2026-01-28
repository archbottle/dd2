package browser

import (
	"fmt"

	"github.com/archbottle/device-detector/regexes"
	"gopkg.in/yaml.v3"
)

// BrowserHints maps app IDs to browser names.
type BrowserHints struct {
	hints map[string]string
}

// NewBrowserHints loads the browser hints from the embedded YAML DB.
func NewBrowserHints() (*BrowserHints, error) {
	data, err := regexes.FS.ReadFile("client/hints/browsers.yml")
	if err != nil {
		return nil, fmt.Errorf("reading browser hints file: %w", err)
	}

	var hints map[string]string
	if err := yaml.Unmarshal(data, &hints); err != nil {
		return nil, fmt.Errorf("parsing browser hints YAML: %w", err)
	}

	return &BrowserHints{hints: hints}, nil
}

// NewDefaultBrowserHints is an alias for NewBrowserHints kept for compatibility.
func NewDefaultBrowserHints() (*BrowserHints, error) { return NewBrowserHints() }

// GetBrowserName returns the browser name for an app ID.
func (h *BrowserHints) GetBrowserName(appID string) string {
	if h == nil || h.hints == nil {
		return ""
	}
	return h.hints[appID]
}
