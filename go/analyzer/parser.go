package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Analyzer processes device-detector YAML files
type Analyzer struct {
	regexesDir string
	patterns   []PatternInfo
}

// NewAnalyzer creates a new analyzer for the given regexes directory
func NewAnalyzer(regexesDir string) *Analyzer {
	return &Analyzer{
		regexesDir: regexesDir,
		patterns:   []PatternInfo{},
	}
}

// Analyze processes all YAML files and builds the index
func (a *Analyzer) Analyze() (*AnalysisResult, error) {
	// Parse browsers
	if err := a.parseBrowsers(); err != nil {
		return nil, fmt.Errorf("parsing browsers: %w", err)
	}

	// Parse OS
	if err := a.parseOS(); err != nil {
		return nil, fmt.Errorf("parsing OS: %w", err)
	}

	// Parse devices (mobiles.yml is the main one)
	if err := a.parseDevices(); err != nil {
		return nil, fmt.Errorf("parsing devices: %w", err)
	}

	// Build result
	result := &AnalysisResult{}
	result.BrowserIndex.Keywords = make(map[string][]string)
	result.OSIndex.Keywords = make(map[string][]string)
	result.DeviceIndex.Keywords = make(map[string][]string)

	re2Safe := 0
	regexp2Needed := 0

	for _, p := range a.patterns {
		if p.IsRE2Safe {
			re2Safe++
		} else {
			regexp2Needed++
		}

		// Build keyword indexes
		var index *KeywordIndex
		switch p.Category {
		case "browser":
			index = &result.BrowserIndex
		case "os":
			index = &result.OSIndex
		case "device":
			index = &result.DeviceIndex
		}

		for _, kw := range p.Keywords {
			index.Keywords[kw] = append(index.Keywords[kw], p.Name)
		}
	}

	// Deduplicate pattern names in indexes
	dedupeIndex(&result.BrowserIndex)
	dedupeIndex(&result.OSIndex)
	dedupeIndex(&result.DeviceIndex)

	result.Summary.TotalPatterns = len(a.patterns)
	result.Summary.RE2SafePatterns = re2Safe
	result.Summary.Regexp2Patterns = regexp2Needed
	result.Summary.IndexedKeywords = len(result.BrowserIndex.Keywords) +
		len(result.OSIndex.Keywords) +
		len(result.DeviceIndex.Keywords)

	result.Patterns = a.patterns

	return result, nil
}

func dedupeIndex(idx *KeywordIndex) {
	for kw, names := range idx.Keywords {
		seen := make(map[string]bool)
		unique := []string{}
		for _, name := range names {
			if !seen[name] {
				seen[name] = true
				unique = append(unique, name)
			}
		}
		sort.Strings(unique)
		idx.Keywords[kw] = unique
	}
}

func (a *Analyzer) parseBrowsers() error {
	path := filepath.Join(a.regexesDir, "client", "browsers.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var browsers []BrowserEntry
	if err := yaml.Unmarshal(data, &browsers); err != nil {
		return fmt.Errorf("unmarshal browsers.yml: %w", err)
	}

	for _, b := range browsers {
		isRE2Safe, lookaroundType := CheckRE2Compatibility(b.Regex)
		keywords := ExtractKeywords(b.Regex)

		a.patterns = append(a.patterns, PatternInfo{
			Category:       "browser",
			Name:           b.Name,
			OriginalRegex:  b.Regex,
			Keywords:       keywords,
			HasLookaround:  !isRE2Safe,
			LookaroundType: lookaroundType,
			IsRE2Safe:      isRE2Safe,
		})
	}

	return nil
}

func (a *Analyzer) parseOS() error {
	path := filepath.Join(a.regexesDir, "oss.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var osList []OSEntry
	if err := yaml.Unmarshal(data, &osList); err != nil {
		return fmt.Errorf("unmarshal oss.yml: %w", err)
	}

	for _, os := range osList {
		isRE2Safe, lookaroundType := CheckRE2Compatibility(os.Regex)
		keywords := ExtractKeywords(os.Regex)

		a.patterns = append(a.patterns, PatternInfo{
			Category:       "os",
			Name:           os.Name,
			OriginalRegex:  os.Regex,
			Keywords:       keywords,
			HasLookaround:  !isRE2Safe,
			LookaroundType: lookaroundType,
			IsRE2Safe:      isRE2Safe,
		})
	}

	return nil
}

func (a *Analyzer) parseDevices() error {
	path := filepath.Join(a.regexesDir, "device", "mobiles.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// mobiles.yml has a different structure: map of brand -> DeviceBrand
	var devices map[string]DeviceBrand
	if err := yaml.Unmarshal(data, &devices); err != nil {
		return fmt.Errorf("unmarshal mobiles.yml: %w", err)
	}

	for brand, device := range devices {
		// Main brand regex
		isRE2Safe, lookaroundType := CheckRE2Compatibility(device.Regex)
		keywords := ExtractKeywords(device.Regex)

		a.patterns = append(a.patterns, PatternInfo{
			Category:       "device",
			Name:           brand,
			OriginalRegex:  device.Regex,
			Keywords:       keywords,
			HasLookaround:  !isRE2Safe,
			LookaroundType: lookaroundType,
			IsRE2Safe:      isRE2Safe,
		})

		// Model-specific regexes
		for _, model := range device.Models {
			modelRE2Safe, modelLookaround := CheckRE2Compatibility(model.Regex)
			modelKeywords := ExtractKeywords(model.Regex)

			modelName := brand + " - " + model.Model
			a.patterns = append(a.patterns, PatternInfo{
				Category:       "device",
				Name:           modelName,
				OriginalRegex:  model.Regex,
				Keywords:       modelKeywords,
				HasLookaround:  !modelRE2Safe,
				LookaroundType: modelLookaround,
				IsRE2Safe:      modelRE2Safe,
			})
		}
	}

	return nil
}
