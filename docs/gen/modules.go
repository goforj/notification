package gen

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// CheckModules verifies the complete module inventory and intentional sibling replacements.
func CheckModules(root string) error {
	expected, err := Modules(root)
	if err != nil {
		return err
	}
	sort.Strings(expected)
	var actual []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".artifacts" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == "go.mod" {
			relative, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}
			actual = append(actual, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(actual)
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		return fmt.Errorf("module manifest mismatch: expected %v, found %v", expected, actual)
	}
	for _, name := range actual {
		if name != "integration" && !strings.HasPrefix(name, "driver/") && !strings.HasPrefix(name, "examples/") {
			continue
		}
		relative, err := filepath.Rel(filepath.Join(root, name), root)
		if err != nil {
			return err
		}
		replacement := filepath.ToSlash(relative)
		data, err := os.ReadFile(filepath.Join(root, name, "go.mod"))
		if err != nil {
			return err
		}
		if !regexp.MustCompile(`(?m)^replace\s+github\.com/goforj/notification\s+=>\s+` + regexp.QuoteMeta(replacement) + `\s*$`).Match(data) {
			return fmt.Errorf("%s: unpublished root replacement missing", name)
		}
	}
	return nil
}
