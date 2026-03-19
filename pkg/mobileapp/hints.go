package mobileapp

import (
	"fmt"

	"github.com/archbottle/dd2/regexes"
	"gopkg.in/yaml.v3"
)

// AppHints maps app IDs to mobile app names.
type AppHints struct {
	hints map[string]string
}

// NewAppHints loads mobile app hints from the embedded YAML DB.
func NewAppHints() (*AppHints, error) {
	data, err := regexes.FS.ReadFile("client/hints/apps.yml")
	if err != nil {
		return nil, fmt.Errorf("reading mobile app hints file: %w", err)
	}

	var hints map[string]string
	if err := yaml.Unmarshal(data, &hints); err != nil {
		return nil, fmt.Errorf("parsing mobile app hints YAML: %w", err)
	}

	return &AppHints{hints: hints}, nil
}

// NewDefaultAppHints is an alias for NewAppHints kept for compatibility.
func NewDefaultAppHints() (*AppHints, error) { return NewAppHints() }

// GetAppName returns the mobile app name for an app ID.
func (h *AppHints) GetAppName(appID string) string {
	if h == nil || h.hints == nil {
		return ""
	}
	return h.hints[appID]
}
