package pool

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/extcompare"
)

const (
	compareLabelApplesToApples  = "apples_to_apples=true"
	compareLabelManifestShifted = "manifest_equivalent=false"
	compareLabelNotComparable   = "not_comparable"
)

type CompareStudy struct {
	SchemaVersion string             `json:"schema_version"`
	ManifestPath  string             `json:"manifest_path,omitempty"`
	GeneratedAt   time.Time          `json:"generated_at"`
	Totals        CompareStudyTotals `json:"totals"`
	Repos         []CompareStudyRepo `json:"repos"`
}

type CompareStudyTotals struct {
	Repos                int            `json:"repos"`
	ResultRows           int            `json:"result_rows"`
	ApplesToApplesRepos  int            `json:"apples_to_apples_repos"`
	ManifestShiftedRepos int            `json:"manifest_shifted_repos"`
	NotComparableRepos   int            `json:"not_comparable_repos"`
	StatusCounts         map[string]int `json:"status_counts,omitempty"`
	ToolCounts           map[string]int `json:"tool_counts,omitempty"`
}

type CompareStudyRepo struct {
	Name               string                   `json:"name"`
	Target             string                   `json:"target,omitempty"`
	Lane               string                   `json:"lane,omitempty"`
	Domain             string                   `json:"domain,omitempty"`
	Comparability      extcompare.Comparability `json:"comparability"`
	ComparabilityLabel string                   `json:"comparability_label"`
	Rows               []CompareStudyRow        `json:"rows"`
	Warnings           []string                 `json:"warnings,omitempty"`
}

type CompareStudyRow struct {
	Tool                string   `json:"tool"`
	Status              string   `json:"status"`
	Target              string   `json:"target,omitempty"`
	EffectiveTarget     string   `json:"effective_target,omitempty"`
	TargetMode          string   `json:"target_mode,omitempty"`
	Seconds             float64  `json:"seconds"`
	Total               int      `json:"total"`
	Killed              int      `json:"killed"`
	Survived            int      `json:"survived"`
	NotCovered          int      `json:"not_covered"`
	NotViable           int      `json:"not_viable,omitempty"`
	Errors              int      `json:"errors"`
	TimedOut            int      `json:"timed_out"`
	Score               float64  `json:"score"`
	TestEfficacy        float64  `json:"test_efficacy,omitempty"`
	MutationCoverage    float64  `json:"mutation_coverage,omitempty"`
	DenominatorWarnings []string `json:"denominator_warnings,omitempty"`
	PartialReportUsed   bool     `json:"partial_report_used,omitempty"`
	Exit                int      `json:"exit"`
	Note                string   `json:"note,omitempty"`
	Log                 string   `json:"log,omitempty"`
}

func writeCompareWorkflowArtifacts(manifestPath, outputRoot string, results []CompareResult) (map[string]string, error) {
	study := buildCompareStudy(manifestPath, results)
	jsonPath := studyJSONPath(outputRoot)
	if err := writeJSON(jsonPath, study); err != nil {
		return nil, err
	}
	mdPath := studyMarkdownPath(outputRoot)
	if err := writeCompareMarkdown(mdPath, study); err != nil {
		return nil, err
	}
	return map[string]string{
		"study_json":       jsonPath,
		"summary_markdown": mdPath,
	}, nil
}

func buildCompareStudy(manifestPath string, results []CompareResult) CompareStudy {
	grouped := map[string][]CompareResult{}
	repoNames := make([]string, 0)
	for _, result := range results {
		if _, ok := grouped[result.Repo]; !ok {
			repoNames = append(repoNames, result.Repo)
		}
		grouped[result.Repo] = append(grouped[result.Repo], result)
	}
	sort.Strings(repoNames)
	repos := make([]CompareStudyRepo, 0, len(repoNames))
	statusCounts := map[string]int{}
	toolCounts := map[string]int{}
	totals := CompareStudyTotals{
		Repos:        len(repoNames),
		ResultRows:   len(results),
		StatusCounts: statusCounts,
		ToolCounts:   toolCounts,
	}
	for _, repoName := range repoNames {
		repo := buildCompareStudyRepo(repoName, grouped[repoName])
		repos = append(repos, repo)
		switch repo.ComparabilityLabel {
		case compareLabelApplesToApples:
			totals.ApplesToApplesRepos++
		case compareLabelManifestShifted:
			totals.ManifestShiftedRepos++
		default:
			totals.NotComparableRepos++
		}
		for _, row := range repo.Rows {
			statusCounts[row.Status]++
			toolCounts[row.Tool]++
		}
	}
	return CompareStudy{
		SchemaVersion: "1",
		ManifestPath:  manifestPath,
		GeneratedAt:   time.Now().UTC(),
		Totals:        totals,
		Repos:         repos,
	}
}

