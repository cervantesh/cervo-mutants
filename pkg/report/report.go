package report

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cervantesh/CervoMutants/pkg/engine"
)

func JSON(result engine.RunResult) ([]byte, error) {
	if result.SchemaVersion == "" {
		result.SchemaVersion = "1"
	}
	if result.Thresholds == nil {
		result.Thresholds = map[string]any{}
	}
	return json.MarshalIndent(result, "", "  ")
}

func Summary(result engine.RunResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Mutation score: %.2f%%\nGenerated mutants: %d\nCovered mutants: %d\nExecuted mutants: %d\nEffective mutants: %d\nScore denominator: %d\nKilled: %d\nSurvived: %d\nNot covered: %d\nQuarantined: %d\nTimed out: %d\nCompile errors: %d\nTest efficacy: %.2f%%\nMutation coverage: %.2f%%\nHigh-risk survivors: %d\nNew survivors: %d\nLong-standing survivors: %d\nSuppression audits: report_only=%d lower_priority=%d suppress=%d quarantine_required=%d\n",
		result.Summary.Score,
		result.Summary.GeneratedMutants,
		result.Summary.CoveredMutants,
		result.Summary.ExecutedMutants,
		result.Summary.EffectiveMutants,
		result.Summary.ScoreDenominator,
		result.Summary.Killed,
		result.Summary.Survived,
		result.Summary.NotCovered,
		result.Summary.Quarantined,
		result.Summary.TimedOut,
		result.Summary.CompileError,
		result.Summary.TestEfficacy,
		result.Summary.MutationCoverage,
		result.Summary.HighRiskSurvivors,
		result.Summary.NewSurvivors,
		result.Summary.LongStandingSurvivors,
		result.Summary.SuppressionReportOnly,
		result.Summary.SuppressionLowerPriority,
		result.Summary.SuppressionSuppressed,
		result.Summary.SuppressionQuarantineRequired,
	)
	if result.Summary.DenominatorHealth.Generated > 0 || len(result.Summary.DenominatorHealth.Warnings) > 0 {
		health := result.Summary.DenominatorHealth
		fmt.Fprintf(&b, "Denominator health: healthy=%t generated=%d covered=%d executed=%d effective=%d score_denominator=%d killed=%d survived=%d not_covered=%d timed_out=%d compile_error=%d\n",
			health.Healthy,
			health.Generated,
			health.Covered,
			health.Executed,
			health.Effective,
			health.ScoreDenominator,
			health.Killed,
			health.Survived,
			health.NotCovered,
			health.TimedOut,
			health.CompileError,
		)
		if len(health.Warnings) > 0 {
			fmt.Fprintf(&b, "Denominator warnings: %s\n", strings.Join(health.Warnings, ", "))
		}
	}
	if len(result.Summary.EquivalentRiskStats) > 0 {
		b.WriteString("Equivalent-risk statistics:\n")
		keys := make([]string, 0, len(result.Summary.EquivalentRiskStats))
		for key := range result.Summary.EquivalentRiskStats {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&b, "- %s: %d\n", key, result.Summary.EquivalentRiskStats[key])
		}
	}
	if len(result.Summary.MutatorStats) > 0 {
		b.WriteString("Mutator statistics:\n")
		keys := make([]string, 0, len(result.Summary.MutatorStats))
		for key := range result.Summary.MutatorStats {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			stat := result.Summary.MutatorStats[key]
			fmt.Fprintf(&b, "- %s: total=%d killed=%d survived=%d not_covered=%d timed_out=%d compile_error=%d recommendation=%s equivalent_risk=%s\n",
				key,
				stat.Total,
				stat.Killed,
				stat.Survived,
				stat.NotCovered,
				stat.TimedOut,
				stat.CompileError,
				stat.Recommendation,
				stat.EquivalentRisk,
			)
		}
	}
	if result.Environment.OS != "" {
		fmt.Fprintf(&b, "Environment: os=%s arch=%s go=%s isolation=%s workers=%d timeout=%s wsl=%t onedrive=%t\n",
			result.Environment.OS,
			result.Environment.Arch,
			result.Environment.GoVersion,
			result.Environment.Isolation,
			result.Environment.Workers,
			result.Environment.TestTimeout,
			result.Environment.WSL,
			result.Environment.WindowsOneDrive,
		)
	}
	return b.String()
}

