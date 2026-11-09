// Package gen projects documentation from Go declarations, executable examples and measured evidence.
package gen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Root locates the library without selecting the documentation tooling module.
func Root() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return rootFrom(directory)
}

// rootFrom walks upward to the authoritative module declaration.
func rootFrom(directory string) (string, error) {
	for {
		data, err := os.ReadFile(filepath.Join(directory, "go.mod"))
		if err == nil && strings.Contains(string(data), "module github.com/goforj/notification\n") {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("notification module root not found")
		}
		directory = parent
	}
}

// SourceFingerprint binds evidence to every Go source, test and module requirement.
func SourceFingerprint(root string) (string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".artifacts" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" || entry.Name() == "go.mod" || entry.Name() == "go.sum" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	digest := sha256.New()
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		digest.Write([]byte(filepath.ToSlash(relative)))
		digest.Write([]byte{0})
		digest.Write(data)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// Example retains the single marked snippet and its actual selector calls.
type Example struct {
	Name    string
	Snippet string
	Calls   map[string]bool
}

// Examples discovers the same authoritative programs executed by the shared runner.
func Examples(root string) ([]Example, error) {
	paths, err := filepath.Glob(filepath.Join(root, "examples", "*", "main.go"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("missing runnable examples")
	}
	var examples []Example
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, data, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		code, err := Snippet(string(data))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		file, err = parser.ParseFile(token.NewFileSet(), path, "package main\nfunc example() {\n"+code+"\n}", 0)
		if err != nil {
			return nil, err
		}
		calls := map[string]bool{}
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
					calls[selector.Sel.Name] = true
				}
			}
			return true
		})
		examples = append(examples, Example{filepath.Base(filepath.Dir(path)), code, calls})
	}
	return examples, nil
}

// Snippet extracts the tested region without maintaining another workflow implementation.
func Snippet(source string) (string, error) {
	const start = "// snippet:start"
	const end = "// snippet:end"
	if strings.Count(source, start) != 1 || strings.Count(source, end) != 1 {
		return "", fmt.Errorf("expected exactly one snippet region")
	}
	_, region, _ := strings.Cut(source, start)
	region, _, found := strings.Cut(region, end)
	if !found || !strings.Contains(region, "fmt.Println(") {
		return "", fmt.Errorf("snippet needs adjacent expected output")
	}
	lines := strings.Split(strings.Trim(region, "\n"), "\n")
	indent := -1
	output := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		size := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent < 0 || size < indent {
			indent = size
		}
		if strings.Contains(line, "fmt.Println(") && index+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[index+1]), "// ") {
			output = true
		}
	}
	if !output {
		return "", fmt.Errorf("snippet needs adjacent expected output")
	}
	for index, line := range lines {
		if len(strings.TrimSpace(line)) > 0 {
			lines[index] = line[indent:]
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

// HasSymbol validates an indexed operation against its documented Go declaration.
func HasSymbol(root, directory, symbol string) error {
	packages, err := parser.ParseDir(token.NewFileSet(), filepath.Join(root, directory), func(info fs.FileInfo) bool { return !strings.HasSuffix(info.Name(), "_test.go") }, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				function, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				name := function.Name.Name
				if function.Recv != nil {
					name = receiverName(function.Recv.List[0].Type) + "." + name
				}
				if name == symbol {
					if function.Doc == nil {
						return fmt.Errorf("missing GoDoc for %s", symbol)
					}
					return nil
				}
			}
		}
	}
	return fmt.Errorf("unknown API %s in %s", symbol, directory)
}

// receiverName preserves receiver ownership through pointer and generic syntax.
func receiverName(expression ast.Expr) string {
	switch node := expression.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.StarExpr:
		return receiverName(node.X)
	case *ast.IndexExpr:
		return receiverName(node.X)
	case *ast.IndexListExpr:
		return receiverName(node.X)
	}
	return ""
}
