package reporter

import (
	"bytes"
	"html/template"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Device Detector Compatibility Report</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            background: #f5f5f5;
            padding: 20px;
        }
        .container { max-width: 1200px; margin: 0 auto; }
        .header {
            background: #fff;
            border-radius: 8px;
            padding: 24px;
            margin-bottom: 20px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .header h1 { font-size: 24px; font-weight: 600; margin-bottom: 8px; }
        .header .meta { color: #666; font-size: 14px; }
        .overall-stats {
            background: #fff;
            border-radius: 8px;
            padding: 24px;
            margin-bottom: 20px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .stat-row {
            display: flex;
            align-items: center;
            gap: 20px;
            flex-wrap: wrap;
        }
        .stat-box {
            text-align: center;
            padding: 16px 24px;
            background: #f8f9fa;
            border-radius: 6px;
            min-width: 120px;
        }
        .stat-box .value { font-size: 32px; font-weight: 700; }
        .stat-box .label {
            font-size: 12px;
            color: #666;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .stat-box.success .value { color: #22863a; }
        .stat-box.failure .value { color: #cb2431; }
        .stat-box.neutral .value { color: #0366d6; }
        .progress-container { flex: 1; min-width: 200px; }
        .progress-bar {
            height: 24px;
            background: #e1e4e8;
            border-radius: 12px;
            overflow: hidden;
            margin-bottom: 8px;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #22863a, #28a745);
            border-radius: 12px;
        }
        .progress-text { font-size: 14px; color: #666; }
        .parser-section {
            background: #fff;
            border-radius: 8px;
            margin-bottom: 16px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .parser-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 16px 20px;
            cursor: pointer;
            user-select: none;
            border-bottom: 1px solid #e1e4e8;
        }
        .parser-header:hover { background: #f8f9fa; }
        .parser-title { display: flex; align-items: center; gap: 12px; }
        .parser-title h3 { font-size: 16px; font-weight: 600; }
        .parser-stats { display: flex; gap: 16px; align-items: center; }
        .badge {
            padding: 4px 10px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 500;
        }
        .badge.pass { background: #dcffe4; color: #22863a; }
        .badge.fail { background: #ffeef0; color: #cb2431; }
        .badge.pct { background: #f1f8ff; color: #0366d6; }
        .toggle-icon {
            font-size: 12px;
            color: #666;
            transition: transform 0.2s ease;
        }
        .parser-section.expanded .toggle-icon { transform: rotate(180deg); }
        .failures-list { display: none; padding: 0; }
        .parser-section.expanded .failures-list { display: block; }
        .failure-item {
            border-bottom: 1px solid #e1e4e8;
            padding: 16px 20px;
        }
        .failure-item:last-child { border-bottom: none; }
        .failure-item:hover { background: #fafbfc; }
        .failure-header {
            font-size: 13px;
            color: #666;
            margin-bottom: 8px;
        }
        .ua-string {
            font-family: "SF Mono", Monaco, "Courier New", monospace;
            font-size: 12px;
            background: #f6f8fa;
            padding: 8px 12px;
            border-radius: 4px;
            margin-bottom: 12px;
            word-break: break-all;
            color: #24292e;
        }
        .diff-table {
            width: 100%;
            font-size: 13px;
            border-collapse: collapse;
        }
        .diff-table th {
            text-align: left;
            padding: 6px 12px;
            background: #f6f8fa;
            font-weight: 500;
            border: 1px solid #e1e4e8;
        }
        .diff-table td {
            padding: 6px 12px;
            border: 1px solid #e1e4e8;
        }
        .diff-table .field { font-weight: 500; width: 120px; }
        .diff-table .expected { background: #dcffe4; }
        .diff-table .actual { background: #ffeef0; }
        .diff-table .match { background: #fff; }
        .all-passed {
            padding: 20px;
            color: #22863a;
            text-align: center;
            font-style: italic;
        }
        @media (max-width: 768px) {
            body { padding: 12px; }
            .stat-row { flex-direction: column; align-items: stretch; }
            .stat-box { min-width: auto; }
            .parser-header {
                flex-direction: column;
                align-items: flex-start;
                gap: 12px;
            }
            .parser-stats { width: 100%; justify-content: space-between; }
            .diff-table { font-size: 11px; }
            .diff-table th, .diff-table td { padding: 4px 8px; }
            .ua-string { font-size: 10px; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Device Detector Compatibility Report</h1>
            <div class="meta">Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05 MST"}} | Go Implementation vs PHP Reference</div>
        </div>

        <div class="overall-stats">
            <div class="stat-row">
                <div class="stat-box neutral">
                    <div class="value">{{.TotalTests}}</div>
                    <div class="label">Total Tests</div>
                </div>
                <div class="stat-box success">
                    <div class="value">{{.PassedTests}}</div>
                    <div class="label">Passed</div>
                </div>
                <div class="stat-box failure">
                    <div class="value">{{.FailedTests}}</div>
                    <div class="label">Failed</div>
                </div>
                <div class="progress-container">
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: {{printf "%.1f" .Compatibility}}%"></div>
                    </div>
                    <div class="progress-text">{{printf "%.1f" .Compatibility}}% Compatible</div>
                </div>
            </div>
        </div>

        {{range .Parsers}}
        <div class="parser-section{{if gt .Failed 0}} expanded{{end}}">
            <div class="parser-header" onclick="toggleSection(this)">
                <div class="parser-title">
                    <h3>{{.Name}}</h3>
                </div>
                <div class="parser-stats">
                    <span class="badge pct">{{printf "%.1f" .Percent}}%</span>
                    <span class="badge pass">{{.Passed}} passed</span>
                    {{if gt .Failed 0}}<span class="badge fail">{{.Failed}} failed</span>{{end}}
                    <span class="toggle-icon">&#9660;</span>
                </div>
            </div>
            <div class="failures-list">
                {{if eq .Failed 0}}
                <p class="all-passed">All tests passed!</p>
                {{else}}
                {{range .Failures}}
                <div class="failure-item">
                    <div class="failure-header">Case #{{.CaseIndex}}</div>
                    <div class="ua-string">{{.UserAgent}}</div>
                    <table class="diff-table">
                        <tr>
                            <th class="field">Field</th>
                            <th>Expected</th>
                            <th>Actual</th>
                        </tr>
                        {{range .Fields}}
                        <tr>
                            <td class="field">{{.Name}}</td>
                            {{if .Matches}}
                            <td class="match">{{.Expected}}</td>
                            <td class="match">{{.Actual}}</td>
                            {{else}}
                            <td class="expected">{{.Expected}}</td>
                            <td class="actual">{{.Actual}}</td>
                            {{end}}
                        </tr>
                        {{end}}
                    </table>
                </div>
                {{end}}
                {{end}}
            </div>
        </div>
        {{end}}
    </div>

    <script>
        function toggleSection(header) {
            header.parentElement.classList.toggle('expanded');
        }
    </script>
</body>
</html>`

// RenderHTML generates the HTML report from a Report
func RenderHTML(report *Report) (string, error) {
	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, report); err != nil {
		return "", err
	}

	return buf.String(), nil
}
