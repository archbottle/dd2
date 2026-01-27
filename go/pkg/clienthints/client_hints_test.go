package clienthints

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHeaders tests standard HTTP headers parsing (lowercase format).
// PHP equivalent: ClientHintsTest::testHeaders()
func TestHeaders(t *testing.T) {
	headers := map[string]interface{}{
		"sec-ch-ua":                  `"Opera";v="83", " Not;A Brand";v="99", "Chromium";v="98"`,
		"sec-ch-ua-mobile":           "?0",
		"sec-ch-ua-platform":         "Windows",
		"sec-ch-ua-platform-version": "14.0.0",
	}

	ch := Factory(headers)

	assert.False(t, ch.IsMobile())
	assert.Equal(t, "Windows", ch.GetOperatingSystem())
	assert.Equal(t, "14.0.0", ch.GetOperatingSystemVersion())
	assert.Equal(t, map[string]string{
		"Opera":        "83",
		" Not;A Brand": "99",
		"Chromium":     "98",
	}, ch.GetBrandList())
}

// TestHeadersHttp tests PHP $_SERVER style headers (HTTP_ prefix).
// PHP equivalent: ClientHintsTest::testHeadersHttp()
func TestHeadersHttp(t *testing.T) {
	headers := map[string]interface{}{
		"HTTP_SEC_CH_UA_FULL_VERSION_LIST": `" Not A;Brand";v="99.0.0.0", "Chromium";v="98.0.4758.82", "Opera";v="98.0.4758.82"`,
		"HTTP_SEC_CH_UA":                   `" Not A;Brand";v="99", "Chromium";v="98", "Opera";v="84"`,
		"HTTP_SEC_CH_UA_MOBILE":            "?1",
		"HTTP_SEC_CH_UA_MODEL":             "DN2103",
		"HTTP_SEC_CH_UA_PLATFORM":          "Ubuntu",
		"HTTP_SEC_CH_UA_PLATFORM_VERSION":  "3.7",
		"HTTP_SEC_CH_UA_FULL_VERSION":      "98.0.14335.105",
		"HTTP_SEC_CH_UA_FORM_FACTORS":      `"Desktop"`,
	}

	ch := Factory(headers)

	assert.True(t, ch.IsMobile())
	assert.Equal(t, "Ubuntu", ch.GetOperatingSystem())
	assert.Equal(t, "3.7", ch.GetOperatingSystemVersion())
	// Full version list takes priority over sec-ch-ua
	assert.Equal(t, map[string]string{
		" Not A;Brand": "99.0.0.0",
		"Chromium":     "98.0.4758.82",
		"Opera":        "98.0.4758.82",
	}, ch.GetBrandList())
	assert.Equal(t, "DN2103", ch.GetModel())
	assert.Equal(t, []string{"desktop"}, ch.GetFormFactors())
}

// TestHeadersJavascript tests JavaScript navigator.userAgentData API format.
// PHP equivalent: ClientHintsTest::testHeadersJavascript()
func TestHeadersJavascript(t *testing.T) {
	headers := map[string]interface{}{
		"fullVersionList": []interface{}{
			map[string]interface{}{"brand": " Not A;Brand", "version": "99.0.0.0"},
			map[string]interface{}{"brand": "Chromium", "version": "99.0.4844.51"},
			map[string]interface{}{"brand": "Google Chrome", "version": "99.0.4844.51"},
		},
		"formFactors":     []interface{}{"Desktop"},
		"mobile":          false,
		"model":           "",
		"platform":        "Windows",
		"platformVersion": "10.0.0",
	}

	ch := Factory(headers)

	assert.False(t, ch.IsMobile())
	assert.Equal(t, "Windows", ch.GetOperatingSystem())
	assert.Equal(t, "10.0.0", ch.GetOperatingSystemVersion())
	assert.Equal(t, map[string]string{
		" Not A;Brand":  "99.0.0.0",
		"Chromium":      "99.0.4844.51",
		"Google Chrome": "99.0.4844.51",
	}, ch.GetBrandList())
	assert.Equal(t, "", ch.GetModel())
	assert.Equal(t, []string{"desktop"}, ch.GetFormFactors())
}

// TestIncorrectVersionListIsDiscarded tests that invalid entries discard the entire list.
// PHP equivalent: ClientHintsTest::testIncorrectVersionListIsDiscarded()
func TestIncorrectVersionListIsDiscarded(t *testing.T) {
	headers := map[string]interface{}{
		"fullVersionList": []interface{}{
			map[string]interface{}{"brand": " Not A;Brand", "version": "99.0.0.0"},
			map[string]interface{}{"brand": "Chromium", "version": "99.0.4844.51"},
			map[string]interface{}{"version": "99.0.4844.51"}, // this entry lacks a brand
		},
	}

	ch := Factory(headers)

	assert.Equal(t, map[string]string{}, ch.GetBrandList())
}

