// Command modules verifies the complete module inventory and development replacements.
package main

import (
	"fmt"
	"os"

	"github.com/goforj/notification/docs/gen"
)

// main reports drift before repository validation silently omits a nested module.
func main() {
	root, err := gen.Root()
	if err == nil {
		err = gen.CheckModules(root)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Module inventory and local root replacements verified")
}
