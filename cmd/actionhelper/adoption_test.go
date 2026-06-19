package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAdoptionFeedbackIssueBackfilledAndLegacyBodies(t *testing.T) {
	backfilled := parseAdoptionFeedbackIssue(githubIssueExport{
		Number: 294,
		Title:  "backfilled",
		State:  "CLOSED",
		URL:    "https://example.test/issues/294",
		Body: `## Repository profile
Compact library

## Adoption stage
First useful report

## Repository
github.com/example/lib

## Mutation target
./...

## Install path
GitHub Action

## Primary blocker class
Signal noise or equivalent-risk survivors

## Suggested outcome
Documentation clarification

## Upstream issue or discussion
None

## External response status
No upstream thread opened

## External response last checked
2026-06-19
`,
	})
	if backfilled.RepositoryProfile != "Compact library" || backfilled.ExternalResponseStatus != "No upstream thread opened" {
		t.Fatalf("unexpected backfilled parse: %+v", backfilled)
	}
	if backfilled.HasUpstreamThread {
		t.Fatalf("expected no upstream thread in backfilled issue: %+v", backfilled)
	}
	if len(backfilled.MissingSections) != 0 {
		t.Fatalf("unexpected missing sections for backfilled issue: %+v", backfilled.MissingSections)
	}

	legacy := parseAdoptionFeedbackIssue(githubIssueExport{
		Number: 295,
		Title:  "legacy",
		State:  "CLOSED",
		URL:    "https://example.test/issues/295",
		Body: `## Repository profile
Compact library

## Adoption stage
First useful report

## Repository
github.com/example/legacy

## Mutation target
./...

## Install path
GitHub Action

## Primary blocker class
Signal noise or equivalent-risk survivors

## Suggested outcome
Documentation clarification
`,
	})
	if legacy.ExternalResponseStatus != "Unspecified" {
		t.Fatalf("expected unspecified external status for legacy issue: %+v", legacy)
	}
	if legacy.HasUpstreamThread || legacy.UpstreamThread != "" {
		t.Fatalf("expected empty upstream thread for legacy issue: %+v", legacy)
	}
	if len(legacy.MissingSections) != 0 {
		t.Fatalf("optional missing sections should not count as missing: %+v", legacy.MissingSections)
	}
}

func TestBuildAdoptionSummaryAndMarkdown(t *testing.T) {
	issuesPath := filepath.Join(t.TempDir(), "issues.json")
	writeJSONForTest(t, issuesPath, []githubIssueExport{
		{
			Number:   294,
			Title:    "[Adoption Feedback] repo a",
			State:    "CLOSED",
			URL:      "https://example.test/issues/294",
			ClosedAt: "2026-06-19T22:07:54Z",
			Body: `## Repository profile
Compact library

## Adoption stage
First useful report

## Repository
github.com/example/a

## Mutation target
./...

## Install path
GitHub Action

## Primary blocker class
Signal noise or equivalent-risk survivors

## Suggested outcome
Documentation clarification

## Upstream issue or discussion
None

## External response status
No upstream thread opened

## External response last checked
2026-06-19
`,
		},
		{
			Number:   295,
			Title:    "[Adoption Feedback] repo b",
			State:    "OPEN",
			URL:      "https://example.test/issues/295",
			ClosedAt: "",
			Body: `## Repository profile
Medium service

## Adoption stage
Pull request lane

## Repository
github.com/example/b

## Mutation target
./pkg/service

## Install path
go install

## Primary blocker class
Review UX or report usability

## Suggested outcome
Product or code change

## Upstream issue or discussion
https://github.com/example/b/issues/123

## External response status
Maintainer replied or asked follow-up questions

## External response last checked
2026-06-20
`,
		},
	})

	summary, err := buildAdoptionSummary(issuesPath, "#313", "2026-06-19T23:00:00Z")
	if err != nil {
		t.Fatalf("buildAdoptionSummary returned error: %v", err)
	}
	if summary.Aggregate.TotalIssues != 2 || summary.Aggregate.OpenIssues != 1 || summary.Aggregate.ClosedIssues != 1 {
		t.Fatalf("unexpected issue counts: %+v", summary.Aggregate)
	}
	if summary.Aggregate.IssuesWithUpstreamThread != 1 || summary.Aggregate.IssuesWithoutUpstreamThread != 1 {
		t.Fatalf("unexpected upstream thread counts: %+v", summary.Aggregate)
	}
	if summary.Aggregate.IssuesWithMaintainerReply != 1 || summary.Aggregate.IssuesWithoutMaintainerReply != 1 {
		t.Fatalf("unexpected maintainer reply counts: %+v", summary.Aggregate)
	}
	if summary.Aggregate.ExternalResponseStatuses["No upstream thread opened"] != 1 || summary.Aggregate.ExternalResponseStatuses["Maintainer replied or asked follow-up questions"] != 1 {
		t.Fatalf("unexpected external response statuses: %+v", summary.Aggregate.ExternalResponseStatuses)
	}
	markdown := renderAdoptionSummaryMarkdown(summary)
	for _, want := range []string{
		"# Adoption Feedback Summary",
		"- Issues with upstream thread: **1**",
		"- Issues with maintainer reply: **1**",
		"## #294 [Adoption Feedback] repo a",
		"## #295 [Adoption Feedback] repo b",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("summary markdown missing %q:\n%s", want, markdown)
		}
	}
}

func TestAdoptionHelperCommandsWriteJSONAndMarkdown(t *testing.T) {
	issuesPath := filepath.Join(t.TempDir(), "issues.json")
	writeJSONForTest(t, issuesPath, []githubIssueExport{
		{
			Number: 294,
			Title:  "[Adoption Feedback] repo a",
			State:  "CLOSED",
			URL:    "https://example.test/issues/294",
			Body: `## Repository profile
Compact library

## Adoption stage
First useful report

## Repository
github.com/example/a

## Mutation target
./...

## Install path
GitHub Action

## Primary blocker class
Signal noise or equivalent-risk survivors

## Suggested outcome
Documentation clarification

## Upstream issue or discussion
None

## External response status
No upstream thread opened

## External response last checked
2026-06-19
`,
		},
	})

	var jsonOut bytes.Buffer
	if err := cmdBuildAdoptionSummary([]string{
		"--issues-json", issuesPath,
		"--tracking-issue", "#313",
		"--generated-at", "2026-06-19T23:05:00Z",
	}, &jsonOut); err != nil {
		t.Fatalf("cmdBuildAdoptionSummary returned error: %v", err)
	}
	var summary adoptionSummary
	if err := json.Unmarshal(jsonOut.Bytes(), &summary); err != nil {
		t.Fatalf("cmdBuildAdoptionSummary did not emit valid JSON: %v", err)
	}
	if summary.Aggregate.TotalIssues != 1 {
		t.Fatalf("unexpected command summary output: %+v", summary.Aggregate)
	}

	summaryPath := filepath.Join(t.TempDir(), "adoption-summary.json")
	writeJSONForTest(t, summaryPath, summary)
	var markdownOut bytes.Buffer
	if err := cmdRenderAdoptionSummaryMarkdown([]string{"--path", summaryPath}, &markdownOut); err != nil {
		t.Fatalf("cmdRenderAdoptionSummaryMarkdown returned error: %v", err)
	}
	if !strings.Contains(markdownOut.String(), "## #294 [Adoption Feedback] repo a") {
		t.Fatalf("adoption summary markdown missing issue heading:\n%s", markdownOut.String())
	}
}
