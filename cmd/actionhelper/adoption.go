package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

type githubIssueExport struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	State    string `json:"state"`
	URL      string `json:"url"`
	ClosedAt string `json:"closedAt,omitempty"`
	Body     string `json:"body"`
}

type adoptionSummary struct {
	SchemaVersion string                  `json:"schema_version"`
	TrackingIssue string                  `json:"tracking_issue,omitempty"`
	SourcePath    string                  `json:"source_path,omitempty"`
	GeneratedAt   string                  `json:"generated_at"`
	Issues        []adoptionFeedbackIssue `json:"issues"`
	Aggregate     adoptionAggregate       `json:"aggregate"`
}

type adoptionFeedbackIssue struct {
	Number                    int      `json:"number"`
	Title                     string   `json:"title"`
	State                     string   `json:"state"`
	URL                       string   `json:"url"`
	ClosedAt                  string   `json:"closed_at,omitempty"`
	RepositoryProfile         string   `json:"repository_profile"`
	AdoptionStage             string   `json:"adoption_stage"`
	Repository                string   `json:"repository"`
	MutationTarget            string   `json:"mutation_target"`
	InstallPath               string   `json:"install_path"`
	PrimaryBlockerClass       string   `json:"primary_blocker_class"`
	SuggestedOutcome          string   `json:"suggested_outcome"`
	UpstreamThread            string   `json:"upstream_thread,omitempty"`
	HasUpstreamThread         bool     `json:"has_upstream_thread"`
	ExternalResponseStatus    string   `json:"external_response_status"`
	ExternalResponseLastCheck string   `json:"external_response_last_checked,omitempty"`
	MissingSections           []string `json:"missing_sections,omitempty"`
}

type adoptionAggregate struct {
	TotalIssues                  int            `json:"total_issues"`
	OpenIssues                   int            `json:"open_issues"`
	ClosedIssues                 int            `json:"closed_issues"`
	RepositoryProfiles           map[string]int `json:"repository_profiles"`
	AdoptionStages               map[string]int `json:"adoption_stages"`
	InstallPaths                 map[string]int `json:"install_paths"`
	PrimaryBlockerClasses        map[string]int `json:"primary_blocker_classes"`
	SuggestedOutcomes            map[string]int `json:"suggested_outcomes"`
	ExternalResponseStatuses     map[string]int `json:"external_response_statuses"`
	MissingSectionCounts         map[string]int `json:"missing_section_counts"`
	IssuesWithUpstreamThread     int            `json:"issues_with_upstream_thread"`
	IssuesWithoutUpstreamThread  int            `json:"issues_without_upstream_thread"`
	IssuesWithMaintainerReply    int            `json:"issues_with_maintainer_reply"`
	IssuesWithoutMaintainerReply int            `json:"issues_without_maintainer_reply"`
}

var adoptionIssueFields = []struct {
	label       string
	required    bool
	assignValue func(*adoptionFeedbackIssue, string)
}{
	{label: "Repository profile", required: true, assignValue: func(issue *adoptionFeedbackIssue, value string) {
		issue.RepositoryProfile = normalizeIssueField(value, "Unspecified")
	}},
	{label: "Adoption stage", required: true, assignValue: func(issue *adoptionFeedbackIssue, value string) {
		issue.AdoptionStage = normalizeIssueField(value, "Unspecified")
	}},
	{label: "Repository", required: true, assignValue: func(issue *adoptionFeedbackIssue, value string) {
		issue.Repository = normalizeIssueField(value, "Unspecified")
	}},
	{label: "Mutation target", required: true, assignValue: func(issue *adoptionFeedbackIssue, value string) {
		issue.MutationTarget = normalizeIssueField(value, "Unspecified")
	}},
	{label: "Install path", required: true, assignValue: func(issue *adoptionFeedbackIssue, value string) {
		issue.InstallPath = normalizeIssueField(value, "Unspecified")
	}},
	{label: "Primary blocker class", required: true, assignValue: func(issue *adoptionFeedbackIssue, value string) {
		issue.PrimaryBlockerClass = normalizeIssueField(value, "Unspecified")
	}},
	{label: "Suggested outcome", required: true, assignValue: func(issue *adoptionFeedbackIssue, value string) {
		issue.SuggestedOutcome = normalizeIssueField(value, "Unspecified")
	}},
	{label: "Upstream issue or discussion", required: false, assignValue: func(issue *adoptionFeedbackIssue, value string) {
		value = strings.TrimSpace(value)
		if strings.EqualFold(value, "none") {
			value = ""
		}
		issue.UpstreamThread = value
		issue.HasUpstreamThread = value != ""
	}},
	{label: "External response status", required: false, assignValue: func(issue *adoptionFeedbackIssue, value string) {
		issue.ExternalResponseStatus = normalizeIssueField(value, "Unspecified")
	}},
	{label: "External response last checked", required: false, assignValue: func(issue *adoptionFeedbackIssue, value string) {
		issue.ExternalResponseLastCheck = strings.TrimSpace(value)
	}},
}

