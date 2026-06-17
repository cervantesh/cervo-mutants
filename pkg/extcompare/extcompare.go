package extcompare

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

type ToolResult struct {
	Tool              string            `json:"tool"`
	Completed         bool              `json:"completed"`
	Status            string            `json:"status"`
	Target            string            `json:"target,omitempty"`
	EffectiveTarget   string            `json:"effective_target,omitempty"`
	TargetMode        string            `json:"target_mode,omitempty"`
	NotComparable     bool              `json:"not_comparable,omitempty"`
	Total             int               `json:"total"`
	Killed            int               `json:"killed"`
	Survived          int               `json:"survived"`
	NotCovered        int               `json:"not_covered"`
	NotViable         int               `json:"not_viable"`
	Errors            int               `json:"errors"`
	Skipped           int               `json:"skipped"`
	TimedOut          int               `json:"timed_out"`
	Score             float64           `json:"score"`
	TestEfficacy      float64           `json:"test_efficacy"`
	MutationCoverage  float64           `json:"mutation_coverage"`
	DenominatorHealth DenominatorHealth `json:"denominator_health"`
	Notes             []string          `json:"notes,omitempty"`
}

type DenominatorHealth struct {
	Effective        int      `json:"effective"`
	ScoreDenominator int      `json:"score_denominator"`
	Killed           int      `json:"killed"`
	Survived         int      `json:"survived"`
	NotCovered       int      `json:"not_covered"`
	TimedOut         int      `json:"timed_out"`
	Errors           int      `json:"errors"`
	Healthy          bool     `json:"healthy"`
	Warnings         []string `json:"warnings,omitempty"`
}

type Study struct {
	SchemaVersion string        `json:"schema_version"`
	Comparability Comparability `json:"comparability"`
	Results       []ToolResult  `json:"results"`
}

type Comparability struct {
	ApplesToApples     bool     `json:"apples_to_apples"`
	ManifestEquivalent bool     `json:"manifest_equivalent"`
	EffectiveTargets   []string `json:"effective_targets,omitempty"`
	TargetModes        []string `json:"target_modes,omitempty"`
	Warnings           []string `json:"warnings,omitempty"`
}

func ParseCervo(path string) (ToolResult, error) {
	var report struct {
		Summary struct {
			Total        int     `json:"total"`
			Killed       int     `json:"killed"`
			Survived     int     `json:"survived"`
			NotCovered   int     `json:"not_covered"`
			TimedOut     int     `json:"timed_out"`
			CompileError int     `json:"compile_error"`
			Skipped      int     `json:"skipped"`
			Score        float64 `json:"score"`
			TestEfficacy float64 `json:"test_efficacy"`
		} `json:"summary"`
	}
	if err := readJSON(path, &report); err != nil {
		return ToolResult{}, err
	}
	result := ToolResult{
		Tool:       "cervo-mutants",
		Completed:  true,
		Status:     "ok",
		Total:      report.Summary.Total,
		Killed:     report.Summary.Killed,
		Survived:   report.Summary.Survived,
		NotCovered: report.Summary.NotCovered,
		TimedOut:   report.Summary.TimedOut,
		Errors:     report.Summary.CompileError,
		Skipped:    report.Summary.Skipped,
		Score:      report.Summary.Score,
	}
	result.TestEfficacy = report.Summary.TestEfficacy
	if result.TestEfficacy == 0 {
		result.TestEfficacy = scoreFrom(result.Killed, result.Survived)
	}
	result.DenominatorHealth = denominatorHealth(result)
	result.MutationCoverage = mutationCoverage(result)
	return result, nil
}

