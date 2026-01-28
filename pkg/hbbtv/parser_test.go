package hbbtv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsHbbTv matches PHP HbbTvTest::testIsHbbTv.
func TestIsHbbTv(t *testing.T) {
	factory, err := NewParserFactory()
	assert.NoError(t, err, "failed to create factory")

	ua := "Opera/9.80 (Linux mips ; U; HbbTV/1.1.1 (; Philips; ; ; ; ) CE-HTML/1.0 NETTV/3.2.1; en) Presto/2.6.33 Version/10.70" // nolint:lll // UA

	got := factory.IsHbbTv(ua)
	want := "1.1.1"

	assert.Equal(t, want, got, "IsHbbTv()")
}
