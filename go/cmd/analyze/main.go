package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/archbottle/device-detector/analyzer"
	"gopkg.in/yaml.v3"
)

func main() {
	regexesDir := flag.String("regexes", "", "Path to device-detector regexes directory")
	outputDir := flag.String("output", "./output", "Output directory for generated files")
	flag.Parse()

	if *regexesDir == "" {
		// Try to find it relative to working directory
		candidates := []string{
			"../device-detector/regexes",
			"../../device-detector/regexes",
			"device-detector/regexes",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				*regexesDir = c
				break
			}
		}
	}

	if *regexesDir == "" {
		log.Fatal("Could not find regexes directory. Use -regexes flag.")
	}

	log.Printf("Analyzing regexes from: %s", *regexesDir)

	a := analyzer.NewAnalyzer(*regexesDir)
	result, err := a.Analyze()
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output dir: %v", err)
	}

	// Write summary
	log.Printf("\n=== ANALYSIS SUMMARY ===")
	log.Printf("Total patterns: %d", result.Summary.TotalPatterns)
	log.Printf("RE2-safe (Go regexp): %d (%.1f%%)",
		result.Summary.RE2SafePatterns,
		float64(result.Summary.RE2SafePatterns)/float64(result.Summary.TotalPatterns)*100)
	log.Printf("Needs regexp2: %d (%.1f%%)",
		result.Summary.Regexp2Patterns,
		float64(result.Summary.Regexp2Patterns)/float64(result.Summary.TotalPatterns)*100)
	log.Printf("Indexed keywords: %d", result.Summary.IndexedKeywords)

	// Write keyword indexes to separate files
	if err := writeYAML(filepath.Join(*outputDir, "browser_index.yml"), result.BrowserIndex); err != nil {
		log.Fatalf("Failed to write browser index: %v", err)
	}
	log.Printf("Wrote browser_index.yml (%d keywords)", len(result.BrowserIndex.Keywords))

	if err := writeYAML(filepath.Join(*outputDir, "os_index.yml"), result.OSIndex); err != nil {
		log.Fatalf("Failed to write OS index: %v", err)
	}
	log.Printf("Wrote os_index.yml (%d keywords)", len(result.OSIndex.Keywords))

	if err := writeYAML(filepath.Join(*outputDir, "device_index.yml"), result.DeviceIndex); err != nil {
		log.Fatalf("Failed to write device index: %v", err)
	}
	log.Printf("Wrote device_index.yml (%d keywords)", len(result.DeviceIndex.Keywords))

	// Write regexp2 patterns (those needing special handling)
	regexp2Patterns := []analyzer.PatternInfo{}
	for _, p := range result.Patterns {
		if p.HasLookaround {
			regexp2Patterns = append(regexp2Patterns, p)
		}
	}
	if err := writeYAML(filepath.Join(*outputDir, "regexp2_patterns.yml"), regexp2Patterns); err != nil {
		log.Fatalf("Failed to write regexp2 patterns: %v", err)
	}
	log.Printf("Wrote regexp2_patterns.yml (%d patterns)", len(regexp2Patterns))

	// Write full analysis (can be large)
	if err := writeYAML(filepath.Join(*outputDir, "full_analysis.yml"), result); err != nil {
		log.Fatalf("Failed to write full analysis: %v", err)
	}
	log.Printf("Wrote full_analysis.yml")

	fmt.Println("\nDone! Check the output directory for generated files.")
}

func writeYAML(path string, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	return encoder.Encode(data)
}
