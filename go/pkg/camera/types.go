// Package camera provides digital camera detection based on Device Detector regexes.
package camera

// Match represents the parsed camera device match.
type Match struct {
	// Type is the Device Detector device type name, e.g. "camera".
	Type string `json:"type"`
	// Brand is the full brand name, e.g. "Samsung".
	Brand string `json:"brand"`
	// Model is the detected model string.
	Model string `json:"model"`
}
