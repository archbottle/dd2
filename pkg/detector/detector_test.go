package detector

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/archbottle/device-detector/pkg/clienthints"
	"github.com/archbottle/device-detector/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// clientFixture matches the PHP client fixture format (browser, feed_reader, etc.).
type clientFixture struct {
	UserAgent string                 `yaml:"user_agent"`
	Headers   common.YAMLHTTPHeader `yaml:"headers"`
	Client    clientExpect           `yaml:"client"`
}

type clientExpect struct {
	Type          string `yaml:"type"`
	Name          string `yaml:"name"`
	Version       string `yaml:"version"`
	Engine        string `yaml:"engine"`
	EngineVersion string `yaml:"engine_version"`
	Family        string `yaml:"family"` // NOTE: This is excluded from comparison in PHP test
}

// fullParseFixture matches the PHP testParse fixture format (full integration test).
// Note: OS/Client/Device can be empty arrays in PHP fixtures, so we use interface{} and convert.
type fullParseFixture struct {
	UserAgent     string      `yaml:"user_agent"`
	Headers       interface{} `yaml:"headers"` // Can be map[string]string or complex client hints structure
	OS            interface{} `yaml:"os"`      // Can be fullParseOS or empty array
	Client        interface{} `yaml:"client"`  // Can be fullParseClient or empty array
	Device        interface{} `yaml:"device"`  // Can be fullParseDevice or empty array
	OSFamily      string      `yaml:"os_family"`
	BrowserFamily string      `yaml:"browser_family"`
}

// Helper methods to extract typed values from interface{} fields
func (f *fullParseFixture) getOS() fullParseOS {
	if f.OS == nil {
		return fullParseOS{}
	}
	if m, ok := f.OS.(map[string]interface{}); ok {
		return fullParseOS{
			Name:     stringFromInterface(m["name"]),
			Version:  stringFromInterface(m["version"]),
			Platform: stringFromInterface(m["platform"]),
		}
	}
	return fullParseOS{}
}

func (f *fullParseFixture) getClient() fullParseClient {
	if f.Client == nil {
		return fullParseClient{}
	}
	if m, ok := f.Client.(map[string]interface{}); ok {
		return fullParseClient{
			Type:          stringFromInterface(m["type"]),
			Name:          stringFromInterface(m["name"]),
			Version:       stringFromInterface(m["version"]),
			Engine:        stringFromInterface(m["engine"]),
			EngineVersion: stringFromInterface(m["engine_version"]),
		}
	}
	return fullParseClient{}
}

func (f *fullParseFixture) getDevice() fullParseDevice {
	if f.Device == nil {
		return fullParseDevice{}
	}
	if m, ok := f.Device.(map[string]interface{}); ok {
		return fullParseDevice{
			Type:  stringFromInterface(m["type"]),
			Brand: stringFromInterface(m["brand"]),
			Model: stringFromInterface(m["model"]),
		}
	}
	return fullParseDevice{}
}

func stringFromInterface(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// getHeaders extracts simple string headers from the fixture.
// Complex client hints structures (with brands arrays) are ignored for now.
func (f *fullParseFixture) getHeaders() http.Header {
	if f.Headers == nil {
		return nil
	}
	if m, ok := f.Headers.(map[string]interface{}); ok {
		result := make(http.Header)
		for k, v := range m {
			if s, ok := v.(string); ok {
				result.Set(k, s)
			}
			// Skip complex structures (arrays, nested maps)
		}
		if len(result) > 0 {
			return result
		}
	}
	return nil
}

type fullParseOS struct {
	Name     string
	Version  string
	Platform string
}

type fullParseClient struct {
	Type          string
	Name          string
	Version       string
	Engine        string
	EngineVersion string
}

type fullParseDevice struct {
	Type  string
	Brand string
	Model string
}

// deviceFixture matches the PHP device fixture format.
type deviceFixture struct {
	UserAgent string       `yaml:"user_agent"`
	Device    deviceExpect `yaml:"device"`
}

type deviceExpect struct {
	Type  string `yaml:"type"`
	Brand string `yaml:"brand"`
	Model string `yaml:"model"`
}

// typeMethodFixture matches the PHP fixture format.
type typeMethodFixture struct {
	UserAgent string `yaml:"user_agent"`
	Check     []bool `yaml:"check"`
}


func loadTypeMethodFixtures(t *testing.T) []typeMethodFixture {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")

	fixturePath := filepath.Join(filepath.Dir(filename), "fixtures", "type-methods.yml")
	data, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "failed to read fixture file")

	var fixtures []typeMethodFixture
	err = yaml.Unmarshal(data, &fixtures)
	require.NoError(t, err, "failed to parse fixture YAML")

	return fixtures
}

