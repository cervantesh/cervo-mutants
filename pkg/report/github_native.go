package report

import (
	"encoding/json"
	"fmt"
	"os"
	pathpkg "path"
	"sort"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

const (
	sarifSchemaURL    = "https://schemastore.azurewebsites.net/schemas/json/sarif-2.1.0-rtm.5.json"
	projectRepository = "https://github.com/cervantesh/cervo-mutants"
)

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool               sarifTool                   `json:"tool"`
	Results            []sarifResult               `json:"results"`
	Artifacts          []sarifArtifact             `json:"artifacts,omitempty"`
	Invocations        []sarifInvocation           `json:"invocations,omitempty"`
	Properties         map[string]any              `json:"properties,omitempty"`
	OriginalURIBaseIDs map[string]sarifArtifactRef `json:"originalUriBaseIds,omitempty"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string                     `json:"name"`
	Version        string                     `json:"version,omitempty"`
	InformationURI string                     `json:"informationUri,omitempty"`
	Rules          []sarifReportingDescriptor `json:"rules,omitempty"`
}

type sarifReportingDescriptor struct {
	ID               string         `json:"id"`
	Name             string         `json:"name,omitempty"`
	ShortDescription sarifMessage   `json:"shortDescription,omitempty"`
	FullDescription  sarifMessage   `json:"fullDescription,omitempty"`
	HelpURI          string         `json:"helpUri,omitempty"`
	Properties       map[string]any `json:"properties,omitempty"`
}

type sarifResult struct {
	RuleID              string            `json:"ruleId"`
	Level               string            `json:"level,omitempty"`
	Message             sarifMessage      `json:"message"`
	Locations           []sarifLocation   `json:"locations,omitempty"`
	PartialFingerprints map[string]string `json:"partialFingerprints,omitempty"`
	Properties          map[string]any    `json:"properties,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactRef `json:"artifactLocation"`
	Region           sarifRegion      `json:"region,omitempty"`
}

type sarifArtifactRef struct {
	URI       string `json:"uri,omitempty"`
	URIBaseID string `json:"uriBaseId,omitempty"`
}

type sarifRegion struct {
	StartLine int `json:"startLine,omitempty"`
}

type sarifArtifact struct {
	Location sarifArtifactRef `json:"location"`
}

type sarifInvocation struct {
	ExecutionSuccessful bool             `json:"executionSuccessful"`
	CommandLine         string           `json:"commandLine,omitempty"`
	WorkingDirectory    sarifArtifactRef `json:"workingDirectory,omitempty"`
}

func SARIF(result engine.RunResult) ([]byte, error) {
	rules := map[string]sarifReportingDescriptor{}
	artifacts := make([]sarifArtifact, 0)
	artifactSeen := map[string]bool{}
	sarifResults := make([]sarifResult, 0)
	for _, mutant := range result.Mutants {
		if !sarifRelevantStatus(mutant.Status) {
			continue
		}
		rule := sarifRule(mutant)
		rules[rule.ID] = rule
		entry := sarifResult{
			RuleID:  rule.ID,
			Level:   sarifLevel(mutant),
			Message: sarifMessage{Text: sarifMessageText(mutant)},
			PartialFingerprints: map[string]string{
				"mutant_id": mutant.MutantID,
			},
			Properties: sarifProperties(mutant),
		}
		if location, ok := sarifLocationFor(result, mutant); ok {
			entry.Locations = []sarifLocation{location}
			if location.PhysicalLocation.ArtifactLocation.URI != "" && !artifactSeen[location.PhysicalLocation.ArtifactLocation.URI] {
				artifactSeen[location.PhysicalLocation.ArtifactLocation.URI] = true
				artifacts = append(artifacts, sarifArtifact{Location: location.PhysicalLocation.ArtifactLocation})
			}
		}
		sarifResults = append(sarifResults, entry)
	}
	ruleList := make([]sarifReportingDescriptor, 0, len(rules))
	for _, rule := range rules {
		ruleList = append(ruleList, rule)
	}
	sort.Slice(ruleList, func(i, j int) bool {
		return ruleList[i].ID < ruleList[j].ID
	})
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Location.URI < artifacts[j].Location.URI
	})
	run := sarifRun{
		Tool: sarifTool{Driver: sarifDriver{
			Name:           "cervomut",
			Version:        result.Environment.ToolVersion,
			InformationURI: projectRepository,
			Rules:          ruleList,
		}},
		Results:     sarifResults,
		Artifacts:   artifacts,
		Invocations: []sarifInvocation{sarifInvocationFor(result)},
		Properties: map[string]any{
			"raw_score":        result.Summary.Score,
			"actionable_score": result.Summary.Actionable.ActionableScore,
			"survived":         result.Summary.Survived,
			"timed_out":        result.Summary.TimedOut,
			"not_covered":      result.Summary.NotCovered,
		},
	}
	if baseIDs := sarifOriginalURIBaseIDs(result.Environment.WorkingDir); len(baseIDs) > 0 {
		run.OriginalURIBaseIDs = baseIDs
	}
	log := sarifLog{
		Version: "2.1.0",
		Schema:  sarifSchemaURL,
		Runs:    []sarifRun{run},
	}
	return json.MarshalIndent(log, "", "  ")
}