// TestNewWithStringHeaders tests the New() convenience function.
func TestNewWithStringHeaders(t *testing.T) {
	headers := map[string]string{
		"sec-ch-ua":          `"Chrome";v="120", "Chromium";v="120"`,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": "macOS",
	}

	ch := New(headers)

	assert.False(t, ch.IsMobile())
	assert.Equal(t, "macOS", ch.GetOperatingSystem())
	assert.Equal(t, map[string]string{
		"Chrome":   "120",
		"Chromium": "120",
	}, ch.GetBrandList())
}

// TestMobileTrueValues tests various true values for mobile flag.
func TestMobileTrueValues(t *testing.T) {
	testCases := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"?1", "?1", true},
		{"1", "1", true},
		{"true string", "true", true},
		{"bool true", true, true},
		{"?0", "?0", false},
		{"0", "0", false},
		{"empty", "", false},
		{"bool false", false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ch := Factory(map[string]interface{}{
				"sec-ch-ua-mobile": tc.value,
			})
			assert.Equal(t, tc.expected, ch.IsMobile())
		})
	}
}

// TestFormFactorsParsing tests various form factors formats.
func TestFormFactorsParsing(t *testing.T) {
	testCases := []struct {
		name     string
		value    interface{}
		expected []string
	}{
		{
			name:     "string with quotes",
			value:    `"Desktop"`,
			expected: []string{"desktop"},
		},
		{
			name:     "string multiple values",
			value:    `"EInk", "Watch"`,
			expected: []string{"eink", "watch"},
		},
		{
			name:     "array of strings",
			value:    []string{"Desktop", "Mobile"},
			expected: []string{"desktop", "mobile"},
		},
		{
			name:     "array of interfaces",
			value:    []interface{}{"Tablet", "Automotive"},
			expected: []string{"tablet", "automotive"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ch := Factory(map[string]interface{}{
				"sec-ch-ua-form-factors": tc.value,
			})
			assert.Equal(t, tc.expected, ch.GetFormFactors())
		})
	}
}

// TestAppHeader tests the X-Requested-With header parsing.
func TestAppHeader(t *testing.T) {
	// Normal app name should be captured
	ch := Factory(map[string]interface{}{
		"x-requested-with": "com.example.app",
	})
	assert.Equal(t, "com.example.app", ch.GetApp())

	// XMLHttpRequest should be ignored
	ch = Factory(map[string]interface{}{
		"x-requested-with": "XMLHttpRequest",
	})
	assert.Equal(t, "", ch.GetApp())

	// Case insensitive
	ch = Factory(map[string]interface{}{
		"x-requested-with": "xmlhttprequest",
	})
	assert.Equal(t, "", ch.GetApp())
}

// TestArchitectureAndBitness tests architecture and bitness parsing.
func TestArchitectureAndBitness(t *testing.T) {
	ch := Factory(map[string]interface{}{
		"sec-ch-ua-arch":    `"x86"`,
		"sec-ch-ua-bitness": `"64"`,
	})

	assert.Equal(t, "x86", ch.GetArchitecture())
	assert.Equal(t, "64", ch.GetBitness())
}

// TestFullVersionPriority tests that full-version-list takes priority over sec-ch-ua.
func TestFullVersionPriority(t *testing.T) {
	headers := map[string]interface{}{
		"sec-ch-ua":                   `"Chrome";v="98"`,
		"sec-ch-ua-full-version-list": `"Chrome";v="98.0.4758.102"`,
	}

	ch := Factory(headers)

	// Full version should be used
	brandList := ch.GetBrandList()
	assert.Equal(t, "98.0.4758.102", brandList["Chrome"])
}

// TestEmptyHeaders tests that empty headers produce empty results.
func TestEmptyHeaders(t *testing.T) {
	ch := Factory(map[string]interface{}{})

	assert.False(t, ch.IsMobile())
	assert.Equal(t, "", ch.GetOperatingSystem())
	assert.Equal(t, "", ch.GetOperatingSystemVersion())
	assert.Equal(t, "", ch.GetModel())
	assert.Equal(t, "", ch.GetApp())
	assert.Equal(t, "", ch.GetArchitecture())
	assert.Equal(t, "", ch.GetBitness())
	assert.Equal(t, map[string]string{}, ch.GetBrandList())
	assert.Equal(t, []string{}, ch.GetFormFactors())
}

// TestNilValues tests that nil values are handled gracefully.
func TestNilValues(t *testing.T) {
	headers := map[string]interface{}{
		"sec-ch-ua-platform": nil,
		"sec-ch-ua-mobile":   nil,
	}

	ch := Factory(headers)

	assert.Equal(t, "", ch.GetOperatingSystem())
	assert.False(t, ch.IsMobile())
}
