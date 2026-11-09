package gen

import (
	"fmt"
	"html"
	"strings"
)

// featureOrder keeps the opening table centered on the common send workflow.
var featureOrder = []string{"intent", "policy", "channels", "identity", "diagnostics", "testing"}

// featuresTable gives icons true middle alignment and keeps each searchable name on one line.
func featuresTable(features map[string]Presentation) (string, error) {
	var out strings.Builder
	out.WriteString("<table>\n<thead><tr><th align=\"left\" width=\"220\">Feature</th><th align=\"left\">What you get</th></tr></thead>\n<tbody>\n")
	rendered := 0
	for _, name := range featureOrder {
		feature, ok := features[name]
		if !ok {
			continue
		}
		if feature.Description == "" {
			return "", fmt.Errorf("missing feature description: %s", name)
		}
		fmt.Fprintf(&out, "<tr><td width=\"220\" nowrap><img src=\"docs/assets/badges/%s.svg\" width=\"22\" height=\"22\" alt=\"\" align=\"absmiddle\"> <strong>%s</strong></td><td>%s</td></tr>\n", name, html.EscapeString(feature.Label), html.EscapeString(feature.Description))
		rendered++
	}
	if rendered != len(features) {
		return "", fmt.Errorf("unknown feature presentation")
	}
	out.WriteString("</tbody>\n</table>")
	return out.String(), nil
}

// acceptanceSummary keeps the overview short while preserving the full native contract in the guide.
func acceptanceSummary(driver Driver) string {
	if driver.AcceptanceSummary != "" {
		return driver.AcceptanceSummary
	}
	return driver.Acceptance
}

// verificationSummary distinguishes protocol fixtures from real servers without repeating the legend.
func verificationSummary(driver Driver) string {
	switch {
	case strings.HasPrefix(driver.Verification, "Real "):
		return strings.TrimSuffix(strings.TrimPrefix(driver.Verification, "Real "), " server verified")
	case strings.Contains(driver.Verification, "Mail composition"):
		return "Composition tests"
	case strings.Contains(driver.Verification, "Local capture"):
		return "Local capture"
	case strings.Contains(driver.Verification, "Local protocol"):
		return "Protocol tests"
	default:
		return driver.Verification
	}
}

// driverGuide retains every configuration, native acknowledgement and verification detail behind the overview.
func driverGuide(drivers []Driver, presentations map[string]Presentation) string {
	var out strings.Builder
	out.WriteString("# Driver configuration and delivery contracts\n\nGenerated from the driver catalog by `go -C docs run ./readme`. See [portable delivery contracts](contracts.md) and [executed integration evidence](integration-results.json). Provider acceptance never proves recipient delivery.\n")
	for _, driver := range drivers {
		fmt.Fprintf(&out, "\n## %s\n\n[Package source](../%s)\n\n- Configuration: %s.\n- Accepted when: %s.\n- Verification: %s.\n", presentations[driver.ID].Label, driver.Package, driver.Configuration, driver.Acceptance, driver.Verification)
		if driver.ProtocolSource != "" {
			fmt.Fprintf(&out, "- [Provider protocol reference](%s).\n", driver.ProtocolSource)
		}
	}
	return out.String()
}
