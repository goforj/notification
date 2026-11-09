package examples

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPrograms discovers and executes the authoritative example programs.
// Child binaries preserve race and coverage instrumentation from the test run.
func TestPrograms(t *testing.T) {
	paths, err := filepath.Glob("*/main.go")
	if err != nil || len(paths) == 0 {
		t.Fatal("missing runnable examples")
	}
	for _, path := range paths {
		t.Run(filepath.Dir(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var expected []string
			lines := strings.Split(string(source), "\n")
			for i, line := range lines {
				if strings.Contains(line, "fmt.Println(") {
					for j := i + 1; j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "// "); j++ {
						line := strings.TrimPrefix(strings.TrimSpace(lines[j]), "// ")
						if strings.HasPrefix(line, "snippet:") {
							break
						}
						expected = append(expected, line)
					}
				}
			}
			if len(expected) == 0 {
				t.Fatal("missing inline expected output")
			}
			bin := filepath.Join(t.TempDir(), "example")
			args := []string{"build"}
			if raceEnabled {
				args = append(args, "-race")
			}
			if os.Getenv("NOTIFICATION_EXAMPLE_COVERAGE") != "" {
				args = append(args, "-cover", "-covermode=atomic", "-coverpkg=github.com/goforj/notification/...")
			}
			target := "./" + filepath.Dir(path)
			nested := false
			if _, err := os.Stat(filepath.Join(filepath.Dir(path), "go.mod")); err == nil {
				target = "."
				nested = true
			}
			args = append(args, "-o", bin, target)
			build := exec.Command("go", args...)
			if nested {
				build.Dir = filepath.Dir(path)
			}
			if output, err := build.CombinedOutput(); err != nil {
				t.Fatalf("build: %v\n%s", err, output)
			}
			run := exec.Command(bin)
			if directory := os.Getenv("NOTIFICATION_EXAMPLE_COVERAGE"); directory != "" {
				run.Env = append(os.Environ(), "GOCOVERDIR="+directory)
			}
			out, err := run.CombinedOutput()
			if err != nil {
				t.Fatalf("run: %v\n%s", err, out)
			}
			want := strings.Join(expected, "\n") + "\n"
			if !bytes.Equal(out, []byte(want)) {
				t.Fatalf("output mismatch:\n%s", out)
			}
		})
	}
}
