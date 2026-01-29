package operatingsystem

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/archbottle/dd2/pkg/clienthints"
	"github.com/archbottle/dd2/pkg/common"
)

// Pre-compiled regex patterns for platform detection
var (
	armPattern       *regexp.Regexp
	armPatternOnce   sync.Once
	loongPattern     *regexp.Regexp
	loongPatternOnce sync.Once
	mipsPattern      *regexp.Regexp
	mipsPatternOnce  sync.Once
	sh4Pattern       *regexp.Regexp
	sh4PatternOnce   sync.Once
	sparc64Pattern   *regexp.Regexp
	sparc64Once      sync.Once
	x64Pattern       *regexp.Regexp
	x64PatternOnce   sync.Once
	x86Pattern       *regexp.Regexp
	x86PatternOnce   sync.Once

	// Patterns for restoreUserAgentFromClientHints
	androidHintsFragmentPattern     *regexp.Regexp
	androidHintsFragmentPatternOnce sync.Once
	androidReplacePattern           *regexp.Regexp
	androidReplacePatternOnce       sync.Once
	desktopFragmentPattern          *regexp.Regexp
	desktopFragmentPatternOnce      sync.Once
	desktopExcludePattern           *regexp.Regexp
	desktopExcludePatternOnce       sync.Once
)

func getArmPattern() *regexp.Regexp {
	armPatternOnce.Do(func() {
		armPattern = regexp.MustCompile(`(?i)arm[ _;)ev]|.*arm$|.*arm64|aarch64|Apple ?TV|Watch ?OS|Watch1,[12]`)
	})
	return armPattern
}

func getLoongPattern() *regexp.Regexp {
	loongPatternOnce.Do(func() {
		loongPattern = regexp.MustCompile(`(?i)loongarch64`)
	})
	return loongPattern
}

func getMipsPattern() *regexp.Regexp {
	mipsPatternOnce.Do(func() {
		mipsPattern = regexp.MustCompile(`(?i)mips`)
	})
	return mipsPattern
}

func getSh4Pattern() *regexp.Regexp {
	sh4PatternOnce.Do(func() {
		sh4Pattern = regexp.MustCompile(`(?i)sh4`)
	})
	return sh4Pattern
}

func getSparc64Pattern() *regexp.Regexp {
	sparc64Once.Do(func() {
		sparc64Pattern = regexp.MustCompile(`(?i)sparc64`)
	})
	return sparc64Pattern
}

func getX64Pattern() *regexp.Regexp {
	x64PatternOnce.Do(func() {
		x64Pattern = regexp.MustCompile(`(?i)64-?bit|WOW64|(?:Intel)?x64|WINDOWS_64|win64|.*amd64|.*x86_?64`)
	})
	return x64Pattern
}

func getX86Pattern() *regexp.Regexp {
	x86PatternOnce.Do(func() {
		x86Pattern = regexp.MustCompile(`(?i).*32bit|.*win32|(?:i[0-9]|x)86|i86pc`)
	})
	return x86Pattern
}

func getAndroidHintsFragmentPattern() *regexp.Regexp {
	androidHintsFragmentPatternOnce.Do(func() {
		// Pattern: Android (?:1[0-6][.\d]*; K(?: Build/|[;)])|1[0-6]\)) AppleWebKit
		androidHintsFragmentPattern = regexp.MustCompile(`(?i)Android (?:1[0-6][.\d]*; K(?: Build/|[;)])|1[0-6]\)) AppleWebKit`)
	})
	return androidHintsFragmentPattern
}

func getAndroidReplacePattern() *regexp.Regexp {
	androidReplacePatternOnce.Do(func() {
		// Pattern: Android (?:10[.\d]*; K|1[1-5])
		androidReplacePattern = regexp.MustCompile(`(?i)Android (?:10[.\d]*; K|1[1-5])`)
	})
	return androidReplacePattern
}

func getDesktopFragmentPattern() *regexp.Regexp {
	desktopFragmentPatternOnce.Do(func() {
		desktopFragmentPattern = regexp.MustCompile(`(?i)(?:Windows (?:NT|IoT)|X11; Linux x86_64)`)
	})
	return desktopFragmentPattern
}

func getDesktopExcludePattern() *regexp.Regexp {
	desktopExcludePatternOnce.Do(func() {
		desktopExcludePattern = regexp.MustCompile(`(?i)CE-HTML| Mozilla/|Andr[o0]id|Tablet|Mobile|iPhone|Windows Phone|ricoh|OculusBrowser|PicoBrowser|Lenovo|compatible; MSIE|Trident/|Tesla/|XBOX|FBMD/|ARM; ?([^)]+)`)
	})
	return desktopExcludePattern
}

