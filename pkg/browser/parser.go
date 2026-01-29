package browser

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/archbottle/dd2/pkg/clienthints"
	"github.com/archbottle/dd2/pkg/common"
)

// Pre-compiled regex patterns for Parse() - compiled once on first use
var (
	iridiumVersionPattern       *regexp.Regexp
	iridiumVersionPatternOnce   sync.Once
	secureVersion15Pattern      *regexp.Regexp
	secureVersion15PatternOnce  sync.Once
	secureVersion114Pattern     *regexp.Regexp
	secureVersion114PatternOnce sync.Once
	chromeSafariPattern         *regexp.Regexp
	chromeSafariPatternOnce     sync.Once
	cypressPhantomPattern       *regexp.Regexp
	cypressPhantomPatternOnce   sync.Once
)

func getIridiumVersionPattern() *regexp.Regexp {
	iridiumVersionPatternOnce.Do(func() {
		iridiumVersionPattern = regexp.MustCompile(`^202[0-4]`)
	})
	return iridiumVersionPattern
}

func getSecureVersion15Pattern() *regexp.Regexp {
	secureVersion15PatternOnce.Do(func() {
		secureVersion15Pattern = regexp.MustCompile(`^15`)
	})
	return secureVersion15Pattern
}

func getSecureVersion114Pattern() *regexp.Regexp {
	secureVersion114PatternOnce.Do(func() {
		secureVersion114Pattern = regexp.MustCompile(`^114`)
	})
	return secureVersion114Pattern
}

func getChromeSafariPattern() *regexp.Regexp {
	chromeSafariPatternOnce.Do(func() {
		chromeSafariPattern = regexp.MustCompile(`(?i)Chrome/.+ Safari/537.36`)
	})
	return chromeSafariPattern
}

func getCypressPhantomPattern() *regexp.Regexp {
	cypressPhantomPatternOnce.Do(func() {
		cypressPhantomPattern = regexp.MustCompile(`Cypress|PhantomJS`)
	})
	return cypressPhantomPattern
}

// Parser parses a single user agent for browser client information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory     *ParserFactory
	userAgent   string
	clientHints *clienthints.ClientHints
}

