package gen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// CheckConsumer resolves every runtime module from the public proxy and runs an independent application.
// Revision is a published tag or commit. Local replacements and module caches cannot satisfy this check.
func CheckConsumer(ctx context.Context, root, revision string, output io.Writer) error {
	if !regexp.MustCompile(`^([a-f0-9]{7,40}|v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?)$`).MatchString(revision) {
		return fmt.Errorf("revision must be a published tag or commit")
	}
	modules, err := consumerModules(root)
	if err != nil {
		return err
	}
	var catalog struct {
		Drivers []Driver `json:"drivers"`
	}
	if err := readJSON(root, "docs/providers/drivers.json", &catalog); err != nil {
		return err
	}

	if _, err := consumerSource(catalog.Drivers); err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "notification-consumer-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	environment := consumerEnvironment(filepath.Join(directory, "modcache"))
	for index, phase := range consumerPhases(modules, catalog.Drivers) {
		target := filepath.Join(directory, fmt.Sprintf("consumer-%d", index))
		if err := os.Mkdir(target, 0700); err != nil {
			return err
		}
		if err := checkConsumerAt(ctx, target, revision, phase, environment, output); err != nil {
			return err
		}
	}
	return nil
}

// consumerPhase separates a full composition from each independently resolved runtime module.
type consumerPhase struct {
	modules  []string
	required []string
	drivers  []Driver
}

// consumerPhases prevents the combined consumer's root requirement from masking an adapter's invalid minimum.
func consumerPhases(modules []string, drivers []Driver) []consumerPhase {
	phases := []consumerPhase{{modules, modules, drivers}}
	var rootDrivers []Driver
	for _, driver := range drivers {
		optional := false
		for _, module := range modules[1:] {
			if module == modules[0]+"/"+driver.Package {
				optional = true
				break
			}
		}
		if !optional {
			rootDrivers = append(rootDrivers, driver)
		}
	}
	phases = append(phases, consumerPhase{modules: modules[:1], required: modules[:1], drivers: rootDrivers})
	for _, module := range modules[1:] {
		phases = append(phases, consumerPhase{modules: []string{module}, required: []string{modules[0], module}, drivers: []Driver{{Package: strings.TrimPrefix(module, modules[0]+"/")}}})
	}
	return phases
}

// checkConsumerAt exercises one public selection without carrying module requirements between phases.
func checkConsumerAt(ctx context.Context, directory, revision string, phase consumerPhase, environment []string, output io.Writer) error {
	source, err := consumerSource(phase.drivers)
	if err != nil {
		return err
	}
	for name, data := range map[string][]byte{"go.mod": []byte("module notification-consumer\n\ngo 1.24.0\n"), "main.go": source} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
			return err
		}
	}
	goCommand := filepath.Join(runtime.GOROOT(), "bin", "go")
	run := func(stdout io.Writer, args ...string) error {
		command := exec.CommandContext(ctx, goCommand, args...)
		command.Dir = directory
		command.Env = environment
		command.Stdout = stdout
		command.Stderr = output
		return command.Run()
	}
	arguments := []string{"get"}
	for _, module := range phase.modules {
		arguments = append(arguments, module+"@"+revision)
	}
	fmt.Fprintf(output, "Resolving %s with public checksum verification...\n", strings.Join(phase.modules, ", "))
	if err := run(output, arguments...); err != nil {
		return fmt.Errorf("public module resolution failed: %w", err)
	}
	var selected bytes.Buffer
	if err := run(&selected, "list", "-m", "-json", "all"); err != nil {
		return err
	}
	if err := checkConsumerSelection(selected.Bytes(), phase.required); err != nil {
		return err
	}
	fmt.Fprintf(output, "Compiling %d drivers and exercising Sender...\n", len(phase.drivers))
	return run(output, "run", ".")
}

// consumerModules follows the shared inventory rather than maintaining a second optional-driver list.
func consumerModules(root string) ([]string, error) {
	if err := CheckModules(root); err != nil {
		return nil, err
	}
	inventory, err := Modules(root)
	if err != nil {
		return nil, err
	}
	const base = "github.com/goforj/notification"
	modules := []string{base}
	for _, name := range inventory {
		if !strings.HasPrefix(name, "driver/") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, name, "go.mod"))
		if err != nil {
			return nil, err
		}
		module := base + "/" + name
		if !strings.Contains(string(data), "module "+module+"\n") {
			return nil, fmt.Errorf("unexpected runtime module declaration: %s", name)
		}
		modules = append(modules, module)
	}
	return modules, nil
}

// consumerSource compiles the complete driver catalog while checking the small public application API.
func consumerSource(drivers []Driver) ([]byte, error) {
	if len(drivers) == 0 {
		return nil, fmt.Errorf("driver catalog is empty")
	}
	var source strings.Builder
	source.WriteString("package main\nimport (\n\"context\"\n\"fmt\"\n\"github.com/goforj/notification\"\n")
	seen := map[string]bool{}
	for _, driver := range drivers {
		if !regexp.MustCompile(`^driver/[a-z][a-z0-9]*$`).MatchString(driver.Package) || seen[driver.Package] {
			return nil, fmt.Errorf("invalid or duplicate driver package: %s", driver.Package)
		}
		seen[driver.Package] = true
		source.WriteString("_ " + strconv.Quote("github.com/goforj/notification/"+driver.Package) + "\n")
	}
	source.WriteString(`)
// main checks the public API without contacting a notification provider.
func main(){
 calls:=0
 sender:=notification.New(notification.SenderFunc(func(ctx context.Context,message notification.Message)error{
  if ctx==nil || message.Title!="Release check" || message.Body!="All drivers compile" {return fmt.Errorf("public message contract changed")}
  calls++
  return nil
 }))
 if err:=sender.Send(nil,notification.Message{Title:"Release check",Body:"All drivers compile"});err!=nil{panic(err)}
 if calls!=1{panic("unexpected send count")}
 fmt.Println("Public consumer verified")
 // Public consumer verified
}
`)
	return format.Source([]byte(source.String()))
}

// consumerEnvironment isolates resolution from workspaces, private-module settings and previous candidate archives.
func consumerEnvironment(cache string) []string {
	overrides := map[string]string{"GOWORK": "off", "GOENV": "off", "GOFLAGS": "-modcacherw", "GOROOT": runtime.GOROOT(), "GOTOOLCHAIN": "local", "GOMODCACHE": cache, "GOPROXY": "https://proxy.golang.org", "GOSUMDB": "sum.golang.org", "GOPRIVATE": "", "GONOPROXY": "none", "GONOSUMDB": "none"}
	var environment []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if _, replace := overrides[name]; !replace {
			environment = append(environment, entry)
		}
	}
	var names []string
	for name := range overrides {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		environment = append(environment, name+"="+overrides[name])
	}
	return environment
}

// checkConsumerSelection rejects missing published-module selections and all replacement resolution.
func checkConsumerSelection(data []byte, required []string) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	selected := map[string]bool{}
	for {
		var module struct {
			Path, Version string
			Replace       *json.RawMessage
		}
		err := decoder.Decode(&module)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if module.Replace != nil {
			return fmt.Errorf("consumer resolved a replacement: %s", module.Path)
		}
		if module.Version != "" {
			selected[module.Path] = true
		}
	}
	for _, module := range required {
		if !selected[module] {
			return fmt.Errorf("consumer did not resolve a published version of %s", module)
		}
	}
	return nil
}