// Parser parses a single user agent for OS information.
// Created via ParserFactory.NewParser() - do not instantiate directly.
type Parser struct {
	factory     *ParserFactory
	userAgent   string
	clientHints *clienthints.ClientHints
}

// Parse detects the operating system and returns a Match, or nil if not detected.
// Mirrors PHP: DeviceDetector\Parser\OperatingSystem::parse(): ?array
func (p *Parser) Parse() *Match {
	if p.factory == nil || p.userAgent == "" {
		return nil
	}

	// Restore user agent from client hints (inject device model if available)
	p.restoreUserAgentFromClientHints()

	osFromClientHints := p.parseOsFromClientHints()
	osFromUserAgent := p.parseOsFromUserAgent()

	var name, version, short string

	if osFromClientHints.Name != "" {
		name = osFromClientHints.Name
		version = osFromClientHints.Version

		// Use version from user agent if none was provided in client hints, but OS family from useragent matches
		if version == "" && GetOsFamily(name) == GetOsFamily(osFromUserAgent.Name) {
			version = osFromUserAgent.Version
		}

		// On Windows, version 0.0.0 can be either 7, 8 or 8.1
		if name == "Windows" && version == "0.0.0" {
			if osFromUserAgent.Version == "10" {
				version = ""
			} else {
				version = osFromUserAgent.Version
			}
		}

		// If the OS name detected from client hints matches the OS family from user agent
		// but the OS name is another, we use the one from user agent, as it might be more detailed
		if GetOsFamily(osFromUserAgent.Name) == name && osFromUserAgent.Name != name {
			name = osFromUserAgent.Name

			if name == "LeafOS" || name == "HarmonyOS" {
				version = ""
			}

			if name == "PICO OS" {
				version = osFromUserAgent.Version
			}

			if name == "Fire OS" && osFromClientHints.Version != "" {
				majorVersion := getMajorVersion(version)
				if v, ok := FireOsVersionMapping[version]; ok {
					version = v
				} else if v, ok := FireOsVersionMapping[majorVersion]; ok {
					version = v
				} else {
					version = ""
				}
			}
		}

		short = osFromClientHints.ShortName

		// Chrome OS is in some cases reported as Linux in client hints, we fix this only if the version matches
		if name == "GNU/Linux" && osFromUserAgent.Name == "Chrome OS" && osFromClientHints.Version == osFromUserAgent.Version {
			name = osFromUserAgent.Name
			short = osFromUserAgent.ShortName
		}

		// Chrome OS is in some cases reported as Android in client hints
		if name == "Android" && osFromUserAgent.Name == "Chrome OS" {
			name = osFromUserAgent.Name
			version = ""
			short = osFromUserAgent.ShortName
		}

		// Meta Horizon is reported as Linux in client hints
		if name == "GNU/Linux" && osFromUserAgent.Name == "Meta Horizon" {
			name = osFromUserAgent.Name
			short = osFromUserAgent.ShortName
		}
	} else if osFromUserAgent.Name != "" {
		name = osFromUserAgent.Name
		version = osFromUserAgent.Version
		short = osFromUserAgent.ShortName
	} else {
		return nil
	}

	platform := p.parsePlatform()
	family := GetOsFamily(short)

	// Check for Android apps via client hints
	if p.clientHints != nil {
		app := p.clientHints.GetApp()
		if containsString(AndroidApps, app) && name != "Android" {
			name = "Android"
			family = "Android"
			short = "AND"
			version = ""
		}

		if app == "org.lineageos.jelly" && name != "Lineage OS" {
			majorVersion := getMajorVersion(version)
			name = "Lineage OS"
			family = "Android"
			short = "LEN"
			if v, ok := LineageOsVersionMapping[version]; ok {
				version = v
			} else if v, ok := LineageOsVersionMapping[majorVersion]; ok {
				version = v
			} else {
				version = ""
			}
		}

		if app == "org.mozilla.tv.firefox" && name != "Fire OS" {
			majorVersion := getMajorVersion(version)
			name = "Fire OS"
			family = "Android"
			short = "FIR"
			if v, ok := FireOsVersionMapping[version]; ok {
				version = v
			} else if v, ok := FireOsVersionMapping[majorVersion]; ok {
				version = v
			} else {
				version = ""
			}
		}
	}

	result := &Match{
		Name:      name,
		ShortName: short,
		Version:   version,
		Platform:  platform,
		Family:    family,
	}

	// Ensure short name is correct for the detected name
	if _, exists := OperatingSystems[result.ShortName]; !exists {
		for osShort, osName := range OperatingSystems {
			if osName == result.Name {
				result.ShortName = osShort
				break
			}
		}
	}

	return result
}