// Parse detects browsers and returns a Match, or nil if not detected.
// Mirrors PHP: DeviceDetector\Parser\Client\Browser::parse(): ?array
func (p *Parser) Parse() *Match {
	if p.factory == nil || p.userAgent == "" {
		return nil
	}

	browserFromClientHints := p.parseBrowserFromClientHints()
	browserFromUserAgent := p.parseBrowserFromUserAgent()

	var name, version, short, engine, engineVersion string

	// Use client hints in favor of user agent data if possible
	if browserFromClientHints.Name != "" && browserFromClientHints.Version != "" {
		name = browserFromClientHints.Name
		version = browserFromClientHints.Version
		short = browserFromClientHints.ShortName

		// If the version reported from the client hints is YYYY or YYYY.MM (e.g., 2022 or 2022.04),
		// then it is the Iridium browser
		if getIridiumVersionPattern().MatchString(version) {
			name = "Iridium"
			short = "I1"
		}

		// https://bbs.360.cn/thread-16096544-1-1.html
		if getSecureVersion15Pattern().MatchString(version) {
			if getSecureVersion114Pattern().MatchString(browserFromUserAgent.Version) {
				name = "360 Secure Browser"
				short = "3B"
				engine = browserFromUserAgent.Engine
				engineVersion = browserFromUserAgent.EngineVersion
			}
		}

		// If client hints report the following browsers, we use the version from useragent
		if browserFromUserAgent.Version != "" {
			useUAVersion := []string{"A0", "AL", "HP", "JR", "MU", "OM", "OP", "VR"}
			for _, s := range useUAVersion {
				if short == s {
					version = browserFromUserAgent.Version
					break
				}
			}
		}

		if name == "Vewd Browser" {
			engine = browserFromUserAgent.Engine
			engineVersion = browserFromUserAgent.EngineVersion
		}

		// If client hints report Chromium, but user agent detects a Chromium based browser, we favor this instead
		if (name == "Chromium" || name == "Chrome Webview") && browserFromUserAgent.Name != "" {
			notChromiumBased := []string{"CR", "CV", "AN", "CM"}
			isChromiumBased := true
			for _, s := range notChromiumBased {
				if browserFromUserAgent.ShortName == s {
					isChromiumBased = false
					break
				}
			}
			if isChromiumBased {
				name = browserFromUserAgent.Name
				short = browserFromUserAgent.ShortName
				version = browserFromUserAgent.Version
			}
		}

		// Fix mobile browser names e.g. Chrome => Chrome Mobile
		if name+" Mobile" == browserFromUserAgent.Name {
			name = browserFromUserAgent.Name
			short = browserFromUserAgent.ShortName
		}

		// If user agent detects another browser, but the family matches, we use the detected engine from user agent
		if name != browserFromUserAgent.Name &&
			GetBrowserFamily(name) == GetBrowserFamily(browserFromUserAgent.Name) {
			engine = browserFromUserAgent.Engine
			engineVersion = browserFromUserAgent.EngineVersion
		}

		if name == browserFromUserAgent.Name {
			engine = browserFromUserAgent.Engine
			engineVersion = browserFromUserAgent.EngineVersion
		}

		// In case the user agent reports a more detailed version, we try to use this instead
		if browserFromUserAgent.Version != "" &&
			strings.HasPrefix(browserFromUserAgent.Version, version) &&
			compareVersions(version, browserFromUserAgent.Version) < 0 {
			version = browserFromUserAgent.Version
		}

		if name == "DuckDuckGo Privacy Browser" {
			version = ""
		}

		// In case client hints report a more detailed engine version, we try to use this instead
		if engine == "Blink" && name != "Iridium" &&
			compareVersions(engineVersion, browserFromClientHints.Version) < 0 {
			engineVersion = browserFromClientHints.Version
		}
	} else {
		name = browserFromUserAgent.Name
		version = browserFromUserAgent.Version
		short = browserFromUserAgent.ShortName
		engine = browserFromUserAgent.Engine
		engineVersion = browserFromUserAgent.EngineVersion
	}

	family := GetBrowserFamily(short)

	// Check browser hints from app ID
	appHash := p.parseBrowserHints()
	if appHash != "" && name != appHash {
		name = appHash
		version = ""
		short = GetBrowserShortName(name)

		if getChromeSafariPattern().MatchString(p.userAgent) {
			engine = "Blink"
			if f := GetBrowserFamily(short); f != "" {
				family = f
			} else {
				family = "Chrome"
			}
			engineVersion = p.buildEngineVersion(engine)
		}

		if short == "" {
			// This should never happen - browser name missing from AvailableBrowsers
			return nil
		}
	}

	// Exclude certain patterns
	if name == "" {
		return nil
	}

	if getCypressPhantomPattern().MatchString(p.userAgent) {
		return nil
	}

	// Exclude Blink engine version for Flow Browser
	if engine == "Blink" && name == "Flow Browser" {
		engineVersion = ""
	}

	// The browser simulate ua for Android OS
	if name == "Every Browser" {
		family = "Chrome"
		engine = "Blink"
		engineVersion = ""
	}

	// This browser simulates user-agent of Firefox
	if name == "TV-Browser Internet" && engine == "Gecko" {
		family = "Chrome"
		engine = "Blink"
		engineVersion = ""
	}

	if (name == "Yaani Browser" || name == "Wolvic") && engine == "Blink" {
		family = "Chrome"
	}

	if (name == "Yaani Browser" || name == "Wolvic") && engine == "Gecko" {
		family = "Firefox"
	}

	return &Match{
		Type:          "browser",
		Name:          name,
		ShortName:     short,
		Version:       version,
		Engine:        engine,
		EngineVersion: engineVersion,
		Family:        family,
	}
}

