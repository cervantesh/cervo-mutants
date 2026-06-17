package pool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/extcompare"
)

var defaultCompareNames = []string{"cobra", "pflag", "moby", "hugo", "prometheus", "terraform", "grpc-go", "echo", "logrus", "validator", "decimal", "gjson"}
var defaultCompareTools = []string{"cervomut", "gremlins", "gomu", "go-mutesting"}

type CompareOptions struct {
	ManifestPath               string
	WorkRoot                   string
	OutputRoot                 string
	Names                      []string
	Tools                      []string
	Workers                    int
	CompareTargetMode          string
	GremlinsTargetMode         string
	GremlinsTimeoutCoefficient int
	GomuWorkers                int
	GoMutestingWorkers         int
	TimeoutSeconds             int
	MinFreeMemoryMB            int
	MinFreeCommitMB            int
	KillBelowFreeMemoryMB      int
	KillBelowFreeCommitMB      int
	MaxUsedMemoryMB            int
	MaxCommittedMemoryMB       int
	MaxProcessTreeMemoryMB     int
	MemoryWaitSeconds          int
	MemoryPollSeconds          int
	GoMemoryLimit              string
	GoMaxProcs                 int
	GoFlags                    string
	Resume                     bool
	CervoBinary                string
	GremlinsBinary             string
	GomuBinary                 string
	GoMutestingBinary          string
	Runner                     CommandRunner
	Monitor                    MemoryMonitor
}

type CompareResult struct {
	Repo                string   `json:"repo"`
	Target              string   `json:"target"`
	EffectiveTarget     string   `json:"effective_target"`
	TargetMode          string   `json:"target_mode"`
	ManifestEquivalent  bool     `json:"manifest_equivalent"`
	ApplesToApplesKey   string   `json:"apples_to_apples_key"`
	Lane                string   `json:"lane"`
	Domain              string   `json:"domain"`
	Tool                string   `json:"tool"`
	Exit                int      `json:"exit"`
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
	Status              string   `json:"status"`
	Note                string   `json:"note"`
	Log                 string   `json:"log"`
}

type compareMetrics struct {
	Total               int
	Killed              int
	Survived            int
	NotCovered          int
	NotViable           int
	Errors              int
	TimedOut            int
	Score               float64
	TestEfficacy        float64
	MutationCoverage    float64
	DenominatorWarnings []string
	PartialReportUsed   bool
	Status              string
}

type toolSpec struct {
	name            string
	exe             string
	args            []string
	report          string
	partialReport   string
	parser          string
	effectiveTarget string
	targetMode      string
}

