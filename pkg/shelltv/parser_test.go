package shelltv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsShellTv matches PHP ShellTvTest::testIsShellTv.
func TestIsShellTv(t *testing.T) {
	factory, err := NewParserFactory()
	assert.NoError(t, err, "failed to create factory")

	// Positive case
	assert.True(t, factory.IsShellTv("Leff Shell LC390TA2A"), "IsShellTv(positive)")

	// Negative case
	assert.False(t, factory.IsShellTv("Leff Shell"), "IsShellTv(negative)")
}
