package gen

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// put creates isolated fixture sources without changing the real checkout.
func put(t *testing.T, root, path, text string) {
	t.Helper()
	target := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

// putJSON uses the same metadata types as production generation.
func putJSON(t *testing.T, root, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, path, string(data))
}

// freshUnit records evidence only after the fixture's authoritative Go source is complete.
func freshUnit(t *testing.T, root string) {
	t.Helper()
	fingerprint, err := SourceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	putJSON(t, root, "docs/unit-results.json", UnitEvidence{SourceFingerprint: fingerprint, CoveredStatements: 1, TotalStatements: 1, UnitTestsPassed: 1, Packages: map[string]PackageCoverage{"github.com/goforj/notification": {Covered: 1, Total: 1}}})
}

// fixture builds a tiny real source/example/template composition for validation failures.
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	put(t, root, "go.mod", "module github.com/goforj/notification\n\ngo 1.24.0\n")
	put(t, root, "fixture.go", "package notification\n// New constructs the fixture.\nfunc New(){}\n")
	put(t, root, "examples/demo/main.go", `package main
func main(){
 // snippet:start
 notification.New()
 fmt.Println("hello")
 // hello
 // snippet:end
}`)
	put(t, root, "docs/readme.tmpl.md", "{{integration-badge}}\n## Demo\n{{snippet:demo}}\n{{drivers}}\n{{api-index}}\n")
	put(t, root, "docs/advanced.tmpl.md", "# Advanced\n")
	putJSON(t, root, "docs/providers/drivers.json", map[string]any{"drivers": []Driver{}})
	putJSON(t, root, "docs/assets/driver-icons/presentation.json", map[string]Presentation{})
	putJSON(t, root, "docs/assets/feature-icons/presentation.json", map[string]Presentation{})
	freshUnit(t, root)
	return root
}

// demoAPI selects the actual operation present inside the authoritative fixture snippet.
func demoAPI() []API { return []API{{"New", ".", "New", "demo", "Demo", "Construction"}} }

// TestExamplesAndAPIs keeps output comments and API navigation bound to one runnable source.
func TestExamplesAndAPIs(t *testing.T) {
	root := fixture(t)
	outputs, err := outputsFor(root, demoAPI())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["README.md"]), "[New](#demo)") || !strings.Contains(string(outputs["README.md"]), "// hello") {
		t.Fatal("missing checked source projection")
	}
	if err := HasSymbol(root, ".", "New"); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"", "// snippet:start\nfmt.Println(1)\n// snippet:end", "// snippet:end\n// snippet:start\nfmt.Println(1)\n// hello", "// snippet:start\nfmt.Println(1)\n// 1\n// snippet:end\n// snippet:end"} {
		if _, err := Snippet(source); err == nil {
			t.Fatal("invalid snippet accepted")
		}
	}
	if err := HasSymbol(root, ".", "Missing"); err == nil {
		t.Fatal("unknown API accepted")
	}
	put(t, root, "fixture.go", "package notification\nfunc New(){}\n")
	if err := HasSymbol(root, ".", "New"); err == nil {
		t.Fatal("undocumented API accepted")
	}
}

// TestProjectionFailures rejects stale, incomplete and misleading documentation inputs.
func TestProjectionFailures(t *testing.T) {
	for _, defect := range []string{"stale", "coverage", "duplicate", "missing-target", "missing-heading", "missing-api", "outside-snippet", "invalid-source", "unresolved", "missing-category", "wrong-section"} {
		t.Run(defect, func(t *testing.T) {
			root := fixture(t)
			apis := demoAPI()
			switch defect {
			case "missing-category":
				apis[0].Category = ""
			case "wrong-section":
				put(t, root, "docs/readme.tmpl.md", "## Demo\n## Other\n{{snippet:demo}}\n{{api-index}}")
			case "stale":
				put(t, root, "changed.go", "package notification\n")
			case "coverage":
				putJSON(t, root, "docs/unit-results.json", UnitEvidence{SourceFingerprint: mustFingerprint(t, root), TotalStatements: 1})
			case "duplicate":
				put(t, root, "docs/advanced.tmpl.md", "{{snippet:demo}}")
			case "missing-target":
				put(t, root, "docs/readme.tmpl.md", "## Demo\n{{api-index}}")
			case "missing-heading":
				put(t, root, "docs/readme.tmpl.md", "{{snippet:demo}}\n{{api-index}}")
			case "missing-api":
				apis[0].Symbol = "Unknown"
			case "outside-snippet":
				put(t, root, "examples/demo/main.go", "package main\nfunc main(){notification.New()\n// snippet:start\nfmt.Println(1)\n// 1\n// snippet:end\n}")
				freshUnit(t, root)
			case "invalid-source":
				put(t, root, "examples/demo/main.go", "broken Go source")
				freshUnit(t, root)
			case "unresolved":
				put(t, root, "docs/advanced.tmpl.md", "{{snippet:missing}}")
			}
			if _, err := outputsFor(root, apis); err == nil {
				t.Fatal("invalid projection accepted")
			}
		})
	}
}