// parseBrowserFromClientHints returns browser info from client hints.
func (p *Parser) parseBrowserFromClientHints() *Match {
	result := &Match{}

	if p.clientHints == nil {
		return result
	}

	brandList := p.clientHints.GetBrandList()
	if len(brandList) == 0 {
		return result
	}

	for brand, brandVersion := range brandList {
		brand = applyClientHintMapping(brand)

		for browserShort, browserName := range AvailableBrowsers {
			if fuzzyCompare(brand, browserName) ||
				fuzzyCompare(brand+" Browser", browserName) ||
				fuzzyCompare(brand, browserName+" Browser") {
				result.Name = browserName
				result.ShortName = browserShort
				result.Version = buildVersion(brandVersion, nil)
				break
			}
		}

		// If we detected a brand that is not Chromium, we use it
		if result.Name != "" && result.Name != "Chromium" && result.Name != "Microsoft Edge" {
			break
		}
	}

	if brandVersion := p.clientHints.GetBrandVersion(); brandVersion != "" {
		result.Version = buildVersion(brandVersion, nil)
	}

	return result
}

// parseBrowserFromUserAgent returns browser info from user agent.
func (p *Parser) parseBrowserFromUserAgent() *Match {
	result := &Match{}

	candidates := p.factory.db.Candidates(p.userAgent)
	var matchedEntry *Entry
	var matches []string

	for _, e := range candidates {
		if e == nil || e.compiled == nil {
			continue
		}

		m, err := e.compiled.FindStringSubmatch(p.userAgent)
		if err != nil || len(m) == 0 {
			continue
		}

		matches = m
		matchedEntry = e
		break
	}

	// Fallback scan in compatibility mode
	if matchedEntry == nil && p.factory.db != nil && p.factory.db.Index != nil && p.factory.db.Mode == common.Compatibility {
		for _, e := range p.factory.patterns {
			if e == nil || e.compiled == nil {
				continue
			}

			m, err := e.compiled.FindStringSubmatch(p.userAgent)
			if err != nil || len(m) == 0 {
				continue
			}

			matches = m
			matchedEntry = e
			break
		}
	}

	if matchedEntry == nil {
		return result
	}

	name := strings.TrimSpace(buildByMatch(matchedEntry.Name, matches))
	browserShort := GetBrowserShortName(name)

	if browserShort == "" {
		return result
	}

	version := buildVersion(matchedEntry.Version, matches)
	engine := p.buildEngine(matchedEntry.Engine, version)
	engineVersion := p.buildEngineVersion(engine)

	result.Name = name
	result.ShortName = browserShort
	result.Version = version
	result.Engine = engine
	result.EngineVersion = engineVersion

	return result
}

// parseBrowserHints returns browser name from app ID hints.
func (p *Parser) parseBrowserHints() string {
	if p.clientHints == nil || p.factory.browserHints == nil {
		return ""
	}

	appID := p.clientHints.GetApp()
	if appID == "" {
		return ""
	}

	return p.factory.browserHints.GetBrowserName(appID)
}

// buildEngine determines the browser engine based on the entry and version.
// This mirrors PHP's Browser::buildEngine().
func (p *Parser) buildEngine(engineSpec *EngineSpec, browserVersion string) string {
	engine := ""

	// If an engine is set as default
	if engineSpec != nil && engineSpec.Default != "" {
		engine = engineSpec.Default
	}

	// Check if engine is set for browser version
	// PHP iterates through versions and picks the one where browserVersion >= version
	// We need to find the highest version that browserVersion is >= to
	if engineSpec != nil && engineSpec.Versions != nil && browserVersion != "" {
		// Collect all matching versions and pick the highest one
		var bestVersion string
		var bestEngine string

		for version, versionEngine := range engineSpec.Versions {
			// Only consider if browserVersion >= this version
			if compareVersions(browserVersion, version) >= 0 {
				// Check if this version is higher than our current best
				if bestVersion == "" || compareVersions(version, bestVersion) > 0 {
					bestVersion = version
					bestEngine = versionEngine
				}
			}
		}

		if bestEngine != "" {
			engine = bestEngine
		}
	}

	// Try to detect the engine using the regexes
	if engine == "" && p.factory.engineParser != nil {
		engine = p.factory.engineParser.Parse(p.userAgent)
	}

	return engine
}

