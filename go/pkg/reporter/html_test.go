package reporter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderHTML_IncludesClientHintsOnFailure(t *testing.T) {
	testCases := []struct {
		name          string
		hints         []HeaderKV
		expectPresent bool
	}{
		{
			name:          "no hints",
			hints:         nil,
			expectPresent: false,
		},
		{
			name: "with hints",
			hints: []HeaderKV{
				{Name: "sec-ch-ua", Value: `"Chromium";v="120"`},
				{Name: "sec-ch-ua-platform", Value: "Linux"},
			},
			expectPresent: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			r := &Report{
				Parsers: []ParserResult{
					{
						Name:   "Browser",
						Passed: 0,
						Failed: 1,
						Failures: []TestFailure{
							{
								CaseIndex:   0,
								UserAgent:   "UA",
								ClientHints: tc.hints,
								Fields:      []FieldDiff{{Name: "Name", Expected: "x", Actual: "y", Matches: false}},
							},
						},
					},
				},
			}
			r.Calculate()

			// when
			html, err := RenderHTML(r)

			// then
			assert.NoError(t, err)
			assert.Contains(t, html, "UA")

			if tc.expectPresent {
				assert.Contains(t, html, "Client hints")
				assert.True(t, strings.Contains(html, "sec-ch-ua") || strings.Contains(html, "sec-ch-ua-platform"))
			} else {
				assert.NotContains(t, html, "Client hints")
			}
		})
	}
}
