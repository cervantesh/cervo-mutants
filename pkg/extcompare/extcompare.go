package extcompare

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

type ToolResult struct {
	Tool       string   `json:"tool"`
	Completed  bool     `json:"completed"`
	Total      int      `json:"total"`
	Killed     int      `json:"killed"`
	Survived   int      `json:"survived"`
	NotCovered int      `json:"not_covered"`
	NotViable  int      `json:"not_viable"`
	Errors     int      `json:"errors"`
	Skipped    int      `json:"skipped"`
	TimedOut   int      `json:"timed_out"`
	Score      float64  `json:"score"`
	Notes      []string `json:"notes,omitempty"`
}

type Study struct {
	SchemaVersion string       `json:"schema_version"`
	Results       []ToolResult `json:"results"`
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
		} `json:"summary"`
	}
	if err := readJSON(path, &report); err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Tool:       "cervo-mutant",
		Completed:  true,
		Total:      report.Summary.Total,
		Killed:     report.Summary.Killed,
		Survived:   report.Summary.Survived,
		NotCovered: report.Summary.NotCovered,
		TimedOut:   report.Summary.TimedOut,
		Errors:     report.Summary.CompileError,
		Skipped:    report.Summary.Skipped,
		Score:      report.Summary.Score,
	}, nil
}

func ParseGremlins(path string) (ToolResult, error) {
	var report map[string]any
	if err := readJSON(path, &report); err != nil {
		return ToolResult{}, err
	}
	total := intField(report, "total_mutants", "total", "mutants")
	killed := intField(report, "killed", "killed_mutants")
	survived := intField(report, "survived", "survived_mutants")
	notCovered := intField(report, "not_covered", "notCovered", "uncovered")
	score := floatField(report, "mutation_score", "score")
	if score == 0 {
		score = scoreFrom(killed, survived)
	}
	return ToolResult{Tool: "gremlins", Completed: true, Total: total, Killed: killed, Survived: survived, NotCovered: notCovered, Score: score}, nil
}

func ParseGomu(path string) (ToolResult, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{}, err
	}
	return parseKeyValueText("gomu", string(text)), nil
}

func ParseGoMutesting(path string) (ToolResult, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{}, err
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
	return result, nil
}

func Write(path string, results []ToolResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Study{SchemaVersion: "1", Results: results}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
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
	result := ToolResult{Tool: tool, Completed: true}
	result.Total = regexpInt(`(?i)(?:total|mutants)\s*[:=]\s*(\d+)`, text)
	result.Killed = regexpInt(`(?i)killed\s*[:=]\s*(\d+)`, text)
	result.Survived = regexpInt(`(?i)survived\s*[:=]\s*(\d+)`, text)
	result.NotCovered = regexpInt(`(?i)(?:not_covered|uncovered)\s*[:=]\s*(\d+)`, text)
	result.TimedOut = regexpInt(`(?i)(?:timed_out|timeout|timedout)\s*[:=]\s*(\d+)`, text)
	result.Score = regexpFloat(`(?i)score\s*[:=]\s*([0-9.]+)`, text)
	if result.Score == 0 {
		result.Score = scoreFrom(result.Killed, result.Survived)
	}
	return result
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
