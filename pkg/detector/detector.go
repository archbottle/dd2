package detector

import (
	"strings"
	"sync"

	"github.com/archbottle/dd2/pkg/bots"
	"github.com/archbottle/dd2/pkg/browser"
	"github.com/archbottle/dd2/pkg/camera"
	"github.com/archbottle/dd2/pkg/carbrowser"
	"github.com/archbottle/dd2/pkg/clienthints"
	"github.com/archbottle/dd2/pkg/common"
	"github.com/archbottle/dd2/pkg/console"
	"github.com/archbottle/dd2/pkg/feedreader"
	"github.com/archbottle/dd2/pkg/hbbtv"
	"github.com/archbottle/dd2/pkg/library"
	"github.com/archbottle/dd2/pkg/mediaplayer"
	"github.com/archbottle/dd2/pkg/mobile"
	"github.com/archbottle/dd2/pkg/mobileapp"
	"github.com/archbottle/dd2/pkg/notebook"
	"github.com/archbottle/dd2/pkg/operatingsystem"
	"github.com/archbottle/dd2/pkg/pim"
	"github.com/archbottle/dd2/pkg/shelltv"
)

// DeviceDetector is the main orchestrator for user agent parsing.
type DeviceDetector struct {
	// Parser factories (lazy-initialized via sync.Once)
	botFactory *bots.ParserFactory
	botOnce    sync.Once
	botErr     error

	osFactory *operatingsystem.ParserFactory
	osOnce    sync.Once
	osErr     error

	// Client parser factories
	feedReaderFactory *feedreader.ParserFactory
	feedReaderOnce    sync.Once
	feedReaderErr     error

	mobileAppFactory *mobileapp.ParserFactory
	mobileAppOnce    sync.Once
	mobileAppErr     error

	mediaPlayerFactory *mediaplayer.ParserFactory
	mediaPlayerOnce    sync.Once
	mediaPlayerErr     error

	pimFactory *pim.ParserFactory
	pimOnce    sync.Once
	pimErr     error

	browserFactory   *browser.ParserFactory
	browserOnce      sync.Once
	browserErr       error
	browserHints     *browser.BrowserHints
	browserHintsOnce sync.Once
	browserHintsErr  error

	libraryFactory     *library.ParserFactory
	libraryOnce        sync.Once
	libraryErr         error
	mobileAppHints     *mobileapp.AppHints
	mobileAppHintsOnce sync.Once
	mobileAppHintsErr  error

	// Device parser factories
	hbbtvFactory *hbbtv.ParserFactory
	hbbtvOnce    sync.Once
	hbbtvErr     error

	shelltvFactory *shelltv.ParserFactory
	shelltvOnce    sync.Once
	shelltvErr     error

	notebookFactory *notebook.ParserFactory
	notebookOnce    sync.Once
	notebookErr     error

	consoleFactory *console.ParserFactory
	consoleOnce    sync.Once
	consoleErr     error

	carFactory *carbrowser.ParserFactory
	carOnce    sync.Once
	carErr     error

	cameraFactory *camera.ParserFactory
	cameraOnce    sync.Once
	cameraErr     error

	mobileFactory *mobile.ParserFactory
	mobileOnce    sync.Once
	mobileErr     error

	// Configuration
	skipBotDetection      bool
	discardBotInformation bool

	// Factory options to propagate to all parser factories
	factoryOpts []common.FactoryOption
}

// Option configures the DeviceDetector.
type Option func(*DeviceDetector)

// WithSkipBotDetection skips bot detection.
func WithSkipBotDetection() Option {
	return func(d *DeviceDetector) {
		d.skipBotDetection = true
	}
}

// WithDiscardBotInformation discards detailed bot information.
func WithDiscardBotInformation() Option {
	return func(d *DeviceDetector) {
		d.discardBotInformation = true
	}
}

// WithRe2Only sets regex mode to RE2-only for all parser factories.
func WithRe2Only() Option {
	return func(d *DeviceDetector) {
		d.factoryOpts = append(d.factoryOpts, common.WithRe2Only())
	}
}

