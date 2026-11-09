package gen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// InventorySource pins the exact framework source behind the provider extraction.
type InventorySource struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// InventoryCounts distinguishes true adapters from compatibility-envelope aliases.
type InventoryCounts struct {
	BackendIDs        int `json:"backend_ids"`
	UIIDs             int `json:"ui_ids"`
	DedicatedIDs      int `json:"dedicated_ids"`
	DedicatedAdapters int `json:"dedicated_adapters"`
	GenericAliasIDs   int `json:"generic_alias_ids"`
}

// InventoryProvider preserves every stored provider ID and its honest support scope.
type InventoryProvider struct {
	ID                 string  `json:"id"`
	DemoImplementation string  `json:"demo_implementation"`
	DemoConstructor    string  `json:"demo_constructor"`
	DemoAdapter        string  `json:"demo_adapter"`
	DemoUIExposed      bool    `json:"demo_ui_exposed"`
	CanonicalID        string  `json:"canonical_id"`
	LibraryStatus      string  `json:"library_status"`
	LibrarySurface     *string `json:"library_surface"`
}

// Inventory is the complete framework snapshot without claiming alias protocols are native.
type Inventory struct {
	SchemaVersion    int                 `json:"schema_version"`
	SourceRepository string              `json:"source_repository"`
	SourceCommit     string              `json:"source_commit"`
	SourceFiles      []InventorySource   `json:"source_files"`
	Counts           InventoryCounts     `json:"counts"`
	Scope            string              `json:"scope"`
	Providers        []InventoryProvider `json:"providers"`
}

// matches centralizes small extraction expressions without importing framework dependencies.
func matches(pattern, text string) [][]string {
	return regexp.MustCompile(pattern).FindAllStringSubmatch(text, -1)
}

