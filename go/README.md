# Device Detector Regex Analyzer

A Go tool that analyzes the PHP device-detector YAML regex files and produces:

1. **Keyword indexes** - Maps literal strings to patterns that might match
2. **RE2 compatibility report** - Identifies patterns that need `regexp2` library
3. **Pattern analysis** - Full breakdown of all patterns with metadata

## Quick Start

```bash
# Run the analyzer
go run ./cmd/analyze -regexes ../php/regexes -output ./output

# Run tests
go test ./... -v
```

## Output Files

| File | Description |
|------|-------------|
| `browser_index.yml` | Keywords → browser patterns |
| `device_index.yml` | Keywords → device patterns |
| `os_index.yml` | Keywords → OS patterns |
| `regexp2_patterns.yml` | Patterns that need regexp2 (have lookarounds) |
| `full_analysis.yml` | Complete analysis with all patterns |

## How the Keyword Index Works

### The Problem

Device-detector has ~17,000+ regex patterns. Testing every UA against all patterns is slow.

### The Solution

Extract **literal keywords** from each regex. For example:

| Regex | Extracted Keywords |
|-------|-------------------|
| `Chrome/(\d+)` | `Chrome/` |
| `SM-G[0-9]+` | `SM-G` |
| `SAMSUNG\|Galaxy` | `SAMSUNG`, `Galaxy` |

### Usage in Detection

```go
// Instead of testing all 17,000 patterns:
for _, pattern := range allPatterns {
    if pattern.Match(userAgent) { ... }  // SLOW
}

// Use keyword index to find candidates first:
candidates := findCandidates(userAgent, keywordIndex)  // ~50-200 patterns
for _, pattern := range candidates {
    if pattern.Match(userAgent) { ... }  // FAST
}
```

## Analysis Results (as of current device-detector)

```
Total patterns:           17,839
RE2-safe (Go regexp):     17,691 (99.2%)
Needs regexp2:               148 (0.8%)
Indexed keywords:         24,710
```

### RE2 Incompatible Patterns

Only ~148 patterns use lookaround assertions that Go's `regexp` doesn't support:

- `(?<!...)` - Negative lookbehind
- `(?<=...)` - Positive lookbehind  
- `(?!...)` - Negative lookahead
- `(?=...)` - Positive lookahead

These patterns require the `github.com/dlclark/regexp2` library.

## Project Structure

```
dd2/
├── go/
│   ├── cmd/              # CLI tools (analyze, compat-report, etc.)
│   ├── pkg/              # Parsers + shared logic
│   └── regexes/          # YAML copied/ported from php/regexes for Go consumption
└── php/                  # Upstream device-detector (git submodule)
```

## Example: Keyword Lookup Demo

From `TestDemoKeywordLookup`:

```
Input UA: Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.36 Chrome/91.0

Keywords found:
  "Chrome/"  → [Chrome, Brave, Chromium, ...]
  "SM-G"     → [Samsung Galaxy S10, Samsung Galaxy S9, ...]
  "Android"  → [Android Browser, Chrome Mobile, ...]

Result: 
  Candidate browsers: 39 (vs 700+ total browser patterns)
  Candidate devices: 111 (vs 17,000+ total device patterns)
```

This reduces pattern matching work by **~90%** for typical user agents.