func RunCompare(ctx context.Context, opts CompareOptions) (RunSummary[CompareResult], error) {
	manifest, err := LoadManifest(opts.ManifestPath)
	if err != nil {
		return RunSummary[CompareResult]{}, err
	}
	names := opts.Names
	if len(names) == 0 {
		names = defaultCompareNames
	}
	tools := opts.Tools
	if len(tools) == 0 {
		tools = defaultCompareTools
	}
	repos := FilterRepos(manifest, names, 0)
	if err := os.MkdirAll(opts.OutputRoot, 0o755); err != nil {
		return RunSummary[CompareResult]{}, err
	}
	runner := opts.Runner
	if runner == nil {
		runner = RealCommandRunner{Monitor: opts.Monitor}
	}
	monitor := opts.Monitor
	if monitor == nil {
		monitor = systemMemoryMonitor{}
	}
	adjustMemoryThresholds(&opts, monitor)

	results := make([]CompareResult, 0)
	summaryPath := summaryPath(opts.OutputRoot)
	if opts.Resume {
		if data, readErr := os.ReadFile(summaryPath); readErr == nil {
			_ = json.Unmarshal(data, &results)
		}
	}
	toolWanted := map[string]bool{}
	for _, tool := range tools {
		if tool = strings.TrimSpace(tool); tool != "" {
			toolWanted[tool] = true
		}
	}

	for _, repo := range repos {
		repoDir := filepath.Join(opts.WorkRoot, repo.Name)
		if _, statErr := os.Stat(repoDir); statErr != nil {
			results = append(results, CompareResult{
				Repo:   repo.Name,
				Tool:   "all",
				Exit:   127,
				Status: "missing_repo",
				Note:   "repo checkout missing",
			})
			if err := writeJSON(summaryPath, results); err != nil {
				return RunSummary[CompareResult]{}, err
			}
			continue
		}
		repoOut := filepath.Join(opts.OutputRoot, repo.Name)
		if err := os.MkdirAll(repoOut, 0o755); err != nil {
			return RunSummary[CompareResult]{}, err
		}
		selectedTools := buildToolSpecs(repo, repoOut, opts)
		for _, tool := range selectedTools {
			if !toolWanted[tool.name] {
				continue
			}
			if opts.Resume && hasCompareResult(results, repo.Name, tool.name) {
				continue
			}
			_ = os.Remove(tool.report)
			logPath := filepath.Join(repoOut, tool.name+".log")
			started := time.Now()
			exit, runErr := runSimpleCommand(ctx, runner, CommandSpec{
				Path:                   tool.exe,
				Args:                   tool.args,
				Dir:                    repoDir,
				LogPath:                logPath,
				Timeout:                time.Duration(opts.TimeoutSeconds) * time.Second,
				MinFreeMemoryMB:        opts.MinFreeMemoryMB,
				MinFreeCommitMB:        opts.MinFreeCommitMB,
				KillBelowFreeMemoryMB:  opts.KillBelowFreeMemoryMB,
				KillBelowFreeCommitMB:  opts.KillBelowFreeCommitMB,
				MemoryWait:             time.Duration(opts.MemoryWaitSeconds) * time.Second,
				MemoryPoll:             time.Duration(opts.MemoryPollSeconds) * time.Second,
				MaxProcessTreeMemoryMB: opts.MaxProcessTreeMemoryMB,
				Env:                    goEnvOverrides(opts),
			})
			if runErr != nil {
				return RunSummary[CompareResult]{}, runErr
			}
			metrics, parseErr := parseToolMetrics(tool, repoOut, logPath)
			if parseErr != nil {
				return RunSummary[CompareResult]{}, parseErr
			}
			status, note := classifyCompareStatus(tool.parser, exit, metrics, logPath, tool.report)
			results = append(results, CompareResult{
				Repo:                repo.Name,
				Target:              repo.Target,
				EffectiveTarget:     tool.effectiveTarget,
				TargetMode:          tool.targetMode,
				ManifestEquivalent:  repo.Target == tool.effectiveTarget,
				ApplesToApplesKey:   tool.targetMode + ":" + tool.effectiveTarget,
				Lane:                repo.Lane,
				Domain:              repo.Domain,
				Tool:                tool.name,
				Exit:                exit,
				Seconds:             roundSeconds(started),
				Total:               metrics.Total,
				Killed:              metrics.Killed,
				Survived:            metrics.Survived,
				NotCovered:          metrics.NotCovered,
				NotViable:           metrics.NotViable,
				Errors:              metrics.Errors,
				TimedOut:            metrics.TimedOut,
				Score:               metrics.Score,
				TestEfficacy:        metrics.TestEfficacy,
				MutationCoverage:    metrics.MutationCoverage,
				DenominatorWarnings: append([]string{}, metrics.DenominatorWarnings...),
				PartialReportUsed:   metrics.PartialReportUsed,
				Status:              status,
				Note:                note,
				Log:                 logPath,
			})
			if err := writeJSON(summaryPath, results); err != nil {
				return RunSummary[CompareResult]{}, err
			}
		}
	}

	if err := writeJSON(summaryPath, results); err != nil {
		return RunSummary[CompareResult]{}, err
	}
	artifacts, err := writeCompareWorkflowArtifacts(opts.ManifestPath, opts.OutputRoot, results)
	if err != nil {
		return RunSummary[CompareResult]{Results: results, SummaryPath: summaryPath}, err
	}
	return RunSummary[CompareResult]{Results: results, SummaryPath: summaryPath, Artifacts: artifacts}, nil
}

func buildToolSpecs(repo Repo, repoOut string, opts CompareOptions) []toolSpec {
	cervoTarget := comparisonTarget(repo.Target, opts.CompareTargetMode)
	gremlinsTarget := comparisonTarget(repo.Target, gremlinsMode(opts))
	otherTarget := comparisonTarget(repo.Target, opts.CompareTargetMode)
	specs := []toolSpec{{
		name:            "cervomut",
		exe:             opts.CervoBinary,
		args:            []string{"run", cervoTarget, "--policy", "comparison-safe", "--workers", strconv.Itoa(opts.Workers), "--out", filepath.Join(repoOut, "cervomut")},
		report:          filepath.Join(repoOut, "cervomut", "mutation-report.json"),
		partialReport:   filepath.Join(repoOut, "cervomut", "partial-mutation-report.json"),
		parser:          "cervo",
		effectiveTarget: cervoTarget,
		targetMode:      targetModeForTool("cervomut", opts),
	}, {
		name:            "gremlins",
		exe:             opts.GremlinsBinary,
		args:            gremlinsArgs(gremlinsTarget, repoOut, opts),
		report:          filepath.Join(repoOut, "gremlins.json"),
		parser:          "gremlins",
		effectiveTarget: gremlinsTarget,
		targetMode:      targetModeForTool("gremlins", opts),
	}, {
		name:            "gomu",
		exe:             opts.GomuBinary,
		args:            []string{"run", otherTarget, "--workers", strconv.Itoa(opts.GomuWorkers), "--timeout", strconv.Itoa(opts.TimeoutSeconds), "--threshold", "0", "--fail-on-gate=false", "--output", "json"},
		report:          filepath.Join(filepath.Join(opts.WorkRoot, repo.Name), "mutation-report.json"),
		parser:          "gomu",
		effectiveTarget: otherTarget,
		targetMode:      targetModeForTool("gomu", opts),
	}, {
		name:            "go-mutesting",
		exe:             opts.GoMutestingBinary,
		args:            []string{"/noop", "/quiet", "/no-diffs", "/logger-summary-json", "/logger-agentic-json", "/exec-timeout:" + strconv.Itoa(opts.TimeoutSeconds), "/workers:" + strconv.Itoa(opts.GoMutestingWorkers), otherTarget},
		report:          filepath.Join(filepath.Join(opts.WorkRoot, repo.Name), "report.json"),
		parser:          "go-mutesting",
		effectiveTarget: otherTarget,
		targetMode:      targetModeForTool("go-mutesting", opts),
	}}
	return specs
}