// mustFingerprint simplifies fixture evidence without guessing source identities.
func mustFingerprint(t *testing.T, root string) string {
	t.Helper()
	value, err := SourceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

// TestFingerprintAndRoot binds modules and tests while ignoring generated assets and artifacts.
func TestFingerprintAndRoot(t *testing.T) {
	root := fixture(t)
	original := mustFingerprint(t, root)
	put(t, root, ".artifacts/ignored.go", "ignored")
	put(t, root, "docs/assets/ignored.svg", "image")
	if mustFingerprint(t, root) != original {
		t.Fatal("generated artifacts changed source evidence")
	}
	put(t, root, "docs/go.mod", "module github.com/goforj/notification/docs\n")
	found, err := rootFrom(filepath.Join(root, "docs"))
	if err != nil || found != root {
		t.Fatal("selected the tooling module", found, err)
	}
	if mustFingerprint(t, root) == original {
		t.Fatal("module requirements omitted")
	}
	if _, err := rootFrom(t.TempDir()); err == nil {
		t.Fatal("missing module root accepted")
	}
}

// TestDriverEvidenceAndSearchableNames preserves native evidence and searchable names beside aligned icons.
func TestDriverEvidenceAndSearchableNames(t *testing.T) {
	root := fixture(t)
	put(t, root, "driver/ntfy/ntfy.go", "package ntfy\n")
	put(t, root, "docs/assets/driver-icons/ntfy.svg", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M0 0h24v24z"/></svg>`)
	putJSON(t, root, "docs/assets/driver-icons/presentation.json", map[string]Presentation{"ntfy": {Label: "ntfy", Icon: "ntfy.svg", Accent: "#317F6F"}})
	putJSON(t, root, "docs/providers/drivers.json", map[string]any{"drivers": []Driver{{ID: "ntfy", Package: "driver/ntfy", Verification: "Real server verified"}}})
	servers := map[string]Server{"ntfy": {Version: "1", Image: "pinned"}}
	putJSON(t, root, "integration/versions.json", servers)
	freshUnit(t, root)
	evidence := IntegrationEvidence{SourceFingerprint: mustFingerprint(t, root), Servers: servers, PassedTests: []string{"TestNtfy"}}
	putJSON(t, root, "docs/integration-results.json", evidence)
	if _, err := outputsFor(root, demoAPI()); err == nil {
		t.Fatal("partial native evidence accepted")
	}
	evidence.PassedTests = append(evidence.PassedTests, "TestGotify")
	putJSON(t, root, "docs/integration-results.json", evidence)
	outputs, err := outputsFor(root, demoAPI())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(outputs["docs/assets/badges/integration-tests.svg"], []byte("integration tests: 2")) || !bytes.Contains(outputs["README.md"], []byte("Executed native integration tests")) {
		t.Fatal("native integration badge is not backed by the executed inventory")
	}
	if !strings.Contains(string(outputs["README.md"]), `<strong>ntfy</strong>`) || !strings.Contains(string(outputs["README.md"]), `align="absmiddle"`) || bytes.Contains(outputs["docs/assets/badges/driver-ntfy.svg"], []byte("<text")) {
		t.Fatal("driver name is not searchable beside an aligned icon badge")
	}
	if !strings.Contains(string(outputs["README.md"]), `align="absmiddle"> <a href="driver/ntfy"><strong>ntfy</strong></a>`) {
		t.Fatal("driver icon or gap was included in the text link")
	}
	if strings.Contains(string(outputs["README.md"]), ">Name</th>") {
		t.Fatal("duplicate name column retained")
	}
	servers["matrix"] = Server{Version: "1", Image: "pinned"}
	putJSON(t, root, "integration/versions.json", servers)
	evidence.Servers = servers
	putJSON(t, root, "docs/integration-results.json", evidence)
	if _, err := outputsFor(root, demoAPI()); err == nil {
		t.Fatal("Matrix evidence omitted")
	}
	evidence.PassedTests = append(evidence.PassedTests, "TestMatrix")
	putJSON(t, root, "docs/integration-results.json", evidence)
	if _, err := outputsFor(root, demoAPI()); err != nil {
		t.Fatal(err)
	}
	putJSON(t, root, "docs/assets/driver-icons/presentation.json", map[string]Presentation{})
	if _, err := outputsFor(root, demoAPI()); err == nil {
		t.Fatal("missing presentation accepted")
	}
}

// TestArtworkAndStatusBadges checks standalone vectors, raster embedding and accessible titles.
func TestArtworkAndStatusBadges(t *testing.T) {
	root := t.TempDir()
	path := "docs/assets/driver-icons/mark.svg"
	put(t, root, path, `<svg xmlns="http://www.w3.org/2000/svg" xml:space="preserve" viewBox="0 0 24 24"><foreignObject width="1"/><path d="M0 0h24v24z"/></svg>`)
	p := Presentation{Label: "A & B", Icon: "mark.svg", Fill: "#123456", Accent: "#123456"}
	for _, labelled := range []bool{false, true} {
		data, err := ArtworkBadge(root, "driver-icons", p, labelled)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "foreignObject") || !strings.Contains(string(data), "A &amp; B") || strings.Contains(string(data), "<text") != labelled {
			t.Fatal("invalid artwork projection")
		}
		var check struct{ XMLName xml.Name }
		if err := xml.Unmarshal(data, &check); err != nil {
			t.Fatal(err)
		}
	}
	p.Icon = "mark.png"
	put(t, root, "docs/assets/driver-icons/mark.png", "PNG fixture")
	if data, err := ArtworkBadge(root, "driver-icons", p, false); err != nil || !strings.Contains(string(data), "data:image/png;base64,") {
		t.Fatal("raster was not embedded", err)
	}
	for _, defect := range []string{"missing", "invalid-xml", "not-svg", "format", "accent", "hex"} {
		t.Run(defect, func(t *testing.T) {
			invalid := p
			switch defect {
			case "missing":
				invalid.Icon = "absent.svg"
			case "invalid-xml":
				invalid.Icon = "bad.svg"
				put(t, root, "docs/assets/driver-icons/bad.svg", "<svg>")
			case "not-svg":
				invalid.Icon = "bad.svg"
				put(t, root, "docs/assets/driver-icons/bad.svg", "<html/>")
			case "format":
				invalid.Icon = "mark.txt"
				put(t, root, "docs/assets/driver-icons/mark.txt", "invalid")
			case "accent":
				invalid.Accent = "invalid"
			case "hex":
				invalid.Accent = "#zzzzzz"
			}
			if _, err := ArtworkBadge(root, "driver-icons", invalid, false); err == nil {
				t.Fatal("invalid artwork accepted")
			}
		})
	}
	if !strings.Contains(string(StatusBadge("A & B", "100%", "#123456")), "A &amp; B") {
		t.Fatal("status title not escaped")
	}
}

// TestGroupedAPIIndex keeps categories compact while preserving each operation's own destination.
func TestGroupedAPIIndex(t *testing.T) {
	root := fixture(t)
	apis := demoAPI()
	apis = append(apis,
		API{"Notifier.Send", ".", "New", "demo", "Demo", "Sending"},
		API{"driver.New", ".", "New", "demo", "Demo", "Construction"},
	)
	outputs, err := outputsFor(root, apis)
	if err != nil {
		t.Fatal(err)
	}
	readme := string(outputs["README.md"])
	want := "| Category | Functions |\n| ---: | --- |\n| Construction | [New](#demo) · [driver.New](#demo) |\n| Sending | [Notifier.Send](#demo) |"
	if !strings.Contains(readme, want) {
		t.Fatal("grouped navigation changed", readme)
	}
}

// TestSectionContainsExample rejects unrelated headings and accepts nested examples at their actual destination.
func TestSectionContainsExample(t *testing.T) {
	for _, entry := range []struct {
		name, markdown, heading string
		found                   bool
	}{
		{"section", "## Sending\ncall()\n## Next\n", "Sending", true},
		{"subsection", "## Push\n### APNs\ncall()\n### SNS\nother()", "APNs", true},
		{"last subsection", "## Push\n### APNs\ncall()", "APNs", true},
		{"wrong section", "## Sending\nother()\n## Next\ncall()", "Sending", false},
		{"wrong subsection", "## Push\n### APNs\nother()\n### SNS\ncall()", "APNs", false},
		{"missing", "## Sending\ncall()", "Unknown", false},
		{"fenced heading", "## Sending\n```go\n### Sending\ncall()\n```\n## Next", "Sending", true},
		{"fenced target", "```go\n## Sending\ncall()\n```", "Sending", false},
	} {
		t.Run(entry.name, func(t *testing.T) {
			if got := sectionContainsExample(entry.markdown, entry.heading, "call()"); got != entry.found {
				t.Fatal("example destination mismatch", got)
			}
		})
	}
}
