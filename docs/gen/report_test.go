package gen

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCoverageMerge preserves exact statement counts and overlapping execution evidence.
func TestCoverageMerge(t *testing.T) {
	blocks := map[string]coverageBlock{}
	for _, data := range []string{"mode: atomic\nexample.org/library/file.go:1.1,2.1 2 1\n", "mode: atomic\nexample.org/library/file.go:1.1,2.1 2 3\n"} {
		if err := mergeProfile(blocks, strings.NewReader(data)); err != nil {
			t.Fatal(err)
		}
	}
	if block := blocks["example.org/library/file.go:1.1,2.1"]; block.Statements != 2 || block.Count != 4 {
		t.Fatal(block)
	}
	if !bytes.Contains(profile(blocks), []byte("2 4\n")) {
		t.Fatal("lost merged execution count")
	}
	for _, data := range []string{"", "mode: set\n", "mode: atomic\ninvalid\n", "mode: atomic\ninvalid 1 1\n", "mode: atomic\nexample.org/library/file.go:1.1,2.1 3 1\n", "mode: atomic\nexample.org/library/file.go:1.1,2.1 -1 1\n", "mode: atomic\nexample.org/library/file.go:1.1,2.1 2 -1\n"} {
		if err := mergeProfile(blocks, strings.NewReader(data)); err == nil {
			t.Fatal("invalid coverage accepted", data)
		}
	}
}

// TestExecutedEvents rejects failed or malformed runs instead of counting test declarations.
func TestExecutedEvents(t *testing.T) {
	data := `{"Action":"pass","Package":"fixture","Test":"TestOne"}` + "\n" + `{"Action":"pass","Package":"fixture","Test":"TestOne"}` + "\n" + `{"Action":"pass","Package":"fixture","Test":"TestOne/subtest"}`
	passed, err := passedTests(strings.NewReader(data))
	if err != nil || len(passed) != 2 {
		t.Fatal(passed, err)
	}
	for _, bad := range []string{`{"Action":"fail","Package":"fixture","Test":"TestOne"}`, "invalid"} {
		if _, err := passedTests(strings.NewReader(bad)); err == nil {
			t.Fatal("failed events accepted")
		}
	}
}

// TestUnitReport separates command coverage without excluding any runtime library statements.
func TestUnitReport(t *testing.T) {
	root := fixture(t)
	put(t, root, "scripts/module-manifest.txt", ".\n")
	put(t, root, ".artifacts/unit-0.out", "mode: atomic\ngithub.com/goforj/notification/fixture.go:1.1,2.1 2 1\ngithub.com/goforj/notification/examples/demo/main.go:1.1,2.1 3 1\n")
	put(t, root, ".artifacts/examples.out", "mode: atomic\ngithub.com/goforj/notification/fixture.go:1.1,2.1 2 3\n")
	put(t, root, ".artifacts/unit-0.json", `{"Action":"pass","Package":"fixture","Test":"TestOne"}`)
	put(t, root, ".artifacts/unit-source.txt", mustFingerprint(t, root))
	evidence, err := ReportUnit(root)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.CoveredStatements != 2 || evidence.TotalStatements != 2 || evidence.CommandStatements != 3 || evidence.UnitTestsPassed != 1 || evidence.CoveragePercent != 100 {
		t.Fatal(evidence)
	}
	if len(evidence.Packages) != 1 || evidence.Packages["github.com/goforj/notification"] != (PackageCoverage{Covered: 2, Total: 2}) {
		t.Fatal("package coverage includes command code or loses measured weights", evidence.Packages)
	}
	before, err := os.ReadFile(filepath.Join(root, "docs/unit-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, ".artifacts/unit-0.out", "mode: atomic\ngithub.com/goforj/notification/fixture.go:1.1,2.1 2 0\n")
	put(t, root, ".artifacts/examples.out", "mode: atomic\n")
	if _, err := ReportUnit(root); err == nil {
		t.Fatal("partial coverage accepted")
	}
	after, err := os.ReadFile(filepath.Join(root, "docs/unit-results.json"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed run replaced passing evidence")
	}
	put(t, root, ".artifacts/unit-0.json", "")
	if _, err := ReportUnit(root); err == nil || !strings.Contains(err.Error(), "no executed unit tests") {
		t.Fatal("empty test run accepted", err)
	}
	put(t, root, ".artifacts/unit-0.json", `{"Action":"pass","Package":"fixture","Test":"TestOne"}`)
	put(t, root, "changed.go", "package notification\n")
	if _, err := ReportUnit(root); err == nil || !strings.Contains(err.Error(), "source is stale") {
		t.Fatal("old artifacts relabelled", err)
	}
}

// TestIntegrationReport requires both native service tests from unchanged source.
func TestIntegrationReport(t *testing.T) {
	root := fixture(t)
	path := filepath.Join(root, ".artifacts/integration.json")
	putJSON(t, root, "integration/versions.json", map[string]Server{"ntfy": {Version: "1", Image: "pinned"}})
	put(t, root, ".artifacts/integration.json", `{"Action":"pass","Package":"fixture","Test":"TestNtfy"}`)
	put(t, root, ".artifacts/integration-source.txt", mustFingerprint(t, root))
	if err := ReportIntegration(root, path); err == nil {
		t.Fatal("partial native suite accepted")
	}
	put(t, root, ".artifacts/integration.json", `{"Action":"pass","Package":"fixture","Test":"TestNtfy"}`+"\n"+`{"Action":"pass","Package":"fixture","Test":"TestGotify"}`)
	if err := ReportIntegration(root, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "docs/integration-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var evidence IntegrationEvidence
	if err := json.Unmarshal(data, &evidence); err != nil || len(evidence.PassedTests) != 2 || evidence.Servers["ntfy"].Image != "pinned" {
		t.Fatal(evidence, err)
	}
	put(t, root, "changed.go", "package notification\n")
	if err := ReportIntegration(root, path); err == nil {
		t.Fatal("changed source relabelled")
	}
}
