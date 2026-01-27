package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/archbottle/device-detector/pkg/reporter"
	"gopkg.in/yaml.v3"
)

func main() {
	// Backward compatible behavior:
	// - If invoked with flags only (no subcommand), run the compat-report command exactly as before.
	// - If invoked with "resources", run resource mapping/copy utilities.
	if len(os.Args) > 1 && os.Args[1] == "resources" {
		if err := resourcesMain(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "resources: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := compatReportMain(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func compatReportMain(args []string) error {
	fs := flag.NewFlagSet("compat-report", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	format := fs.String("format", "html", "Output format: json or html")
	outputFile := fs.String("o", "", "Output file path (default: compatibility-report.{format})")
	inputFile := fs.String("input", "", "Input JSON file (for HTML generation without re-running tests)")
	includeFull := fs.Bool("full", false, "Include full parse integration test (slow, 36K+ tests)")
	if err := fs.Parse(args); err != nil {
		return err
	}

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
			return fmt.Errorf("error loading JSON: %w", err)
		}
	} else {
		fmt.Println("Collecting test results...")
		report, err = reporter.CollectAll(*includeFull)
		if err != nil {
			return fmt.Errorf("error collecting results: %w", err)
		}
		report.GeneratedAt = time.Now()
	}

	// Generate output based on format
	switch *format {
	case "json":
		fmt.Println("Generating JSON report...")
		if err := writeJSON(report, *outputFile); err != nil {
			return fmt.Errorf("error writing JSON: %w", err)
		}

	case "html":
		fmt.Println("Generating HTML report...")
		html, err := reporter.RenderHTML(report)
		if err != nil {
			return fmt.Errorf("error rendering HTML: %w", err)
		}
		if err := os.WriteFile(*outputFile, []byte(html), 0644); err != nil {
			return fmt.Errorf("error writing file: %w", err)
		}

	default:
		return fmt.Errorf("unknown format: %s (use 'json' or 'html')", *format)
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
	return nil
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

type resourcesFile struct {
	Version     int               `yaml:"version"`
	GeneratedAt time.Time         `yaml:"generated_at"`
	Mappings    []resourceMapping `yaml:"mappings"`
}

type resourceMapping struct {
	From   string `yaml:"from"`
	To     string `yaml:"to"`
	SHA256 string `yaml:"sha256,omitempty"`
}

type phpIndex struct {
	ByHash map[string][]string
	ByBase map[string][]string
}

func resourcesMain(args []string) error {
	fs := flag.NewFlagSet("resources", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	rootFlag := fs.String("root", "", "Repo root (defaults to auto-detected by walking up from cwd)")
	resourcesPathFlag := fs.String("resources", "", "Path to resources.yaml (default: <repoRoot>/resources.yaml)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("usage: resources <gen|sync|check> [flags]")
	}

	root := *rootFlag
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		root, err = findRepoRoot(cwd)
		if err != nil {
			return err
		}
	}

	resourcesPath := *resourcesPathFlag
	if resourcesPath == "" {
		resourcesPath = filepath.Join(root, "resources.yaml")
	}

	switch rest[0] {
	case "gen":
		return resourcesGen(root, resourcesPath, rest[1:])
	case "sync":
		return resourcesSync(root, resourcesPath, rest[1:])
	case "check":
		return resourcesCheck(root, resourcesPath, rest[1:])
	default:
		return fmt.Errorf("unknown subcommand %q (expected gen|sync|check)", rest[0])
	}
}

func resourcesGen(root, resourcesPath string, args []string) error {
	fs := flag.NewFlagSet("resources gen", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	goDir := fs.String("go-dir", filepath.Join(root, "go"), "Go subtree to scan for YAML files")
	phpDir := fs.String("php-dir", filepath.Join(root, "php"), "PHP subtree to scan for YAML files")
	includeSHA := fs.Bool("sha256", true, "Include sha256 in the mapping entries")
	if err := fs.Parse(args); err != nil {
		return err
	}

	phpIdx, err := indexPHPYAML(root, *phpDir)
	if err != nil {
		return err
	}

	goFiles, err := listYAMLFiles(*goDir)
	if err != nil {
		return err
	}

	var mappings []resourceMapping
	for _, goAbs := range goFiles {
		goRel, err := filepath.Rel(root, goAbs)
		if err != nil {
			return err
		}
		goRel = filepath.ToSlash(goRel)

		best, ok, err := findPHPSrcForGoYAML(root, goRel, phpIdx)
		if err != nil {
			return err
		}
		if !ok {
			continue // no plausible PHP source file found
		}

		m := resourceMapping{
			From: best,
			To:   goRel,
		}
		if *includeSHA {
			srcAbs := filepath.Join(root, filepath.FromSlash(best))
			sb, err := os.ReadFile(srcAbs)
			if err != nil {
				return err
			}
			m.SHA256 = sha256Hex(sb)
		}
		mappings = append(mappings, m)
	}

	sort.Slice(mappings, func(i, j int) bool {
		if mappings[i].To == mappings[j].To {
			return mappings[i].From < mappings[j].From
		}
		return mappings[i].To < mappings[j].To
	})

	out := resourcesFile{
		Version:     1,
		GeneratedAt: time.Now().UTC(),
		Mappings:    mappings,
	}

	buf, err := yaml.Marshal(out)
	if err != nil {
		return err
	}

	// Ensure deterministic file header/newline.
	content := append([]byte("---\n"), buf...)
	if len(content) == 0 || content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}

	if err := os.WriteFile(resourcesPath, content, 0644); err != nil {
		return err
	}

	fmt.Printf("Wrote %d mappings to %s\n", len(mappings), resourcesPath)
	return nil
}

func findPHPSrcForGoYAML(root, goRel string, phpIdx *phpIndex) (phpRel string, ok bool, err error) {
	goAbs := filepath.Join(root, filepath.FromSlash(goRel))
	gb, err := os.ReadFile(goAbs)
	if err != nil {
		return "", false, err
	}
	gh := sha256Hex(gb)

	// 1) Exact content match (strongest signal for "copied from PHP").
	if cands := phpIdx.ByHash[gh]; len(cands) > 0 {
		return pickBestPHPMatch(goRel, cands), true, nil
	}

	// 2) Path-based match for regex DBs (php and go share the same regexes/ layout).
	if strings.HasPrefix(goRel, "go/regexes/") {
		want := "php/" + strings.TrimPrefix(goRel, "go/")
		if fileExists(filepath.Join(root, filepath.FromSlash(want))) {
			return want, true, nil
		}
	}

	// 3) Heuristic mapping for Go parser fixtures -> PHP test fixtures.
	//    go/pkg/<parser>/fixtures/<file>.yml
	if strings.HasPrefix(goRel, "go/pkg/") && strings.Contains(goRel, "/fixtures/") {
		parts := strings.Split(goRel, "/")
		if len(parts) >= 5 {
			parser := parts[2]
			file := parts[len(parts)-1]

			var want string
			switch parser {
			case "browser", "feedreader", "library", "mediaplayer", "mobileapp", "pim":
				want = "php/Tests/Parser/Client/fixtures/" + file
			case "camera", "carbrowser", "console", "notebook":
				want = "php/Tests/Parser/Device/fixtures/" + file
			case "operatingsystem", "vendorfragment", "detector":
				want = "php/Tests/Parser/fixtures/" + file
			case "bots":
				// PHP fixture lives under Tests/fixtures/ in this repo.
				want = "php/Tests/fixtures/" + file
			}
			if want != "" && fileExists(filepath.Join(root, filepath.FromSlash(want))) {
				return want, true, nil
			}
		}
	}

	// 4) Fallback: basename match in PHP tree.
	base := pathBase(goRel)
	if cands := phpIdx.ByBase[base]; len(cands) > 0 {
		return pickBestPHPMatch(goRel, cands), true, nil
	}

	return "", false, nil
}

func resourcesSync(root, resourcesPath string, args []string) error {
	fs := flag.NewFlagSet("resources sync", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dryRun := fs.Bool("dry-run", false, "Print actions without writing files")
	check := fs.Bool("check", false, "Verify destinations match sources (sha256) after sync")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rf, err := readResourcesFile(resourcesPath)
	if err != nil {
		return err
	}

	for _, m := range rf.Mappings {
		src := filepath.Join(root, filepath.FromSlash(m.From))
		dst := filepath.Join(root, filepath.FromSlash(m.To))

		if *dryRun {
			fmt.Printf("copy %s -> %s\n", m.From, m.To)
			continue
		}

		if err := copyFile(src, dst); err != nil {
			return err
		}
	}

	if *check {
		return resourcesCheck(root, resourcesPath, nil)
	}
	return nil
}

func resourcesCheck(root, resourcesPath string, args []string) error {
	fs := flag.NewFlagSet("resources check", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	rf, err := readResourcesFile(resourcesPath)
	if err != nil {
		return err
	}

	var bad int
	for _, m := range rf.Mappings {
		src := filepath.Join(root, filepath.FromSlash(m.From))
		dst := filepath.Join(root, filepath.FromSlash(m.To))

		sb, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "missing source: %s (%v)\n", m.From, err)
			bad++
			continue
		}
		db, err := os.ReadFile(dst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "missing dest: %s (%v)\n", m.To, err)
			bad++
			continue
		}

		sh := sha256Hex(sb)
		dh := sha256Hex(db)
		if sh != dh {
			fmt.Fprintf(os.Stderr, "hash mismatch: %s -> %s\n", m.From, m.To)
			bad++
			continue
		}
	}

	if bad > 0 {
		return fmt.Errorf("%d resource(s) failed verification", bad)
	}
	fmt.Printf("OK: %d mapping(s) verified\n", len(rf.Mappings))
	return nil
}

func readResourcesFile(path string) (*resourcesFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rf resourcesFile
	if err := yaml.Unmarshal(b, &rf); err != nil {
		return nil, err
	}
	return &rf, nil
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0644)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum[:])
}

func listYAMLFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func hashIndexYAML(repoRoot, root string) (map[string][]string, error) {
	files, err := listYAMLFiles(root)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string, len(files))
	for _, abs := range files {
		b, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		h := sha256Hex(b)
		rel, err := filepath.Rel(repoRoot, abs)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		out[h] = append(out[h], rel)
	}
	for h := range out {
		sort.Strings(out[h])
	}
	return out, nil
}

func indexPHPYAML(repoRoot, root string) (*phpIndex, error) {
	files, err := listYAMLFiles(root)
	if err != nil {
		return nil, err
	}
	idx := &phpIndex{
		ByHash: make(map[string][]string, len(files)),
		ByBase: make(map[string][]string, len(files)),
	}
	for _, abs := range files {
		b, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		h := sha256Hex(b)
		rel, err := filepath.Rel(repoRoot, abs)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		idx.ByHash[h] = append(idx.ByHash[h], rel)
		idx.ByBase[pathBase(rel)] = append(idx.ByBase[pathBase(rel)], rel)
	}
	for h := range idx.ByHash {
		sort.Strings(idx.ByHash[h])
	}
	for b := range idx.ByBase {
		sort.Strings(idx.ByBase[b])
	}
	return idx, nil
}

func pickBestPHPMatch(goRel string, phpCandidates []string) string {
	// Prefer same relative suffix, e.g. go/regexes/oss.yml -> php/regexes/oss.yml
	if strings.HasPrefix(goRel, "go/") {
		want := "php/" + strings.TrimPrefix(goRel, "go/")
		for _, c := range phpCandidates {
			if c == want {
				return c
			}
		}
	}

	goBase := pathBase(goRel)
	for _, c := range phpCandidates {
		if pathBase(c) == goBase {
			return c
		}
	}
	return phpCandidates[0]
}

func pathBase(p string) string {
	// Input is slash-separated.
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func findRepoRoot(start string) (string, error) {
	dir := start
	for i := 0; i < 20; i++ {
		if isDir(filepath.Join(dir, "php")) && isDir(filepath.Join(dir, "go")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found (expected directories: php/ and go/)")
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
