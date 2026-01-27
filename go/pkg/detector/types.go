// Package detector provides the main DeviceDetector that orchestrates
// bot, client, and device detection.
package detector

// DeviceType represents the detected device type.
type DeviceType int

const (
	DeviceTypeDesktop             DeviceType = 0
	DeviceTypeSmartphone          DeviceType = 1
	DeviceTypeTablet              DeviceType = 2
	DeviceTypeFeaturePhone        DeviceType = 3
	DeviceTypeConsole             DeviceType = 4
	DeviceTypeTV                  DeviceType = 5
	DeviceTypeCarBrowser          DeviceType = 6
	DeviceTypeSmartDisplay        DeviceType = 7
	DeviceTypeCamera              DeviceType = 8
	DeviceTypePortableMediaPlayer DeviceType = 9
	DeviceTypePhablet             DeviceType = 10
	DeviceTypeSmartSpeaker        DeviceType = 11
	DeviceTypeWearable            DeviceType = 12
	DeviceTypePeripheral          DeviceType = 13
	DeviceTypeUnknown             DeviceType = -1
)

// DeviceTypeNames maps device type constants to their string names.
var DeviceTypeNames = map[DeviceType]string{
	DeviceTypeDesktop:             "desktop",
	DeviceTypeSmartphone:          "smartphone",
	DeviceTypeTablet:              "tablet",
	DeviceTypeFeaturePhone:        "feature phone",
	DeviceTypeConsole:             "console",
	DeviceTypeTV:                  "tv",
	DeviceTypeCarBrowser:          "car browser",
	DeviceTypeSmartDisplay:        "smart display",
	DeviceTypeCamera:              "camera",
	DeviceTypePortableMediaPlayer: "portable media player",
	DeviceTypePhablet:             "phablet",
	DeviceTypeSmartSpeaker:        "smart speaker",
	DeviceTypeWearable:            "wearable",
	DeviceTypePeripheral:          "peripheral",
}

// DeviceTypeFromString converts a string device type to DeviceType constant.
func DeviceTypeFromString(s string) DeviceType {
	for dt, name := range DeviceTypeNames {
		if name == s {
			return dt
		}
	}
	return DeviceTypeUnknown
}

// BotInfo contains information about a detected bot.
type BotInfo struct {
	Name     string
	Category string
	URL      string
	Producer *BotProducer
}

// BotProducer contains information about a bot's producer.
type BotProducer struct {
	Name string
	URL  string
}

// ClientInfo contains information about the detected client (browser, app, etc.).
type ClientInfo struct {
	Type    string // browser, mobile app, feed reader, etc.
	Name    string
	Version string
	Engine  string
}

// ClientMatch is a unified client match type that can hold results from any client parser.
// This is used by the DeviceDetector to return client detection results.
type ClientMatch struct {
	Type          string // browser, mobile app, feed reader, library, media player, pim
	Name          string
	Version       string
	Engine        string // only for browsers
	EngineVersion string // only for browsers
	Family        string // only for browsers (but excluded from testParseClient comparison)
}

// OSInfo contains information about the detected operating system.
type OSInfo struct {
	Name    string
	Version string
	Family  string
}

// DeviceInfo contains information about the detected device.
type DeviceInfo struct {
	Type  DeviceType
	Brand string
	Model string
}

// Result contains the full detection result.
type Result struct {
	Bot    *BotInfo
	Client *ClientInfo
	OS     *OSInfo
	Device *DeviceInfo
}

// FullInfo is the complete detection result matching PHP's getInfoFromUserAgent() output.
// This is the structure used for the testParse integration test.
type FullInfo struct {
	UserAgent     string          `yaml:"user_agent"`
	OS            *FullInfoOS     `yaml:"os"`
	Client        *FullInfoClient `yaml:"client"`
	Device        *FullInfoDevice `yaml:"device"`
	OSFamily      string          `yaml:"os_family"`
	BrowserFamily string          `yaml:"browser_family"`
}

// FullInfoOS matches the PHP os output structure.
type FullInfoOS struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	Platform string `yaml:"platform"`
}

// FullInfoClient matches the PHP client output structure.
type FullInfoClient struct {
	Type          string `yaml:"type"`
	Name          string `yaml:"name"`
	Version       string `yaml:"version"`
	Engine        string `yaml:"engine"`
	EngineVersion string `yaml:"engine_version"`
}

// FullInfoDevice matches the PHP device output structure.
type FullInfoDevice struct {
	Type  string `yaml:"type"`
	Brand string `yaml:"brand"`
	Model string `yaml:"model"`
}