func cmdBuildAdoptionSummary(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("build-adoption-summary", flag.ContinueOnError)
	issuesPath := fs.String("issues-json", "", "path to GitHub issue export JSON")
	trackingIssue := fs.String("tracking-issue", "", "tracking issue")
	generatedAt := fs.String("generated-at", "", "summary generation timestamp")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	summary, err := buildAdoptionSummary(*issuesPath, *trackingIssue, *generatedAt)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(summary)
}

func cmdRenderAdoptionSummaryMarkdown(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("render-adoption-summary-markdown", flag.ContinueOnError)
	path := fs.String("path", "", "path to adoption-summary.json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	var summary adoptionSummary
	if err := readJSONFile(*path, &summary); err != nil {
		return err
	}
	_, err := io.WriteString(stdout, renderAdoptionSummaryMarkdown(summary))
	return err
}

func buildAdoptionSummary(issuesPath, trackingIssue, generatedAt string) (adoptionSummary, error) {
	if strings.TrimSpace(issuesPath) == "" {
		return adoptionSummary{}, fmt.Errorf("issues-json path must not be empty")
	}
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := os.ReadFile(issuesPath)
	if err != nil {
		return adoptionSummary{}, err
	}
	var exported []githubIssueExport
	if err := json.Unmarshal(data, &exported); err != nil {
		return adoptionSummary{}, err
	}
	issues := make([]adoptionFeedbackIssue, 0, len(exported))
	for _, raw := range exported {
		issues = append(issues, parseAdoptionFeedbackIssue(raw))
	}
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Number < issues[j].Number
	})
	summary := adoptionSummary{
		SchemaVersion: "1",
		TrackingIssue: trackingIssue,
		SourcePath:    issuesPath,
		GeneratedAt:   generatedAt,
		Issues:        issues,
		Aggregate: adoptionAggregate{
			RepositoryProfiles:       map[string]int{},
			AdoptionStages:           map[string]int{},
			InstallPaths:             map[string]int{},
			PrimaryBlockerClasses:    map[string]int{},
			SuggestedOutcomes:        map[string]int{},
			ExternalResponseStatuses: map[string]int{},
			MissingSectionCounts:     map[string]int{},
		},
	}
	summary.Aggregate.TotalIssues = len(issues)
	for _, issue := range issues {
		if strings.EqualFold(issue.State, "open") {
			summary.Aggregate.OpenIssues++
		} else {
			summary.Aggregate.ClosedIssues++
		}
		summary.Aggregate.RepositoryProfiles[issue.RepositoryProfile]++
		summary.Aggregate.AdoptionStages[issue.AdoptionStage]++
		summary.Aggregate.InstallPaths[issue.InstallPath]++
		summary.Aggregate.PrimaryBlockerClasses[issue.PrimaryBlockerClass]++
		summary.Aggregate.SuggestedOutcomes[issue.SuggestedOutcome]++
		summary.Aggregate.ExternalResponseStatuses[issue.ExternalResponseStatus]++
		for _, missing := range issue.MissingSections {
			summary.Aggregate.MissingSectionCounts[missing]++
		}
		if issue.HasUpstreamThread {
			summary.Aggregate.IssuesWithUpstreamThread++
		} else {
			summary.Aggregate.IssuesWithoutUpstreamThread++
		}
		if externalResponseCountsAsReply(issue.ExternalResponseStatus) {
			summary.Aggregate.IssuesWithMaintainerReply++
		} else {
			summary.Aggregate.IssuesWithoutMaintainerReply++
		}
	}
	return summary, nil
}

func parseAdoptionFeedbackIssue(raw githubIssueExport) adoptionFeedbackIssue {
	issue := adoptionFeedbackIssue{
		Number:                 raw.Number,
		Title:                  raw.Title,
		State:                  normalizeIssueField(raw.State, "unknown"),
		URL:                    raw.URL,
		ClosedAt:               strings.TrimSpace(raw.ClosedAt),
		RepositoryProfile:      "Unspecified",
		AdoptionStage:          "Unspecified",
		Repository:             "Unspecified",
		MutationTarget:         "Unspecified",
		InstallPath:            "Unspecified",
		PrimaryBlockerClass:    "Unspecified",
		SuggestedOutcome:       "Unspecified",
		ExternalResponseStatus: "Unspecified",
	}
	sections := parseIssueSections(raw.Body)
	for _, field := range adoptionIssueFields {
		value, ok := sections[field.label]
		if !ok {
			if field.required {
				issue.MissingSections = append(issue.MissingSections, field.label)
			}
			continue
		}
		field.assignValue(&issue, value)
	}
	return issue
}