// InventoryDemo reads source templates directly and never renders into the framework checkout.
func InventoryDemo(root, checkout string) (Inventory, error) {
	inventory := Inventory{SchemaVersion: 1, SourceRepository: "goforj/goforj", Scope: "Complete demo ID inventory, not a native protocol or hosted-delivery certification."}
	directory := filepath.Join(checkout, "templates/demo/internal/notification")
	paths := map[string]bool{}
	read := func(path string) (string, error) {
		data, err := os.ReadFile(path)
		if err == nil {
			paths[path] = true
		}
		return string(data), err
	}
	sources := map[string]string{}
	for name, path := range map[string]string{"constants": filepath.Join(directory, "channel_repo.go.tmpl"), "catalog": filepath.Join(directory, "provider_catalog.go.tmpl"), "manager": filepath.Join(directory, "manager.go.tmpl"), "generic": filepath.Join(directory, "generic_webhook_compat_provider.go.tmpl"), "ui": filepath.Join(checkout, "templates/demo/frontend/src/lib/notification-providers.ts")} {
		text, err := read(path)
		if err != nil {
			return inventory, err
		}
		sources[name] = text
	}
	constants := map[string]string{}
	for _, match := range matches(`(Provider\w+)\s*=\s*"([^"]+)"`, sources["constants"]) {
		constants[match[1]] = match[2]
	}
	provider := func(token string) string {
		token = strings.TrimSpace(token)
		if value, ok := constants[token]; ok {
			return value
		}
		return strings.Trim(token, `"`)
	}
	ids := map[string]bool{}
	catalog, _, found := strings.Cut(sources["catalog"], "func NormalizeProviderID")
	if !found {
		return inventory, fmt.Errorf("missing provider normalization boundary")
	}
	for _, match := range matches(`(?m)^\s*("[^"]+"|Provider\w+):\s*\{\}`, catalog) {
		ids[provider(match[1])] = true
	}
	_, special, found := strings.Cut(sources["catalog"], "func IsSupportedNotificationProvider")
	if !found {
		return inventory, fmt.Errorf("missing supported-provider boundary")
	}
	special, _, _ = strings.Cut(special, "func IsWebhookCompatProvider")
	for _, match := range matches(`normalized == (Provider\w+)`, special) {
		ids[provider(match[1])] = true
	}
	ui := map[string]bool{}
	array := matches(`(?s)const webhookCompatProviderIDs[^=]*=\s*\[(.*?)\]`, sources["ui"])
	if len(array) != 1 {
		return inventory, fmt.Errorf("missing UI provider array")
	}
	for _, match := range matches(`'([^']+)'`, array[0][1]) {
		ui[match[1]] = true
	}
	for _, match := range matches(`\bid:\s*'([^']+)'`, sources["ui"]) {
		ui[match[1]] = true
	}
	if len(ids) == 0 || len(ui) == 0 {
		return inventory, fmt.Errorf("empty demo provider catalog")
	}
	for name := range ui {
		if !ids[name] {
			return inventory, fmt.Errorf("UI provider missing from backend: %s", name)
		}
	}
	constructors := map[string]string{}
	files, err := filepath.Glob(filepath.Join(directory, "*.go.tmpl"))
	if err != nil {
		return inventory, err
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go.tmpl") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return inventory, err
		}
		for _, match := range matches(`func (New\w+)\(`, string(data)) {
			constructors[match[1]] = path
		}
	}
	_, manager, found := strings.Cut(sources["manager"], "func (m *Manager) providerForChannel")
	if !found {
		return inventory, fmt.Errorf("missing provider manager boundary")
	}
	manager, _, _ = strings.Cut(manager, "func notificationHeaders")
	direct := map[string]string{}
	adapterNames := map[string]bool{}
	for _, part := range strings.Split(manager, "\n\tcase ")[1:] {
		tokens, body, found := strings.Cut(part, ":\n")
		if !found {
			return inventory, fmt.Errorf("invalid dedicated provider case")
		}
		body, _, _ = strings.Cut(body, "\n\tdefault:")
		constructor := matches(`return (New\w+)\(`, body)
		if len(constructor) == 0 {
			return inventory, fmt.Errorf("dedicated constructor could not be resolved")
		}
		name := constructor[0][1]
		path, ok := constructors[name]
		if !ok {
			return inventory, fmt.Errorf("dedicated constructor missing: %s", name)
		}
		paths[path] = true
		adapterNames[name] = true
		for _, token := range strings.Split(tokens, ",") {
			id := provider(token)
			if !ids[id] {
				return inventory, fmt.Errorf("manager provider missing from backend: %s", id)
			}
			direct[id] = name
		}
	}
	data, err := exec.Command("git", "-C", checkout, "rev-parse", "HEAD").Output()
	if err != nil {
		return inventory, err
	}
	inventory.SourceCommit = strings.TrimSpace(string(data))
	sourcePaths := make([]string, 0, len(paths))
	for path := range paths {
		sourcePaths = append(sourcePaths, path)
	}
	sort.Strings(sourcePaths)
	for _, path := range sourcePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return inventory, err
		}
		digest := sha256.Sum256(data)
		relative, err := filepath.Rel(checkout, path)
		if err != nil {
			return inventory, err
		}
		inventory.SourceFiles = append(inventory.SourceFiles, InventorySource{filepath.ToSlash(relative), hex.EncodeToString(digest[:])})
	}
	var libraryCatalog struct {
		Drivers []Driver `json:"drivers"`
	}
	if err := readJSON(root, "docs/providers/drivers.json", &libraryCatalog); err != nil {
		return inventory, err
	}
	library := map[string]string{}
	for _, driver := range libraryCatalog.Drivers {
		library[driver.ID] = driver.Package + " (" + driver.Verification + ")"
	}
	library["smtp"] = library["email"]
	names := make([]string, 0, len(ids))
	for name := range ids {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entry := InventoryProvider{ID: name, DemoImplementation: "generic_webhook_compat", DemoConstructor: "NewGenericWebhookCompatProvider", DemoAdapter: "templates/demo/internal/notification/generic_webhook_compat_provider.go.tmpl", DemoUIExposed: ui[name], CanonicalID: name, LibraryStatus: "planned"}
		if constructor, ok := direct[name]; ok {
			entry.DemoImplementation = "dedicated"
			entry.DemoConstructor = constructor
			path, err := filepath.Rel(checkout, constructors[constructor])
			if err != nil {
				return inventory, err
			}
			entry.DemoAdapter = filepath.ToSlash(path)
		}
		if name == "smtp" {
			entry.CanonicalID = "email"
		}
		if surface, ok := library[name]; ok {
			entry.LibraryStatus = "available"
			entry.LibrarySurface = &surface
		} else if name == "sendgrid" || name == "resend" {
			entry.LibraryStatus = "mail_composition_candidate"
			surface := "driver/mail with sibling mail" + name + " (not verified here)"
			entry.LibrarySurface = &surface
		}
		inventory.Providers = append(inventory.Providers, entry)
	}
	inventory.Counts = InventoryCounts{len(ids), len(ui), len(direct), len(adapterNames), len(ids) - len(direct)}
	return inventory, nil
}

