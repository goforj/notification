package gen

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// packageCoverageBadges renders public package measurements inline with the existing status badges.
func packageCoverageBadges(unit UnitEvidence, outputs map[string][]byte) (string, error) {
	const module = "github.com/goforj/notification"
	var names []string
	for name, coverage := range unit.Packages {
		if name != module && (!strings.HasPrefix(name, module+"/") || strings.HasPrefix(name, module+"/internal/")) {
			continue
		}
		if coverage.Total <= 0 || coverage.Covered < 0 || coverage.Covered > coverage.Total {
			return "", fmt.Errorf("invalid package coverage for %s", name)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return "", fmt.Errorf("missing measured public package coverage; run make test")
	}
	sort.Strings(names)
	var badges strings.Builder
	for _, name := range names {
		coverage := unit.Packages[name]
		label := path.Base(name)
		identifier := strings.ReplaceAll(strings.TrimPrefix(name, module+"/"), "/", "-")
		if name == module {
			identifier = "notification"
		}
		asset := "docs/assets/badges/coverage-" + identifier + ".svg"
		outputs[asset] = StatusBadge(label, fmt.Sprintf("%.1f%%", 100*float64(coverage.Covered)/float64(coverage.Total)), "#2e8540")
		fmt.Fprintf(&badges, "  <a href=\"docs/unit-results.json\"><img src=\"%s\" alt=\"%s unit statement coverage\"></a>\n", asset, label)
	}
	return strings.TrimSuffix(badges.String(), "\n"), nil
}
