package gen

import (
	"strings"
	"testing"
)

// TestReadableFeatureTable preserves searchable names, no-wrap labels and true middle icon alignment.
func TestReadableFeatureTable(t *testing.T) {
	features := map[string]Presentation{"intent": {Label: "Simple messages", Description: "Send a title & body."}}
	table, err := featuresTable(features)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`nowrap`, `align="absmiddle"`, `<strong>Simple messages</strong>`, "Send a title &amp; body."} {
		if !strings.Contains(table, required) {
			t.Fatal("missing readable table contract", required)
		}
	}
	features["intent"] = Presentation{Label: "Simple messages"}
	if _, err := featuresTable(features); err == nil {
		t.Fatal("missing feature description accepted")
	}
	if _, err := featuresTable(map[string]Presentation{"unknown": {Description: "unknown"}}); err == nil {
		t.Fatal("unknown feature accepted")
	}
}

// TestCompactDriverSummaries preserves precise evidence scope and all detailed configuration in the guide.
func TestCompactDriverSummaries(t *testing.T) {
	for _, entry := range []struct{ source, want string }{
		{"Real Gotify 3.1.1 server verified", "Gotify 3.1.1"},
		{"Mail composition tests; hosted verification pending", "Composition tests"},
		{"Local capture tests", "Local capture"},
		{"Local protocol tests; hosted verification pending", "Protocol tests"},
		{"Local protocol tests; paired-session verification pending", "Protocol tests"},
		{"Future verification scope", "Future verification scope"},
	} {
		if got := verificationSummary(Driver{Verification: entry.source}); got != entry.want {
			t.Fatal(got, entry.want)
		}
	}
	driver := Driver{Module: "example/optional", ID: "fixture", Package: "driver/fixture", Configuration: "configured endpoint", Acceptance: "native identity", AcceptanceSummary: "Message ID", Verification: "pending", ProtocolSource: "https://example.com/protocol"}
	if acceptanceSummary(driver) != "Message ID" {
		t.Fatal("summary missing")
	}
	driver.AcceptanceSummary = ""
	if acceptanceSummary(driver) != "native identity" {
		t.Fatal("full contract fallback missing")
	}
	guide := driverGuide([]Driver{driver}, map[string]Presentation{"fixture": {Label: "Fixture"}})
	for _, required := range []string{"example/optional", "configured endpoint", "native identity", "pending", "https://example.com/protocol"} {
		if !strings.Contains(guide, required) {
			t.Fatal("detailed contract lost", required)
		}
	}
	driver.ProtocolSource = ""
	if strings.Contains(driverGuide([]Driver{driver}, map[string]Presentation{"fixture": {Label: "Fixture"}}), "Provider protocol reference") {
		t.Fatal("invented upstream reference")
	}
}