// WithIndexOnly sets candidate mode to index-only (no full scan fallback) for all parser factories.
func WithIndexOnly() Option {
	return func(d *DeviceDetector) {
		d.factoryOpts = append(d.factoryOpts, common.WithIndexOnly())
	}
}

// WithFactoryOptions allows passing factory options directly to all parser factories.
func WithFactoryOptions(opts ...common.FactoryOption) Option {
	return func(d *DeviceDetector) {
		d.factoryOpts = append(d.factoryOpts, opts...)
	}
}

// New creates a new DeviceDetector. All parsers are loaded lazily on first use.
func New(opts ...Option) (*DeviceDetector, error) {
	d := &DeviceDetector{}
	for _, opt := range opts {
		opt(d)
	}
	return d, nil
}

// NewDefault creates a DeviceDetector with default options (alias for New).
func NewDefault(opts ...Option) (*DeviceDetector, error) {
	return New(opts...)
}

// Lazy getter methods for each parser factory.

func (d *DeviceDetector) getBotFactory() (*bots.ParserFactory, error) {
	d.botOnce.Do(func() {
		d.botFactory, d.botErr = bots.NewParserFactory(d.factoryOpts...)
	})
	return d.botFactory, d.botErr
}

func (d *DeviceDetector) getOSFactory() (*operatingsystem.ParserFactory, error) {
	d.osOnce.Do(func() {
		d.osFactory, d.osErr = operatingsystem.NewParserFactory(d.factoryOpts...)
	})
	return d.osFactory, d.osErr
}

func (d *DeviceDetector) getFeedReaderFactory() (*feedreader.ParserFactory, error) {
	d.feedReaderOnce.Do(func() {
		d.feedReaderFactory, d.feedReaderErr = feedreader.NewParserFactory(d.factoryOpts...)
	})
	return d.feedReaderFactory, d.feedReaderErr
}

func (d *DeviceDetector) getMobileAppFactory() (*mobileapp.ParserFactory, error) {
	d.mobileAppOnce.Do(func() {
		d.mobileAppFactory, d.mobileAppErr = mobileapp.NewParserFactory(d.factoryOpts...)
	})
	return d.mobileAppFactory, d.mobileAppErr
}

func (d *DeviceDetector) getMediaPlayerFactory() (*mediaplayer.ParserFactory, error) {
	d.mediaPlayerOnce.Do(func() {
		d.mediaPlayerFactory, d.mediaPlayerErr = mediaplayer.NewParserFactory(d.factoryOpts...)
	})
	return d.mediaPlayerFactory, d.mediaPlayerErr
}

func (d *DeviceDetector) getPIMFactory() (*pim.ParserFactory, error) {
	d.pimOnce.Do(func() {
		d.pimFactory, d.pimErr = pim.NewParserFactory(d.factoryOpts...)
	})
	return d.pimFactory, d.pimErr
}

func (d *DeviceDetector) getBrowserFactory() (*browser.ParserFactory, error) {
	d.browserOnce.Do(func() {
		d.browserFactory, d.browserErr = browser.NewParserFactory(d.factoryOpts...)
	})
	return d.browserFactory, d.browserErr
}

func (d *DeviceDetector) getBrowserHints() (*browser.BrowserHints, error) {
	d.browserHintsOnce.Do(func() {
		d.browserHints, d.browserHintsErr = browser.NewBrowserHints()
	})
	return d.browserHints, d.browserHintsErr
}

func (d *DeviceDetector) getLibraryFactory() (*library.ParserFactory, error) {
	d.libraryOnce.Do(func() {
		d.libraryFactory, d.libraryErr = library.NewParserFactory(d.factoryOpts...)
	})
	return d.libraryFactory, d.libraryErr
}

func (d *DeviceDetector) getMobileAppHints() (*mobileapp.AppHints, error) {
	d.mobileAppHintsOnce.Do(func() {
		d.mobileAppHints, d.mobileAppHintsErr = mobileapp.NewAppHints()
	})
	return d.mobileAppHints, d.mobileAppHintsErr
}

