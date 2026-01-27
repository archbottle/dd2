package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/archbottle/device-detector/pkg/bots"
	"github.com/archbottle/device-detector/pkg/browser"
	"github.com/archbottle/device-detector/pkg/camera"
	"github.com/archbottle/device-detector/pkg/carbrowser"
	"github.com/archbottle/device-detector/pkg/clienthints"
	"github.com/archbottle/device-detector/pkg/console"
	"github.com/archbottle/device-detector/pkg/detector"
	"github.com/archbottle/device-detector/pkg/feedreader"
	"github.com/archbottle/device-detector/pkg/library"
	"github.com/archbottle/device-detector/pkg/mediaplayer"
	"github.com/archbottle/device-detector/pkg/mobileapp"
	"github.com/archbottle/device-detector/pkg/notebook"
	"github.com/archbottle/device-detector/pkg/operatingsystem"
	"github.com/archbottle/device-detector/pkg/pim"
	"github.com/archbottle/device-detector/pkg/vendorfragment"
	"gopkg.in/yaml.v3"
)

// getBaseDir returns the base directory for regexes and fixtures
func getBaseDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "..")
}

// allMatch returns true if all fields match
func allMatch(fields []FieldDiff) bool {
	for _, f := range fields {
		if !f.Matches {
			return false
		}
	}
	return true
}

// safeStr returns empty string if pointer is nil
func safeStr(m interface{}, field string) string {
	if m == nil {
		return ""
	}
	return field
}

// ============================================================================
// Browser Collector
// ============================================================================

type browserFixture struct {
	UserAgent string            `yaml:"user_agent"`
	Headers   map[string]string `yaml:"headers"`
	Client    struct {
		Type          string `yaml:"type"`
		Name          string `yaml:"name"`
		Version       string `yaml:"version"`
		Engine        string `yaml:"engine"`
		EngineVersion string `yaml:"engine_version"`
		Family        string `yaml:"family"`
	} `yaml:"client"`
}

