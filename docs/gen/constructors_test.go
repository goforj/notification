package gen

import (
	"strings"
	"testing"
)

// TestConstructorCatalog covers the complete supported inventory, including aliased and optional-module constructors.
func TestConstructorCatalog(t *testing.T) {
	root, err := Root()
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Drivers []Driver `json:"drivers"`
	}
	if err := readJSON(root, "docs/providers/drivers.json", &catalog); err != nil {
		t.Fatal(err)
	}
	apis, examples, content, err := constructorCatalog(root, catalog.Drivers)
	if err != nil {
		t.Fatal(err)
	}
	if len(apis) != len(catalog.Drivers) || len(examples) != len(catalog.Drivers) {
		t.Fatal("constructor inventory is incomplete")
	}
	seen := map[string]bool{}
	for _, api := range apis {
		if seen[api.Package] || api.Category != "Driver constructors" {
			t.Fatal("duplicate or misplaced constructor", api)
		}
		seen[api.Package] = true
		example, ok := examples[api.Example]
		if !ok || !example.Calls["New"] || !strings.Contains(example.Snippet, api.Label+"(") || !sectionContainsExample(content, api.Heading, example.Snippet) {
			t.Fatal("constructor destination is not backed by executable source", api)
		}
		if err := HasSymbol(root, api.Package, api.Symbol); err != nil {
			t.Fatal(err)
		}
	}
	for _, driver := range catalog.Drivers {
		if !seen[driver.Package] {
			t.Fatal("unindexed constructor", driver.Package)
		}
	}
}

// TestConstructorSourceFailures requires a matching imported package and a real constructor call.
func TestConstructorSourceFailures(t *testing.T) {
	for _, entry := range []struct {
		name, source string
	}{
		{"missing source", ""},
		{"invalid Go", "package"},
		{"wrong import", "package main\nimport demo \"example.com/unrelated\"\nfunc main(){demo.New()}"},
		{"missing call", "package main\nimport \"github.com/goforj/notification/driver/demo\"\nfunc main(){other.New()}"},
	} {
		t.Run(entry.name, func(t *testing.T) {
			root := t.TempDir()
			if entry.source != "" {
				put(t, root, "examples/providers/main.go", entry.source)
			}
			if _, _, _, err := constructorCatalog(root, []Driver{{Package: "driver/demo"}}); err == nil {
				t.Fatal("invalid constructor source accepted")
			}
		})
	}
}