func ParseGremlins(path string) (ToolResult, error) {
	var report map[string]any
	if err := readJSON(path, &report); err != nil {
		return ToolResult{}, err
	}
	total := intField(report, "mutants_total", "total_mutants", "total", "mutants")
	killed := intField(report, "mutants_killed", "killed", "killed_mutants")
	survived := intField(report, "mutants_lived", "survived", "survived_mutants")
	notCovered := intField(report, "mutants_not_covered", "not_covered", "notCovered", "uncovered")
	notViable := intField(report, "mutants_not_viable", "not_viable", "notViable")
	skipped := intField(report, "mutants_skipped", "skipped")
	timedOut := intField(report, "mutants_timed_out", "timed_out", "timedOut")
	if timedOut == 0 {
		timedOut = countGremlinsStatus(report, "TIMED OUT")
	}
	score := floatField(report, "test_efficacy", "mutation_score", "score")
	if score == 0 {
		score = scoreFrom(killed, survived)
	}
	result := ToolResult{Tool: "gremlins", Completed: true, Status: "ok", Total: total, Killed: killed, Survived: survived, NotCovered: notCovered, NotViable: notViable, Skipped: skipped, TimedOut: timedOut, Score: score, TestEfficacy: score}
	if total == 0 && killed == 0 && survived == 0 {
		switch {
		case timedOut > 0:
			result.Status = "all_timed_out"
			result.Notes = append(result.Notes, "report exists but all observed mutations timed out")
		case notCovered > 0:
			result.Status = "not_covered_only"
			result.Notes = append(result.Notes, "report exists but only not-covered mutants were counted")
		default:
			result.Status = "no_results"
			result.Notes = append(result.Notes, "report exists but has no effective mutants")
		}
	}
	result.DenominatorHealth = denominatorHealth(result)
	result.MutationCoverage = mutationCoverage(result)
	return result, nil
}

func ParseGomu(path string) (ToolResult, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{}, err
	}
	var report struct {
		TotalMutants int `json:"totalMutants"`
		Results      []struct {
			Status string `json:"status"`
		} `json:"results"`
	}
	if json.Unmarshal(text, &report) == nil && report.TotalMutants > 0 {
		result := ToolResult{Tool: "gomu", Completed: true, Status: "ok", Total: report.TotalMutants}
		for _, item := range report.Results {
			switch item.Status {
			case "KILLED":
				result.Killed++
			case "SURVIVED":
				result.Survived++
			case "ERROR":
				result.Errors++
			case "NOT_VIABLE":
				result.NotViable++
			case "SKIPPED":
				result.Skipped++
			case "TIMEOUT", "TIMED_OUT":
				result.TimedOut++
			}
		}
		result.Score = scoreFrom(result.Killed, result.Survived)
		result.TestEfficacy = result.Score
		result.DenominatorHealth = denominatorHealth(result)
		result.MutationCoverage = mutationCoverage(result)
		return result, nil
	}
	return parseKeyValueText("gomu", string(text)), nil
}

func ParseGoMutesting(path string) (ToolResult, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{}, err
	}
	var report struct {
		Stats struct {
			TotalMutantsCount int     `json:"totalMutantsCount"`
			KilledCount       int     `json:"killedCount"`
			NotCoveredCount   int     `json:"notCoveredCount"`
			EscapedCount      int     `json:"escapedCount"`
			ErrorCount        int     `json:"errorCount"`
			SkippedCount      int     `json:"skippedCount"`
			TimeOutCount      int     `json:"timeOutCount"`
			MSI               float64 `json:"msi"`
		} `json:"stats"`
	}
	if json.Unmarshal(text, &report) == nil && report.Stats.TotalMutantsCount > 0 {
		result := ToolResult{
			Tool:       "go-mutesting",
			Completed:  true,
			Status:     "ok",
			Total:      report.Stats.TotalMutantsCount,
			Killed:     report.Stats.KilledCount,
			Survived:   report.Stats.EscapedCount,
			NotCovered: report.Stats.NotCoveredCount,
			Errors:     report.Stats.ErrorCount,
			Skipped:    report.Stats.SkippedCount,
			TimedOut:   report.Stats.TimeOutCount,
			Score:      report.Stats.MSI * 100,
		}
		result.TestEfficacy = result.Score
		result.DenominatorHealth = denominatorHealth(result)
		result.MutationCoverage = mutationCoverage(result)
		return result, nil
	}
	result := parseKeyValueText("go-mutesting", string(text))
	if result.Total == 0 {
		result.Total = regexpInt(`(?i)(\d+)\s+total`, string(text))
	}
	if result.Killed == 0 {
		result.Killed = regexpInt(`(?i)(\d+)\s+killed`, string(text))
	}
	if result.Survived == 0 {
		result.Survived = regexpInt(`(?i)(\d+)\s+survived`, string(text))
	}
	if result.TimedOut == 0 {
		result.TimedOut = regexpInt(`(?i)(\d+)\s+timed\s*out`, string(text))
	}
	if result.Score == 0 {
		result.Score = regexpFloat(`(?i)score\s+is\s+([0-9.]+)%`, string(text))
	}
	if result.Score == 0 {
		result.Score = scoreFrom(result.Killed, result.Survived)
	}
	result.Completed = true
	if result.Status == "" {
		result.Status = "ok"
	}
	result.TestEfficacy = result.Score
	result.DenominatorHealth = denominatorHealth(result)
	result.MutationCoverage = mutationCoverage(result)
	return result, nil
}