// buildEngineVersion extracts the engine version from the user agent.
func (p *Parser) buildEngineVersion(engine string) string {
	return ParseEngineVersion(p.userAgent, engine)
}

// buildByMatch substitutes $1..$n with corresponding capture groups.
func buildByMatch(template string, matches []string) string {
	if template == "" || len(matches) == 0 {
		return template
	}
	out := template
	// Replace from high to low to avoid $10 being partially replaced as $1 + "0"
	for i := len(matches) - 1; i >= 1; i-- {
		out = strings.ReplaceAll(out, "$"+strconv.Itoa(i), matches[i])
	}
	return out
}

// buildVersion processes a version template with matches.
func buildVersion(template string, matches []string) string {
	v := strings.TrimSpace(buildByMatch(template, matches))
	if v == "" {
		return ""
	}
	v = strings.ReplaceAll(v, "_", ".")
	v = strings.Trim(v, " .")

	// If version still contains unreplaced placeholder like $1, $2, etc., return empty
	// This happens when the regex has no capture group but version template references one
	if strings.Contains(v, "$") {
		return ""
	}

	return v
}

// compareVersions compares two version strings.
// Returns -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2.
// This mirrors PHP's version_compare behavior where "112" < "112.0.0.0"
// (a longer version string with the same prefix is considered greater).
func compareVersions(v1, v2 string) int {
	if v1 == "" && v2 == "" {
		return 0
	}
	if v1 == "" {
		return -1
	}
	if v2 == "" {
		return 1
	}

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		var has1, has2 bool

		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
			has1 = true
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
			has2 = true
		}

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}

		// If values are equal but one has the part and one doesn't,
		// the one with more parts is greater (PHP behavior)
		if has1 && !has2 {
			return 1
		}
		if !has1 && has2 {
			return -1
		}
	}

	return 0
}

// applyClientHintMapping applies the client hint mapping.
func applyClientHintMapping(brand string) string {
	for browserName, hintNames := range ClientHintMapping {
		for _, hintName := range hintNames {
			if strings.EqualFold(brand, hintName) {
				return browserName
			}
		}
	}
	return brand
}

// fuzzyCompare performs a case-insensitive comparison with some normalization.
func fuzzyCompare(s1, s2 string) bool {
	// Normalize strings for comparison
	normalize := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "-", "")
		s = strings.ReplaceAll(s, "_", "")
		return s
	}
	return normalize(s1) == normalize(s2)
}

// GetBrowserShortName returns the short name for a browser.
func GetBrowserShortName(name string) string {
	nameLower := strings.ToLower(name)
	for short, browserName := range AvailableBrowsers {
		if strings.ToLower(browserName) == nameLower {
			return short
		}
	}
	return ""
}

// GetBrowserFamily returns the family for a browser (by short name or name).
func GetBrowserFamily(browserLabel string) string {
	// Check if browserLabel is a browser name (convert to short)
	if _, exists := AvailableBrowsers[browserLabel]; !exists {
		// It's a name, find the short code
		for short, name := range AvailableBrowsers {
			if name == browserLabel {
				browserLabel = short
				break
			}
		}
	}

	for family, codes := range BrowserFamilies {
		for _, code := range codes {
			if code == browserLabel {
				return family
			}
		}
	}
	return ""
}

// IsMobileOnlyBrowser checks if a browser is mobile-only.
func IsMobileOnlyBrowser(browser string) bool {
	// Check by short code
	for _, code := range MobileOnlyBrowsers {
		if code == browser {
			return true
		}
	}
	// Check by name
	if short := GetBrowserShortName(browser); short != "" {
		for _, code := range MobileOnlyBrowsers {
			if code == short {
				return true
			}
		}
	}
	return false
}

// GetAvailableBrowsers returns all available browsers.
func GetAvailableBrowsers() map[string]string {
	return AvailableBrowsers
}

// GetAvailableBrowserFamilies returns all available browser families.
func GetAvailableBrowserFamilies() map[string][]string {
	return BrowserFamilies
}

// GetAvailableEngines returns all available browser engines.
func GetAvailableEngines() []string {
	return AvailableEngines
}
