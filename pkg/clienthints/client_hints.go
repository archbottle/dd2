package clienthints

import (
	"net/http"
	"regexp"
	"strings"
)

// ClientHints represents the parsed client hints data
type ClientHints struct {
	model           string
	platform        string
	platformVersion string
	uaFullVersion   string
	fullVersionList []BrandVersion
	mobile          bool
	architecture    string
	bitness         string
	app             string
	formFactors     []string
}

// BrandVersion represents a browser brand and its version.
type BrandVersion struct {
	Brand   string
	Version string
}

// Factory creates a ClientHints object from headers.
// It supports three header formats:
// 1. Standard HTTP headers: sec-ch-ua, sec-ch-ua-mobile, etc.
// 2. PHP $_SERVER style: HTTP_SEC_CH_UA, HTTP_SEC_CH_UA_MOBILE, etc.
// 3. JavaScript navigator.userAgentData API format:
//   - fullVersionList: []map[string]string{{"brand": "...", "version": "..."}, ...}
//   - mobile: bool
//   - platform: string
//   - platformVersion: string
//   - formFactors: []string
func Factory(headers map[string]interface{}) *ClientHints {
	ch := &ClientHints{
		fullVersionList: []BrandVersion{},
		formFactors:     []string{},
	}

	for name, value := range headers {
		if value == nil {
			continue
		}

		lowerName := strings.ToLower(strings.ReplaceAll(name, "_", "-"))

		switch lowerName {
		case "http-sec-ch-ua-arch", "sec-ch-ua-arch", "arch", "architecture":
			if s, ok := value.(string); ok {
				ch.architecture = strings.Trim(s, "\"")
			}

		case "http-sec-ch-ua-bitness", "sec-ch-ua-bitness", "bitness":
			if s, ok := value.(string); ok {
				ch.bitness = strings.Trim(s, "\"")
			}

		case "http-sec-ch-ua-mobile", "sec-ch-ua-mobile", "mobile":
			switch v := value.(type) {
			case string:
				ch.mobile = (v == "1" || v == "?1" || v == "true")
			case bool:
				ch.mobile = v
			}

		case "http-sec-ch-ua-model", "sec-ch-ua-model", "model":
			if s, ok := value.(string); ok {
				ch.model = strings.Trim(s, "\"")
			}

		case "http-sec-ch-ua-full-version", "sec-ch-ua-full-version", "uafullversion":
			if s, ok := value.(string); ok {
				ch.uaFullVersion = strings.Trim(s, "\"")
			}

		case "http-sec-ch-ua-platform", "sec-ch-ua-platform", "platform":
			if s, ok := value.(string); ok {
				ch.platform = strings.Trim(s, "\"")
			}

		case "http-sec-ch-ua-platform-version", "sec-ch-ua-platform-version", "platformversion":
			if s, ok := value.(string); ok {
				ch.platformVersion = strings.Trim(s, "\"")
			}

		case "http-sec-ch-ua", "sec-ch-ua", "http-sec-ch-ua-full-version-list", "sec-ch-ua-full-version-list", "fullversionlist":
			// If fullVersionList is already set (by full-version-list header), don't overwrite with sec-ch-ua
			if len(ch.fullVersionList) > 0 && (lowerName == "http-sec-ch-ua" || lowerName == "sec-ch-ua") {
				continue
			}

			if s, ok := value.(string); ok {
				// Parse string format: "Brand";v="Version", "Brand2";v="Version2"
				list := parseBrandVersionString(s)
				if len(list) > 0 {
					ch.fullVersionList = list
				}
			} else if arr, ok := value.([]interface{}); ok {
				// JavaScript format: [{brand: "...", version: "..."}, ...]
				list, valid := parseBrandVersionArray(arr)
				if valid && len(list) > 0 {
					ch.fullVersionList = list
				}
			} else if arr, ok := value.([]map[string]string); ok {
				// Direct map format
				list, valid := parseBrandVersionMapArray(arr)
				if valid && len(list) > 0 {
					ch.fullVersionList = list
				}
			}

		case "http-x-requested-with", "x-requested-with":
			if s, ok := value.(string); ok {
				if strings.ToLower(s) != "xmlhttprequest" {
					ch.app = s
				}
			}

		case "formfactors", "http-sec-ch-ua-form-factors", "sec-ch-ua-form-factors":
			switch v := value.(type) {
			case string:
				// Parse string format: "Desktop", "Mobile"
				re := regexp.MustCompile(`"([a-zA-Z]+)"`)
				matches := re.FindAllStringSubmatch(v, -1)
				for _, m := range matches {
					ch.formFactors = append(ch.formFactors, strings.ToLower(m[1]))
				}
			case []interface{}:
				// JavaScript array format: ["Desktop", "Mobile"]
				for _, item := range v {
					if s, ok := item.(string); ok {
						ch.formFactors = append(ch.formFactors, strings.ToLower(s))
					}
				}
			case []string:
				// Direct string array
				for _, s := range v {
					ch.formFactors = append(ch.formFactors, strings.ToLower(s))
				}
			}
		}
	}

	return ch
}