// TestTypeMethods tests the boolean classification methods.
// PHP: DeviceDetectorTest::testTypeMethods
func TestTypeMethods(t *testing.T) {
	// Create detector with discardBotInformation (as PHP test does)
	dd, err := New(WithDiscardBotInformation())
	require.NoError(t, err, "failed to create DeviceDetector")

	fixtures := loadTypeMethodFixtures(t)
	require.NotEmpty(t, fixtures, "no fixtures loaded")

	t.Logf("Running %d type method test cases", len(fixtures))

	passed := 0
	failed := 0

	for i, tc := range fixtures {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			result := dd.Parse(tc.UserAgent, nil)

			// Check format: [isBot(), isMobile(), isDesktop(), isTablet(), isTV(), isWearable()]
			got := []bool{
				result.IsBot(),
				result.IsMobile(),
				result.IsDesktop(),
				result.IsTablet(),
				result.IsTV(),
				result.IsWearable(),
			}

			if assert.Equal(t, tc.Check, got, "UA: %s\nExpected: %v\nGot: %v", tc.UserAgent, tc.Check, got) {
				passed++
			} else {
				failed++
				t.Logf("Mismatch for UA: %s", tc.UserAgent)
				t.Logf("  Expected: isBot=%v, isMobile=%v, isDesktop=%v, isTablet=%v, isTV=%v, isWearable=%v",
					tc.Check[0], tc.Check[1], tc.Check[2], tc.Check[3], tc.Check[4], tc.Check[5])
				t.Logf("  Got:      isBot=%v, isMobile=%v, isDesktop=%v, isTablet=%v, isTV=%v, isWearable=%v",
					got[0], got[1], got[2], got[3], got[4], got[5])

				// Debug info
				if result.os != nil {
					t.Logf("  OS: %s (family: %s)", result.os.Name, result.os.Family)
				}
				if result.client != nil {
					t.Logf("  Client: %s (type: %s)", result.client.Name, result.client.Type)
				}
				t.Logf("  Device type: %v", result.device)
			}
		})
	}

	t.Logf("Results: %d/%d passed (%.1f%%)", passed, len(fixtures), float64(passed)/float64(len(fixtures))*100)
}

// loadFullParseFixtures loads all fixtures from the PHP Tests/fixtures/ directory.
// This excludes bots.yml as per PHP's testParse behavior.
func loadFullParseFixtures(t *testing.T) []fullParseFixture {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")

	// PHP fixture files are in device-detector/Tests/fixtures/
	fixturesDir := filepath.Join(filepath.Dir(filename), "..", "..", "..", "device-detector", "Tests", "fixtures")

	// Get all yml files
	files, err := filepath.Glob(filepath.Join(fixturesDir, "*.yml"))
	require.NoError(t, err, "failed to glob fixture files")

	var allFixtures []fullParseFixture

	for _, file := range files {
		// Skip bots.yml as per PHP test
		if filepath.Base(file) == "bots.yml" {
			continue
		}

		data, err := os.ReadFile(file)
		require.NoError(t, err, "failed to read fixture file: %s", file)

		var fixtures []fullParseFixture
		err = yaml.Unmarshal(data, &fixtures)
		require.NoError(t, err, "failed to parse fixture YAML: %s", file)

		allFixtures = append(allFixtures, fixtures...)
	}

	return allFixtures
}

