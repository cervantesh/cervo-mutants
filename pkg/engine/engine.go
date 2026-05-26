package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/config"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/discover"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/isolate"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/mutator"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/quarantine"
)

type Engine struct {
	cfg config.Config
}

func New(cfg config.Config) *Engine {
	return &Engine{cfg: cfg}
}

func (e *Engine) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	targets := req.Targets
	if len(targets) == 0 {
		targets = e.cfg.Scope.Include
	}
	discovered, err := discover.Discover(targets)
	if err != nil {
		return RunResult{}, err
	}
	mutants, err := e.generateMutants(discovered)
	if err != nil {
		return RunResult{}, err
	}
	sort.Slice(mutants, func(i, j int) bool { return mutants[i].ID < mutants[j].ID })
	if e.cfg.Limits.MaxMutants > 0 && len(mutants) > e.cfg.Limits.MaxMutants {
		mutants = mutants[:e.cfg.Limits.MaxMutants]
	}
	quarantined, expired, err := e.loadQuarantine()
	if err != nil {
		return RunResult{}, err
	}
	result := RunResult{
		SchemaVersion: "1",
		Thresholds:    map[string]any{"fail_under": e.cfg.CI.FailUnder},
		Mutants:       []MutantResult{},
		Quarantine:    QuarantineStats{Active: len(quarantined), Expired: expired},
	}
	if req.DryRun {
		for _, mutant := range mutants {
			result.Mutants = append(result.Mutants, MutantResult{MutantID: mutant.ID, Status: StatusSkipped, StatusReason: "dry-run", Mutant: mutant})
		}
		result.Summary = summarize(result.Mutants)
		return result, nil
	}
	baselineResult, err := e.runBaseline(ctx, targets)
	if err != nil && e.cfg.Tests.BaselineRequired {
		return RunResult{}, err
	}
	_ = baselineResult
	start := time.Now()
	for _, mutant := range mutants {
		if quarantined[mutant.ID] {
			result.Mutants = append(result.Mutants, MutantResult{MutantID: mutant.ID, Status: StatusQuarantined, StatusReason: "mutant is in active quarantine", Mutant: mutant})
			continue
		}
		if e.cfg.Execution.Budget > 0 && time.Since(start) >= e.cfg.Execution.Budget {
			result.Mutants = append(result.Mutants, MutantResult{MutantID: mutant.ID, Status: StatusSkipped, StatusReason: "budget exhausted", Mutant: mutant})
			continue
		}
		mutantResult, err := e.runMutant(ctx, mutant)
		if err != nil {
			return RunResult{}, err
		}
		result.Mutants = append(result.Mutants, mutantResult)
	}
	result.Summary = summarize(result.Mutants)
	if e.cfg.Baseline.Enabled {
		if prev, ok, err := e.loadBaseline(); err == nil && ok {
			result.Baseline = compareBaseline(prev, result)
		} else {
			result.Baseline = BaselineComparison{Enabled: true, CurrentScore: result.Summary.Score}
		}
	}
	if err := e.writeReports(result); err != nil {
		return RunResult{}, err
	}
	return result, nil
}

func (e *Engine) Affected(ctx context.Context, req AffectedRequest) (AffectedResult, error) {
	discovered, err := discover.Discover(req.Targets)
	if err != nil {
		return AffectedResult{}, err
	}
	mutants, err := e.generateMutants(discovered)
	if err != nil {
		return AffectedResult{}, err
	}
	packages := map[string]bool{}
	files := map[string]bool{}
	for _, file := range discovered.Files {
		if file.IsTest {
			continue
		}
		packages[file.Package] = true
		files[file.Path] = true
	}
	result := AffectedResult{Modules: discovered.Modules, EstimatedMutants: len(mutants)}
	for pkg := range packages {
		result.Packages = append(result.Packages, pkg)
	}
	for file := range files {
		result.Files = append(result.Files, file)
	}
	return result, nil
}

func (e *Engine) Explain(ctx context.Context, req ExplainRequest) (ExplainResult, error) {
	if strings.TrimSpace(req.MutantID) == "" {
		return ExplainResult{}, errors.New("mutant id is required")
	}
	return ExplainResult{
		MutantID:    req.MutantID,
		Explanation: "This mutant changes program behavior. If tests still pass, the current suite executes the code without asserting the changed outcome.",
		Suggestion:  "Add an assertion that fails for the mutated expression and passes for the original behavior.",
	}, nil
}