// parseOsFromClientHints returns OS info from client hints.
func (p *Parser) parseOsFromClientHints() *Match {
	result := &Match{}

	if p.clientHints == nil || p.clientHints.GetOperatingSystem() == "" {
		return result
	}

	hintName := applyClientHintMapping(p.clientHints.GetOperatingSystem())

	for osShort, osName := range OperatingSystems {
		if fuzzyCompare(hintName, osName) {
			result.Name = osName
			result.ShortName = osShort
			break
		}
	}

	version := p.clientHints.GetOperatingSystemVersion()

	if result.Name == "Windows" {
		version = parseWindowsVersionFromHints(version)
	}

	// On Windows, version 0.0.0 can be either 7, 8 or 8.1, so we return 0.0.0
	if result.Name != "Windows" && version != "0.0.0" {
		if intVer, _ := strconv.Atoi(strings.Split(version, ".")[0]); intVer == 0 {
			version = ""
		}
	}

	result.Version = buildVersion(version, nil)

	return result
}

// parseWindowsVersionFromHints converts client hints version to Windows version.
func parseWindowsVersionFromHints(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return version
	}

	majorVersion, _ := strconv.Atoi(parts[0])
	minorVersion := 0
	if len(parts) > 1 {
		minorVersion, _ = strconv.Atoi(parts[1])
	}

	if majorVersion == 0 {
		minorVersionMapping := map[int]string{1: "7", 2: "8", 3: "8.1"}
		if v, ok := minorVersionMapping[minorVersion]; ok {
			return v
		}
		return version
	} else if majorVersion > 0 && majorVersion < 11 {
		return "10"
	} else if majorVersion > 10 {
		return "11"
	}

	return version
}

// parseOsFromUserAgent returns OS info from user agent.
func (p *Parser) parseOsFromUserAgent() *Match {
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
	shortData := getShortOsData(name)
	name = shortData.Name
	short := shortData.Short

	version := ""
	if matchedEntry.Version != "" {
		version = buildVersion(matchedEntry.Version, matches)
	}

	// Check for version sub-patterns
	for _, vEntry := range matchedEntry.Versions {
		if vEntry.compiled == nil {
			continue
		}

		vMatches, err := vEntry.compiled.FindStringSubmatch(p.userAgent)
		if err != nil || len(vMatches) == 0 {
			continue
		}

		if vEntry.Name != "" {
			name = strings.TrimSpace(buildByMatch(vEntry.Name, vMatches))
			shortData = getShortOsData(name)
			name = shortData.Name
			short = shortData.Short
		}

		if vEntry.Version != "" {
			version = buildVersion(vEntry.Version, vMatches)
		}

		break
	}

	result.Name = name
	result.ShortName = short
	result.Version = version

	return result
}

// parsePlatform detects the OS platform (x86, x64, ARM, etc.)
func (p *Parser) parsePlatform() string {
	// Use architecture from client hints if available
	if p.clientHints != nil && p.clientHints.GetArchitecture() != "" {
		arch := strings.ToLower(p.clientHints.GetArchitecture())

		if strings.Contains(arch, "arm") {
			return "ARM"
		}

		if strings.Contains(arch, "loongarch64") {
			return "LoongArch64"
		}

		if strings.Contains(arch, "mips") {
			return "MIPS"
		}

		if strings.Contains(arch, "sh4") {
			return "SuperH"
		}

		if strings.Contains(arch, "sparc64") {
			return "SPARC64"
		}

		if strings.Contains(arch, "x64") ||
			(strings.Contains(arch, "x86") && p.clientHints.GetBitness() == "64") {
			return "x64"
		}

		if strings.Contains(arch, "x86") {
			return "x86"
		}
	}

	// Fall back to user agent detection
	if getArmPattern().MatchString(p.userAgent) {
		return "ARM"
	}

	if getLoongPattern().MatchString(p.userAgent) {
		return "LoongArch64"
	}

	if getMipsPattern().MatchString(p.userAgent) {
		return "MIPS"
	}

	if getSh4Pattern().MatchString(p.userAgent) {
		return "SuperH"
	}

	if getSparc64Pattern().MatchString(p.userAgent) {
		return "SPARC64"
	}

	if getX64Pattern().MatchString(p.userAgent) {
		return "x64"
	}

	if getX86Pattern().MatchString(p.userAgent) {
		return "x86"
	}

	return ""
}

