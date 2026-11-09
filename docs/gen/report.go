package gen

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// UnitEvidence retains exact library counts separately from executed example commands.
type UnitEvidence struct {
	Packages                 map[string]PackageCoverage `json:"packages"`
	Toolchain                string                     `json:"toolchain"`
	SourceFingerprint        string                     `json:"source_fingerprint"`
	UnitTestsPassed          int                        `json:"unit_tests_passed"`
	CoveredStatements        int                        `json:"covered_statements"`
	TotalStatements          int                        `json:"total_statements"`
	CommandStatements        int                        `json:"command_statements"`
	CoveredCommandStatements int                        `json:"covered_command_statements"`
	CoveragePercent          float64                    `json:"coverage_percent"`
	Modules                  []string                   `json:"modules"`
}

// PackageCoverage preserves exact measured statement counts for one runtime library package.
type PackageCoverage struct {
	Covered int `json:"covered_statements"`
	Total   int `json:"total_statements"`
}

// Server identifies the exact upstream version and immutable integration image.
type Server struct {
	Version string `json:"version"`
	Image   string `json:"image"`
}

// IntegrationEvidence records successful real-server execution without implying hosted verification.
type IntegrationEvidence struct {
	VerifiedAt        string            `json:"verified_at"`
	SourceFingerprint string            `json:"source_fingerprint"`
	Toolchain         string            `json:"toolchain"`
	PassedTests       []string          `json:"passed_tests"`
	Servers           map[string]Server `json:"servers"`
	Scope             string            `json:"scope"`
}

// testEvent decodes the Go tool's event stream rather than counting test declarations.
type testEvent struct{ Action, Package, Test, Output string }

// passedTests validates the full event stream and retains distinct executed test identities.
func passedTests(reader io.Reader) (map[string]bool, error) {
	decoder := json.NewDecoder(reader)
	passed := map[string]bool{}
	for {
		var event testEvent
		err := decoder.Decode(&event)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if event.Action == "fail" {
			return nil, fmt.Errorf("test execution failed: %s %s", event.Package, event.Test)
		}
		if event.Action == "pass" && event.Test != "" {
			passed[event.Package+"\x00"+event.Test] = true
		}
	}
	return passed, nil
}

// coverageBlock preserves the exact statement weight when merging compatible profiles.
type coverageBlock struct{ Statements, Count int }

// mergeProfile merges real execution counts and rejects incompatible instrumentation.
func mergeProfile(blocks map[string]coverageBlock, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	if !scanner.Scan() || scanner.Text() != "mode: atomic" {
		return fmt.Errorf("expected atomic coverage profile")
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 || strings.LastIndex(fields[0], "/") < 1 {
			return fmt.Errorf("invalid coverage block")
		}
		statements, err := strconv.Atoi(fields[1])
		if err != nil || statements < 0 {
			return fmt.Errorf("invalid coverage statement count")
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil || count < 0 {
			return fmt.Errorf("invalid execution count")
		}
		prior, exists := blocks[fields[0]]
		if exists && prior.Statements != statements {
			return fmt.Errorf("incompatible coverage instrumentation")
		}
		blocks[fields[0]] = coverageBlock{statements, prior.Count + count}
	}
	return scanner.Err()
}

// profile formats stable coverage output for the Go tools and Codecov.
func profile(blocks map[string]coverageBlock) []byte {
	names := make([]string, 0, len(blocks))
	for name := range blocks {
		names = append(names, name)
	}
	sort.Strings(names)
	var out strings.Builder
	out.WriteString("mode: atomic\n")
	for _, name := range names {
		block := blocks[name]
		fmt.Fprintf(&out, "%s %d %d\n", name, block.Statements, block.Count)
	}
	return []byte(out.String())
}

// Modules reads the shared module inventory used by all repository checks.
func Modules(root string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, "scripts/module-manifest.txt"))
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(data)), nil
}

// toolchain records the actual compiler instead of a configured or guessed Go version.
func toolchain() (string, error) {
	data, err := exec.Command("go", "version").Output()
	return strings.TrimSpace(string(data)), err
}

// writeJSON writes measured evidence with stable formatting after its producer succeeds.
func writeJSON(root, path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, path), append(data, '\n'), 0644)
}