func (e *Engine) generateMutants(discovered discover.Result) ([]Mutant, error) {
	var mutants []Mutant
	for _, file := range discovered.Files {
		if file.IsTest {
			continue
		}
		data, err := os.ReadFile(file.Path)
		if err != nil {
			return nil, err
		}
		generated, err := mutator.Generate(file.Package, file.Path, data, e.cfg.Mutators.Profile)
		if err != nil {
			return nil, err
		}
		for i := range generated {
			generated[i].Module = file.ModuleDir
			generated[i].Package = file.Package
			mutants = append(mutants, Mutant{
				ID:          generated[i].ID,
				Module:      generated[i].Module,
				Package:     generated[i].Package,
				File:        generated[i].File,
				Line:        generated[i].Line,
				Function:    generated[i].Function,
				Operator:    generated[i].Operator,
				Original:    generated[i].Original,
				Mutated:     generated[i].Mutated,
				StartOffset: generated[i].StartOffset,
				EndOffset:   generated[i].EndOffset,
				Diff:        generated[i].Diff,
				Fingerprint: generated[i].Fingerprint,
				Hint:        generated[i].Hint,
			})
		}
	}
	return mutants, nil
}

func (e *Engine) runBaseline(ctx context.Context, targets []string) (MutantResult, error) {
	moduleDir, err := moduleForTargets(targets)
	if err != nil {
		return MutantResult{}, err
	}
	job := MutantJob{ID: "baseline", WorkDir: moduleDir, TestCommand: e.cfg.Tests.Command, Timeout: e.cfg.Tests.Timeout.String()}
	result, err := e.runTest(ctx, job)
	if err != nil {
		return MutantResult{}, err
	}
	if result.Status != StatusSurvived {
		return result, errors.New("baseline tests failed before mutation")
	}
	return result, nil
}

func (e *Engine) runMutant(ctx context.Context, mutant Mutant) (MutantResult, error) {
	plan := e.selectTests(mutant)
	key := localKey(mutant.Fingerprint, strings.Join(plan.Command, " "), e.cfg.Mutators.Profile)
	if e.cfg.Cache.Enabled && e.cfg.Cache.Mode != "off" {
		if cached, ok, err := e.getCached(key); err == nil && ok {
			result := cached
			result.Status = StatusCached
			result.StatusReason = "result reused from incremental cache"
			return result, nil
		}
	}
	workdir, err := isolate.CopyModule(mutant.Module)
	if err != nil {
		return MutantResult{}, err
	}
	defer isolate.Cleanup(workdir)
	rel, err := filepath.Rel(mutant.Module, mutant.File)
	if err != nil {
		return MutantResult{}, err
	}
	targetFile := filepath.Join(workdir, rel)
	if err := applyDiffReplacement(targetFile, mutant); err != nil {
		return MutantResult{}, err
	}
	result, err := e.runTest(ctx, MutantJob{ID: mutant.ID, Mutant: mutant, WorkDir: workdir, TestCommand: plan.Command, Timeout: e.cfg.Tests.Timeout.String()})
	if err == nil && e.cfg.Cache.Enabled && e.cfg.Cache.Mode == "incremental" {
		_ = e.putCached(key, result)
	}
	return result, err
}

func (e *Engine) selectTests(mutant Mutant) TestPlan {
	command := append([]string{}, e.cfg.Tests.Command...)
	if len(command) == 0 {
		command = []string{"go", "test", "./..."}
	}
	if e.cfg.Selection.Mode == "package" && len(command) >= 3 && command[0] == "go" && command[1] == "test" && mutant.Package != "" {
		command[2] = mutant.Package
		return TestPlan{Command: command, Reason: "package selected from mutant file"}
	}
	if e.cfg.Selection.Mode == "coverage" {
		return TestPlan{Command: command, Reason: "coverage timing data unavailable; package fallback selected"}
	}
	return TestPlan{Command: command, Reason: "all tests selected"}
}

func (e *Engine) runTest(ctx context.Context, job MutantJob) (MutantResult, error) {
	timeout := e.cfg.Tests.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if len(job.TestCommand) == 0 {
		return MutantResult{}, errors.New("test command is empty")
	}
	start := time.Now()
	cmd := exec.CommandContext(runCtx, job.TestCommand[0], job.TestCommand[1:]...)
	cmd.Dir = job.WorkDir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	text := output.String()
	if max := e.cfg.Reports.MaxOutputBytes; max > 0 && len(text) > max {
		text = text[:max]
	}
	status := StatusKilled
	reason := "tests failed with mutant applied"
	if runCtx.Err() == context.DeadlineExceeded {
		status = StatusTimedOut
		reason = "test command timed out"
	} else if err == nil {
		status = StatusSurvived
		reason = "tests passed with mutant applied"
	} else if !strings.Contains(text, "FAIL") {
		status = StatusCompileError
		reason = "test command failed before running assertions"
	}
	return MutantResult{
		MutantID:     job.Mutant.ID,
		Status:       status,
		Duration:     time.Since(start),
		TestCommand:  job.TestCommand,
		StatusReason: reason,
		Output:       text,
		Mutant:       job.Mutant,
	}, nil
}