// InventoryMarkdown projects every ID without promoting generic aliases to native support.
func InventoryMarkdown(inventory Inventory) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Uptime Gopher provider inventory\n\nGenerated by `go -C docs run ./inventory -goforj ../goforj` from the GoForj demo templates.\n\nSource commit: `%s`. Exact source file hashes are retained in [the JSON snapshot](uptime-gopher.json).\n\nThe backend recognizes %d IDs; the UI exposes %d. %d IDs use %d dedicated adapters, while %d IDs use the generic webhook compatibility envelope.\n\n", inventory.SourceCommit, inventory.Counts.BackendIDs, inventory.Counts.UIIDs, inventory.Counts.DedicatedIDs, inventory.Counts.DedicatedAdapters, inventory.Counts.GenericAliasIDs)
	out.WriteString("Dedicated means provider-specific code exists. It does not establish that the current upstream API accepts its payload or that hosted delivery was tested. Generic compatibility means the demo sends its own provider/text/message envelope to a configured URL; it is not native provider support.\n\nLibrary surfaces below distinguish real-server verification from protocol fixtures and pending hosted verification. SendGrid and Resend are candidates for composition through sibling Mail drivers, not newly verified notification integrations.\n\n## Dedicated demo adapters\n\n| Provider ID | Demo UI | Current notification library surface |\n| --- | --- | --- |\n")
	for _, entry := range inventory.Providers {
		if entry.DemoImplementation != "dedicated" {
			continue
		}
		surface := "Planned"
		if entry.LibrarySurface != nil {
			surface = *entry.LibrarySurface
		}
		ui := "No"
		if entry.DemoUIExposed {
			ui = "Yes"
		}
		fmt.Fprintf(&out, "| %s | %s | %s |\n", entry.ID, ui, surface)
	}
	out.WriteString("\nSMTP and Email share the same demo adapter. Their stored provider IDs remain distinct for migration.\n\n## Generic webhook compatibility aliases\n\n| Provider ID | Demo UI | Native support status |\n| --- | --- | --- |\n")
	for _, entry := range inventory.Providers {
		if entry.DemoImplementation != "generic_webhook_compat" {
			continue
		}
		surface := "Requires provider protocol review and implementation"
		if entry.LibrarySurface != nil {
			surface = *entry.LibrarySurface
		}
		ui := "No"
		if entry.DemoUIExposed {
			ui = "Yes"
		}
		fmt.Fprintf(&out, "| %s | %s | %s |\n", entry.ID, ui, surface)
	}
	fmt.Fprintf(&out, "\nThe extraction scope is all %d IDs in this snapshot, with aliases, retired services, and delegated Mail transports recorded explicitly. See [the provider and API review](../provider-review.md) for sequencing and compatibility decisions.\n", inventory.Counts.BackendIDs)
	return out.String()
}

// WriteInventory updates both projections only after successful source extraction.
func WriteInventory(root, checkout string) error {
	inventory, err := InventoryDemo(root, checkout)
	if err != nil {
		return err
	}
	if err := writeJSON(root, "docs/providers/uptime-gopher.json", inventory); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "docs/providers/uptime-gopher.md"), []byte(InventoryMarkdown(inventory)), 0644)
}
