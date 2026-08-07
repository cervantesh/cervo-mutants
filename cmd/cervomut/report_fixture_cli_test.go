package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/pkg/report"
	"github.com/cervantesh/cervo-mutants/pkg/report/reporttest"
)

// TestReportCommandsFromCanonicalFixture proves CLI report paths can load the
// shared public-report fixture without running the mutation engine (#357).
func TestReportCommandsFromCanonicalFixture(t *testing.T) {
	out := t.TempDir()
	fixture := reporttest.CanonicalPublicReport()
	data, err := report.JSON(fixture)
	if err != nil {
		t.Fatalf("report.JSON: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, mutationReportFileName), data, 0o644); err != nil {
		t.Fatalf("write mutation-report.json: %v", err)
	}

	summary := captureStdout(t, func() {
		if err := run([]string{"report", "summary", "--out", out}); err != nil {
			t.Fatalf("report summary: %v", err)
		}
	})
	for _, want := range []string{
		"Effective mutation score: 33.33%",
		"Raw mutation score: 50.00%",
		"Survived: 2",
		"Actionable survivors: 1",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("report summary missing %q:\n%s", want, summary)
		}
	}

	survivors := captureStdout(t, func() {
		if err := run([]string{"report", "survivors", "--out", out}); err != nil {
			t.Fatalf("report survivors: %v", err)
		}
	})
	for _, want := range []string{
		"pkg/review.go",
		"conditionals-boundary",
	} {
		if !strings.Contains(survivors, want) {
			t.Fatalf("report survivors missing %q:\n%s", want, survivors)
		}
	}
}
