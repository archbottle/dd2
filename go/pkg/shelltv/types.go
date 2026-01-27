// Package shelltv provides Shell TV detection.
package shelltv

// Match represents the result of a successful Shell TV detection.
type Match struct {
	IsShellTv bool `json:"is_shell_tv"`
}