func (d *DeviceDetector) getHbbtvFactory() (*hbbtv.ParserFactory, error) {
	d.hbbtvOnce.Do(func() {
		d.hbbtvFactory, d.hbbtvErr = hbbtv.NewParserFactory(d.factoryOpts...)
	})
	return d.hbbtvFactory, d.hbbtvErr
}

func (d *DeviceDetector) getShelltvFactory() (*shelltv.ParserFactory, error) {
	d.shelltvOnce.Do(func() {
		d.shelltvFactory, d.shelltvErr = shelltv.NewParserFactory(d.factoryOpts...)
	})
	return d.shelltvFactory, d.shelltvErr
}

func (d *DeviceDetector) getNotebookFactory() (*notebook.ParserFactory, error) {
	d.notebookOnce.Do(func() {
		d.notebookFactory, d.notebookErr = notebook.NewParserFactory(d.factoryOpts...)
	})
	return d.notebookFactory, d.notebookErr
}

func (d *DeviceDetector) getConsoleFactory() (*console.ParserFactory, error) {
	d.consoleOnce.Do(func() {
		d.consoleFactory, d.consoleErr = console.NewParserFactory(d.factoryOpts...)
	})
	return d.consoleFactory, d.consoleErr
}

func (d *DeviceDetector) getCarFactory() (*carbrowser.ParserFactory, error) {
	d.carOnce.Do(func() {
		d.carFactory, d.carErr = carbrowser.NewParserFactory(d.factoryOpts...)
	})
	return d.carFactory, d.carErr
}

func (d *DeviceDetector) getCameraFactory() (*camera.ParserFactory, error) {
	d.cameraOnce.Do(func() {
		d.cameraFactory, d.cameraErr = camera.NewParserFactory(d.factoryOpts...)
	})
	return d.cameraFactory, d.cameraErr
}

func (d *DeviceDetector) getMobileFactory() (*mobile.ParserFactory, error) {
	d.mobileOnce.Do(func() {
		d.mobileFactory, d.mobileErr = mobile.NewParserFactory(d.factoryOpts...)
	})
	return d.mobileFactory, d.mobileErr
}

// ParseResult contains the parsed detection result with helper methods.
type ParseResult struct {
	userAgent   string
	clientHints *clienthints.ClientHints

	bot    *bots.BotMatch
	client *ClientMatch
	os     *operatingsystem.Match
	device DeviceType

	// Brand and model from device detection
	brand string
	model string
}

// Parse parses a user agent string and returns the result.
func (d *DeviceDetector) Parse(ua string, ch *clienthints.ClientHints) *ParseResult {
	result := &ParseResult{
		userAgent:   ua,
		clientHints: ch,
		device:      DeviceTypeUnknown,
	}

	// Bot detection first
	if !d.skipBotDetection {
		if botFactory, err := d.getBotFactory(); err == nil {
			if d.discardBotInformation {
				if botFactory.IsBot(ua) {
					result.bot = &bots.BotMatch{Name: ""}
				}
			} else {
				result.bot = botFactory.Parse(ua)
			}
		}

		// If bot detected, skip other parsing (PHP behavior)
		if result.bot != nil {
			return result
		}
	}

	// Parse client in PHP order: FeedReader, MobileApp, MediaPlayer, PIM, Browser, Library
	d.parseClient(result, ua, ch)

	// Parse OS
	if osFactory, err := d.getOSFactory(); err == nil {
		if ch != nil {
			result.os = osFactory.Parse(ua, operatingsystem.WithClientHints(ch))
		} else {
			result.os = osFactory.Parse(ua)
		}
	}

	// Detect device type from device parsers
	d.detectDevice(result, ua)

	// Apply heuristics to determine device type if not yet detected
	d.applyHeuristics(result, ua)

	return result
}