func applyDiffReplacement(path string, mutant Mutant) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if mutant.StartOffset < 0 || mutant.EndOffset > len(data) || mutant.StartOffset >= mutant.EndOffset {
		return errors.New("mutant patch offsets are invalid")
	}
	if !strings.Contains(string(data[mutant.StartOffset:mutant.EndOffset]), mutant.Original) {
		return errors.New("original token not found at mutant patch offsets")
	}
	segment := string(data[mutant.StartOffset:mutant.EndOffset])
	replaced := strings.Replace(segment, mutant.Original, mutant.Mutated, 1)
	text := string(data[:mutant.StartOffset]) + replaced + string(data[mutant.EndOffset:])
	return os.WriteFile(path, []byte(text), 0o644)
}

func moduleForTargets(targets []string) (string, error) {
	if len(targets) == 0 {
		return os.Getwd()
	}
	root := strings.TrimSuffix(targets[0], "/...")
	if root == "./..." {
		root = "."
	}
	return filepath.Abs(root)
}

func summarize(results []MutantResult) Summary {
	var s Summary
	s.Total = len(results)
	for _, result := range results {
		switch result.Status {
		case StatusKilled:
			s.Killed++
		case StatusSurvived:
			s.Survived++
		case StatusTimedOut:
			s.TimedOut++
		case StatusCompileError:
			s.CompileError++
		case StatusSkipped:
			s.Skipped++
		case StatusIgnored:
			s.Ignored++
		case StatusQuarantined:
			s.Quarantined++
		case StatusCached:
			s.Cached++
		}
	}
	denom := s.Total - s.Ignored - s.Quarantined - s.Skipped
	if denom > 0 {
		s.Score = float64(s.Killed) / float64(denom) * 100
		s.EffectiveScore = s.Score
	}
	return s
}

func (e *Engine) writeReports(result RunResult) error {
	if e.cfg.Reports.Output == "" {
		return nil
	}
	if err := os.MkdirAll(e.cfg.Reports.Output, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(e.cfg.Reports.Output, "mutation-report.json"), data, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.cfg.Reports.Output, "summary.txt"), []byte("Mutation score generated by cervomut\n"), 0o644)
}

func (e *Engine) loadQuarantine() (map[string]bool, int, error) {
	active := map[string]bool{}
	if !e.cfg.Quarantine.Enabled {
		return active, 0, nil
	}
	data, err := os.ReadFile(e.cfg.Quarantine.Path)
	if os.IsNotExist(err) {
		return active, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var entries []quarantine.Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, 0, err
	}
	policy := quarantine.Policy{
		RequireReason: e.cfg.Quarantine.RequireReason,
		RequireOwner:  e.cfg.Quarantine.RequireOwner,
		RequireIssue:  e.cfg.Quarantine.RequireIssue,
		FailOnExpired: e.cfg.Quarantine.FailOnExpired,
		MaxRenewals:   e.cfg.Quarantine.MaxRenewals,
	}
	now := time.Now()
	if err := quarantine.Validate(entries, policy, now); err != nil {
		return nil, 0, err
	}
	expired := 0
	for _, entry := range entries {
		if entry.ExpiresAt.After(now) {
			active[entry.MutantID] = true
		} else {
			expired++
		}
	}
	return active, expired, nil
}

func (e *Engine) loadBaseline() (RunResult, bool, error) {
	data, err := os.ReadFile(e.cfg.Baseline.Path)
	if os.IsNotExist(err) {
		return RunResult{}, false, nil
	}
	if err != nil {
		return RunResult{}, false, err
	}
	var result RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return RunResult{}, false, err
	}
	return result, true, nil
}

func compareBaseline(previous, current RunResult) BaselineComparison {
	seen := map[string]Status{}
	for _, mutant := range previous.Mutants {
		seen[mutant.MutantID] = mutant.Status
	}
	comparison := BaselineComparison{Enabled: true, PreviousScore: previous.Summary.Score, CurrentScore: current.Summary.Score, Regression: current.Summary.Score < previous.Summary.Score}
	for _, mutant := range current.Mutants {
		if mutant.Status == StatusSurvived && seen[mutant.MutantID] != StatusSurvived {
			comparison.NewSurvivors = append(comparison.NewSurvivors, mutant.MutantID)
		}
	}
	return comparison
}

func (e *Engine) getCached(key string) (MutantResult, bool, error) {
	data, err := os.ReadFile(filepath.Join(e.cfg.Cache.Path, key+".json"))
	if os.IsNotExist(err) {
		return MutantResult{}, false, nil
	}
	if err != nil {
		return MutantResult{}, false, err
	}
	var result MutantResult
	if err := json.Unmarshal(data, &result); err != nil {
		return MutantResult{}, false, err
	}
	return result, true, nil
}

func (e *Engine) putCached(key string, result MutantResult) error {
	if err := os.MkdirAll(e.cfg.Cache.Path, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.cfg.Cache.Path, key+".json"), data, 0o644)
}

func localKey(parts ...string) string {
	return strings.NewReplacer("\\", "/", " ", "_", ":", "_").Replace(strings.Join(parts, "_"))
}
