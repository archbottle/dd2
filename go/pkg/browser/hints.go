package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// BrowserHints maps app IDs to browser names.
type BrowserHints struct {
	hints map[string]string
}

// NewBrowserHints loads the browser hints from the YAML file.
func NewBrowserHints(hintsPath string) (*BrowserHints, error) {
	data, err := os.ReadFile(hintsPath)
	if err != nil {
		return nil, fmt.Errorf("reading browser hints file: %w", err)
	}

	var hints map[string]string
	if err := yaml.Unmarshal(data, &hints); err != nil {
		return nil, fmt.Errorf("parsing browser hints YAML: %w", err)
	}

	return &BrowserHints{hints: hints}, nil
}

// NewDefaultBrowserHints creates browser hints using the repo-local path.
func NewDefaultBrowserHints() (*BrowserHints, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to get caller info")
	}
	hintsPath := filepath.Join(filepath.Dir(filename), "..", "..", "regexes", "client", "hints", "browsers.yml")
	return NewBrowserHints(hintsPath)
}

// GetBrowserName returns the browser name for an app ID.
func (h *BrowserHints) GetBrowserName(appID string) string {
	if h == nil || h.hints == nil {
		return ""
	}
	return h.hints[appID]
}