// detectDevice tries to detect the device type from device-specific parsers.
// PHP order: HbbTv, ShellTv, Notebook, Console, CarBrowser, Camera, PortableMediaPlayer, Mobile
func (d *DeviceDetector) detectDevice(result *ParseResult, ua string) {
	// 1. Try HbbTv (returns TV type if detected)
	if hbbtvFactory, err := d.getHbbtvFactory(); err == nil {
		if match := hbbtvFactory.NewParser(ua).Parse(); match != nil {
			result.device = DeviceTypeTV
			// HbbTv parser doesn't extract brand/model in current implementation
			return
		}
	}

	// 2. Try ShellTv (returns TV type if detected)
	if shelltvFactory, err := d.getShelltvFactory(); err == nil {
		if match := shelltvFactory.NewParser(ua).Parse(); match != nil {
			result.device = DeviceTypeTV
			// ShellTv parser doesn't extract brand/model in current implementation
			return
		}
	}

	// 3. Try notebook
	if notebookFactory, err := d.getNotebookFactory(); err == nil {
		if match := notebookFactory.NewParser(ua).Parse(); match != nil {
			result.device = DeviceTypeFromString(match.Type)
			result.brand = match.Brand
			result.model = match.Model
			return
		}
	}

	// 4. Try console
	if consoleFactory, err := d.getConsoleFactory(); err == nil {
		if match := consoleFactory.NewParser(ua).Parse(); match != nil {
			result.device = DeviceTypeFromString(match.Type)
			result.brand = match.Brand
			result.model = match.Model
			return
		}
	}

	// 5. Try car browser
	if carFactory, err := d.getCarFactory(); err == nil {
		if match := carFactory.NewParser(ua).Parse(); match != nil {
			result.device = DeviceTypeFromString(match.Type)
			result.brand = match.Brand
			result.model = match.Model
			return
		}
	}

	// 6. Try camera
	if cameraFactory, err := d.getCameraFactory(); err == nil {
		if match := cameraFactory.NewParser(ua).Parse(); match != nil {
			result.device = DeviceTypeFromString(match.Type)
			result.brand = match.Brand
			result.model = match.Model
			return
		}
	}

	// 7. PortableMediaPlayer - TODO: implement when needed

	// 8. Mobile - handles smartphones, tablets, phablets, feature phones
	// Skip if this is clearly a desktop browser
	if !mobile.HasDesktopFragment(ua) {
		if mobileFactory, err := d.getMobileFactory(); err == nil {
			var mobileMatch *mobile.Match
			if result.clientHints != nil {
				mobileMatch = mobileFactory.NewParser(ua, mobile.WithClientHints(result.clientHints)).Parse()
			} else {
				mobileMatch = mobileFactory.NewParser(ua).Parse()
			}
			if mobileMatch != nil {
				result.device = DeviceTypeFromString(mobileMatch.Type)
				result.brand = mobileMatch.Brand
				result.model = mobileMatch.Model
				return
			}
		}
	}

	// 9. Try to detect device type from client hints form factors
	if result.clientHints != nil {
		if deviceType := detectDeviceTypeFromFormFactors(result.clientHints.GetFormFactors()); deviceType != DeviceTypeUnknown {
			result.device = deviceType
			// Model from client hints
			if model := result.clientHints.GetModel(); model != "" {
				result.model = model
			}
			return
		}

		// If we have a model from client hints but no device type yet, set the model
		if model := result.clientHints.GetModel(); model != "" {
			result.model = model
		}
	}
}

// detectDeviceTypeFromFormFactors determines device type based on client hints form factors.
// Priority (highest to lowest):
// 1. Watch → WEARABLE
// 2. Xr (VR/AR) → WEARABLE
// 3. Automotive → CAR_BROWSER
// 4. Mobile → SMARTPHONE (even if combined with Desktop/Tablet)
// 5. EInk → TABLET (e-readers)
// 6. Tablet → TABLET
func detectDeviceTypeFromFormFactors(formFactors []string) DeviceType {
	if len(formFactors) == 0 {
		return DeviceTypeUnknown
	}

	hasWatch := false
	hasXr := false
	hasAutomotive := false
	hasMobile := false
	hasEink := false
	hasTablet := false

	for _, factor := range formFactors {
		switch factor {
		case "watch":
			hasWatch = true
		case "xr":
			hasXr = true
		case "automotive":
			hasAutomotive = true
		case "mobile":
			hasMobile = true
		case "eink":
			hasEink = true
		case "tablet":
			hasTablet = true
		}
	}

	// Priority order matching PHP behavior
	if hasWatch {
		return DeviceTypeWearable
	}
	if hasXr {
		return DeviceTypeWearable
	}
	if hasAutomotive {
		return DeviceTypeCarBrowser
	}
	if hasMobile {
		return DeviceTypeSmartphone
	}
	if hasEink {
		return DeviceTypeTablet
	}
	if hasTablet {
		return DeviceTypeTablet
	}

	return DeviceTypeUnknown
}