func GitHubSummary(result engine.RunResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## CervoMutants Mutation Summary\n\n")
	fmt.Fprintf(&b, "- Raw score: **%.2f%%**\n", result.Summary.Score)
	fmt.Fprintf(&b, "- Actionable score: **%.2f%%**\n", result.Summary.Actionable.ActionableScore)
	fmt.Fprintf(&b, "- Survivors: **%d** total, **%d** true actionable review units\n", result.Summary.Survived, result.Summary.Actionable.TrueActionableSurvivors)
	fmt.Fprintf(&b, "- Equivalent-risk survivors: **%d**\n", result.Summary.Actionable.EquivalentRiskSurvivors)
	fmt.Fprintf(&b, "- Platform-sensitive survivors: **%d**\n", result.Summary.PlatformSensitiveSurvivors)
	fmt.Fprintf(&b, "- Non-progress timeouts: **%d**\n", result.Summary.NonProgressTimeouts)
	if result.Baseline.Enabled {
		fmt.Fprintf(&b, "- Baseline regression: **%t**\n", result.Baseline.Regression)
		fmt.Fprintf(&b, "- Baseline new survivors: **%d**\n", len(result.Baseline.NewSurvivors))
	}
	if len(result.Summary.DenominatorHealth.Warnings) > 0 {
		fmt.Fprintf(&b, "- Denominator warnings: `%s`\n", strings.Join(result.Summary.DenominatorHealth.Warnings, "`, `"))
	}
	topSurvivors := githubTopSurvivors(result.Mutants, 5)
	if len(topSurvivors) > 0 {
		b.WriteString("\n### Top Survivor Queue\n\n")
		b.WriteString("| Rank | Mutant | Actionability | Operator | Location | Skip guidance |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
		for _, survivor := range topSurvivors {
			location := fmt.Sprintf("%s:%d", sarifSlashPath(survivor.Mutant.File), survivor.Mutant.Line)
			fmt.Fprintf(&b, "| %d | `%s` | `%s` | `%s` | `%s` | %s |\n",
				survivor.SurvivorRank,
				escapeMarkdownCell(survivor.MutantID),
				escapeMarkdownCell(survivor.Actionability),
				escapeMarkdownCell(survivor.Mutant.Operator),
				escapeMarkdownCell(location),
				escapeMarkdownCell(ledgerSuggestedReason(survivor, "review based on the mutation diff and nearby tests")),
			)
		}
	}
	otherSignals := githubSignalRows(result)
	if len(otherSignals) > 0 {
		b.WriteString("\n### Other Review Signals\n\n")
		b.WriteString("| Kind | Count |\n")
		b.WriteString("| --- | ---: |\n")
		for _, row := range otherSignals {
			fmt.Fprintf(&b, "| %s | %d |\n", escapeMarkdownCell(row.label), row.count)
		}
	}
	return b.String()
}

func writeGitHubStepSummary(markdown string) error {
	path := os.Getenv("GITHUB_STEP_SUMMARY")
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return os.WriteFile(path, []byte(markdown), 0o644)
}

type githubSignalRow struct {
	label string
	count int
}

func githubSignalRows(result engine.RunResult) []githubSignalRow {
	rows := []githubSignalRow{
		{label: "timed out", count: result.Summary.TimedOut},
		{label: "not covered", count: result.Summary.NotCovered},
		{label: "compile errors", count: result.Summary.CompileError},
		{label: "memory killed", count: result.Summary.MemoryKilled},
		{label: "pending budget", count: result.Summary.PendingBudget},
		{label: "skipped resource", count: result.Summary.SkippedResource},
	}
	filtered := make([]githubSignalRow, 0, len(rows))
	for _, row := range rows {
		if row.count > 0 {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func githubTopSurvivors(mutants []engine.MutantResult, limit int) []engine.MutantResult {
	survivors := rankedSurvivors(mutants)
	if len(survivors) > limit {
		return survivors[:limit]
	}
	return survivors
}

func escapeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.ReplaceAll(value, "|", "\\|")
}

func sarifRelevantStatus(status engine.Status) bool {
	switch status {
	case engine.StatusSurvived, engine.StatusTimedOut, engine.StatusMemoryKilled, engine.StatusCompileError, engine.StatusNotCovered, engine.StatusPendingBudget, engine.StatusSkippedResource:
		return true
	default:
		return false
	}
}

func sarifInvocationFor(result engine.RunResult) sarifInvocation {
	command := ""
	if len(result.Mutants) > 0 && len(result.Mutants[0].TestCommand) > 0 {
		command = strings.Join(result.Mutants[0].TestCommand, " ")
	}
	invocation := sarifInvocation{
		ExecutionSuccessful: result.Failure == nil,
	}
	if command != "" {
		invocation.CommandLine = command
	}
	if ref, ok := sarifWorkingDirectoryRef(result.Environment.WorkingDir); ok {
		invocation.WorkingDirectory = ref
	}
	return invocation
}

func sarifRule(mutant engine.MutantResult) sarifReportingDescriptor {
	id, name, description := sarifRuleParts(mutant)
	return sarifReportingDescriptor{
		ID:               id,
		Name:             name,
		ShortDescription: sarifMessage{Text: name},
		FullDescription:  sarifMessage{Text: description},
		HelpURI:          projectRepository,
		Properties: map[string]any{
			"status": mutant.Status,
		},
	}
}

func sarifRuleParts(mutant engine.MutantResult) (string, string, string) {
	switch mutant.Status {
	case engine.StatusSurvived:
		return "survived", "survived mutation", "Tests passed with the mutation applied, so the changed behavior still lacks a killing assertion."
	case engine.StatusTimedOut:
		if mutant.FailureKind == "non_progress_loop" {
			return "timed_out.non_progress_loop", "non-progress loop timeout", "The mutation likely broke loop progress and the test run timed out."
		}
		return "timed_out", "mutation timeout", "The mutation run timed out before producing a stable result."
	case engine.StatusMemoryKilled:
		return "memory_killed", "mutation memory kill", "The mutation run exceeded memory constraints or was killed for memory pressure."
	case engine.StatusCompileError:
		return "compile_error", "mutation compile error", "Applying the mutation produced a compile error."
	case engine.StatusNotCovered:
		return "not_covered", "mutation not covered", "No selected test covered this mutation."
	case engine.StatusPendingBudget:
		return "pending_budget", "mutation pending budget", "The mutation run stopped before this mutant because the configured budget was exhausted."
	case engine.StatusSkippedResource:
		return "skipped_resource", "mutation skipped for resources", "The mutation run skipped this mutant because required resources were unavailable."
	default:
		return string(mutant.Status), strings.ReplaceAll(string(mutant.Status), "_", " "), "Mutation result recorded for GitHub-native output."
	}
}

func sarifLevel(mutant engine.MutantResult) string {
	switch mutant.Status {
	case engine.StatusCompileError:
		return "error"
	case engine.StatusSurvived, engine.StatusTimedOut, engine.StatusMemoryKilled:
		return "warning"
	default:
		return "note"
	}
}

func sarifMessageText(mutant engine.MutantResult) string {
	location := sarifSlashPath(mutant.Mutant.File)
	if mutant.Mutant.Line > 0 {
		location = fmt.Sprintf("%s:%d", location, mutant.Mutant.Line)
	}
	text := fmt.Sprintf("%s in %s mutated `%s` to `%s` with status `%s`.", mutant.Mutant.Operator, location, mutant.Mutant.Original, mutant.Mutant.Mutated, mutant.Status)
	if mutant.StatusReason != "" {
		text += " " + mutant.StatusReason
	}
	if mutant.Actionability != "" {
		text += " actionability=" + mutant.Actionability + "."
	}
	return text
}

func sarifProperties(mutant engine.MutantResult) map[string]any {
	properties := map[string]any{
		"mutant_id":            mutant.MutantID,
		"operator":             mutant.Mutant.Operator,
		"status":               mutant.Status,
		"equivalent_risk":      mutant.Mutant.EquivalentRisk,
		"actionability":        mutant.Actionability,
		"suggested_test_scope": mutant.SuggestedTestScope,
	}
	if mutant.Mutant.PlatformSensitive {
		properties["platform_sensitive"] = true
	}
	if mutant.Mutant.NonProgressRisk != "" {
		properties["non_progress_risk"] = mutant.Mutant.NonProgressRisk
	}
	if mutant.Mutant.SemanticGroup != "" {
		properties["semantic_group"] = mutant.Mutant.SemanticGroup
	}
	return properties
}

func sarifLocationFor(result engine.RunResult, mutant engine.MutantResult) (sarifLocation, bool) {
	ref, ok := sarifArtifactRefFor(result.Environment.WorkingDir, mutant.Mutant.File)
	if !ok {
		return sarifLocation{}, false
	}
	location := sarifLocation{
		PhysicalLocation: sarifPhysicalLocation{
			ArtifactLocation: ref,
		},
	}
	if mutant.Mutant.Line > 0 {
		location.PhysicalLocation.Region = sarifRegion{StartLine: mutant.Mutant.Line}
	}
	return location, true
}

func sarifArtifactRefFor(workingDir, path string) (sarifArtifactRef, bool) {
	if strings.TrimSpace(path) == "" {
		return sarifArtifactRef{}, false
	}
	cleanPath := sarifSlashPath(path)
	cleanWorkingDir := sarifSlashPath(workingDir)
	if cleanWorkingDir != "" && sarifIsAbs(cleanWorkingDir) && sarifIsAbs(cleanPath) {
		if rel, ok := sarifRelativeToBase(cleanWorkingDir, cleanPath); ok {
			return sarifArtifactRef{URI: rel, URIBaseID: "SRCROOT"}, true
		}
	}
	return sarifArtifactRef{URI: cleanPath}, true
}

func sarifOriginalURIBaseIDs(workingDir string) map[string]sarifArtifactRef {
	ref, ok := sarifWorkingDirectoryRef(workingDir)
	if !ok {
		return nil
	}
	return map[string]sarifArtifactRef{"SRCROOT": ref}
}

func sarifWorkingDirectoryRef(workingDir string) (sarifArtifactRef, bool) {
	cleanWorkingDir := sarifSlashPath(workingDir)
	if cleanWorkingDir == "" {
		return sarifArtifactRef{}, false
	}
	return sarifArtifactRef{URI: cleanWorkingDir}, true
}

func sarifSlashPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "/")
	return pathpkg.Clean(value)
}

func sarifIsAbs(value string) bool {
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return true
	}
	return sarifHasWindowsDrive(value)
}

func sarifRelativeToBase(base, target string) (string, bool) {
	base = strings.TrimSuffix(base, "/")
	if base == "" || target == "" || target == base {
		return "", false
	}
	compareBase := base
	compareTarget := target
	if sarifHasWindowsDrive(base) || strings.HasPrefix(base, "//") {
		compareBase = strings.ToLower(base)
		compareTarget = strings.ToLower(target)
	}
	prefix := compareBase + "/"
	if !strings.HasPrefix(compareTarget, prefix) {
		return "", false
	}
	rel := target[len(base)+1:]
	if rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

func sarifHasWindowsDrive(value string) bool {
	return len(value) >= 3 && isASCIIAlpha(value[0]) && value[1] == ':' && value[2] == '/'
}

func isASCIIAlpha(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}
