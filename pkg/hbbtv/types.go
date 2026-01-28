// Package hbbtv provides HbbTV (Hybrid Broadcast Broadband TV) detection.
package hbbtv

// Match represents the result of a successful HbbTV detection.
// It mirrors the PHP behavior of returning a version string from isHbbTv().
type Match struct {
	Version string `json:"version"`
}
