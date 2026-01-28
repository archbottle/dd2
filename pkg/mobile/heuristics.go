package mobile

import (
	"regexp"
	"strings"

	"github.com/archbottle/device-detector/pkg/clienthints"
)

// Pre-compiled regexes for heuristics (compiled once at package init).
var (
	// Desktop fragment detection: Windows NT/IoT or X11 Linux x86_64
	reDesktopFragment = regexp.MustCompile(`(?i)(?:Windows (?:NT|IoT)|X11; Linux x86_64)`)

	// Exclusion patterns: if present, it's NOT a desktop (even if desktop fragment matches)
	// CE-HTML, Mozilla/, Android, Tablet, Mobile, iPhone, Windows Phone, etc.
	reDesktopExclude = regexp.MustCompile(`(?i)CE-HTML| Mozilla/|Andr[o0]id|Tablet|Mobile|iPhone|Windows Phone|ricoh|OculusBrowser|PicoBrowser|Lenovo|compatible; MSIE|Trident/|Tesla/|XBOX|FBMD/|ARM; ?(?:[^)]+)`)

	// Client hints fragment: Android 10-16 with "; K" (Chromium privacy mode)
	// Pattern: Android 10.x; K Build/ or Android 10; K) or Android 10-16)
	reClientHintsFragment = regexp.MustCompile(`(?i)Android (?:1[0-6][.\d]*; K(?: Build/|[;)])|1[0-6]\)) AppleWebKit`)

	// Pattern for restoring UA: replace "Android 10; K" or similar with actual version/model
	reRestoreAndroid = regexp.MustCompile(`(?i)Android (?:10[.\d]*; K|1[1-6])`)

	// Pattern for restoring UA for desktop: inject model into X11 fragment
	reRestoreX11 = regexp.MustCompile(`(?i)X11; Linux x86_64`)
)

// HasDesktopFragment returns true if the user agent appears to be from a desktop browser.
// When true, mobile device parsing should be skipped entirely.
//
// PHP: AbstractDeviceParser::hasDesktopFragment()
func HasDesktopFragment(ua string) bool {
	// Must have Windows NT/IoT or X11 Linux x86_64
	if !reDesktopFragment.MatchString(ua) {
		return false
	}

	// But must NOT have any exclusion patterns (Android, Mobile, XBOX, etc.)
	return !reDesktopExclude.MatchString(ua)
}

// HasClientHintsFragment returns true if the user agent uses the Android client hints format.
// This is detected by the "Android 10+; K" pattern that Chromium uses for privacy.
//
// Special case: Telegram-Android spoofs this pattern, so we exclude it.
//
// PHP: AbstractDeviceParser::hasUserAgentClientHintsFragment()
func HasClientHintsFragment(ua string) bool {
	if !reClientHintsFragment.MatchString(ua) {
		return false
	}

	// Telegram-Android lies about client hints, exclude it
	return !strings.Contains(strings.ToLower(ua), "telegram-android/")
}

// RestoreUserAgent injects client hints data (device model, OS version) back into the UA string.
// This allows YAML regex patterns to match against the reconstructed UA.
//
// PHP: AbstractDeviceParser::restoreUserAgentFromClientHints()
func RestoreUserAgent(ua string, ch *clienthints.ClientHints) string {
	if ch == nil {
		return ua
	}

	model := ch.GetModel()
	if model == "" {
		return ua
	}

	// Get OS version from client hints (or default to "10")
	osVersion := ch.GetOperatingSystemVersion()
	if osVersion == "" {
		osVersion = "10"
	}

	// For Android with client hints, replace "Android 10; K" with "Android {version}; {model}"
	if HasClientHintsFragment(ua) {
		replacement := "Android " + osVersion + "; " + model
		return reRestoreAndroid.ReplaceAllString(ua, replacement)
	}

	// For Desktop (X11 Linux) with client hints, inject model after x86_64
	if HasDesktopFragment(ua) {
		replacement := "X11; Linux x86_64; " + model
		return reRestoreX11.ReplaceAllString(ua, replacement)
	}

	return ua
}

// ClientHintFormFactors maps client hint form factors to device types.
// Used when client hints provide explicit device type information.
//
// PHP: AbstractDeviceParser::$clientHintFormFactorsMapping
var ClientHintFormFactors = map[string]string{
	"automotive": "car browser",
	"xr":         "wearable",
	"watch":      "wearable",
	"mobile":     "smartphone",
	"tablet":     "tablet",
	"desktop":    "desktop",
	"eink":       "tablet", // E-readers treated as tablets
}

// GetDeviceTypeFromFormFactor returns the device type for a client hints form factor.
// Returns empty string if the form factor is not recognized.
func GetDeviceTypeFromFormFactor(formFactor string) string {
	return ClientHintFormFactors[strings.ToLower(formFactor)]
}
