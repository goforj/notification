package gen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

// constructorExample extracts the actual driver call from its executable program rather than maintaining another configuration.
func constructorExample(root string, driver Driver) (API, Example, error) {
	program := "providers"
	switch driver.Package {
	case "driver/log":
		program = "quickstart"
	case "driver/webhook":
		program = "webhook"
	case "driver/slack":
		program = "transport"
	case "driver/pagerduty":
		program = "incident"
	case "driver/mail", "driver/apns", "driver/sns", "driver/webpush":
		program = filepath.Base(driver.Package)
	}
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, filepath.Join(root, "examples", program, "main.go"), nil, 0)
	if err != nil {
		return API{}, Example{}, err
	}
	name := ""
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err == nil && path == "github.com/goforj/notification/"+driver.Package {
			name = filepath.Base(driver.Package)
			if imported.Name != nil {
				name = imported.Name.Name
			}
		}
	}
	var constructor *ast.CallExpr
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || constructor != nil {
			return constructor == nil
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "New" {
			return true
		}
		owner, ok := selector.X.(*ast.Ident)
		if ok && owner.Name == name {
			constructor = call
		}
		return constructor == nil
	})
	if constructor == nil {
		return API{}, Example{}, fmt.Errorf("%s.New is missing from executable example %s", driver.Package, program)
	}
	var source bytes.Buffer
	if err := format.Node(&source, positions, constructor); err != nil {
		return API{}, Example{}, err
	}
	label := name + ".New"
	example := Example{Name: "constructor:" + driver.Package, Snippet: source.String(), Calls: map[string]bool{"New": true}}
	return API{label, driver.Package, "New", example.Name, label, "Driver constructors"}, example, nil
}

// constructorCatalog keeps every supported driver's navigation and configuration bound to executable source.
func constructorCatalog(root string, drivers []Driver) ([]API, map[string]Example, string, error) {
	var apis []API
	examples := map[string]Example{}
	var content strings.Builder
	for _, driver := range drivers {
		api, example, err := constructorExample(root, driver)
		if err != nil {
			return nil, nil, "", err
		}
		apis = append(apis, api)
		examples[example.Name] = example
		fmt.Fprintf(&content, "### %s\n\n```go\n%s\n```\n\n", api.Heading, example.Snippet)
	}
	return apis, examples, strings.TrimSpace(content.String()), nil
}