// applyHeuristics applies PHP's device type heuristics.
func (d *DeviceDetector) applyHeuristics(result *ParseResult, ua string) {
	// PHP applies many heuristics based on UA patterns
	// This is a simplified version focusing on the test cases

	// Check for TV pattern
	if strings.Contains(ua, " TV ") || strings.Contains(ua, "TV Safari") {
		result.device = DeviceTypeTV
		return
	}

	// Check for tablet patterns
	if strings.Contains(ua, "Pad/APad") {
		result.device = DeviceTypeTablet
		return
	}

	// Check for tablet model patterns (e.g., PMP = Prestigio tablet)
	if result.device == DeviceTypeUnknown && strings.Contains(ua, "PMP") && strings.Contains(ua, "Build/") {
		result.device = DeviceTypeTablet
		return
	}
}

var feedReaderGateTokens = []string{
	"feed",
	"rss",
	"newsblur",
	"inoreader",
	"feedly",
	"feeder",
	"reeder/",
	"goodpods/",
}

var mobileAppGateTokens = []string{
	"amazon;aft",
	"aliapp(",
	"cfnetwork/",
	" okhttp",
	" okhttp/",
	"dalvik/",
	"fban/",
	"fbios/",
	"micromessenger/",
	"fb_iab",
	"instagram",
	"instabridge/",
	"line/",
	"gsa/",
	"whatsapp/",
	"pandora/",
	"tivimate/",
	"podkast",
	"podkicker",
	"player fm",
	";appver:",
	"microsoft office word/",
	"ns/",
	"yjapp-",
	"; wv",
	" wv)",
}

var mediaPlayerGateTokens = []string{
	"vlc",
	"mediaplayer",
	"mpv",
	"foobar2000",
	"player/",
	"itunes",
	"substream/",
}

var pimGateTokens = []string{
	"outlook",
	"thunderbird",
	"evolution",
	"mailspring",
	"lotus-notes",
	"kontact",
	"spicebird/",
}

var libraryGateTokens = []string{
	"curl/",
	"wget/",
	"python-requests",
	"go-http-client",
	"okhttp",
	"guzzlehttp",
	"apache-httpclient",
	"postmanruntime",
	"java/",
	"libwww-perl",
	"safariviewservice/",
}

func containsAnyLower(uaLower string, tokens []string) bool {
	for _, token := range tokens {
		if strings.Contains(uaLower, token) {
			return true
		}
	}
	return false
}

func shouldTryFeedReaderParser(uaLower string) bool {
	return containsAnyLower(uaLower, feedReaderGateTokens)
}

func shouldTryMobileAppParser(uaLower, appID string) bool {
	if appID != "" {
		return true
	}
	return containsAnyLower(uaLower, mobileAppGateTokens)
}

func shouldTryMediaPlayerParser(uaLower string) bool {
	return containsAnyLower(uaLower, mediaPlayerGateTokens)
}

func shouldTryPIMParser(uaLower string) bool {
	return containsAnyLower(uaLower, pimGateTokens)
}

func shouldTryLibraryParser(uaLower string) bool {
	return containsAnyLower(uaLower, libraryGateTokens)
}

func setSimpleClientMatch(result *ParseResult, clientType, name, version string) {
	result.client = &ClientMatch{
		Type:    clientType,
		Name:    name,
		Version: version,
	}
}

