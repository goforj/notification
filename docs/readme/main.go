// Command readme generates checked README examples, API links and local badges.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/goforj/notification/docs/gen"
)

// main fails visibly when source, examples or verification evidence have drifted.
func main() {
	check := flag.Bool("check", false, "verify generated files without writing")
	flag.Parse()
	root, err := gen.Root()
	if err == nil {
		err = gen.Generate(root, *check)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