// TestParse tests the full DeviceDetector pipeline validating ALL output fields.
// PHP: DeviceDetectorTest::testParse
// This is the comprehensive integration test that validates the complete detection output.
func TestParse(t *testing.T) {
	dd, err := New()
	require.NoError(t, err, "failed to create DeviceDetector")

	fixtures := loadFullParseFixtures(t)
	require.NotEmpty(t, fixtures, "no fixtures loaded")

	t.Logf("Running %d full parse test cases", len(fixtures))

	// Use atomic counters for parallel tests
	var passed, failed int64

	for i, tc := range fixtures {
		i, tc := i, tc // capture loop vars for parallel execution
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			t.Parallel() // Run tests in parallel

			// Build client hints if headers present
			var ch *clienthints.ClientHints
			headers := tc.getHeaders()
			if len(headers) > 0 {
				ch = clienthints.New(headers)
			}

			result := dd.Parse(tc.UserAgent, ch)
			got := result.GetFullInfo()

			// Extract expected values using helper methods
			expOS := tc.getOS()
			expClient := tc.getClient()
			expDevice := tc.getDevice()

			// Compare all fields
			var mismatches []string

			// OS comparison
			if expOS.Name != got.OS.Name {
				mismatches = append(mismatches, "OS.Name: "+expOS.Name+" vs "+got.OS.Name)
			}
			if expOS.Version != got.OS.Version {
				mismatches = append(mismatches, "OS.Version: "+expOS.Version+" vs "+got.OS.Version)
			}
			if expOS.Platform != got.OS.Platform {
				mismatches = append(mismatches, "OS.Platform: "+expOS.Platform+" vs "+got.OS.Platform)
			}

			// Client comparison
			if expClient.Type != got.Client.Type {
				mismatches = append(mismatches, "Client.Type: "+expClient.Type+" vs "+got.Client.Type)
			}
			if expClient.Name != got.Client.Name {
				mismatches = append(mismatches, "Client.Name: "+expClient.Name+" vs "+got.Client.Name)
			}
			if expClient.Version != got.Client.Version {
				mismatches = append(mismatches, "Client.Version: "+expClient.Version+" vs "+got.Client.Version)
			}
			if expClient.Engine != got.Client.Engine {
				mismatches = append(mismatches, "Client.Engine: "+expClient.Engine+" vs "+got.Client.Engine)
			}
			if expClient.EngineVersion != got.Client.EngineVersion {
				mismatches = append(mismatches, "Client.EngineVersion: "+expClient.EngineVersion+" vs "+got.Client.EngineVersion)
			}

			// Device comparison
			if expDevice.Type != got.Device.Type {
				mismatches = append(mismatches, "Device.Type: "+expDevice.Type+" vs "+got.Device.Type)
			}
			if expDevice.Brand != got.Device.Brand {
				mismatches = append(mismatches, "Device.Brand: "+expDevice.Brand+" vs "+got.Device.Brand)
			}
			if expDevice.Model != got.Device.Model {
				mismatches = append(mismatches, "Device.Model: "+expDevice.Model+" vs "+got.Device.Model)
			}

			// Family comparison
			if tc.OSFamily != got.OSFamily {
				mismatches = append(mismatches, "OSFamily: "+tc.OSFamily+" vs "+got.OSFamily)
			}
			if tc.BrowserFamily != got.BrowserFamily {
				mismatches = append(mismatches, "BrowserFamily: "+tc.BrowserFamily+" vs "+got.BrowserFamily)
			}

			if len(mismatches) == 0 {
				atomic.AddInt64(&passed, 1)
			} else {
				atomic.AddInt64(&failed, 1)
				t.Errorf("Mismatch for UA: %s", tc.UserAgent)
				for _, m := range mismatches {
					t.Errorf("  %s", m)
				}
			}
		})
	}

	// Note: Due to parallel execution, final count is logged after all subtests complete
	t.Cleanup(func() {
		total := int(passed + failed)
		t.Logf("Results: %d/%d passed (%.1f%%)", passed, total, float64(passed)/float64(total)*100)
	})
}