func Survivors(result engine.RunResult) string {
	var b strings.Builder
	survivors := make([]engine.MutantResult, 0)
	for _, item := range result.Mutants {
		if item.Status != engine.StatusSurvived {
			continue
		}
		survivors = append(survivors, item)
	}
	sort.SliceStable(survivors, func(i, j int) bool {
		if survivors[i].SurvivorRank == 0 {
			return false
		}
		if survivors[j].SurvivorRank == 0 {
			return true
		}
		return survivors[i].SurvivorRank < survivors[j].SurvivorRank
	})
	for _, mutant := range survivors {
		fmt.Fprintf(&b, "#%d %.1f %s %s:%d %s %s -> %s actionability=%s scope=%s (%s)\n", mutant.SurvivorRank, mutant.RankScore, mutant.MutantID, mutant.Mutant.File, mutant.Mutant.Line, mutant.Mutant.Operator, mutant.Mutant.Original, mutant.Mutant.Mutated, mutant.Actionability, mutant.SuggestedTestScope, mutant.RankReason)
	}
	return b.String()
}

func HTML(result engine.RunResult) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>cervomut report</title></head><body>")
	b.WriteString("<h1>cervomut mutation report</h1>")
	b.WriteString("<pre>")
	b.WriteString(html.EscapeString(Summary(result)))
	b.WriteString("</pre><table><thead><tr><th>Status</th><th>Mutant</th><th>Diff</th></tr></thead><tbody>")
	for _, mutant := range result.Mutants {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(string(mutant.Status)))
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(mutant.MutantID))
		b.WriteString("</td><td><pre>")
		b.WriteString(html.EscapeString(mutant.Mutant.Diff))
		b.WriteString("</pre></td></tr>")
	}
	b.WriteString("</tbody></table></body></html>")
	return b.String()
}

type junitTestsuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Cases    []junitTestcase `xml:"testcase"`
}

type junitTestcase struct {
	Name    string        `xml:"name,attr"`
	Failure *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

func JUnit(result engine.RunResult) ([]byte, error) {
	suite := junitTestsuite{Name: "cervomut", Tests: len(result.Mutants)}
	for _, mutant := range result.Mutants {
		tc := junitTestcase{Name: mutant.MutantID}
		if mutant.Status == engine.StatusSurvived || mutant.Status == engine.StatusTimedOut || mutant.Status == engine.StatusNotCovered {
			suite.Failures++
			tc.Failure = &junitFailure{Message: string(mutant.Status), Text: mutant.StatusReason}
		}
		suite.Cases = append(suite.Cases, tc)
	}
	return xml.MarshalIndent(suite, "", "  ")
}

func WriteAll(dir string, result engine.RunResult) error {
	return WriteFormats(dir, result, []string{"summary", "json", "junit", "html"})
}

func WriteFormats(dir string, result engine.RunResult, formats []string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if len(formats) == 0 {
		formats = []string{"summary", "json"}
	}
	files := map[string][]byte{}
	for _, format := range formats {
		switch strings.TrimSpace(format) {
		case "summary":
			files["summary.txt"] = []byte(Summary(result))
			files["survivors.txt"] = []byte(Survivors(result))
		case "json":
			jsonData, err := JSON(result)
			if err != nil {
				return err
			}
			files["mutation-report.json"] = jsonData
		case "junit":
			junitData, err := JUnit(result)
			if err != nil {
				return err
			}
			files["junit.xml"] = junitData
		case "html":
			files["index.html"] = []byte(HTML(result))
		}
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
