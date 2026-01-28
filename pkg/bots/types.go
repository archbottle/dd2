// Package bots provides bot detection functionality for user agent parsing.
package bots

// Producer represents the producer/company that created the bot.
type Producer struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// BotEntry represents a single bot definition from the YAML regex file.
type BotEntry struct {
	Regex    string    `yaml:"regex"`
	Name     string    `yaml:"name"`
	Category string    `yaml:"category,omitempty"`
	URL      string    `yaml:"url,omitempty"`
	Producer *Producer `yaml:"producer,omitempty"`

	// position tracks the original order in the YAML file for first-match-wins semantics
	position int
}

// GetRegex implements the common.Pattern interface.
func (b *BotEntry) GetRegex() string {
	return b.Regex
}

// GetPosition implements the common.OrderedPattern interface.
func (b *BotEntry) GetPosition() int {
	return b.position
}

// SetPosition implements the common.OrderedPattern interface.
func (b *BotEntry) SetPosition(pos int) {
	b.position = pos
}

// BotMatch represents the result of a successful bot detection.
// This matches the PHP return structure.
type BotMatch struct {
	Name     string    `json:"name"`
	Category string    `json:"category,omitempty"`
	URL      string    `json:"url,omitempty"`
	Producer *Producer `json:"producer,omitempty"`
}
