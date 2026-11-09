package gen

import (
	"bytes"
	"strings"
	"testing"
)

// TestPackageCoverageBadges preserves measured ratios without exposing internal implementation packages.
func TestPackageCoverageBadges(t *testing.T) {
	unit := UnitEvidence{Packages: map[string]PackageCoverage{
		"github.com/goforj/notification":                 {Covered: 4, Total: 4},
		"github.com/goforj/notification/driver/mail":     {Covered: 3, Total: 4},
		"github.com/goforj/notification/internal/native": {Covered: 1, Total: 1},
		"another/module": {Covered: 1, Total: 1},
	}}
	outputs := map[string][]byte{}
	markup, err := packageCoverageBadges(unit, outputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(outputs) != 2 || strings.Count(markup, "<img ") != 2 || strings.Contains(markup, "<br>") {
		t.Fatal("public badges are missing or forced onto a separate row", markup)
	}
	if !bytes.Contains(outputs["docs/assets/badges/coverage-driver-mail.svg"], []byte("mail: 75.0%")) || !bytes.Contains(outputs["docs/assets/badges/coverage-notification.svg"], []byte("notification: 100.0%")) {
		t.Fatal("badges do not match measured package ratios")
	}
	again, err := packageCoverageBadges(unit, map[string][]byte{})
	if err != nil || markup != again {
		t.Fatal("package badge generation is not deterministic", err)
	}
}

// TestPackageCoverageBadgesRejectInvalidCounts prevents missing evidence from being rendered as full coverage.
func TestPackageCoverageBadgesRejectInvalidCounts(t *testing.T) {
	for _, coverage := range []PackageCoverage{{}, {Covered: -1, Total: 1}, {Covered: 2, Total: 1}} {
		if _, err := packageCoverageBadges(UnitEvidence{Packages: map[string]PackageCoverage{"github.com/goforj/notification": coverage}}, map[string][]byte{}); err == nil {
			t.Fatal("invalid statement counts accepted", coverage)
		}
	}
	if _, err := packageCoverageBadges(UnitEvidence{}, map[string][]byte{}); err == nil {
		t.Fatal("missing package measurements accepted")
	}
}