func parseIssueSections(body string) map[string]string {
	sections := map[string]string{}
	var current string
	var content []string
	flush := func() {
		if current == "" {
			return
		}
		sections[current] = strings.TrimSpace(strings.Join(content, "\n"))
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			content = content[:0]
			continue
		}
		if current != "" {
			content = append(content, line)
		}
	}
	flush()
	return sections
}

func renderAdoptionSummaryMarkdown(summary adoptionSummary) string {
	var b strings.Builder
	b.WriteString("# Adoption Feedback Summary\n\n")
	if summary.TrackingIssue != "" {
		fmt.Fprintf(&b, "- Tracking issue: **%s**\n", summary.TrackingIssue)
	}
	if summary.SourcePath != "" {
		fmt.Fprintf(&b, "- Source JSON: `%s`\n", summary.SourcePath)
	}
	fmt.Fprintf(&b, "- Total issues: **%d**\n", summary.Aggregate.TotalIssues)
	fmt.Fprintf(&b, "- Open issues: **%d**\n", summary.Aggregate.OpenIssues)
	fmt.Fprintf(&b, "- Closed issues: **%d**\n", summary.Aggregate.ClosedIssues)
	fmt.Fprintf(&b, "- Issues with upstream thread: **%d**\n", summary.Aggregate.IssuesWithUpstreamThread)
	fmt.Fprintf(&b, "- Issues without upstream thread: **%d**\n", summary.Aggregate.IssuesWithoutUpstreamThread)
	fmt.Fprintf(&b, "- Issues with maintainer reply: **%d**\n", summary.Aggregate.IssuesWithMaintainerReply)
	fmt.Fprintf(&b, "- Issues without maintainer reply: **%d**\n", summary.Aggregate.IssuesWithoutMaintainerReply)
	if len(summary.Aggregate.ExternalResponseStatuses) > 0 {
		fmt.Fprintf(&b, "- External response statuses: `%s`\n", formatStatusCounts(summary.Aggregate.ExternalResponseStatuses))
	}
	if len(summary.Aggregate.PrimaryBlockerClasses) > 0 {
		fmt.Fprintf(&b, "- Primary blocker classes: `%s`\n", formatStatusCounts(summary.Aggregate.PrimaryBlockerClasses))
	}
	if len(summary.Aggregate.RepositoryProfiles) > 0 {
		fmt.Fprintf(&b, "- Repository profiles: `%s`\n", formatStatusCounts(summary.Aggregate.RepositoryProfiles))
	}
	if len(summary.Aggregate.SuggestedOutcomes) > 0 {
		fmt.Fprintf(&b, "- Suggested outcomes: `%s`\n", formatStatusCounts(summary.Aggregate.SuggestedOutcomes))
	}
	if len(summary.Aggregate.MissingSectionCounts) > 0 {
		fmt.Fprintf(&b, "- Missing issue sections: `%s`\n", formatStatusCounts(summary.Aggregate.MissingSectionCounts))
	}
	b.WriteString("\n")
	for _, issue := range summary.Issues {
		fmt.Fprintf(&b, "## #%d %s\n", issue.Number, issue.Title)
		fmt.Fprintf(&b, "- State: `%s`\n", issue.State)
		fmt.Fprintf(&b, "- Repository profile: `%s`\n", issue.RepositoryProfile)
		fmt.Fprintf(&b, "- Adoption stage: `%s`\n", issue.AdoptionStage)
		fmt.Fprintf(&b, "- Install path: `%s`\n", issue.InstallPath)
		fmt.Fprintf(&b, "- Primary blocker: `%s`\n", issue.PrimaryBlockerClass)
		fmt.Fprintf(&b, "- Suggested outcome: `%s`\n", issue.SuggestedOutcome)
		if issue.UpstreamThread == "" {
			b.WriteString("- Upstream thread: none\n")
		} else {
			fmt.Fprintf(&b, "- Upstream thread: `%s`\n", issue.UpstreamThread)
		}
		fmt.Fprintf(&b, "- External response status: `%s`\n", issue.ExternalResponseStatus)
		if issue.ExternalResponseLastCheck != "" {
			fmt.Fprintf(&b, "- External response last checked: `%s`\n", issue.ExternalResponseLastCheck)
		}
		if len(issue.MissingSections) > 0 {
			fmt.Fprintf(&b, "- Missing sections: `%s`\n", strings.Join(issue.MissingSections, ", "))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func normalizeIssueField(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func externalResponseCountsAsReply(status string) bool {
	switch strings.TrimSpace(status) {
	case "Maintainer replied or asked follow-up questions", "Maintainer acknowledged and accepted the finding", "Maintainer rejected, narrowed, or closed the upstream thread":
		return true
	default:
		return false
	}
}