// loadClientFixtures loads client fixtures from the PHP test fixture files.
func loadClientFixtures(t *testing.T) []clientFixture {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")

	// PHP fixture files are in device-detector/Tests/Parser/Client/fixtures/
	baseDir := filepath.Join(filepath.Dir(filename), "..", "..", "..", "device-detector", "Tests", "Parser", "Client", "fixtures")

	fixtureFiles := []string{
		"browser.yml",
		"feed_reader.yml",
		"library.yml",
		"mediaplayer.yml",
		"mobile_app.yml",
		"pim.yml",
	}

	var allFixtures []clientFixture

	for _, file := range fixtureFiles {
		fixturePath := filepath.Join(baseDir, file)
		data, err := os.ReadFile(fixturePath)
		require.NoError(t, err, "failed to read fixture file: %s", file)

		var fixtures []clientFixture
		err = yaml.Unmarshal(data, &fixtures)
		require.NoError(t, err, "failed to parse fixture YAML: %s", file)

		allFixtures = append(allFixtures, fixtures...)
	}

	return allFixtures
}

// TestParseClient tests the full DeviceDetector pipeline validating client output.
// PHP: DeviceDetectorTest::testParseClient
// This runs the FULL detection pipeline but only validates the client field.
// NOTE: The 'family' field is excluded from comparison (as PHP does).
func TestParseClient(t *testing.T) {
	dd, err := New()
	require.NoError(t, err, "failed to create DeviceDetector")

	fixtures := loadClientFixtures(t)
	require.NotEmpty(t, fixtures, "no fixtures loaded")

	t.Logf("Running %d client test cases", len(fixtures))

	passed := 0
	failed := 0

	for i, tc := range fixtures {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			// Build client hints if headers present
			var ch *clienthints.ClientHints
			if len(tc.Headers.Header()) > 0 {
				ch = clienthints.New(tc.Headers.Header())
			}

			result := dd.Parse(tc.UserAgent, ch)

			// PHP assertion: assertArrayNotHasKey('bot', $uaInfo)
			// Ensure none of the client fixtures are incorrectly detected as bots
			if !assert.False(t, result.IsBot(), "UA incorrectly detected as bot: %s", tc.UserAgent) {
				failed++
				return
			}

			// Get client info
			client := result.GetClient()

			// Handle nil client (no match)
			var gotType, gotName, gotVersion, gotEngine, gotEngineVersion string
			if client != nil {
				gotType = client.Type
				gotName = client.Name
				gotVersion = client.Version
				gotEngine = client.Engine
				gotEngineVersion = client.EngineVersion
			}

			// Compare with expected (NOTE: family is excluded from comparison, as per PHP test)
			typeOK := tc.Client.Type == gotType
			nameOK := tc.Client.Name == gotName
			versionOK := tc.Client.Version == gotVersion
			engineOK := tc.Client.Engine == gotEngine
			engineVersionOK := tc.Client.EngineVersion == gotEngineVersion

			if typeOK && nameOK && versionOK && engineOK && engineVersionOK {
				passed++
			} else {
				failed++
				t.Errorf("Client mismatch for UA: %s", tc.UserAgent)
				if !typeOK {
					t.Errorf("  Type: expected %q, got %q", tc.Client.Type, gotType)
				}
				if !nameOK {
					t.Errorf("  Name: expected %q, got %q", tc.Client.Name, gotName)
				}
				if !versionOK {
					t.Errorf("  Version: expected %q, got %q", tc.Client.Version, gotVersion)
				}
				if !engineOK {
					t.Errorf("  Engine: expected %q, got %q", tc.Client.Engine, gotEngine)
				}
				if !engineVersionOK {
					t.Errorf("  EngineVersion: expected %q, got %q", tc.Client.EngineVersion, gotEngineVersion)
				}
			}
		})
	}

	t.Logf("Results: %d/%d passed (%.1f%%)", passed, len(fixtures), float64(passed)/float64(len(fixtures))*100)
}