// ReportUnit merges library and child-program coverage while testing tooling independently.
func ReportUnit(root string) (UnitEvidence, error) {
	evidence := UnitEvidence{Packages: map[string]PackageCoverage{}}
	modules, err := Modules(root)
	if err != nil {
		return evidence, err
	}
	evidence.Modules = modules
	commands := map[string]bool{}
	for _, module := range modules {
		command := exec.Command("go", "-C", module, "list", "-f", "{{.Name}} {{.ImportPath}}", "./...")
		command.Dir = root
		data, err := command.Output()
		if err != nil {
			return evidence, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "main" {
				commands[fields[1]] = true
			}
		}
	}
	blocks := map[string]coverageBlock{}
	paths, err := filepath.Glob(filepath.Join(root, ".artifacts/unit-[0-9]*.out"))
	if err != nil {
		return evidence, err
	}
	paths = append(paths, filepath.Join(root, ".artifacts/examples.out"))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return evidence, err
		}
		if err := mergeProfile(blocks, bytes.NewReader(data)); err != nil {
			return evidence, err
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".artifacts/all-unit-coverage.out"), profile(blocks), 0644); err != nil {
		return evidence, err
	}
	library := map[string]coverageBlock{}
	for location, block := range blocks {
		directory := location[:strings.LastIndex(location, "/")]
		if commands[directory] {
			evidence.CommandStatements += block.Statements
			if block.Count > 0 {
				evidence.CoveredCommandStatements += block.Statements
			}
			continue
		}
		library[location] = block
		coverage := evidence.Packages[directory]
		coverage.Total += block.Statements
		evidence.TotalStatements += block.Statements
		if block.Count > 0 {
			evidence.CoveredStatements += block.Statements
			coverage.Covered += block.Statements
		}
		evidence.Packages[directory] = coverage
	}
	if err := os.WriteFile(filepath.Join(root, ".artifacts/unit-coverage.out"), profile(library), 0644); err != nil {
		return evidence, err
	}
	events, err := filepath.Glob(filepath.Join(root, ".artifacts/unit-*.json"))
	if err != nil {
		return evidence, err
	}
	passed := map[string]bool{}
	for _, path := range events {
		data, err := os.ReadFile(path)
		if err != nil {
			return evidence, err
		}
		tests, err := passedTests(bytes.NewReader(data))
		if err != nil {
			return evidence, err
		}
		for name := range tests {
			passed[name] = true
		}
	}
	if len(passed) == 0 {
		return evidence, fmt.Errorf("no executed unit tests found")
	}
	evidence.UnitTestsPassed = len(passed)
	evidence.SourceFingerprint, err = SourceFingerprint(root)
	if err != nil {
		return evidence, err
	}
	if err := checkRunFingerprint(filepath.Join(root, ".artifacts/unit-source.txt"), evidence.SourceFingerprint); err != nil {
		return evidence, err
	}
	evidence.Toolchain, err = toolchain()
	if err != nil {
		return evidence, err
	}
	if evidence.TotalStatements == 0 || evidence.CoveredStatements != evidence.TotalStatements {
		return evidence, fmt.Errorf("unit statement coverage must be exactly 100%% (%d/%d)", evidence.CoveredStatements, evidence.TotalStatements)
	}
	evidence.CoveragePercent = 100
	return evidence, writeJSON(root, "docs/unit-results.json", evidence)
}

// ReportIntegration accepts only complete successful executions of both native service fixtures.
func ReportIntegration(root, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tests, err := passedTests(bytes.NewReader(data))
	if err != nil {
		return err
	}
	names := map[string]bool{}
	for identity := range tests {
		_, name, _ := strings.Cut(identity, "\x00")
		names[name] = true
	}
	if !names["TestNtfy"] || !names["TestGotify"] {
		return fmt.Errorf("required native integration tests did not execute")
	}
	evidence := IntegrationEvidence{VerifiedAt: time.Now().UTC().Format(time.RFC3339Nano), Scope: "Real disposable ntfy and Gotify server tests. All other hosted or paired-session verification remains pending."}
	for name := range names {
		evidence.PassedTests = append(evidence.PassedTests, name)
	}
	sort.Strings(evidence.PassedTests)
	evidence.SourceFingerprint, err = SourceFingerprint(root)
	if err != nil {
		return err
	}
	if err := checkRunFingerprint(strings.TrimSuffix(path, ".json")+"-source.txt", evidence.SourceFingerprint); err != nil {
		return err
	}
	evidence.Toolchain, err = toolchain()
	if err != nil {
		return err
	}
	if err := readJSON(root, "integration/versions.json", &evidence.Servers); err != nil {
		return err
	}
	return writeJSON(root, "docs/integration-results.json", evidence)
}

// checkRunFingerprint refuses to relabel old execution artifacts as evidence for new source.
func checkRunFingerprint(path, current string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(data)) != current {
		return fmt.Errorf("test run source is stale; rerun the tests")
	}
	return nil
}