func gremlinsArgs(target, repoOut string, opts CompareOptions) []string {
	args := []string{"unleash", target, "--workers", strconv.Itoa(opts.Workers), "--threshold-efficacy", "0", "--threshold-mcover", "0", "--output", filepath.Join(repoOut, "gremlins.json")}
	if opts.GremlinsTimeoutCoefficient > 1 {
		args = append(args, "--timeout-coefficient", strconv.Itoa(opts.GremlinsTimeoutCoefficient))
	}
	return args
}

func parseToolMetrics(tool toolSpec, repoOut, logPath string) (compareMetrics, error) {
	switch tool.parser {
	case "cervo":
		path := tool.report
		partial := false
		if _, err := os.Stat(path); err != nil {
			if _, partialErr := os.Stat(tool.partialReport); partialErr == nil {
				path = tool.partialReport
				partial = true
			}
		}
		if _, err := os.Stat(path); err != nil {
			return compareMetrics{PartialReportUsed: partial}, nil
		}
		parsed, err := extcompare.ParseCervo(path)
		if err != nil {
			return compareMetrics{}, err
		}
		return compareMetrics{
			Total:               parsed.Total,
			Killed:              parsed.Killed,
			Survived:            parsed.Survived,
			NotCovered:          parsed.NotCovered,
			Errors:              parsed.Errors,
			TimedOut:            parsed.TimedOut,
			Score:               parsed.Score,
			TestEfficacy:        parsed.TestEfficacy,
			MutationCoverage:    parsed.MutationCoverage,
			DenominatorWarnings: append([]string{}, parsed.DenominatorHealth.Warnings...),
			PartialReportUsed:   partial,
		}, nil
	case "gremlins":
		if _, err := os.Stat(tool.report); err != nil {
			return compareMetrics{}, nil
		}
		parsed, err := extcompare.ParseGremlins(tool.report)
		if err != nil {
			return compareMetrics{}, err
		}
		return compareMetrics{
			Total:               parsed.Total,
			Killed:              parsed.Killed,
			Survived:            parsed.Survived,
			NotCovered:          parsed.NotCovered,
			NotViable:           parsed.NotViable,
			Errors:              parsed.Errors,
			TimedOut:            parsed.TimedOut,
			Score:               parsed.Score,
			TestEfficacy:        parsed.TestEfficacy,
			MutationCoverage:    parsed.MutationCoverage,
			DenominatorWarnings: append([]string{}, parsed.DenominatorHealth.Warnings...),
			Status:              parsed.Status,
		}, nil
	case "gomu":
		if _, err := os.Stat(tool.report); err != nil {
			return compareMetrics{}, nil
		}
		if err := copyFile(tool.report, filepath.Join(repoOut, "gomu-mutation-report.json")); err != nil {
			return compareMetrics{}, err
		}
		parsed, err := extcompare.ParseGomu(tool.report)
		if err != nil {
			return compareMetrics{}, err
		}
		return compareMetrics{
			Total:               parsed.Total,
			Killed:              parsed.Killed,
			Survived:            parsed.Survived,
			NotCovered:          parsed.NotCovered,
			NotViable:           parsed.NotViable,
			Errors:              parsed.Errors,
			TimedOut:            parsed.TimedOut,
			Score:               parsed.Score,
			TestEfficacy:        parsed.TestEfficacy,
			MutationCoverage:    parsed.MutationCoverage,
			DenominatorWarnings: append([]string{}, parsed.DenominatorHealth.Warnings...),
		}, nil
	case "go-mutesting":
		if _, err := os.Stat(tool.report); err != nil {
			return compareMetrics{}, nil
		}
		if err := copyFile(tool.report, filepath.Join(repoOut, "go-mutesting-report.json")); err != nil {
			return compareMetrics{}, err
		}
		parsed, err := extcompare.ParseGoMutesting(tool.report)
		if err != nil {
			return compareMetrics{}, err
		}
		return compareMetrics{
			Total:               parsed.Total,
			Killed:              parsed.Killed,
			Survived:            parsed.Survived,
			NotCovered:          parsed.NotCovered,
			NotViable:           parsed.NotViable,
			Errors:              parsed.Errors,
			TimedOut:            parsed.TimedOut,
			Score:               parsed.Score,
			TestEfficacy:        parsed.TestEfficacy,
			MutationCoverage:    parsed.MutationCoverage,
			DenominatorWarnings: append([]string{}, parsed.DenominatorHealth.Warnings...),
		}, nil
	default:
		return compareMetrics{}, nil
	}
}

