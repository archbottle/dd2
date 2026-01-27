// compat-report generates compatibility reports for the Go device detector implementation.
//
// Usage:
//
//	# Generate JSON report (slow - runs all tests)
//	go run ./cmd/compat-report -format json -o report.json [-full]
//
//	# Generate HTML from existing JSON (fast - no tests)
//	go run ./cmd/compat-report -format html -input report.json -o report.html
//
//	# Generate HTML directly (runs tests)
//	go run ./cmd/compat-report -format html -o report.html [-full]
//
// The report shows:
//   - Overall compatibility percentage
//   - Per-parser breakdown (pass/fail counts)
//   - Detailed failure information (UA, expected vs actual)
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/archbottle/device-detector/pkg/reporter"
)

func main() {
	format := flag.String("format", "html", "Output format: json or html")
	outputFile := flag.String("o", "", "Output file path (default: compatibility-report.{format})")
	inputFile := flag.String("input", "", "Input JSON file (for HTML generation without re-running tests)")
	includeFull := flag.Bool("full", false, "Include full parse integration test (slow, 36K+ tests)")
	flag.Parse()

	// Determine default output filename
	if *outputFile == "" {
		if *format == "json" {
			*outputFile = "compatibility-report.json"
		} else {
			*outputFile = "compatibility-report.html"
		}
	}

	var report *reporter.Report
	var err error

	// Either load from JSON or collect fresh results
	if *inputFile != "" {
		fmt.Printf("Loading report from %s...\n", *inputFile)
		report, err = loadJSON(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Collecting test results...")
		report, err = reporter.CollectAll(*includeFull)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error collecting results: %v\n", err)
			os.Exit(1)
		}
		report.GeneratedAt = time.Now()
	}

	// Generate output based on format
	switch *format {
	case "json":
		fmt.Println("Generating JSON report...")
		if err := writeJSON(report, *outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JSON: %v\n", err)
			os.Exit(1)
		}

	case "html":
		fmt.Println("Generating HTML report...")
		html, err := reporter.RenderHTML(report)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering HTML: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*outputFile, []byte(html), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s (use 'json' or 'html')\n", *format)
		os.Exit(1)
	}

	fmt.Printf("\nReport generated: %s\n", *outputFile)
	fmt.Printf("Overall: %.1f%% compatible (%d/%d passed)\n",
		report.Compatibility(),
		report.PassedTests,
		report.TotalTests,
	)
	fmt.Println("\nPer-parser breakdown:")
	for _, p := range report.Parsers {
		status := "OK"
		if p.Failed > 0 {
			status = fmt.Sprintf("%d failed", p.Failed)
		}
		fmt.Printf("  %-20s %4d tests, %s\n", p.Name, p.Passed+p.Failed, status)
	}
}

func loadJSON(path string) (*reporter.Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return reporter.ReadJSON(f)
}

func writeJSON(report *reporter.Report, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return report.WriteJSON(f)
}