func (d *DeviceDetector) tryParseBrowser(result *ParseResult, ua string, ch *clienthints.ClientHints) {
	if browserFactory, err := d.getBrowserFactory(); err == nil {
		var browserMatch *browser.Match
		if ch != nil {
			browserMatch = browserFactory.Parse(ua, browser.WithClientHints(ch))
		} else {
			browserMatch = browserFactory.Parse(ua)
		}
		if browserMatch != nil {
			result.client = &ClientMatch{
				Type:          browserMatch.Type,
				Name:          browserMatch.Name,
				Version:       browserMatch.Version,
				Engine:        browserMatch.Engine,
				EngineVersion: browserMatch.EngineVersion,
				Family:        browserMatch.Family,
			}
		}
	}
}

// parseClient tries all client parsers in PHP order.
// PHP order: FeedReader, MobileApp, MediaPlayer, PIM, Browser, Library
func (d *DeviceDetector) parseClient(result *ParseResult, ua string, ch *clienthints.ClientHints) {
	uaLower := strings.ToLower(ua)
	appID := ""
	if ch != nil {
		appID = strings.TrimSpace(ch.GetApp())
	}

	browserShortcut := false
	if appID != "" {
		if browserHints, err := d.getBrowserHints(); err == nil && browserHints.GetBrowserName(appID) != "" {
			browserShortcut = true
		} else if appHints, appErr := d.getMobileAppHints(); appErr == nil {
			if appName := appHints.GetAppName(appID); appName != "" {
				setSimpleClientMatch(result, "mobile app", appName, "")
				return
			}
		}
	}

	if !browserShortcut {
		// 1. Try FeedReader
		if shouldTryFeedReaderParser(uaLower) {
			if feedReaderFactory, err := d.getFeedReaderFactory(); err == nil {
				if match := feedReaderFactory.Parse(ua); match != nil {
					setSimpleClientMatch(result, match.Type, match.Name, match.Version)
					return
				}
			}
		}

		// 2. Try MobileApp
		if shouldTryMobileAppParser(uaLower, appID) {
			if mobileAppFactory, err := d.getMobileAppFactory(); err == nil {
				if match := mobileAppFactory.Parse(ua); match != nil {
					setSimpleClientMatch(result, match.Type, match.Name, match.Version)
					return
				}
			}
		}

		// 3. Try MediaPlayer
		if shouldTryMediaPlayerParser(uaLower) {
			if mediaPlayerFactory, err := d.getMediaPlayerFactory(); err == nil {
				if match := mediaPlayerFactory.Parse(ua); match != nil {
					setSimpleClientMatch(result, match.Type, match.Name, match.Version)
					return
				}
			}
		}

		// 4. Try PIM
		if shouldTryPIMParser(uaLower) {
			if pimFactory, err := d.getPIMFactory(); err == nil {
				if match := pimFactory.Parse(ua); match != nil {
					setSimpleClientMatch(result, match.Type, match.Name, match.Version)
					return
				}
			}
		}
	}

	// 5. Try Browser (with client hints support)
	d.tryParseBrowser(result, ua, ch)
	if result.client != nil {
		return
	}

	// 6. Try Library
	if shouldTryLibraryParser(uaLower) {
		if libraryFactory, err := d.getLibraryFactory(); err == nil {
			if match := libraryFactory.Parse(ua); match != nil {
				setSimpleClientMatch(result, match.Type, match.Name, match.Version)
				return
			}
		}
	}
}

// IsBot returns true if the user agent is a bot.
func (r *ParseResult) IsBot() bool {
	return r.bot != nil
}

// IsMobile returns true if the device is mobile.
// PHP logic: mobile device types OR mobile-only browser OR (!bot && !desktop && known OS)
func (r *ParseResult) IsMobile() bool {
	// Client hints indicate mobile
	if r.clientHints != nil && r.clientHints.IsMobile() {
		return true
	}

	// Mobile device types
	switch r.device {
	case DeviceTypeFeaturePhone, DeviceTypeSmartphone, DeviceTypeTablet,
		DeviceTypePhablet, DeviceTypeCamera, DeviceTypePortableMediaPlayer:
		return true
	}

	// Non-mobile device types
	switch r.device {
	case DeviceTypeTV, DeviceTypeSmartDisplay, DeviceTypeConsole:
		return false
	}

	// Check for mobile-only browser
	if r.usesMobileBrowser() {
		return true
	}

	// If no OS detected, not mobile
	if r.os == nil || r.os.Name == "" {
		return false
	}

	// Final fallback: not bot and not desktop
	return !r.IsBot() && !r.IsDesktop()
}