// loadDeviceFixtures loads device fixtures from the PHP test fixture files.
func loadDeviceFixtures(t *testing.T) []deviceFixture {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")

	// PHP fixture files are in device-detector/Tests/Parser/Device/fixtures/
	baseDir := filepath.Join(filepath.Dir(filename), "..", "..", "..", "device-detector", "Tests", "Parser", "Device", "fixtures")

	fixtureFiles := []string{
		"camera.yml",
		"car_browser.yml",
		"console.yml",
		"notebook.yml",
	}

	var allFixtures []deviceFixture

	for _, file := range fixtureFiles {
		fixturePath := filepath.Join(baseDir, file)
		data, err := os.ReadFile(fixturePath)
		require.NoError(t, err, "failed to read fixture file: %s", file)

		var fixtures []deviceFixture
		err = yaml.Unmarshal(data, &fixtures)
		require.NoError(t, err, "failed to parse fixture YAML: %s", file)

		allFixtures = append(allFixtures, fixtures...)
	}

	return allFixtures
}

// TestParseDevice tests the full DeviceDetector pipeline validating device output.
// PHP: DeviceDetectorTest::testParseDevice
// This runs the FULL detection pipeline but only validates the device field.
func TestParseDevice(t *testing.T) {
	dd, err := New()
	require.NoError(t, err, "failed to create DeviceDetector")

	fixtures := loadDeviceFixtures(t)
	require.NotEmpty(t, fixtures, "no fixtures loaded")

	t.Logf("Running %d device test cases", len(fixtures))

	passed := 0
	failed := 0

	for i, tc := range fixtures {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			result := dd.Parse(tc.UserAgent, nil)

			// PHP assertion: assertArrayNotHasKey('bot', $uaInfo)
			// Ensure none of the device fixtures are incorrectly detected as bots
			if !assert.False(t, result.IsBot(), "UA incorrectly detected as bot: %s", tc.UserAgent) {
				failed++
				return
			}

			// Get device info
			gotType := DeviceTypeNames[result.GetDevice()]
			gotBrand := result.GetBrand()
			gotModel := result.GetModel()

			// Compare with expected
			typeOK := tc.Device.Type == gotType
			brandOK := tc.Device.Brand == gotBrand
			modelOK := tc.Device.Model == gotModel

			if typeOK && brandOK && modelOK {
				passed++
			} else {
				failed++
				t.Errorf("Device mismatch for UA: %s", tc.UserAgent)
				if !typeOK {
					t.Errorf("  Type: expected %q, got %q", tc.Device.Type, gotType)
				}
				if !brandOK {
					t.Errorf("  Brand: expected %q, got %q", tc.Device.Brand, gotBrand)
				}
				if !modelOK {
					t.Errorf("  Model: expected %q, got %q", tc.Device.Model, gotModel)
				}
			}
		})
	}

	t.Logf("Results: %d/%d passed (%.1f%%)", passed, len(fixtures), float64(passed)/float64(len(fixtures))*100)
}

