// Command inventory snapshots Uptime Gopher provider IDs from an explicit framework checkout.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goforj/notification/docs/gen"
)

// main keeps research refreshes separate from ordinary documentation generation.
func main() {
	checkout := flag.String("goforj", "", "framework checkout, resolved relative to the notification root")
	flag.Parse()
	root, err := gen.Root()
	if err == nil && *checkout == "" {
		err = fmt.Errorf("-goforj checkout is required")
	}
	if err == nil {
		path := *checkout
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		err = gen.WriteInventory(root, path)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
