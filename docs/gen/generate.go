package gen

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// API connects a documented operation to the authoritative program demonstrating it.
type API struct{ Label, Package, Symbol, Example, Heading string }

// applicationAPI limits navigation to operations an application developer uses.
var applicationAPI = []API{
	{"New", ".", "New", "quickstart", "Quick Start"},
	{"Fanout", ".", "Fanout", "composition", "Composition"},
	{"Fallback", ".", "Fallback", "composition", "Composition"},
	{"SenderFunc.Send", ".", "SenderFunc.Send", "decorator", "Custom senders"},
	{"DeliveryError.RetryAfter", ".", "DeliveryError.RetryAfter", "composition", "Composition"},
	{"pagerduty.Driver.Trigger", "driver/pagerduty", "Driver.Trigger", "incident", "Incident lifecycle"},
	{"pagerduty.Driver.Acknowledge", "driver/pagerduty", "Driver.Acknowledge", "incident", "Incident lifecycle"},
	{"pagerduty.Driver.Resolve", "driver/pagerduty", "Driver.Resolve", "incident", "Incident lifecycle"},
	{"Notifier.Send", ".", "Notifier.Send", "quickstart", "Quick Start"},
	{"webhook.New", "driver/webhook", "New", "webhook", "Webhook delivery"},
	{"notificationfake.New", "notificationfake", "New", "fake", "Application testing"},
	{"Fake.Messages", "notificationfake", "Fake.Messages", "fake", "Application testing"},
	{"notificationmail.New", "driver/mail", "New", "mail", "Mail delivery"},
}

// Driver records provider-owned handling semantics and executed verification scope.
type Driver struct {
	ID                string `json:"id"`
	AcceptanceSummary string `json:"acceptance_summary"`
	ProtocolSource    string `json:"protocol_source"`
	Package           string `json:"package"`
	Configuration     string `json:"configuration"`
	Acceptance        string `json:"acceptance"`
	Verification      string `json:"verification"`
}