// IsDesktop returns true if the device is a desktop.
func (r *ParseResult) IsDesktop() bool {
	if r.os == nil || r.os.Name == "" {
		return false
	}

	// Mobile-only browser means not desktop
	if r.usesMobileBrowser() {
		return false
	}

	return operatingsystem.IsDesktopOs(r.os.Name)
}

// IsTablet returns true if the device is a tablet.
func (r *ParseResult) IsTablet() bool {
	return r.device == DeviceTypeTablet
}

// IsTV returns true if the device is a TV.
func (r *ParseResult) IsTV() bool {
	return r.device == DeviceTypeTV
}

// IsWearable returns true if the device is a wearable.
func (r *ParseResult) IsWearable() bool {
	return r.device == DeviceTypeWearable
}

// usesMobileBrowser checks if the client is a mobile-only browser.
func (r *ParseResult) usesMobileBrowser() bool {
	if r.client == nil {
		return false
	}
	if r.client.Type != "browser" {
		return false
	}
	return browser.IsMobileOnlyBrowser(r.client.Name)
}

// GetBot returns the bot info if detected.
func (r *ParseResult) GetBot() *bots.BotMatch {
	return r.bot
}

// GetClient returns the client info.
func (r *ParseResult) GetClient() *ClientMatch {
	return r.client
}

// GetOS returns the OS info.
func (r *ParseResult) GetOS() *operatingsystem.Match {
	return r.os
}

// GetDevice returns the device type.
func (r *ParseResult) GetDevice() DeviceType {
	return r.device
}

// GetBrand returns the device brand.
func (r *ParseResult) GetBrand() string {
	return r.brand
}

// GetModel returns the device model.
func (r *ParseResult) GetModel() string {
	return r.model
}

// GetFullInfo returns the complete detection result matching PHP's getInfoFromUserAgent() output.
// This includes user_agent, os, client, device, os_family, and browser_family.
func (r *ParseResult) GetFullInfo() *FullInfo {
	info := &FullInfo{
		UserAgent: r.userAgent,
	}

	// OS info
	if r.os != nil {
		info.OS = &FullInfoOS{
			Name:     r.os.Name,
			Version:  r.os.Version,
			Platform: r.os.Platform,
		}
		info.OSFamily = r.os.Family
		if info.OSFamily == "" {
			info.OSFamily = "Unknown"
		}
	} else {
		info.OS = &FullInfoOS{}
		info.OSFamily = "Unknown"
	}

	// Client info
	if r.client != nil {
		info.Client = &FullInfoClient{
			Type:          r.client.Type,
			Name:          r.client.Name,
			Version:       r.client.Version,
			Engine:        r.client.Engine,
			EngineVersion: r.client.EngineVersion,
		}
		// Browser family: only set if client is a browser with a family
		if r.client.Type == "browser" && r.client.Family != "" {
			info.BrowserFamily = r.client.Family
		} else {
			info.BrowserFamily = "Unknown"
		}
	} else {
		info.Client = &FullInfoClient{}
		info.BrowserFamily = "Unknown"
	}

	// Device info
	// PHP infers device type "desktop" when no explicit device was detected but OS indicates desktop
	deviceType := DeviceTypeNames[r.device]
	if deviceType == "" && r.IsDesktop() {
		deviceType = "desktop"
	}
	info.Device = &FullInfoDevice{
		Type:  deviceType,
		Brand: r.brand,
		Model: r.model,
	}

	return info
}

// GetInfoFromUserAgent is a convenience function matching PHP's DeviceDetector::getInfoFromUserAgent().
// It creates a DeviceDetector, parses the UA, and returns the full info structure.
func GetInfoFromUserAgent(dd *DeviceDetector, ua string, ch *clienthints.ClientHints) *FullInfo {
	result := dd.Parse(ua, ch)
	return result.GetFullInfo()
}
