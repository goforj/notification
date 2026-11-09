package gen

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

// TestConsumerSource requires a complete, valid driver catalog and an executable small-API smoke check.
func TestConsumerSource(t *testing.T) {
	for _, drivers := range [][]Driver{nil, {{Package: "../private"}}, {{Package: "driver/mail"}, {Package: "driver/mail"}}} {
		if _, err := consumerSource(drivers); err == nil {
			t.Fatal("invalid catalog accepted")
		}
	}
	source, err := consumerSource([]Driver{{Package: "driver/mail"}, {Package: "driver/sns"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`_ "github.com/goforj/notification/driver/mail"`, `_ "github.com/goforj/notification/driver/sns"`, `notification.SenderFunc(`, `sender.Send(nil,`, `calls != 1`} {
		if !bytes.Contains(source, []byte(want)) {
			t.Fatal("consumer contract absent", want)
		}
	}
}

// TestConsumerSelection refuses missing module versions or replacement resolution after public downloads.
func TestConsumerSelection(t *testing.T) {
	const module = "github.com/goforj/notification"
	for _, data := range []string{`not json`, ``, `{"Path":"` + module + `"}`, `{"Path":"` + module + `","Version":"v0.1.0","Replace":{"Dir":"/tmp/local"}}`} {
		if err := checkConsumerSelection([]byte(data), []string{module}); err == nil {
			t.Fatal("invalid resolution accepted", data)
		}
	}
	data := `{"Path":"consumer","Main":true}` + "\n" + `{"Path":"` + module + `","Version":"v0.0.0-20261109213702-eafcce898c23"}`
	if err := checkConsumerSelection([]byte(data), []string{module}); err != nil {
		t.Fatal(err)
	}
}

// TestConsumerEnvironment keeps checksum and public-proxy validation independent of the invoking shell.
func TestConsumerEnvironment(t *testing.T) {
	t.Setenv("GOWORK", "/tmp/hidden-workspace")
	t.Setenv("GONOSUMDB", "*")
	t.Setenv("GOMODCACHE", "/tmp/old-cache")
	t.Setenv("GOFLAGS", "-overlay=/tmp/local.json")
	seen := map[string]string{}
	for _, entry := range consumerEnvironment("/tmp/fresh-cache") {
		name, value, _ := strings.Cut(entry, "=")
		if _, exists := seen[name]; exists {
			t.Fatal("duplicate process variable", name)
		}
		seen[name] = value
	}
	for name, want := range map[string]string{"GOWORK": "off", "GOENV": "off", "GOFLAGS": "-modcacherw", "GONOSUMDB": "none", "GOMODCACHE": "/tmp/fresh-cache", "GOPROXY": "https://proxy.golang.org", "GOSUMDB": "sum.golang.org", "GONOPROXY": "none", "GOTOOLCHAIN": "local"} {
		if seen[name] != want {
			t.Fatal("consumer isolation failed", name)
		}
	}
	if os.Getenv("GONOSUMDB") != "*" {
		t.Fatal("parent environment changed")
	}
}

// TestConsumerModuleInventory follows optional modules and rejects mismatched path declarations before network access.
func TestConsumerModuleInventory(t *testing.T) {
	root := fixture(t)
	put(t, root, "scripts/module-manifest.txt", ".\ndriver/mail\n")
	put(t, root, "driver/mail/go.mod", "module wrong/path\nreplace github.com/goforj/notification => ../..\n")
	if _, err := consumerModules(root); err == nil {
		t.Fatal("wrong runtime module path accepted")
	}
	put(t, root, "driver/mail/go.mod", "module github.com/goforj/notification/driver/mail\nreplace github.com/goforj/notification => ../..\n")
	modules, err := consumerModules(root)
	if err != nil || len(modules) != 2 || modules[1] != "github.com/goforj/notification/driver/mail" {
		t.Fatal(modules, err)
	}
	put(t, root, "scripts/module-manifest.txt", ".\n")
	if err := CheckConsumer(context.Background(), root, "eafcce8", new(bytes.Buffer)); err == nil {
		t.Fatal("incomplete inventory accepted")
	}
	for _, revision := range []string{"", "main", "v1.2", "-revision", "deadbeef;touch /tmp/invalid"} {
		if err := CheckConsumer(context.Background(), root, revision, new(bytes.Buffer)); err == nil {
			t.Fatal("invalid revision accepted")
		}
	}
	put(t, root, "scripts/module-manifest.txt", ".\ndriver/mail\n")
	put(t, root, "docs/providers/drivers.json", "not json")
	if err := CheckConsumer(context.Background(), root, "eafcce8", new(bytes.Buffer)); err == nil {
		t.Fatal("invalid catalog JSON accepted")
	}
	putJSON(t, root, "docs/providers/drivers.json", map[string]any{"drivers": []Driver{}})
	if err := CheckConsumer(context.Background(), root, "eafcce8", new(bytes.Buffer)); err == nil {
		t.Fatal("empty catalog accepted")
	}
}