// readJSON decodes authoritative local metadata with its source path in failures.
func readJSON(root, path string, target any) error {
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// Outputs computes all projections before writing so invalid evidence cannot produce partial output.
func Outputs(root string) (map[string][]byte, error) { return outputsFor(root, applicationAPI) }

// outputsFor allows small fixtures to exercise the same validation as the full library.
func outputsFor(root string, apis []API) (map[string][]byte, error) {
	var unit UnitEvidence
	if err := readJSON(root, "docs/unit-results.json", &unit); err != nil {
		return nil, err
	}
	fingerprint, err := SourceFingerprint(root)
	if err != nil {
		return nil, err
	}
	if unit.SourceFingerprint != fingerprint {
		return nil, fmt.Errorf("unit evidence is stale; run make test")
	}
	if unit.TotalStatements == 0 || unit.CoveredStatements != unit.TotalStatements {
		return nil, fmt.Errorf("full library unit coverage is required")
	}
	examples, err := Examples(root)
	if err != nil {
		return nil, err
	}
	templates := map[string]string{}
	for _, name := range []string{"README.md", "docs/advanced.md"} {
		source := "docs/readme.tmpl.md"
		if name != "README.md" {
			source = "docs/advanced.tmpl.md"
		}
		data, err := os.ReadFile(filepath.Join(root, source))
		if err != nil {
			if name != "README.md" && os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		templates[name] = string(data)
	}
	byExample := map[string]Example{}
	for _, example := range examples {
		marker := "{{snippet:" + example.Name + "}}"
		count := 0
		for _, text := range templates {
			count += strings.Count(text, marker)
		}
		if count != 1 {
			return nil, fmt.Errorf("example %s needs one documentation target", example.Name)
		}
		for path, text := range templates {
			templates[path] = strings.ReplaceAll(text, marker, "```go\n"+example.Snippet+"\n```")
		}
		byExample[example.Name] = example
	}
	var catalog struct {
		Drivers []Driver `json:"drivers"`
	}
	if err := readJSON(root, "docs/providers/drivers.json", &catalog); err != nil {
		return nil, err
	}
	nativeTests := 0
	for _, driver := range catalog.Drivers {
		if strings.HasPrefix(driver.Verification, "Real ") {
			var integration IntegrationEvidence
			var versions map[string]Server
			if err := readJSON(root, "docs/integration-results.json", &integration); err != nil {
				return nil, err
			}
			if err := readJSON(root, "integration/versions.json", &versions); err != nil {
				return nil, err
			}
			passed := map[string]bool{}
			for _, name := range integration.PassedTests {
				passed[name] = true
			}
			if integration.SourceFingerprint != fingerprint || !reflect.DeepEqual(integration.Servers, versions) || !passed["TestNtfy"] || !passed["TestGotify"] {
				return nil, fmt.Errorf("native integration evidence is stale; run make test-integration")
			}
			nativeTests = len(passed)
			break
		}
	}
	outputs := map[string][]byte{}
	integrationBadge := ""
	if nativeTests > 0 {
		outputs["docs/assets/badges/integration-tests.svg"] = StatusBadge("integration tests", fmt.Sprint(nativeTests), "#007ec6")
		integrationBadge = `  <a href="docs/integration-results.json"><img src="docs/assets/badges/integration-tests.svg" alt="Executed native integration tests"></a>`
	}
	templates["README.md"] = strings.ReplaceAll(templates["README.md"], "{{integration-badge}}", integrationBadge)
	var presentations map[string]Presentation
	if err := readJSON(root, "docs/assets/driver-icons/presentation.json", &presentations); err != nil {
		return nil, err
	}
	var drivers strings.Builder
	drivers.WriteString("<table>\n<thead><tr><th align=\"left\" width=\"260\">Driver</th><th align=\"left\">Acceptance</th><th align=\"left\">Verification</th></tr></thead>\n<tbody>\n")
	for _, driver := range catalog.Drivers {
		info, err := os.Stat(filepath.Join(root, driver.Package))
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("missing driver package: %s", driver.Package)
		}
		presentation, ok := presentations[driver.ID]
		if !ok {
			return nil, fmt.Errorf("missing driver presentation: %s", driver.ID)
		}
		asset := "docs/assets/badges/driver-" + driver.ID + ".svg"
		badge, err := ArtworkBadge(root, "driver-icons", presentation, false)
		if err != nil {
			return nil, err
		}
		outputs[asset] = badge
		fmt.Fprintf(&drivers, "<tr><td width=\"260\" nowrap><img src=\"%s\" width=\"22\" height=\"22\" alt=\"\" align=\"absmiddle\"> <a href=\"%s\"><strong>%s</strong></a></td><td>%s</td><td>%s</td></tr>\n", asset, driver.Package, html.EscapeString(presentation.Label), html.EscapeString(acceptanceSummary(driver)), html.EscapeString(verificationSummary(driver)))
	}
	drivers.WriteString("</tbody>\n</table>")
	templates["README.md"] = strings.ReplaceAll(templates["README.md"], "{{drivers}}", drivers.String())
	outputs["docs/drivers.md"] = []byte(driverGuide(catalog.Drivers, presentations))
	var features map[string]Presentation
	if err := readJSON(root, "docs/assets/feature-icons/presentation.json", &features); err != nil {
		return nil, err
	}
	featureTable, err := featuresTable(features)
	if err != nil {
		return nil, err
	}
	templates["README.md"] = strings.ReplaceAll(templates["README.md"], "{{features}}", featureTable)
	var index strings.Builder
	index.WriteString("| API | Example |\n| --- | --- |")
	for _, api := range apis {
		if err := HasSymbol(root, api.Package, api.Symbol); err != nil {
			return nil, err
		}
		example, ok := byExample[api.Example]
		operation := api.Symbol[strings.LastIndex(api.Symbol, ".")+1:]
		if !ok || !example.Calls[operation] || !strings.Contains(example.Snippet, "."+operation+"(") {
			return nil, fmt.Errorf("%s is absent from its linked example", api.Label)
		}
		anchor := strings.ReplaceAll(strings.ToLower(api.Heading), " ", "-")
		if !strings.Contains(templates["README.md"], "## "+api.Heading+"\n") {
			return nil, fmt.Errorf("missing anchor %s", anchor)
		}
		fmt.Fprintf(&index, "\n| [%s](#%s) | %s |", api.Label, anchor, api.Heading)
	}
	templates["README.md"] = strings.ReplaceAll(templates["README.md"], "{{api-index}}", index.String())
	for path, text := range templates {
		if strings.Contains(text, "{{snippet:") || strings.Contains(text, "{{api-index}}") || strings.Contains(text, "{{drivers}}") || strings.Contains(text, "{{features}}") {
			return nil, fmt.Errorf("unresolved README marker")
		}
		outputs[path] = []byte(text)
	}
	for _, item := range []struct{ Name, Label, Value, Color string }{{"status", "status", "unreleased", "#6e7681"}, {"go", "Go", "1.24+", "#007d9c"}, {"license", "license", "MIT", "#007ec6"}, {"coverage", "coverage", "100%", "#2e8540"}, {"tests", "unit tests", fmt.Sprint(unit.UnitTestsPassed), "#5368bd"}} {
		outputs["docs/assets/badges/"+item.Name+".svg"] = StatusBadge(item.Label, item.Value, item.Color)
	}
	for name, presentation := range features {
		badge, err := ArtworkBadge(root, "feature-icons", presentation, false)
		if err != nil {
			return nil, err
		}
		outputs["docs/assets/badges/"+name+".svg"] = badge
	}
	return outputs, nil
}

// Generate writes or checks every deterministic projection after all validation succeeds.
func Generate(root string, check bool) error {
	outputs, err := Outputs(root)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(root, name)
		if check {
			current, err := os.ReadFile(path)
			if err != nil || string(current) != string(outputs[name]) {
				return fmt.Errorf("generated file is stale: %s", name)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, outputs[name], 0644); err != nil {
			return err
		}
	}
	return nil
}
