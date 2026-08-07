package report

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
	"github.com/cervantesh/cervo-mutants/pkg/report/reporttest"
)

// fixedGovernanceClock freezes governance expiry timestamps so goldens do not
// depend on wall clock. Public GovernanceReview* APIs still use time.Now().
var fixedGovernanceClock = time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

func TestGoldenPublicReportFormats(t *testing.T) {
	run := reporttest.CanonicalPublicReport()

	jsonData, err := JSON(run)
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}
	junitData, err := JUnit(run)
	if err != nil {
		t.Fatalf("JUnit returned error: %v", err)
	}
	sarifData, err := SARIF(run)
	if err != nil {
		t.Fatalf("SARIF returned error: %v", err)
	}
	governanceJSON, err := governanceReviewJSONAt(run, fixedGovernanceClock)
	if err != nil {
		t.Fatalf("governance JSON returned error: %v", err)
	}
	historyJSON, err := HistoryDashboardJSON(run)
	if err != nil {
		t.Fatalf("history dashboard JSON returned error: %v", err)
	}

	cases := []struct {
		name string
		path string
		data []byte
	}{
		{name: "json", path: filepath.Join("testdata", "public-report.json.golden"), data: jsonData},
		{name: "junit", path: filepath.Join("testdata", "public-report.junit.xml.golden"), data: junitData},
		{name: "html", path: filepath.Join("testdata", "public-report.index.html.golden"), data: []byte(HTML(run))},
		{name: "sarif", path: filepath.Join("testdata", "public-report.sarif.golden"), data: sarifData},
		{name: "github-summary", path: filepath.Join("testdata", "public-report.github-summary.md.golden"), data: []byte(GitHubSummary(run))},
		{name: "test-recommendations", path: filepath.Join("testdata", "public-report.test-recommendations.md.golden"), data: []byte(TestRecommendations(run))},
		// #353–#356: freeze remaining public text/dashboard surfaces against the same fixture.
		{name: "summary", path: filepath.Join("testdata", "public-report.summary.txt.golden"), data: []byte(Summary(run))},
		{name: "survivors", path: filepath.Join("testdata", "public-report.survivors.txt.golden"), data: []byte(Survivors(run))},
		{name: "survivors-actionable", path: filepath.Join("testdata", "public-report.survivors-actionable.txt.golden"), data: []byte(SurvivorsWithOptions(run, SurvivorsOptions{ActionableOnly: true}))},
		{name: "governance-md", path: filepath.Join("testdata", "public-report.governance-review.md.golden"), data: []byte(governanceReviewMarkdownAt(run, fixedGovernanceClock))},
		{name: "governance-json", path: filepath.Join("testdata", "public-report.governance-review.json.golden"), data: governanceJSON},
		{name: "history-json", path: filepath.Join("testdata", "public-report.history-dashboard.json.golden"), data: historyJSON},
		{name: "history-html", path: filepath.Join("testdata", "public-report.history-dashboard.html.golden"), data: []byte(HistoryDashboardHTML(run))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reporttest.AssertGoldenBytes(t, tc.path, tc.data)
		})
	}
}

// goldenReportFixture is kept as a package-local alias so existing report_test
// call sites remain readable while the canonical definition lives in reporttest.
func goldenReportFixture() engine.RunResult {
	return reporttest.CanonicalPublicReport()
}

func TestGoldenFixtureJSONIsStable(t *testing.T) {
	run := reporttest.CanonicalPublicReport()
	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("fixture should be valid JSON: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("fixture JSON should not be empty")
	}
}