// TestDetectDeviceTypeFromClientHints tests device type detection from form factors in client hints.
// PHP equivalent: DeviceDetectorTest::testDetectDeviceTypeFromClientHints
func TestDetectDeviceTypeFromClientHints(t *testing.T) {
	dd, err := New()
	require.NoError(t, err, "failed to create DeviceDetector")

	useragent := "Some Unknown UA"
	deviceName := `"Some Unknown Model"`

	testCases := []struct {
		name    string
		headers map[string]interface{}
		device  DeviceType
	}{
		{
			name: "EInk and Watch = Wearable",
			headers: map[string]interface{}{
				"sec-ch-ua-form-factors": `"EInk", "Watch"`,
				"sec-ch-ua-model":        deviceName,
			},
			device: DeviceTypeWearable,
		},
		{
			name: "EInk alone = Tablet",
			headers: map[string]interface{}{
				"sec-ch-ua-form-factors": `"EInk"`,
				"sec-ch-ua-model":        deviceName,
			},
			device: DeviceTypeTablet,
		},
		{
			name: "Desktop and Mobile = Smartphone",
			headers: map[string]interface{}{
				"sec-ch-ua-form-factors": `"Desktop", "Mobile"`,
				"sec-ch-ua-model":        deviceName,
			},
			device: DeviceTypeSmartphone,
		},
		{
			name: "Unknown and Mobile = Smartphone",
			headers: map[string]interface{}{
				"sec-ch-ua-form-factors": `"Unknown Type", "Mobile"`,
				"sec-ch-ua-model":        deviceName,
			},
			device: DeviceTypeSmartphone,
		},
		{
			name: "Tablet and Mobile = Smartphone",
			headers: map[string]interface{}{
				"sec-ch-ua-form-factors": `"Tablet", "Mobile"`,
				"sec-ch-ua-model":        deviceName,
			},
			device: DeviceTypeSmartphone,
		},
		{
			name: "EInk and Tablet = Tablet",
			headers: map[string]interface{}{
				"sec-ch-ua-form-factors": `"EInk", "Tablet"`,
				"sec-ch-ua-model":        deviceName,
			},
			device: DeviceTypeTablet,
		},
		{
			name: "Tablet and Automotive = CarBrowser",
			headers: map[string]interface{}{
				"sec-ch-ua-form-factors": `"Tablet", "Automotive"`,
				"sec-ch-ua-model":        deviceName,
			},
			device: DeviceTypeCarBrowser,
		},
		{
			name: "EInk and Xr = Wearable",
			headers: map[string]interface{}{
				"sec-ch-ua-form-factors": `"EInk", "Xr"`,
				"sec-ch-ua-model":        deviceName,
			},
			device: DeviceTypeWearable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ch := clienthints.Factory(tc.headers)
			result := dd.Parse(useragent, ch)

			assert.Equal(t, tc.device, result.GetDevice(),
				"Device type mismatch for form factors")
			assert.Equal(t, "Some Unknown Model", result.GetModel(),
				"Model should come from client hints")
			assert.Equal(t, "", result.GetBrand(),
				"Brand should be empty (no regex match)")
			assert.Equal(t, DeviceTypeNames[tc.device], DeviceTypeNames[result.GetDevice()],
				"Device name should match")
		})
	}
}

// TestDetectDeviceTypeFromFormFactors tests the form factor to device type mapping logic.
func TestDetectDeviceTypeFromFormFactors(t *testing.T) {
	testCases := []struct {
		name        string
		formFactors []string
		expected    DeviceType
	}{
		{"empty", []string{}, DeviceTypeUnknown},
		{"watch", []string{"watch"}, DeviceTypeWearable},
		{"xr", []string{"xr"}, DeviceTypeWearable},
		{"automotive", []string{"automotive"}, DeviceTypeCarBrowser},
		{"mobile", []string{"mobile"}, DeviceTypeSmartphone},
		{"eink", []string{"eink"}, DeviceTypeTablet},
		{"tablet", []string{"tablet"}, DeviceTypeTablet},
		{"desktop", []string{"desktop"}, DeviceTypeUnknown}, // desktop alone is unknown
		{"watch beats xr", []string{"xr", "watch"}, DeviceTypeWearable},
		{"xr beats automotive", []string{"automotive", "xr"}, DeviceTypeWearable},
		{"automotive beats mobile", []string{"mobile", "automotive"}, DeviceTypeCarBrowser},
		{"mobile beats eink", []string{"eink", "mobile"}, DeviceTypeSmartphone},
		{"mobile beats tablet", []string{"tablet", "mobile"}, DeviceTypeSmartphone},
		{"eink beats tablet", []string{"tablet", "eink"}, DeviceTypeTablet},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detectDeviceTypeFromFormFactors(tc.formFactors)
			assert.Equal(t, tc.expected, result)
		})
	}
}