func buildCompareStudyRepo(name string, results []CompareResult) CompareStudyRepo {
	if len(results) == 0 {
		return CompareStudyRepo{Name: name, ComparabilityLabel: compareLabelNotComparable}
	}
	rows := make([]CompareStudyRow, 0, len(results))
	tools := make([]extcompare.ToolResult, 0, len(results))
	warnings := make([]string, 0)
	target := results[0].Target
	lane := results[0].Lane
	domain := results[0].Domain
	for _, result := range results {
		rows = append(rows, CompareStudyRow{
			Tool:                result.Tool,
			Status:              result.Status,
			Target:              result.Target,
			EffectiveTarget:     result.EffectiveTarget,
			TargetMode:          result.TargetMode,
			Seconds:             result.Seconds,
			Total:               result.Total,
			Killed:              result.Killed,
			Survived:            result.Survived,
			NotCovered:          result.NotCovered,
			NotViable:           result.NotViable,
			Errors:              result.Errors,
			TimedOut:            result.TimedOut,
			Score:               result.Score,
			TestEfficacy:        result.TestEfficacy,
			MutationCoverage:    result.MutationCoverage,
			DenominatorWarnings: append([]string{}, result.DenominatorWarnings...),
			PartialReportUsed:   result.PartialReportUsed,
			Exit:                result.Exit,
			Note:                result.Note,
			Log:                 result.Log,
		})
		tools = append(tools, extcompare.ToolResult{
			Tool:            result.Tool,
			Status:          result.Status,
			Target:          result.Target,
			EffectiveTarget: result.EffectiveTarget,
			TargetMode:      result.TargetMode,
		})
		if result.Status == "missing_repo" {
			warnings = appendUniqueString(warnings, "missing_repo")
		}
		if result.Note != "" {
			warnings = appendUniqueString(warnings, result.Note)
		}
		for _, warning := range result.DenominatorWarnings {
			warnings = appendUniqueString(warnings, warning)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].Tool < rows[j].Tool
	})
	comp := extcompare.BuildComparability(tools)
	if containsStatus(rows, "missing_repo") {
		comp.ApplesToApples = false
		comp.ManifestEquivalent = false
		comp.Warnings = appendUniqueString(comp.Warnings, "missing_repo")
	}
	label := compareLabelApplesToApples
	switch {
	case !comp.ApplesToApples:
		label = compareLabelNotComparable
	case !comp.ManifestEquivalent:
		label = compareLabelManifestShifted
	}
	return CompareStudyRepo{
		Name:               name,
		Target:             target,
		Lane:               lane,
		Domain:             domain,
		Comparability:      comp,
		ComparabilityLabel: label,
		Rows:               rows,
		Warnings:           warnings,
	}
}

func containsStatus(rows []CompareStudyRow, target string) bool {
	for _, row := range rows {
		if row.Status == target {
			return true
		}
	}
	return false
}

func appendUniqueString(values []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func writeCompareMarkdown(path string, study CompareStudy) error {
	var b strings.Builder
	b.WriteString("# Comparison Study\n\n")
	if study.ManifestPath != "" {
		fmt.Fprintf(&b, "Manifest: `%s`\n\n", study.ManifestPath)
	}
	fmt.Fprintf(&b, "Generated: `%s`\n\n", study.GeneratedAt.Format(time.RFC3339))
	b.WriteString("## Totals\n\n")
	fmt.Fprintf(&b, "- Repos: **%d**\n", study.Totals.Repos)
	fmt.Fprintf(&b, "- Result rows: **%d**\n", study.Totals.ResultRows)
	fmt.Fprintf(&b, "- `apples_to_apples=true`: **%d** repos\n", study.Totals.ApplesToApplesRepos)
	fmt.Fprintf(&b, "- `manifest_equivalent=false`: **%d** repos\n", study.Totals.ManifestShiftedRepos)
	fmt.Fprintf(&b, "- `not_comparable`: **%d** repos\n", study.Totals.NotComparableRepos)
	if len(study.Totals.StatusCounts) > 0 {
		b.WriteString("\n### Status Counts\n\n")
		for _, key := range sortedIntKeys(study.Totals.StatusCounts) {
			fmt.Fprintf(&b, "- `%s`: %d\n", key, study.Totals.StatusCounts[key])
		}
	}
	b.WriteString("\n## Repo Summaries\n\n")
	for _, repo := range study.Repos {
		fmt.Fprintf(&b, "### %s\n\n", repo.Name)
		fmt.Fprintf(&b, "- Target: `%s`\n", repo.Target)
		if repo.Lane != "" {
			fmt.Fprintf(&b, "- Lane: `%s`\n", repo.Lane)
		}
		if repo.Domain != "" {
			fmt.Fprintf(&b, "- Domain: `%s`\n", repo.Domain)
		}
		fmt.Fprintf(&b, "- Comparability: **%s**\n", repo.ComparabilityLabel)
		if len(repo.Comparability.Warnings) > 0 {
			fmt.Fprintf(&b, "- Comparability warnings: `%s`\n", strings.Join(repo.Comparability.Warnings, "`, `"))
		}
		if len(repo.Warnings) > 0 {
			fmt.Fprintf(&b, "- Review warnings: `%s`\n", strings.Join(repo.Warnings, "`, `"))
		}
		b.WriteString("\n| Tool | Status | Seconds | Score | Test efficacy | Mutation coverage | Effective target | Target mode |\n")
		b.WriteString("| --- | --- | ---: | ---: | ---: | ---: | --- | --- |\n")
		for _, row := range repo.Rows {
			fmt.Fprintf(&b, "| `%s` | `%s` | %.2f | %.2f | %.2f | %.2f | `%s` | `%s` |\n",
				row.Tool,
				row.Status,
				row.Seconds,
				row.Score,
				row.TestEfficacy,
				row.MutationCoverage,
				row.EffectiveTarget,
				row.TargetMode,
			)
		}
		b.WriteString("\n")
	}
	return writeTextFile(path, b.String())
}

func writeTextFile(path, body string) error {
	return writeRawFile(path, []byte(body))
}

func writeRawFile(path string, data []byte) error {
	if err := ensureDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

func sortedIntKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
