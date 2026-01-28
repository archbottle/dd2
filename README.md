device-detector (Go)

Go rewrite of the PHP library shipped in `php/`. Parses a User-Agent (plus optional Client Hints) into bot/client/OS/device/brand/model with ~95% compatibility with the PHP library.

### How does it work?

Quite slow. Uses the dclark/regexp2 package for regexes, which is a PCRE-compatible regex engine, required to be compatible with the PHP library. On top of that it uses some optimizations - it uses the RE2 regex engine for the regexec that can be compiled by it, and it uses a keyword index for fast candidate selection, not much helps unfortunately. If you need it to work quicker:

- use `WithIndexOnly()` to only use the keyword index for candidate selection (no full-scan on all regexes). ~4x Faster; drops coverage to ~82%.
- use cache - the library doesn't have it, so bring your own - ristretto is a good choice.

### Install

```bash
go get github.com/archbottle/device-detector@latest
```

Import path for the main API: `github.com/archbottle/device-detector/pkg/detector`.

### High-level library API (what you probably want)

- **Create a detector**: `detector.New(opts...)`
- **Parse a UA**: `dd.Parse(userAgent, clientHints)`
- **Read results**:
  - `result.IsBot()`, `result.IsMobile()`, `result.IsDesktop()`, `result.IsTablet()`, `result.IsTV()`, `result.IsWearable()`
  - `result.GetBot()`, `result.GetClient()`, `result.GetOS()`
  - `result.GetDevice()`, `result.GetBrand()`, `result.GetModel()`
  - `result.GetFullInfo()` (matches the PHP `getInfoFromUserAgent()` structure)

Example:

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/archbottle/device-detector/pkg/clienthints"
	"github.com/archbottle/device-detector/pkg/detector"
)

func main() {
	dd, err := detector.New()
	if err != nil {
		panic(err)
	}

	ua := "Mozilla/5.0 (...)"

	// Optional: build client hints from HTTP request headers.
	var hdr http.Header = http.Header{}
	hdr.Set("Sec-CH-UA", `"Chromium";v="120"`)
	hdr.Set("Sec-CH-UA-Mobile", "?0")
	hdr.Set("Sec-CH-UA-Platform", `"Linux"`)
	ch := clienthints.New(hdr)

	r := dd.Parse(ua, ch)

	if r.IsBot() {
		fmt.Printf("bot=%q\n", r.GetBot().Name)
		return
	}

	if c := r.GetClient(); c != nil {
		fmt.Printf("client=%s %s %s\n", c.Type, c.Name, c.Version)
	}
	if os := r.GetOS(); os != nil {
		fmt.Printf("os=%s %s (%s)\n", os.Name, os.Version, os.Platform)
	}

	fmt.Printf("device=%v brand=%q model=%q\n", r.GetDevice(), r.GetBrand(), r.GetModel())

	// PHP-shaped output:
	info := r.GetFullInfo()
	fmt.Printf("os_family=%q browser_family=%q\n", info.OSFamily, info.BrowserFamily)
}
```

### Detector options (behavior knobs)

Passed to `detector.New(...)`:

- **`detector.WithSkipBotDetection()`**: do not run bot detection.
- **`detector.WithDiscardBotInformation()`**: bot detection becomes a boolean check; detailed bot info is dropped.
- **`detector.WithRe2Only()`**: only use RE2-compatible patterns. For experimenting only, drops coverage to 5%.
- **`detector.WithIndexOnly()`**: only use the keyword index for candidate selection (no full-scan fallback). ~4x Faster; drops coverage to ~82%.
- **`detector.WithFactoryOptions(...)`**: pass `pkg/common` factory options through to all parser factories.

### Client Hints

Package: `github.com/archbottle/device-detector/pkg/clienthints`

- **`clienthints.New(http.Header)`**: build hints from request headers.
- **`clienthints.Factory(map[string]interface{})`**: accepts:
  - Standard headers (`Sec-CH-UA`, `Sec-CH-UA-Mobile`, …)
  - PHP `$_SERVER` style (`HTTP_SEC_CH_UA`, …)
  - JS `navigator.userAgentData`-style payloads (fixtures/tests use this sometimes)

Pass the resulting `*clienthints.ClientHints` into `dd.Parse(ua, ch)`.

### CLI (developer/compat tooling)

Entrypoint is `cmd/main.go`.

Run without a subcommand and you get the default “compat report” behavior:

```bash
go run ./cmd -format html
```

#### `compat-report` (default)

Purpose: run the Go implementation against the fixtures and generate a compatibility report.

```bash
# HTML report (default)
go run ./cmd -format html -o compatibility-report.html

# JSON report
go run ./cmd -format json -o compatibility-report.json

# Include the big full-parse test (~36K+ cases). Slow.
go run ./cmd -full -format html

# Generate HTML from an existing JSON file (no re-run)
go run ./cmd -format html -input compatibility-report.json -o compatibility-report.html
```

Flags:

- **`-format`**: `html` (default) or `json`
- **`-o`**: output path (default depends on format)
- **`-input`**: read JSON report from disk and render HTML from it
- **`-full`**: include the full parse integration test (slow)

Env:

- **`DD2_COMPAT_WORKERS`**: max parallelism for running the individual collectors.

#### `sample-full`

Purpose: run a deterministic sample of the full parse integration test (fast sanity check).

```bash
go run ./cmd sample-full -n 250
go run ./cmd sample-full -n 250 -index-only
go run ./cmd sample-full -n 250 -re2-only
go run ./cmd sample-full -n 250 -index-only -re2-only
```

Flags:

- **`-n`**: number of fixtures to sample (**required**)
- **`-index-only`**: pass `detector.WithIndexOnly()` into the detector
- **`-re2-only`**: pass `detector.WithRe2Only()` into the detector

Exit code is non-zero if any sampled case fails.

#### `resources`

Purpose: mapping/copy utilities for YAML resources between the `php/` subtree and the `go/` subtree, tracked via `resources.yaml`.

```bash
go run ./cmd resources gen
go run ./cmd resources sync
go run ./cmd resources check
```

Top-level flags:

- **`-root`**: repo root (defaults to auto-detected by walking up from cwd)
- **`-resources`**: path to `resources.yaml` (default: `<repoRoot>/resources.yaml`)

Subcommands:

- **`resources gen`**: generate `resources.yaml` mappings by scanning YAML files.
  - Flags: `-go-dir` (default `<root>/go`), `-php-dir` (default `<root>/php`), `-sha256` (default `true`)
- **`resources sync`**: copy mapped files from `from:` → `to:`.
  - Flags: `-dry-run`, `-check` (verify hashes after copying)
- **`resources check`**: verify mapped files match (sha256).

### Repo layout (what’s where)

- **`pkg/`**: Go implementation (import from here).
- **`regexes/`**: embedded runtime regex DB (compiled into the Go binary via `embed`).
- **`php/`**: PHP library + fixtures used for compatibility checking.
- **`cmd/`**: CLI entrypoint (compat reports, sampling, resource sync).
- **`resources.yaml`**: mapping file used by the `resources` command.