// restoreUserAgentFromClientHints modifies the user agent by injecting device model from client hints.
// This mirrors PHP AbstractParser::restoreUserAgentFromClientHints()
func (p *Parser) restoreUserAgentFromClientHints() {
	if p.clientHints == nil {
		return
	}

	deviceModel := p.clientHints.GetModel()
	if deviceModel == "" {
		return
	}

	// Restore Android User Agent
	if p.hasUserAgentClientHintsFragment() {
		osVersion := p.clientHints.GetOperatingSystemVersion()
		if osVersion == "" {
			osVersion = "10"
		}

		replacement := "Android " + osVersion + "; " + deviceModel
		p.userAgent = getAndroidReplacePattern().ReplaceAllString(p.userAgent, replacement)
		return
	}

	// Restore Desktop User Agent
	if !p.hasDesktopFragment() {
		return
	}

	// Replace X11; Linux x86_64 with X11; Linux x86_64; deviceModel
	p.userAgent = strings.Replace(p.userAgent, "X11; Linux x86_64", "X11; Linux x86_64; "+deviceModel, 1)
}

// hasUserAgentClientHintsFragment checks if the user agent contains Android client hints fragment.
// Mirrors PHP AbstractParser::hasUserAgentClientHintsFragment()
func (p *Parser) hasUserAgentClientHintsFragment() bool {
	if getAndroidHintsFragmentPattern().MatchString(p.userAgent) {
		return !strings.Contains(strings.ToLower(p.userAgent), "telegram-android/")
	}
	return false
}

// hasDesktopFragment checks if the user agent is from a desktop browser.
// Mirrors PHP AbstractParser::hasDesktopFragment()
func (p *Parser) hasDesktopFragment() bool {
	return getDesktopFragmentPattern().MatchString(p.userAgent) &&
		!getDesktopExcludePattern().MatchString(p.userAgent)
}

// Helper types and functions

type shortOsResult struct {
	Name  string
	Short string
}

func getShortOsData(name string) shortOsResult {
	short := "UNK"

	for osShort, osName := range OperatingSystems {
		if strings.EqualFold(name, osName) {
			return shortOsResult{Name: osName, Short: osShort}
		}
	}

	return shortOsResult{Name: name, Short: short}
}

// GetOsFamily returns the OS family for the given OS (by short name or name).
func GetOsFamily(osLabel string) string {
	// Check if osLabel is a full name, convert to short
	if _, exists := OperatingSystems[osLabel]; !exists {
		for short, name := range OperatingSystems {
			if name == osLabel {
				osLabel = short
				break
			}
		}
	}

	for family, codes := range OSFamilies {
		for _, code := range codes {
			if code == osLabel {
				return family
			}
		}
	}

	return ""
}

// IsDesktopOs returns true if the OS is desktop only.
func IsDesktopOs(osName string) bool {
	osFamily := GetOsFamily(osName)
	for _, family := range DesktopOsFamilies {
		if family == osFamily {
			return true
		}
	}
	return false
}

// GetNameFromId returns the full name for the given short name.
func GetNameFromId(os string, ver string) string {
	if name, exists := OperatingSystems[os]; exists {
		result := name
		if ver != "" {
			result = result + " " + ver
		}
		return strings.TrimSpace(result)
	}
	return ""
}

// GetAvailableOperatingSystems returns all available operating systems.
func GetAvailableOperatingSystems() map[string]string {
	return OperatingSystems
}

// GetAvailableOperatingSystemFamilies returns all available OS families.
func GetAvailableOperatingSystemFamilies() map[string][]string {
	return OSFamilies
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
	if strings.Contains(v, "$") {
		return ""
	}

	return v
}

// applyClientHintMapping applies the client hint mapping.
func applyClientHintMapping(osName string) string {
	for mappedName, hintNames := range ClientHintMapping {
		for _, hintName := range hintNames {
			if strings.EqualFold(osName, hintName) {
				return mappedName
			}
		}
	}
	return osName
}

// fuzzyCompare performs a case-insensitive comparison with some normalization.
func fuzzyCompare(s1, s2 string) bool {
	normalize := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "-", "")
		s = strings.ReplaceAll(s, "_", "")
		return s
	}
	return normalize(s1) == normalize(s2)
}

// getMajorVersion extracts the major version number from a version string.
func getMajorVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// containsString checks if a string slice contains a specific string.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
