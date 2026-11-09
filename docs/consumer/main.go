// Command consumer verifies public-module installation without local source replacements.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/goforj/notification/docs/gen"
)

// main verifies one published commit or tag without changing the checkout or creating a release.
func main() {
	revision := flag.String("revision", "", "published tag or commit to verify")
	flag.Parse()
	root, err := gen.Root()
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		err = gen.CheckConsumer(ctx, root, *revision, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
