package report

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

type renderedArtifacts map[string][]byte

type postWriteSink interface {
	Run() error
}

type githubStepSummarySink struct {
	markdown string
}

func (s githubStepSummarySink) Run() error {
	return writeGitHubStepSummary(s.markdown)
}

type writePlan struct {
	artifacts renderedArtifacts
	sinks     []postWriteSink
}

func WriteAll(dir string, result engine.RunResult) error {
	return WriteFormats(dir, result, []string{"summary", "json", "junit", "html", "sarif", "github-summary"})
}

func WriteFormats(dir string, result engine.RunResult, formats []string) error {
	return WriteFormatsWithOptions(dir, result, formats, WriteOptions{})
}

func WriteFormatsWithOptions(dir string, result engine.RunResult, formats []string, opts WriteOptions) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	plan, err := renderRequestedFormats(result, formats, opts)
	if err != nil {
		return err
	}
	if err := writeArtifacts(dir, plan.artifacts); err != nil {
		return err
	}
	return runPostWriteSinks(plan.sinks)
}

func renderRequestedFormats(result engine.RunResult, formats []string, opts WriteOptions) (writePlan, error) {
	if len(formats) == 0 {
		formats = []string{"summary", "json"}
	}
	plan := writePlan{artifacts: renderedArtifacts{}}
	for _, format := range formats {
		if err := renderFormat(plan.artifacts, &plan.sinks, result, strings.TrimSpace(format)); err != nil {
			return writePlan{}, err
		}
	}
	if err := appendSupplementalArtifacts(plan.artifacts, result, opts); err != nil {
		return writePlan{}, err
	}
	return plan, nil
}

func renderFormat(artifacts renderedArtifacts, sinks *[]postWriteSink, result engine.RunResult, format string) error {
	switch format {
	case "summary":
		artifacts["summary.txt"] = []byte(Summary(result))
		artifacts["survivors.txt"] = []byte(Survivors(result))
	case "json":
		jsonData, err := JSON(result)
		if err != nil {
			return err
		}
		artifacts["mutation-report.json"] = jsonData
	case "junit":
		junitData, err := JUnit(result)
		if err != nil {
			return err
		}
		artifacts["junit.xml"] = junitData
	case "html":
		artifacts["index.html"] = []byte(HTML(result))
	case "sarif":
		sarifData, err := SARIF(result)
		if err != nil {
			return err
		}
		artifacts["mutation-report.sarif"] = sarifData
	case "github-summary":
		summary := GitHubSummary(result)
		artifacts["github-summary.md"] = []byte(summary)
		*sinks = append(*sinks, githubStepSummarySink{markdown: summary})
	}
	return nil
}

func appendSupplementalArtifacts(artifacts renderedArtifacts, result engine.RunResult, opts WriteOptions) error {
	if opts.ActionableOnly {
		artifacts["survivors-actionable.txt"] = []byte(SurvivorsWithOptions(result, SurvivorsOptions{ActionableOnly: true}))
	}
	artifacts["test-recommendations.md"] = []byte(TestRecommendations(result))
	artifacts["governance-review.md"] = []byte(GovernanceReviewMarkdown(result))
	governanceJSON, err := GovernanceReviewJSON(result)
	if err != nil {
		return err
	}
	artifacts["governance-review.json"] = governanceJSON
	historyJSON, err := HistoryDashboardJSON(result)
	if err != nil {
		return err
	}
	artifacts["history-dashboard.json"] = historyJSON
	artifacts["history-dashboard.html"] = []byte(HistoryDashboardHTML(result))
	ledgerData, err := SemanticTriageLedger(result)
	if err != nil {
		return err
	}
	artifacts["semantic-triage-ledger.json"] = ledgerData
	return nil
}

func writeArtifacts(dir string, artifacts renderedArtifacts) error {
	for name, data := range artifacts {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func runPostWriteSinks(sinks []postWriteSink) error {
	for _, sink := range sinks {
		if err := sink.Run(); err != nil {
			return err
		}
	}
	return nil
}