// parseBrandVersionString parses the Sec-CH-UA header string format.
// Format: "Brand";v="Version", "Brand2";v="Version2"
func parseBrandVersionString(s string) []BrandVersion {
	re := regexp.MustCompile(`^"([^"]+)"; ?v="([^"]+)"(?:, )?`)
	remaining := s
	var list []BrandVersion

	for {
		matches := re.FindStringSubmatch(remaining)
		if matches == nil {
			break
		}
		list = append(list, BrandVersion{Brand: matches[1], Version: matches[2]})
		remaining = remaining[len(matches[0]):]
	}

	return list
}

// parseBrandVersionArray parses the JavaScript navigator.userAgentData.brands format.
// Format: [{brand: "...", version: "..."}, ...]
// Returns the list and a bool indicating if all entries were valid.
// If any entry is invalid (missing brand or version), returns empty list and false.
func parseBrandVersionArray(arr []interface{}) ([]BrandVersion, bool) {
	var list []BrandVersion

	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, false
		}

		brand, hasBrand := m["brand"].(string)
		version, hasVersion := m["version"].(string)

		// PHP behavior: if any entry lacks brand or version, discard entire list
		if !hasBrand || !hasVersion {
			return nil, false
		}

		list = append(list, BrandVersion{Brand: brand, Version: version})
	}

	return list, true
}

// parseBrandVersionMapArray parses a []map[string]string format.
func parseBrandVersionMapArray(arr []map[string]string) ([]BrandVersion, bool) {
	var list []BrandVersion

	for _, m := range arr {
		brand, hasBrand := m["brand"]
		version, hasVersion := m["version"]

		if !hasBrand || !hasVersion || brand == "" {
			return nil, false
		}

		list = append(list, BrandVersion{Brand: brand, Version: version})
	}

	return list, true
}

// New creates a ClientHints object from HTTP headers.
// This is a convenience wrapper around Factory for the common case
// where headers come from an HTTP request.
func New(headers http.Header) *ClientHints {
	// Convert http.Header to interface{} map
	// For multi-valued headers, join with ", " to match HTTP header format
	m := make(map[string]interface{}, len(headers))
	for k, values := range headers {
		if len(values) == 0 {
			continue
		}
		// Join multiple values with ", " (standard HTTP header format)
		joined := strings.Join(values, ", ")
		if joined != "" {
			m[k] = joined
		}
	}
	return Factory(m)
}

func (ch *ClientHints) GetModel() string {
	return ch.model
}

func (ch *ClientHints) GetOperatingSystem() string {
	return ch.platform
}

func (ch *ClientHints) GetOperatingSystemVersion() string {
	return ch.platformVersion
}

func (ch *ClientHints) GetApp() string {
	return ch.app
}

func (ch *ClientHints) GetBrandVersion() string {
	if ch.uaFullVersion != "" {
		return ch.uaFullVersion
	}
	return ""
}

func (ch *ClientHints) GetBrandList() map[string]string {
	result := make(map[string]string)
	for _, bv := range ch.fullVersionList {
		result[bv.Brand] = bv.Version
	}
	return result
}

func (ch *ClientHints) GetArchitecture() string {
	return ch.architecture
}

func (ch *ClientHints) GetBitness() string {
	return ch.bitness
}

func (ch *ClientHints) IsMobile() bool {
	return ch.mobile
}

func (ch *ClientHints) GetFormFactors() []string {
	return ch.formFactors
}
