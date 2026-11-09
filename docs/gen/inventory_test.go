package gen

import (
	"os/exec"
	"strings"
	"testing"
)

// demoFixture creates real template inputs and local Git provenance without rendering an application.
func demoFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	base := "templates/demo/internal/notification/"
	for path, text := range map[string]string{
		"channel_repo.go.tmpl": "const (\nProviderLog = \"log\"\nProviderEmail = \"email\"\nProviderSMTP = \"smtp\"\n)",
		"provider_catalog.go.tmpl": `var providers = map[string]struct{}{
 ProviderSMTP: {},
 "matrix": {},
}
func NormalizeProviderID() {}
func IsSupportedNotificationProvider() { return normalized == ProviderLog || normalized == ProviderEmail }
func IsWebhookCompatProvider() {}`,
		"manager.go.tmpl":                         "func (m *Manager) providerForChannel() {\n switch providerID {\n\tcase ProviderLog:\n return NewLogProvider()\n\tcase ProviderEmail, ProviderSMTP:\n return NewEmailProvider()\n\tdefault:\n return nil\n }\n}\nfunc notificationHeaders() {}",
		"generic_webhook_compat_provider.go.tmpl": "func NewGenericWebhookCompatProvider() {}",
		"providers.go.tmpl":                       "func NewLogProvider() {}",
		"email_provider.go.tmpl":                  "func NewEmailProvider() {}",
	} {
		put(t, root, base+path, text)
	}
	put(t, root, "templates/demo/frontend/src/lib/notification-providers.ts", "const webhookCompatProviderIDs: NotificationProvider[] = ['matrix'];\nconst options = [{ id: 'log' }];")
	for _, args := range [][]string{{"init", "--quiet"}, {"-c", "user.name=Chris Miles", "-c", "user.email=chris.miles.e@gmail.com", "commit", "--allow-empty", "--quiet", "-m", "test: initialize provider inventory fixture"}} {
		command := exec.Command("git", args...)
		command.Dir = root
		if data, err := command.CombinedOutput(); err != nil {
			t.Fatalf("fixture git: %v %s", err, data)
		}
	}
	return root
}

// TestInventory preserves aliases, adapter counts and hashed source provenance.
func TestInventory(t *testing.T) {
	root := fixture(t)
	checkout := demoFixture(t)
	putJSON(t, root, "docs/providers/drivers.json", map[string]any{"drivers": []Driver{{ID: "log", Package: "driver/log", Verification: "Local capture"}, {ID: "email", Package: "driver/mail", Verification: "Mail composition"}}})
	data, err := InventoryDemo(root, checkout)
	if err != nil {
		t.Fatal(err)
	}
	if data.Counts != (InventoryCounts{4, 2, 3, 2, 1}) {
		t.Fatal(data.Counts)
	}
	for _, entry := range data.Providers {
		if entry.ID == "smtp" && entry.CanonicalID != "email" {
			t.Fatal("SMTP identity lost")
		}
		if entry.ID == "matrix" && (entry.DemoImplementation != "generic_webhook_compat" || entry.LibraryStatus != "planned") {
			t.Fatal("compatibility alias promoted")
		}
	}
	if len(data.SourceCommit) != 40 {
		t.Fatal("missing Git provenance")
	}
	for _, source := range data.SourceFiles {
		if len(source.SHA256) != 64 {
			t.Fatal("missing source hash")
		}
	}
	text := InventoryMarkdown(data)
	if !strings.Contains(text, "all 4 IDs") || !strings.Contains(text, "not native provider support") {
		t.Fatal("misleading support projection")
	}
	if err := WriteInventory(root, checkout); err != nil {
		t.Fatal(err)
	}
}

// TestInventorySourceDrift fails when templates no longer establish a complete truthful inventory.
func TestInventorySourceDrift(t *testing.T) {
	for _, defect := range []string{"unknown-ui", "missing-constructor", "unknown-manager", "catalog-boundary", "manager-boundary", "ui-array"} {
		t.Run(defect, func(t *testing.T) {
			root := fixture(t)
			checkout := demoFixture(t)
			base := "templates/demo/internal/notification/"
			switch defect {
			case "unknown-ui":
				put(t, checkout, "templates/demo/frontend/src/lib/notification-providers.ts", "const webhookCompatProviderIDs = ['unknown'];")
			case "missing-constructor":
				put(t, checkout, base+"email_provider.go.tmpl", "package notification")
			case "unknown-manager":
				put(t, checkout, base+"manager.go.tmpl", "func (m *Manager) providerForChannel(){\n\tcase \"unknown\":\nreturn NewLogProvider()\n\tdefault:\n}\nfunc notificationHeaders(){}")
			case "catalog-boundary":
				put(t, checkout, base+"provider_catalog.go.tmpl", "unknown")
			case "manager-boundary":
				put(t, checkout, base+"manager.go.tmpl", "unknown")
			case "ui-array":
				put(t, checkout, "templates/demo/frontend/src/lib/notification-providers.ts", "unknown")
			}
			if _, err := InventoryDemo(root, checkout); err == nil {
				t.Fatal("source drift accepted")
			}
		})
	}
}

// TestModuleInventory verifies every module participates and all intentional root replacements survive.
func TestModuleInventory(t *testing.T) {
	root := t.TempDir()
	put(t, root, "go.mod", "module github.com/goforj/notification\n")
	put(t, root, "scripts/module-manifest.txt", ".\ndriver/mail\ndriver/apns\ndriver/sns\ndriver/webpush\nexamples/mail\nexamples/apns\nexamples/sns\nexamples/webpush\nintegration\ndocs\n")
	for name, replacement := range map[string]string{"driver/mail": "../..", "driver/apns": "../..", "driver/sns": "../..", "driver/webpush": "../..", "examples/mail": "../..", "examples/apns": "../..", "examples/sns": "../..", "examples/webpush": "../..", "integration": ".."} {
		put(t, root, name+"/go.mod", "module fixture\nreplace github.com/goforj/notification => "+replacement+"\n")
	}
	put(t, root, "docs/go.mod", "module github.com/goforj/notification/docs\n")
	if err := CheckModules(root); err != nil {
		t.Fatal(err)
	}
	put(t, root, "driver/mail/go.mod", "module fixture\nreplace github.com/goforj/notification => ../../..\n")
	if err := CheckModules(root); err == nil {
		t.Fatal("incorrect replacement prefix accepted")
	}
	put(t, root, "driver/mail/go.mod", "module fixture\nreplace github.com/goforj/notification => ../..\n")
	put(t, root, "new/go.mod", "module fixture\n")
	if err := CheckModules(root); err == nil {
		t.Fatal("uninventoried module accepted")
	}
	root = fixture(t)
	put(t, root, "scripts/module-manifest.txt", ".\nintegration\n")
	put(t, root, "integration/go.mod", "module fixture\n")
	if err := CheckModules(root); err == nil {
		t.Fatal("missing intentional root replacement accepted")
	}
}
