package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/archbottle/dd2/pkg/detector"
)

// FieldDiff represents the comparison of a single field
type FieldDiff struct {
	Name     string `json:"name"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Matches  bool   `json:"matches"`
}

// HeaderKV represents a name/value header pair (used for client hints rendering).
// Stored as a slice (not a map) to keep output deterministic.
type HeaderKV struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TestFailure represents a single failed test case
type TestFailure struct {
	CaseIndex int    `json:"case_index"`
	UserAgent string `json:"user_agent"`
	// ClientHints contains the request headers used as client hints (if present in fixture).
	ClientHints []HeaderKV  `json:"client_hints,omitempty"`
	Fields      []FieldDiff `json:"fields"`
}

// ParserResult aggregates results for one parser type
type ParserResult struct {
	Name     string        `json:"name"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Failures []TestFailure `json:"failures,omitempty"`
	Stats    *ResultStats  `json:"stats,omitempty"`
}

// ResultStats contains extra aggregated metrics for a collector run.
type ResultStats struct {
	ClientParsers detector.ClientParserStats `json:"client_parsers"`
}

// Percent returns the pass percentage for this parser
func (p *ParserResult) Percent() float64 {
	total := p.Passed + p.Failed
	if total == 0 {
		return 100.0
	}
	return float64(p.Passed) / float64(total) * 100
}

// Report is the complete compatibility report
type Report struct {
	GeneratedAt time.Time      `json:"generated_at"`
	TotalTests  int            `json:"total_tests"`
	PassedTests int            `json:"passed_tests"`
	FailedTests int            `json:"failed_tests"`
	Parsers     []ParserResult `json:"parsers"`
}

// Compatibility returns the overall pass percentage
func (r *Report) Compatibility() float64 {
	if r.TotalTests == 0 {
		return 100.0
	}
	return float64(r.PassedTests) / float64(r.TotalTests) * 100
}

// Calculate populates totals from parser results
func (r *Report) Calculate() {
	r.TotalTests = 0
	r.PassedTests = 0
	r.FailedTests = 0
	for _, p := range r.Parsers {
		r.TotalTests += p.Passed + p.Failed
		r.PassedTests += p.Passed
		r.FailedTests += p.Failed
	}
}

// WriteJSON writes the report as JSON to the given writer
func (r *Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// ReadJSON reads a report from JSON
func ReadJSON(r io.Reader) (*Report, error) {
	var report Report
	if err := json.NewDecoder(r).Decode(&report); err != nil {
		return nil, fmt.Errorf("decoding JSON: %w", err)
	}
	return &report, nil
}