func classifyCompareStatus(parser string, exit int, metrics compareMetrics, logPath, reportPath string) (string, string) {
	status := "ok"
	note := ""
	switch exit {
	case 124:
		status = "timeout"
		note = "timeout"
	case 125:
		status = "skipped"
		note = "skipped before start"
	case 126:
		status = "watchdog_kill"
		note = "memory watchdog kill"
	}
	if parser == "cervo" && metrics.PartialReportUsed {
		if status == "timeout" || status == "watchdog_kill" {
			status = "partial_" + status
		}
		note = appendNote(note, "partial CervoMutants report used")
	}
	if parser == "gremlins" {
		logText := readFile(logPath)
		if strings.Contains(logText, "panic:") {
			return "panic", "panic after coverage or mutation execution"
		}
		if _, err := os.Stat(reportPath); err != nil {
			if strings.Contains(logText, "No results to report") {
				return "no_results", "Gremlins found no covered/reportable mutants"
			}
			if exit == 0 {
				return "no_report", "exit 0 but no JSON report"
			}
			return status, note
		}
		switch metrics.Status {
		case "all_timed_out":
			return "all_timed_out", "report exists but all observed mutations timed out"
		case "not_covered_only":
			return "not_covered_only", "report exists but only not-covered mutants were counted"
		case "no_results":
			return "no_results", "report exists but has no effective mutants"
		}
	}
	return status, note
}

func appendNote(note, extra string) string {
	if note == "" {
		return extra
	}
	if extra == "" {
		return note
	}
	return note + "; " + extra
}

func comparisonTarget(target, mode string) string {
	if mode == "package-root" && target == "./..." {
		return "."
	}
	return target
}

func gremlinsMode(opts CompareOptions) string {
	if opts.GremlinsTargetMode == "package-root" {
		return "package-root"
	}
	return opts.CompareTargetMode
}

func targetModeForTool(tool string, opts CompareOptions) string {
	if tool == "gremlins" && opts.GremlinsTargetMode == "package-root" {
		return "package-root"
	}
	if opts.CompareTargetMode == "" {
		return "manifest"
	}
	return opts.CompareTargetMode
}

func hasCompareResult(results []CompareResult, repo, tool string) bool {
	for _, result := range results {
		if result.Repo == repo && result.Tool == tool {
			return true
		}
	}
	return false
}

func goEnvOverrides(opts CompareOptions) []string {
	env := make([]string, 0, 3)
	if opts.GoMemoryLimit != "" {
		env = append(env, "GOMEMLIMIT="+opts.GoMemoryLimit)
	}
	if opts.GoMaxProcs > 0 {
		env = append(env, "GOMAXPROCS="+strconv.Itoa(opts.GoMaxProcs))
	}
	if opts.GoFlags != "" {
		env = append(env, "GOFLAGS="+opts.GoFlags)
	}
	return env
}

func adjustMemoryThresholds(opts *CompareOptions, monitor MemoryMonitor) {
	if monitor == nil {
		return
	}
	status, err := monitor.Status()
	if err != nil {
		return
	}
	if opts.MaxUsedMemoryMB > 0 && status.TotalMemoryMB > 0 {
		requiredFree := status.TotalMemoryMB - opts.MaxUsedMemoryMB
		if requiredFree > opts.MinFreeMemoryMB {
			opts.MinFreeMemoryMB = requiredFree
		}
		if requiredFree > opts.KillBelowFreeMemoryMB {
			opts.KillBelowFreeMemoryMB = requiredFree
		}
	}
	if opts.MaxCommittedMemoryMB > 0 && status.TotalCommitMB > 0 {
		requiredFree := status.TotalCommitMB - opts.MaxCommittedMemoryMB
		if requiredFree > opts.MinFreeCommitMB {
			opts.MinFreeCommitMB = requiredFree
		}
		if requiredFree > opts.KillBelowFreeCommitMB {
			opts.KillBelowFreeCommitMB = requiredFree
		}
	}
}