func NormalizeGremlinsTarget(target, mode string) (effective string, notComparable bool) {
	return NormalizeTarget(target, mode)
}

func NormalizeTarget(target, mode string) (effective string, notComparable bool) {
	effective = target
	if (mode == "gremlins-package-root" || mode == "package-root") && target == "./..." {
		effective = "."
		notComparable = true
	}
	return effective, notComparable
}

func ApplyTarget(result ToolResult, target, effective string, notComparable bool) ToolResult {
	return ApplyTargetMode(result, target, effective, "manifest", notComparable)
}

func ApplyTargetMode(result ToolResult, target, effective, mode string, notComparable bool) ToolResult {
	result.Target = target
	result.EffectiveTarget = effective
	result.TargetMode = mode
	result.NotComparable = notComparable
	if notComparable {
		result.Notes = append(result.Notes, "manifest target differs from effective tool target")
	}
	return result
}

func Write(path string, results []ToolResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Study{SchemaVersion: "1", Comparability: BuildComparability(results), Results: results}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func BuildComparability(results []ToolResult) Comparability {
	comp := Comparability{ApplesToApples: true, ManifestEquivalent: true}
	effectiveTargets := map[string]bool{}
	targetModes := map[string]bool{}
	for _, result := range results {
		target := result.Target
		effective := result.EffectiveTarget
		mode := result.TargetMode
		if mode == "" {
			mode = "manifest"
		}
		if target == "" && effective == "" {
			comp.ApplesToApples = false
			comp.ManifestEquivalent = false
			comp.Warnings = appendUnique(comp.Warnings, "target_metadata_missing")
			continue
		}
		if effective == "" {
			effective = target
		}
		effectiveTargets[effective] = true
		targetModes[mode] = true
		if target != effective || result.NotComparable {
			comp.ManifestEquivalent = false
		}
	}
	comp.EffectiveTargets = sortedKeys(effectiveTargets)
	comp.TargetModes = sortedKeys(targetModes)
	if len(comp.EffectiveTargets) > 1 {
		comp.ApplesToApples = false
		comp.Warnings = appendUnique(comp.Warnings, "effective_target_mismatch")
	}
	if len(comp.TargetModes) > 1 {
		comp.ApplesToApples = false
		comp.Warnings = appendUnique(comp.Warnings, "target_mode_mismatch")
	}
	if !comp.ManifestEquivalent {
		comp.Warnings = appendUnique(comp.Warnings, "effective_target_differs_from_manifest")
	}
	return comp
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func readJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func parseKeyValueText(tool, text string) ToolResult {
	result := ToolResult{Tool: tool, Completed: true, Status: "ok"}
	result.Total = regexpInt(`(?i)(?:total|mutants)\s*[:=]\s*(\d+)`, text)
	result.Killed = regexpInt(`(?i)killed\s*[:=]\s*(\d+)`, text)
	result.Survived = regexpInt(`(?i)survived\s*[:=]\s*(\d+)`, text)
	result.NotCovered = regexpInt(`(?i)(?:not_covered|uncovered)\s*[:=]\s*(\d+)`, text)
	result.TimedOut = regexpInt(`(?i)(?:timed_out|timeout|timedout)\s*[:=]\s*(\d+)`, text)
	result.Score = regexpFloat(`(?i)score\s*[:=]\s*([0-9.]+)`, text)
	if result.Score == 0 {
		result.Score = scoreFrom(result.Killed, result.Survived)
	}
	result.TestEfficacy = result.Score
	result.DenominatorHealth = denominatorHealth(result)
	result.MutationCoverage = mutationCoverage(result)
	return result
}

func mutationCoverage(result ToolResult) float64 {
	covered := result.Killed + result.Survived + result.TimedOut + result.Errors
	denominator := covered + result.NotCovered
	if denominator == 0 {
		return 0
	}
	return float64(covered) * 100 / float64(denominator)
}

func denominatorHealth(result ToolResult) DenominatorHealth {
	effective := result.Killed + result.Survived
	scoreDenominator := effective + result.TimedOut + result.Errors
	health := DenominatorHealth{
		Effective:        effective,
		ScoreDenominator: scoreDenominator,
		Killed:           result.Killed,
		Survived:         result.Survived,
		NotCovered:       result.NotCovered,
		TimedOut:         result.TimedOut,
		Errors:           result.Errors,
		Healthy:          true,
	}
	if result.Total > 0 && effective == 0 {
		health.Warnings = append(health.Warnings, "no_effective_mutants")
	}
	if effective > 0 && result.TimedOut > effective {
		health.Warnings = append(health.Warnings, "timed_out_exceeds_effective")
	}
	if effective > 0 && result.NotCovered > effective {
		health.Warnings = append(health.Warnings, "not_covered_exceeds_effective")
	}
	if effective > 0 && result.Score >= 90 && (result.TimedOut > effective || result.NotCovered > effective) {
		health.Warnings = append(health.Warnings, "high_score_poor_denominator_health")
	}
	health.Healthy = len(health.Warnings) == 0
	return health
}

func countGremlinsStatus(report map[string]any, status string) int {
	files, ok := report["files"].([]any)
	if !ok {
		return 0
	}
	var count int
	for _, file := range files {
		fileMap, ok := file.(map[string]any)
		if !ok {
			continue
		}
		mutations, ok := fileMap["mutations"].([]any)
		if !ok {
			continue
		}
		for _, mutation := range mutations {
			mutationMap, ok := mutation.(map[string]any)
			if !ok {
				continue
			}
			if mutationMap["status"] == status {
				count++
			}
		}
	}
	return count
}

func intField(values map[string]any, names ...string) int {
	for _, name := range names {
		switch value := values[name].(type) {
		case float64:
			return int(value)
		case int:
			return value
		case string:
			parsed, _ := strconv.Atoi(value)
			return parsed
		}
	}
	return 0
}

func floatField(values map[string]any, names ...string) float64 {
	for _, name := range names {
		switch value := values[name].(type) {
		case float64:
			return value
		case int:
			return float64(value)
		case string:
			parsed, _ := strconv.ParseFloat(value, 64)
			return parsed
		}
	}
	return 0
}

func regexpInt(pattern, text string) int {
	value, _ := strconv.Atoi(regexpMatch(pattern, text))
	return value
}

func regexpFloat(pattern, text string) float64 {
	value, _ := strconv.ParseFloat(regexpMatch(pattern, text), 64)
	return value
}

func regexpMatch(pattern, text string) string {
	match := regexp.MustCompile(pattern).FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func scoreFrom(killed, survived int) float64 {
	denominator := killed + survived
	if denominator == 0 {
		return 0
	}
	return float64(killed) * 100 / float64(denominator)
}