func CollectBrowserResults() (ParserResult, error) {
	result := ParserResult{Name: "Browser"}
	baseDir := getBaseDir()

	// Load factory
	factory, err := browser.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "client", "browsers.yml"),
		filepath.Join(baseDir, "..", "regexes", "client", "browser_engine.yml"),
		filepath.Join(baseDir, "..", "regexes", "client", "hints", "browsers.yml"),
	)
	if err != nil {
		return result, err
	}

	// Load fixtures
	data, err := os.ReadFile(filepath.Join(baseDir, "browser", "fixtures", "browser.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []browserFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		var opts []browser.Option
		if len(tc.Headers) > 0 {
			ch := clienthints.New(tc.Headers)
			opts = append(opts, browser.WithClientHints(ch))
		}

		got := factory.Parse(tc.UserAgent, opts...)

		var gotType, gotName, gotVersion, gotEngine, gotEngineVersion, gotFamily string
		if got != nil {
			gotType = got.Type
			gotName = got.Name
			gotVersion = got.Version
			gotEngine = got.Engine
			gotEngineVersion = got.EngineVersion
			gotFamily = got.Family
		}

		fields := []FieldDiff{
			{Name: "Type", Expected: tc.Client.Type, Actual: gotType, Matches: tc.Client.Type == gotType},
			{Name: "Name", Expected: tc.Client.Name, Actual: gotName, Matches: tc.Client.Name == gotName},
			{Name: "Version", Expected: tc.Client.Version, Actual: gotVersion, Matches: tc.Client.Version == gotVersion},
			{Name: "Engine", Expected: tc.Client.Engine, Actual: gotEngine, Matches: tc.Client.Engine == gotEngine},
			{Name: "EngineVersion", Expected: tc.Client.EngineVersion, Actual: gotEngineVersion, Matches: tc.Client.EngineVersion == gotEngineVersion},
			{Name: "Family", Expected: tc.Client.Family, Actual: gotFamily, Matches: tc.Client.Family == gotFamily},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

// ============================================================================
// Operating System Collector
// ============================================================================

type osFixture struct {
	UserAgent string            `yaml:"user_agent"`
	Headers   map[string]string `yaml:"headers"`
	OS        struct {
		Name      string `yaml:"name"`
		ShortName string `yaml:"short_name"`
		Version   string `yaml:"version"`
		Platform  string `yaml:"platform"`
		Family    string `yaml:"family"`
	} `yaml:"os"`
}

func CollectOSResults() (ParserResult, error) {
	result := ParserResult{Name: "Operating System"}
	baseDir := getBaseDir()

	factory, err := operatingsystem.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "oss.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "operatingsystem", "fixtures", "oss.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []osFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		var opts []operatingsystem.Option
		if len(tc.Headers) > 0 {
			ch := clienthints.New(tc.Headers)
			opts = append(opts, operatingsystem.WithClientHints(ch))
		}

		got := factory.Parse(tc.UserAgent, opts...)

		// Handle empty OS expectation
		if tc.OS.Name == "" && got == nil {
			result.Passed++
			continue
		}

		var gotName, gotShortName, gotVersion, gotPlatform, gotFamily string
		if got != nil {
			gotName = got.Name
			gotShortName = got.ShortName
			gotVersion = got.Version
			gotPlatform = got.Platform
			gotFamily = got.Family
		}

		fields := []FieldDiff{
			{Name: "Name", Expected: tc.OS.Name, Actual: gotName, Matches: tc.OS.Name == gotName},
			{Name: "ShortName", Expected: tc.OS.ShortName, Actual: gotShortName, Matches: tc.OS.ShortName == gotShortName},
			{Name: "Version", Expected: tc.OS.Version, Actual: gotVersion, Matches: tc.OS.Version == gotVersion},
			{Name: "Platform", Expected: tc.OS.Platform, Actual: gotPlatform, Matches: tc.OS.Platform == gotPlatform},
			{Name: "Family", Expected: tc.OS.Family, Actual: gotFamily, Matches: tc.OS.Family == gotFamily},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

// ============================================================================
// Device Collectors (camera, console, carbrowser, notebook)
// ============================================================================

type deviceFixture struct {
	UserAgent string `yaml:"user_agent"`
	Device    struct {
		Type  string `yaml:"type"`
		Brand string `yaml:"brand"`
		Model string `yaml:"model"`
	} `yaml:"device"`
}

func CollectCameraResults() (ParserResult, error) {
	result := ParserResult{Name: "Camera"}
	baseDir := getBaseDir()

	factory, err := camera.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "device", "cameras.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "camera", "fixtures", "camera.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []deviceFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		got := factory.Parse(tc.UserAgent)

		var gotType, gotBrand, gotModel string
		if got != nil {
			gotType = got.Type
			gotBrand = got.Brand
			gotModel = got.Model
		}

		fields := []FieldDiff{
			{Name: "Type", Expected: tc.Device.Type, Actual: gotType, Matches: tc.Device.Type == gotType},
			{Name: "Brand", Expected: tc.Device.Brand, Actual: gotBrand, Matches: tc.Device.Brand == gotBrand},
			{Name: "Model", Expected: tc.Device.Model, Actual: gotModel, Matches: tc.Device.Model == gotModel},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

func CollectConsoleResults() (ParserResult, error) {
	result := ParserResult{Name: "Console"}
	baseDir := getBaseDir()

	factory, err := console.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "device", "consoles.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "console", "fixtures", "console.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []deviceFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		p := factory.NewParser(tc.UserAgent)
		got := p.Parse()

		var gotType, gotBrand, gotModel string
		if got != nil {
			gotType = got.Type
			gotBrand = got.Brand
			gotModel = got.Model
		}

		fields := []FieldDiff{
			{Name: "Type", Expected: tc.Device.Type, Actual: gotType, Matches: tc.Device.Type == gotType},
			{Name: "Brand", Expected: tc.Device.Brand, Actual: gotBrand, Matches: tc.Device.Brand == gotBrand},
			{Name: "Model", Expected: tc.Device.Model, Actual: gotModel, Matches: tc.Device.Model == gotModel},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

func CollectCarBrowserResults() (ParserResult, error) {
	result := ParserResult{Name: "Car Browser"}
	baseDir := getBaseDir()

	factory, err := carbrowser.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "device", "car_browsers.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "carbrowser", "fixtures", "car_browser.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []deviceFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		p := factory.NewParser(tc.UserAgent)
		got := p.Parse()

		var gotType, gotBrand, gotModel string
		if got != nil {
			gotType = got.Type
			gotBrand = got.Brand
			gotModel = got.Model
		}

		fields := []FieldDiff{
			{Name: "Type", Expected: tc.Device.Type, Actual: gotType, Matches: tc.Device.Type == gotType},
			{Name: "Brand", Expected: tc.Device.Brand, Actual: gotBrand, Matches: tc.Device.Brand == gotBrand},
			{Name: "Model", Expected: tc.Device.Model, Actual: gotModel, Matches: tc.Device.Model == gotModel},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

func CollectNotebookResults() (ParserResult, error) {
	result := ParserResult{Name: "Notebook"}
	baseDir := getBaseDir()

	factory, err := notebook.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "device", "notebooks.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "notebook", "fixtures", "notebook.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []deviceFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		p := factory.NewParser(tc.UserAgent)
		got := p.Parse()

		var gotType, gotBrand, gotModel string
		if got != nil {
			gotType = got.Type
			gotBrand = got.Brand
			gotModel = got.Model
		}

		fields := []FieldDiff{
			{Name: "Type", Expected: tc.Device.Type, Actual: gotType, Matches: tc.Device.Type == gotType},
			{Name: "Brand", Expected: tc.Device.Brand, Actual: gotBrand, Matches: tc.Device.Brand == gotBrand},
			{Name: "Model", Expected: tc.Device.Model, Actual: gotModel, Matches: tc.Device.Model == gotModel},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

// ============================================================================
// Client Collectors (feedreader, mobileapp, mediaplayer, pim, library)
// ============================================================================

type clientFixture struct {
	UserAgent string `yaml:"user_agent"`
	Client    struct {
		Type    string `yaml:"type"`
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"client"`
}

func CollectFeedReaderResults() (ParserResult, error) {
	result := ParserResult{Name: "Feed Reader"}
	baseDir := getBaseDir()

	factory, err := feedreader.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "client", "feed_readers.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "feedreader", "fixtures", "feed_reader.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []clientFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		got := factory.Parse(tc.UserAgent)

		var gotType, gotName, gotVersion string
		if got != nil {
			gotType = got.Type
			gotName = got.Name
			gotVersion = got.Version
		}

		fields := []FieldDiff{
			{Name: "Type", Expected: tc.Client.Type, Actual: gotType, Matches: tc.Client.Type == gotType},
			{Name: "Name", Expected: tc.Client.Name, Actual: gotName, Matches: tc.Client.Name == gotName},
			{Name: "Version", Expected: tc.Client.Version, Actual: gotVersion, Matches: tc.Client.Version == gotVersion},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

func CollectMobileAppResults() (ParserResult, error) {
	result := ParserResult{Name: "Mobile App"}
	baseDir := getBaseDir()

	factory, err := mobileapp.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "client", "mobile_apps.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "mobileapp", "fixtures", "mobile_app.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []clientFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		got := factory.Parse(tc.UserAgent)

		var gotType, gotName, gotVersion string
		if got != nil {
			gotType = got.Type
			gotName = got.Name
			gotVersion = got.Version
		}

		fields := []FieldDiff{
			{Name: "Type", Expected: tc.Client.Type, Actual: gotType, Matches: tc.Client.Type == gotType},
			{Name: "Name", Expected: tc.Client.Name, Actual: gotName, Matches: tc.Client.Name == gotName},
			{Name: "Version", Expected: tc.Client.Version, Actual: gotVersion, Matches: tc.Client.Version == gotVersion},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

func CollectMediaPlayerResults() (ParserResult, error) {
	result := ParserResult{Name: "Media Player"}
	baseDir := getBaseDir()

	factory, err := mediaplayer.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "client", "mediaplayers.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "mediaplayer", "fixtures", "mediaplayer.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []clientFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		got := factory.Parse(tc.UserAgent)

		var gotType, gotName, gotVersion string
		if got != nil {
			gotType = got.Type
			gotName = got.Name
			gotVersion = got.Version
		}

		fields := []FieldDiff{
			{Name: "Type", Expected: tc.Client.Type, Actual: gotType, Matches: tc.Client.Type == gotType},
			{Name: "Name", Expected: tc.Client.Name, Actual: gotName, Matches: tc.Client.Name == gotName},
			{Name: "Version", Expected: tc.Client.Version, Actual: gotVersion, Matches: tc.Client.Version == gotVersion},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

func CollectPIMResults() (ParserResult, error) {
	result := ParserResult{Name: "PIM"}
	baseDir := getBaseDir()

	factory, err := pim.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "client", "pim.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "pim", "fixtures", "pim.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []clientFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		got := factory.Parse(tc.UserAgent)

		var gotType, gotName, gotVersion string
		if got != nil {
			gotType = got.Type
			gotName = got.Name
			gotVersion = got.Version
		}

		fields := []FieldDiff{
			{Name: "Type", Expected: tc.Client.Type, Actual: gotType, Matches: tc.Client.Type == gotType},
			{Name: "Name", Expected: tc.Client.Name, Actual: gotName, Matches: tc.Client.Name == gotName},
			{Name: "Version", Expected: tc.Client.Version, Actual: gotVersion, Matches: tc.Client.Version == gotVersion},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

func CollectLibraryResults() (ParserResult, error) {
	result := ParserResult{Name: "Library"}
	baseDir := getBaseDir()

	factory, err := library.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "client", "libraries.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "library", "fixtures", "library.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []clientFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		got := factory.Parse(tc.UserAgent)

		var gotType, gotName, gotVersion string
		if got != nil {
			gotType = got.Type
			gotName = got.Name
			gotVersion = got.Version
		}

		fields := []FieldDiff{
			{Name: "Type", Expected: tc.Client.Type, Actual: gotType, Matches: tc.Client.Type == gotType},
			{Name: "Name", Expected: tc.Client.Name, Actual: gotName, Matches: tc.Client.Name == gotName},
			{Name: "Version", Expected: tc.Client.Version, Actual: gotVersion, Matches: tc.Client.Version == gotVersion},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

// ============================================================================
// Vendor Fragment Collector
// ============================================================================

type vendorFixture struct {
	UserAgent string `yaml:"useragent"`
	Vendor    string `yaml:"vendor"`
}

func CollectVendorFragmentResults() (ParserResult, error) {
	result := ParserResult{Name: "Vendor Fragment"}
	baseDir := getBaseDir()

	factory, err := vendorfragment.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "vendorfragments.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "vendorfragment", "fixtures", "vendorfragments.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []vendorFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		p := factory.NewParser(tc.UserAgent)
		got := p.Parse()

		var gotBrand string
		if got != nil {
			gotBrand = got["brand"]
		}

		fields := []FieldDiff{
			{Name: "Vendor", Expected: tc.Vendor, Actual: gotBrand, Matches: tc.Vendor == gotBrand},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

// ============================================================================
// Bot Collector
// ============================================================================

type botFixture struct {
	UserAgent string `yaml:"user_agent"`
	Bot       struct {
		Name     string `yaml:"name"`
		Category string `yaml:"category,omitempty"`
		URL      string `yaml:"url,omitempty"`
		Producer *struct {
			Name string `yaml:"name"`
			URL  string `yaml:"url"`
		} `yaml:"producer,omitempty"`
	} `yaml:"bot"`
}

func CollectBotResults() (ParserResult, error) {
	result := ParserResult{Name: "Bot"}
	baseDir := getBaseDir()

	factory, err := bots.NewParserFactory(
		filepath.Join(baseDir, "..", "regexes", "bots.yml"),
	)
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "bots", "fixtures", "bots.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []botFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		got := factory.Parse(tc.UserAgent)

		// Handle case where bot is not detected
		if got == nil {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields: []FieldDiff{
					{Name: "Detected", Expected: "true", Actual: "false", Matches: false},
					{Name: "Name", Expected: tc.Bot.Name, Actual: "", Matches: false},
				},
			})
			continue
		}

		// Check producer
		var expectedProducerName, expectedProducerURL string
		var gotProducerName, gotProducerURL string
		if tc.Bot.Producer != nil {
			expectedProducerName = tc.Bot.Producer.Name
			expectedProducerURL = tc.Bot.Producer.URL
		}
		if got.Producer != nil {
			gotProducerName = got.Producer.Name
			gotProducerURL = got.Producer.URL
		}

		fields := []FieldDiff{
			{Name: "Name", Expected: tc.Bot.Name, Actual: got.Name, Matches: tc.Bot.Name == got.Name},
			{Name: "Category", Expected: tc.Bot.Category, Actual: got.Category, Matches: tc.Bot.Category == got.Category},
			{Name: "URL", Expected: tc.Bot.URL, Actual: got.URL, Matches: tc.Bot.URL == got.URL},
			{Name: "Producer.Name", Expected: expectedProducerName, Actual: gotProducerName, Matches: expectedProducerName == gotProducerName},
			{Name: "Producer.URL", Expected: expectedProducerURL, Actual: gotProducerURL, Matches: expectedProducerURL == gotProducerURL},
		}

		if allMatch(fields) {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

// ============================================================================
// Type Methods Collector (testTypeMethods integration test)
// ============================================================================

type typeMethodFixture struct {
	UserAgent string `yaml:"user_agent"`
	Check     []bool `yaml:"check"`
}

func CollectTypeMethodsResults() (ParserResult, error) {
	result := ParserResult{Name: "Type Methods"}
	baseDir := getBaseDir()

	regexesDir := filepath.Join(baseDir, "..", "regexes")
	dd, err := detector.New(regexesDir, detector.WithDiscardBotInformation())
	if err != nil {
		return result, err
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "detector", "fixtures", "type-methods.yml"))
	if err != nil {
		return result, err
	}
	var fixtures []typeMethodFixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return result, err
	}

	for i, tc := range fixtures {
		parsed := dd.Parse(tc.UserAgent, nil)

		// Check format: [isBot(), isMobile(), isDesktop(), isTablet(), isTV(), isWearable()]
		got := []bool{
			parsed.IsBot(),
			parsed.IsMobile(),
			parsed.IsDesktop(),
			parsed.IsTablet(),
			parsed.IsTV(),
			parsed.IsWearable(),
		}

		// Compare arrays
		matches := len(tc.Check) == len(got)
		if matches {
			for j := range tc.Check {
				if tc.Check[j] != got[j] {
					matches = false
					break
				}
			}
		}

		fields := []FieldDiff{
			{Name: "isBot", Expected: fmt.Sprintf("%v", tc.Check[0]), Actual: fmt.Sprintf("%v", got[0]), Matches: tc.Check[0] == got[0]},
			{Name: "isMobile", Expected: fmt.Sprintf("%v", tc.Check[1]), Actual: fmt.Sprintf("%v", got[1]), Matches: tc.Check[1] == got[1]},
			{Name: "isDesktop", Expected: fmt.Sprintf("%v", tc.Check[2]), Actual: fmt.Sprintf("%v", got[2]), Matches: tc.Check[2] == got[2]},
			{Name: "isTablet", Expected: fmt.Sprintf("%v", tc.Check[3]), Actual: fmt.Sprintf("%v", got[3]), Matches: tc.Check[3] == got[3]},
			{Name: "isTV", Expected: fmt.Sprintf("%v", tc.Check[4]), Actual: fmt.Sprintf("%v", got[4]), Matches: tc.Check[4] == got[4]},
			{Name: "isWearable", Expected: fmt.Sprintf("%v", tc.Check[5]), Actual: fmt.Sprintf("%v", got[5]), Matches: tc.Check[5] == got[5]},
		}

		if matches {
			result.Passed++
		} else {
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				CaseIndex: i,
				UserAgent: tc.UserAgent,
				Fields:    fields,
			})
		}
	}

	return result, nil
}

// ============================================================================
// Parse Device Collector (testParseDevice integration test)
// ============================================================================

func CollectParseDeviceResults() (ParserResult, error) {
	result := ParserResult{Name: "Parse Device (Integration)"}
	baseDir := getBaseDir()

	regexesDir := filepath.Join(baseDir, "..", "regexes")
	dd, err := detector.New(regexesDir)
	if err != nil {
		return result, err
	}

	// Load fixtures from PHP test fixture files
	// Path: dd2/device-detector/Tests/Parser/Device/fixtures/
	phpFixturesDir := filepath.Join(baseDir, "..", "..", "device-detector", "Tests", "Parser", "Device", "fixtures")
	fixtureFiles := []string{"camera.yml", "car_browser.yml", "console.yml", "notebook.yml"}

	for _, file := range fixtureFiles {
		data, err := os.ReadFile(filepath.Join(phpFixturesDir, file))
		if err != nil {
			return result, err
		}
		var fixtures []deviceFixture
		if err := yaml.Unmarshal(data, &fixtures); err != nil {
			return result, err
		}

		for i, tc := range fixtures {
			parsed := dd.Parse(tc.UserAgent, nil)

			// PHP assertion: assertArrayNotHasKey('bot', $uaInfo)
			if parsed.IsBot() {
				result.Failed++
				result.Failures = append(result.Failures, TestFailure{
					CaseIndex: i,
					UserAgent: tc.UserAgent,
					Fields: []FieldDiff{
						{Name: "IsBot", Expected: "false", Actual: "true", Matches: false},
					},
				})
				continue
			}

			// Get device info
			gotType := detector.DeviceTypeNames[parsed.GetDevice()]
			gotBrand := parsed.GetBrand()
			gotModel := parsed.GetModel()

			fields := []FieldDiff{
				{Name: "Type", Expected: tc.Device.Type, Actual: gotType, Matches: tc.Device.Type == gotType},
				{Name: "Brand", Expected: tc.Device.Brand, Actual: gotBrand, Matches: tc.Device.Brand == gotBrand},
				{Name: "Model", Expected: tc.Device.Model, Actual: gotModel, Matches: tc.Device.Model == gotModel},
			}

			if allMatch(fields) {
				result.Passed++
			} else {
				result.Failed++
				result.Failures = append(result.Failures, TestFailure{
					CaseIndex: i,
					UserAgent: tc.UserAgent,
					Fields:    fields,
				})
			}
		}
	}

	return result, nil
}

// ============================================================================
// Parse Client Collector (testParseClient integration test)
// ============================================================================

// clientFixtureReport matches the PHP client fixture format for reporting.
type clientFixtureReport struct {
	UserAgent string            `yaml:"user_agent"`
	Headers   map[string]string `yaml:"headers"`
	Client    struct {
		Type          string `yaml:"type"`
		Name          string `yaml:"name"`
		Version       string `yaml:"version"`
		Engine        string `yaml:"engine"`
		EngineVersion string `yaml:"engine_version"`
		Family        string `yaml:"family"`
	} `yaml:"client"`
}

func CollectParseClientResults() (ParserResult, error) {
	result := ParserResult{Name: "Parse Client (Integration)"}
	baseDir := getBaseDir()

	regexesDir := filepath.Join(baseDir, "..", "regexes")
	dd, err := detector.New(regexesDir)
	if err != nil {
		return result, err
	}

	// Load fixtures from PHP test fixture files
	// Path: dd2/device-detector/Tests/Parser/Client/fixtures/
	phpFixturesDir := filepath.Join(baseDir, "..", "..", "device-detector", "Tests", "Parser", "Client", "fixtures")
	fixtureFiles := []string{"browser.yml", "feed_reader.yml", "library.yml", "mediaplayer.yml", "mobile_app.yml", "pim.yml"}

	for _, file := range fixtureFiles {
		data, err := os.ReadFile(filepath.Join(phpFixturesDir, file))
		if err != nil {
			return result, err
		}
		var fixtures []clientFixtureReport
		if err := yaml.Unmarshal(data, &fixtures); err != nil {
			return result, err
		}

		for i, tc := range fixtures {
			// Build client hints if headers present
			var ch *clienthints.ClientHints
			if len(tc.Headers) > 0 {
				ch = clienthints.New(tc.Headers)
			}

			parsed := dd.Parse(tc.UserAgent, ch)

			// PHP assertion: assertArrayNotHasKey('bot', $uaInfo)
			if parsed.IsBot() {
				result.Failed++
				result.Failures = append(result.Failures, TestFailure{
					CaseIndex: i,
					UserAgent: tc.UserAgent,
					Fields: []FieldDiff{
						{Name: "IsBot", Expected: "false", Actual: "true", Matches: false},
					},
				})
				continue
			}

			// Get client info
			client := parsed.GetClient()

			var gotType, gotName, gotVersion, gotEngine, gotEngineVersion string
			if client != nil {
				gotType = client.Type
				gotName = client.Name
				gotVersion = client.Version
				gotEngine = client.Engine
				gotEngineVersion = client.EngineVersion
			}

			// NOTE: family is excluded from comparison (as per PHP test)
			fields := []FieldDiff{
				{Name: "Type", Expected: tc.Client.Type, Actual: gotType, Matches: tc.Client.Type == gotType},
				{Name: "Name", Expected: tc.Client.Name, Actual: gotName, Matches: tc.Client.Name == gotName},
				{Name: "Version", Expected: tc.Client.Version, Actual: gotVersion, Matches: tc.Client.Version == gotVersion},
				{Name: "Engine", Expected: tc.Client.Engine, Actual: gotEngine, Matches: tc.Client.Engine == gotEngine},
				{Name: "EngineVersion", Expected: tc.Client.EngineVersion, Actual: gotEngineVersion, Matches: tc.Client.EngineVersion == gotEngineVersion},
			}

			if allMatch(fields) {
				result.Passed++
			} else {
				result.Failed++
				result.Failures = append(result.Failures, TestFailure{
					CaseIndex: i,
					UserAgent: tc.UserAgent,
					Fields:    fields,
				})
			}
		}
	}

	return result, nil
}

// ============================================================================
// Full Parse Collector (testParse integration test - the comprehensive one)
// ============================================================================

// fullParseFixtureReport matches the PHP testParse fixture format.
// Note: OS/Client/Device can be empty arrays in PHP fixtures, so we use interface{}.
type fullParseFixtureReport struct {
	UserAgent     string      `yaml:"user_agent"`
	Headers       interface{} `yaml:"headers"` // Can be map or complex client hints
	OS            interface{} `yaml:"os"`
	Client        interface{} `yaml:"client"`
	Device        interface{} `yaml:"device"`
	OSFamily      string      `yaml:"os_family"`
	BrowserFamily string      `yaml:"browser_family"`
}

func (f *fullParseFixtureReport) getOS() (name, version, platform string) {
	if m, ok := f.OS.(map[string]interface{}); ok {
		name, _ = m["name"].(string)
		version, _ = m["version"].(string)
		platform, _ = m["platform"].(string)
	}
	return
}

func (f *fullParseFixtureReport) getClient() (typ, name, version, engine, engineVersion string) {
	if m, ok := f.Client.(map[string]interface{}); ok {
		typ, _ = m["type"].(string)
		name, _ = m["name"].(string)
		version, _ = m["version"].(string)
		engine, _ = m["engine"].(string)
		engineVersion, _ = m["engine_version"].(string)
	}
	return
}

func (f *fullParseFixtureReport) getDevice() (typ, brand, model string) {
	if m, ok := f.Device.(map[string]interface{}); ok {
		typ, _ = m["type"].(string)
		brand, _ = m["brand"].(string)
		model, _ = m["model"].(string)
	}
	return
}

func (f *fullParseFixtureReport) getHeaders() map[string]string {
	if f.Headers == nil {
		return nil
	}
	if m, ok := f.Headers.(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range m {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return nil
}

func CollectFullParseResults() (ParserResult, error) {
	result := ParserResult{Name: "Full Parse (Integration)"}
	baseDir := getBaseDir()

	regexesDir := filepath.Join(baseDir, "..", "regexes")
	dd, err := detector.New(regexesDir)
	if err != nil {
		return result, err
	}

	// Load fixtures from all PHP testParse fixture files
	phpFixturesDir := filepath.Join(baseDir, "..", "..", "device-detector", "Tests", "fixtures")
	entries, err := os.ReadDir(phpFixturesDir)
	if err != nil {
		return result, err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
			continue
		}
		if entry.Name() == "bots.yml" {
			continue // bots handled separately
		}

		data, err := os.ReadFile(filepath.Join(phpFixturesDir, entry.Name()))
		if err != nil {
			return result, err
		}
		var fixtures []fullParseFixtureReport
		if err := yaml.Unmarshal(data, &fixtures); err != nil {
			return result, err
		}

		for i, tc := range fixtures {
			// Build client hints if headers present
			var ch *clienthints.ClientHints
			headers := tc.getHeaders()
			if len(headers) > 0 {
				ch = clienthints.New(headers)
			}

			parsed := dd.Parse(tc.UserAgent, ch)
			info := parsed.GetFullInfo()

			// Extract expected values
			expOSName, expOSVer, expOSPlat := tc.getOS()
			expClientType, expClientName, expClientVer, _, _ := tc.getClient()
			expDevType, expDevBrand, expDevModel := tc.getDevice()
			expOSFamily := tc.OSFamily
			expBrowserFamily := tc.BrowserFamily

			// Extract actual values
			var gotOSName, gotOSVer, gotOSPlat, gotOSFamily string
			if info.OS != nil {
				gotOSName = info.OS.Name
				gotOSVer = info.OS.Version
				gotOSPlat = info.OS.Platform
			}
			gotOSFamily = info.OSFamily

			var gotClientType, gotClientName, gotClientVer string
			if info.Client != nil {
				gotClientType = info.Client.Type
				gotClientName = info.Client.Name
				gotClientVer = info.Client.Version
			}
			gotBrowserFamily := info.BrowserFamily

			var gotDevType, gotDevBrand, gotDevModel string
			if info.Device != nil {
				gotDevType = info.Device.Type
				gotDevBrand = info.Device.Brand
				gotDevModel = info.Device.Model
			}

			fields := []FieldDiff{
				{Name: "OS.Name", Expected: expOSName, Actual: gotOSName, Matches: expOSName == gotOSName},
				{Name: "OS.Version", Expected: expOSVer, Actual: gotOSVer, Matches: expOSVer == gotOSVer},
				{Name: "OS.Platform", Expected: expOSPlat, Actual: gotOSPlat, Matches: expOSPlat == gotOSPlat},
				{Name: "OS.Family", Expected: expOSFamily, Actual: gotOSFamily, Matches: expOSFamily == gotOSFamily},
				{Name: "Client.Type", Expected: expClientType, Actual: gotClientType, Matches: expClientType == gotClientType},
				{Name: "Client.Name", Expected: expClientName, Actual: gotClientName, Matches: expClientName == gotClientName},
				{Name: "Client.Version", Expected: expClientVer, Actual: gotClientVer, Matches: expClientVer == gotClientVer},
				{Name: "Browser.Family", Expected: expBrowserFamily, Actual: gotBrowserFamily, Matches: expBrowserFamily == gotBrowserFamily},
				{Name: "Device.Type", Expected: expDevType, Actual: gotDevType, Matches: expDevType == gotDevType},
				{Name: "Device.Brand", Expected: expDevBrand, Actual: gotDevBrand, Matches: expDevBrand == gotDevBrand},
				{Name: "Device.Model", Expected: expDevModel, Actual: gotDevModel, Matches: expDevModel == gotDevModel},
			}

			if allMatch(fields) {
				result.Passed++
			} else {
				result.Failed++
				// Limit failures stored to prevent memory bloat (only store first 100)
				if len(result.Failures) < 100 {
					result.Failures = append(result.Failures, TestFailure{
						CaseIndex: i,
						UserAgent: tc.UserAgent,
						Fields:    fields,
					})
				}
			}
		}
	}

	return result, nil
}

// ============================================================================
// Collect All
// ============================================================================

// CollectAll runs all collectors and returns combined results.
// If includeFull is true, includes the comprehensive full parse test (36K+ tests, slow).
func CollectAll(includeFull bool) (*Report, error) {
	report := &Report{}

	collectors := []func() (ParserResult, error){
		CollectBotResults,
		CollectBrowserResults,
		CollectOSResults,
		CollectCameraResults,
		CollectConsoleResults,
		CollectCarBrowserResults,
		CollectNotebookResults,
		CollectFeedReaderResults,
		CollectMobileAppResults,
		CollectMediaPlayerResults,
		CollectPIMResults,
		CollectLibraryResults,
		CollectVendorFragmentResults,
		CollectTypeMethodsResults,
		CollectParseDeviceResults,
		CollectParseClientResults,
	}

	if includeFull {
		collectors = append(collectors, CollectFullParseResults)
	}

	for _, collect := range collectors {
		result, err := collect()
		if err != nil {
			return nil, err
		}
		report.Parsers = append(report.Parsers, result)
	}

	report.Calculate()
	return report, nil
}
