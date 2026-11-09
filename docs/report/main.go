// Command report records evidence from executed unit and native integration tests.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/goforj/notification/docs/gen"
)

// main refuses to publish evidence from missing, failed or incomplete executions.
func main() {
	integration := flag.String("integration", "", "Go JSON event file from successful native tests")
	fingerprint := flag.Bool("fingerprint", false, "print the current Go source fingerprint")
	flag.Parse()
	root, err := gen.Root()
	if err == nil {
		switch {
		case *fingerprint:
			var value string
			value, err = gen.SourceFingerprint(root)
			if err == nil {
				fmt.Println(value)
			}
		case *integration != "":
			err = gen.ReportIntegration(root, *integration)
		default:
			var evidence gen.UnitEvidence
			evidence, err = gen.ReportUnit(root)
			if err == nil {
				fmt.Printf("Unit tests passed: %d; coverage: %d/%d statements\n", evidence.UnitTestsPassed, evidence.CoveredStatements, evidence.TotalStatements)
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
