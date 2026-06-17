package engine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/discover"
	"github.com/cervantesh/cervo-mutants/pkg/isolate"
	"github.com/cervantesh/cervo-mutants/pkg/mutator"
	"github.com/cervantesh/cervo-mutants/pkg/quarantine"
)

type Engine struct {
	cfg                  config.Config
	mutantGenerator      mutator.Generator
	suppressionEvaluator SuppressionEvaluator
	survivorRanker       SurvivorRanker
	timingMu             sync.Mutex
	checkpointMu         sync.Mutex
	checkpointScope      []Mutant
	sliceMeta            SliceMetadata
}

var (
	discoverMutantsForRun = func(e *Engine, targets []string) ([]Mutant, error) {
		return e.discoverMutants(targets)
	}
	runBaselineForRun = func(e *Engine, ctx context.Context, targets []string) (MutantResult, error) {
		return e.runBaseline(ctx, targets)
	}
	runMutantsForRun = func(e *Engine, ctx context.Context, mutants []Mutant, quarantined map[string]bool) ([]MutantResult, error) {
		return e.runMutants(ctx, mutants, quarantined)
	}
	writeReportsForRun = func(e *Engine, result RunResult) error {
		return e.writeReports(result)
	}
)

func New(cfg config.Config) *Engine {
	return NewWithOptions(cfg)
}

func NewWithOptions(cfg config.Config, opts ...EngineOption) *Engine {
	e := &Engine{
		cfg:                  cfg,
		mutantGenerator:      mutator.DefaultGenerator(),
		suppressionEvaluator: DefaultSuppressionEvaluator(cfg),
		survivorRanker:       DefaultSurvivorRanker(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

func (e *Engine) Run(ctx context.Context, req RunRequest) (result RunResult, err error) {
	defer recoverEnginePanic("run", &err)

	targets := e.runTargets(req.Targets)
	mutants, err := discoverMutantsForRun(e, targets)
	if err != nil {
		return RunResult{}, wrapStageError("discovery_error", err)
	}
	mutants, sliceMeta := e.applySlicing(mutants)
	e.sliceMeta = sliceMeta
	e.scheduleMutants(mutants)
	if e.cfg.Limits.MaxMutants > 0 && len(mutants) > e.cfg.Limits.MaxMutants {
		mutants = mutants[:e.cfg.Limits.MaxMutants]
	}
	e.setCheckpointScope(mutants)
	quarantined, expired, err := e.loadQuarantine()
	if err != nil {
		return RunResult{}, wrapStageError("environment_error", err)
	}
	result = RunResult{
		SchemaVersion: "1",
		Environment:   e.environment(len(mutants)),
		Slice:         sliceMeta,
		Checkpoint:    e.checkpoint(mutants, "final"),
		Thresholds:    map[string]any{"fail_under": e.cfg.CI.FailUnder},
		Mutants:       []MutantResult{},
		Quarantine: QuarantineStats{
			Active:        len(quarantined),
			Expired:       expired,
			Path:          e.cfg.Quarantine.Path,
			ExpireAfter:   e.cfg.Quarantine.ExpireAfter.String(),
			RequireReason: e.cfg.Quarantine.RequireReason,
			RequireOwner:  e.cfg.Quarantine.RequireOwner,
			RequireIssue:  e.cfg.Quarantine.RequireIssue,
			FailOnExpired: e.cfg.Quarantine.FailOnExpired,
			MaxRenewals:   e.cfg.Quarantine.MaxRenewals,
		},
	}
	if req.DryRun {
		return e.dryRunResult(result, mutants), nil
	}
	baselineResult, err := runBaselineForRun(e, ctx, targets)
	if err != nil && e.cfg.Tests.BaselineRequired {
		return RunResult{}, wrapStageError("runner_error", err)
	}
	_ = baselineResult
	mutantResults, err := runMutantsForRun(e, ctx, mutants, quarantined)
	if err != nil {
		return RunResult{}, wrapStageError("runner_error", err)
	}
	result.Mutants = mutantResults
	result.StoppedReason, result.LastCompletedMutant = runStopMetadata(result.Mutants)
	result.History = e.applyHistory(result.Mutants)
	e.applySurvivorRanking(result.Mutants)
	result.Summary = summarize(result.Mutants)
	result.Summary.NewSurvivors = result.History.NewSurvivors
	result.Summary.LongStandingSurvivors = result.History.LongStandingSurvivors
	if e.cfg.Baseline.Enabled {
		if prev, ok, err := e.loadBaseline(); err == nil && ok {
			result.Baseline = compareBaseline(prev, result)
		} else {
			result.Baseline = BaselineComparison{Enabled: true, CurrentScore: result.Summary.Score}
		}
	}
	if err := writeReportsForRun(e, result); err != nil {
		return RunResult{}, wrapStageError("environment_error", err)
	}
	return result, nil
}

func (e *Engine) runTargets(targets []string) []string {
	if len(targets) == 0 {
		return e.cfg.Scope.Include
	}
	return targets
}

func (e *Engine) discoverMutants(targets []string) ([]Mutant, error) {
	discovered, err := discover.Discover(targets)
	if err != nil {
		return nil, err
	}
	return e.generateMutants(discovered)
}

func (e *Engine) dryRunResult(result RunResult, mutants []Mutant) RunResult {
	for _, mutant := range mutants {
		result.Mutants = append(result.Mutants, MutantResult{MutantID: mutant.ID, Status: StatusSkipped, StatusReason: "dry-run", Mutant: mutant})
	}
	result.StoppedReason, result.LastCompletedMutant = runStopMetadata(result.Mutants)
	e.applySurvivorRanking(result.Mutants)
	result.Summary = summarize(result.Mutants)
	return result
}

func (e *Engine) Affected(ctx context.Context, req AffectedRequest) (result AffectedResult, err error) {
	defer recoverEnginePanic("affected", &err)
	discovered, err := discover.Discover(req.Targets)
	if err != nil {
		return AffectedResult{}, wrapStageError("discovery_error", err)
	}
	mutants, err := e.generateMutants(discovered)
	if err != nil {
		return AffectedResult{}, wrapStageError("discovery_error", err)
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
	result = AffectedResult{Modules: discovered.Modules, EstimatedMutants: len(mutants)}
	for pkg := range packages {
		result.Packages = append(result.Packages, pkg)
	}
	for file := range files {
		result.Files = append(result.Files, file)
	}
	return result, nil
}

func (e *Engine) Explain(ctx context.Context, req ExplainRequest) (result ExplainResult, err error) {
	defer recoverEnginePanic("explain", &err)
	if strings.TrimSpace(req.MutantID) == "" {
		return ExplainResult{}, errors.New("mutant id is required")
	}
	return ExplainResult{
		MutantID:    req.MutantID,
		Explanation: "This mutant changes program behavior. If tests still pass, the current suite executes the code without asserting the changed outcome.",
		Suggestion:  "Add an assertion that fails for the mutated expression and passes for the original behavior.",
	}, nil
}

func (e *Engine) runBaseline(ctx context.Context, targets []string) (MutantResult, error) {
	moduleDir, err := moduleForTargets(targets)
	if err != nil {
		return MutantResult{}, err
	}
	command := append([]string{}, e.cfg.Tests.Command...)
	if e.cfg.Selection.Mode == "coverage" || e.cfg.Selection.Prefilter {
		profile := e.cfg.Selection.CoverageProfile
		if !filepath.IsAbs(profile) {
			profile = filepath.Join(moduleDir, profile)
		}
		_ = os.MkdirAll(filepath.Dir(profile), 0o755)
		command = withCoverProfile(command, profile)
	}
	job := MutantJob{ID: "baseline", WorkDir: moduleDir, TestCommand: command, Timeout: e.cfg.Tests.Timeout.String()}
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
	if !plan.CoversMutant {
		return MutantResult{
			MutantID:           mutant.ID,
			Status:             StatusNotCovered,
			TestCommand:        plan.Command,
			StatusReason:       "coverage profile did not execute mutant file",
			SelectionReason:    plan.Reason,
			CoverageSource:     plan.CoverageSource,
			Mutant:             mutant,
			SuggestedTestScope: suggestedTestScope(mutant),
			NearestTests:       mutant.NearbyTests,
		}, nil
	}
	key, err := e.cacheKey(mutant, plan)
	if err != nil {
		return MutantResult{}, err
	}
	if e.cfg.Cache.Enabled && e.cfg.Cache.Mode != "off" {
		if cached, ok, err := e.getCached(key); err == nil && ok {
			result := cached
			result.PreviousStatus = result.Status
			result.Status = StatusCached
			result.StatusReason = "result reused from incremental cache"
			return result, nil
		}
	}
	workdir, command, cleanup, err := e.prepareMutation(mutant, plan.Command)
	if err != nil {
		return MutantResult{}, err
	}
	defer cleanup()
	result, err := e.runTest(ctx, MutantJob{ID: mutant.ID, Mutant: mutant, WorkDir: workdir, TestCommand: command, Timeout: e.cfg.Tests.Timeout.String()})
	result.SelectionReason = plan.Reason
	result.CoverageSource = plan.CoverageSource
	result.SuggestedTestScope = suggestedTestScope(mutant)
	result.NearestTests = mutant.NearbyTests
	applySemanticResultMetadata(&result)
	e.recordTiming(mutant.ID, result.Duration)
	if err == nil && e.cfg.Cache.Enabled && e.cfg.Cache.Mode == "incremental" {
		_ = e.putCached(key, result)
	}
	return result, err
}

func (e *Engine) prepareMutation(mutant Mutant, command []string) (string, []string, func(), error) {
	if e.cfg.Execution.Isolation == "overlay" {
		return prepareOverlayMutation(mutant, command, e.cfg.Execution.TempRoot)
	}
	workdir, err := isolate.CopyModuleWithRoot(mutant.Module, e.cfg.Execution.TempRoot)
	if err != nil {
		return "", nil, noopCleanup, err
	}
	cleanup := func() { _ = isolate.Cleanup(workdir) }
	targetFile, err := isolate.ContainedTargetPath(mutant.Module, workdir, mutant.File)
	if err != nil {
		cleanup()
		return "", nil, noopCleanup, err
	}
	if err := applyDiffReplacement(targetFile, mutant); err != nil {
		cleanup()
		return "", nil, noopCleanup, err
	}
	return workdir, command, cleanup, nil
}

func prepareOverlayMutation(mutant Mutant, command []string, tempRoot string) (string, []string, func(), error) {
	tmp, err := isolate.CreateTempDir(mutant.Module, tempRoot, "cervomut-overlay-*")
	if err != nil {
		return "", nil, noopCleanup, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	rel, err := filepath.Rel(mutant.Module, mutant.File)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		cleanup()
		return "", nil, noopCleanup, errors.New("mutant file is outside module")
	}
	mutatedPath := filepath.Join(tmp, rel)
	if err := os.MkdirAll(filepath.Dir(mutatedPath), 0o755); err != nil {
		cleanup()
		return "", nil, noopCleanup, err
	}
	data, err := os.ReadFile(mutant.File)
	if err != nil {
		cleanup()
		return "", nil, noopCleanup, err
	}
	if err := os.WriteFile(mutatedPath, data, 0o644); err != nil {
		cleanup()
		return "", nil, noopCleanup, err
	}
	if err := applyDiffReplacement(mutatedPath, mutant); err != nil {
		cleanup()
		return "", nil, noopCleanup, err
	}
	overlayPath := filepath.Join(tmp, "overlay.json")
	overlay := struct {
		Replace map[string]string `json:"Replace"`
	}{Replace: map[string]string{mutant.File: mutatedPath}}
	overlayData, err := json.MarshalIndent(overlay, "", "  ")
	if err != nil {
		cleanup()
		return "", nil, noopCleanup, err
	}
	if err := os.WriteFile(overlayPath, overlayData, 0o644); err != nil {
		cleanup()
		return "", nil, noopCleanup, err
	}
	return mutant.Module, withOverlayFlag(command, overlayPath), cleanup, nil
}

func noopCleanup() {
	// No temporary resources were allocated before the failure.
}

func withOverlayFlag(command []string, overlayPath string) []string {
	next := append([]string{}, command...)
	if len(next) >= 2 && next[0] == "go" && next[1] == "test" {
		return append(append([]string{}, next[:2]...), append([]string{"-overlay", overlayPath}, next[2:]...)...)
	}
	return next
}

type testCommandEnvPlan struct {
	Env        []string
	Applied    bool
	GoFlags    string
	GOMAXPROCS string
}

func effectiveTestCommandEnv(goos, isolation string, workers int, command, baseEnv []string) testCommandEnvPlan {
	if goos != "windows" || !isGoTestCommand(command) {
		return testCommandEnvPlan{Env: append([]string{}, baseEnv...)}
	}
	if isolation != config.IsolationTempWorkdir && workers <= 2 {
		return testCommandEnvPlan{Env: append([]string{}, baseEnv...)}
	}
	values := map[string]string{}
	order := make([]string, 0, len(baseEnv))
	for _, entry := range baseEnv {
		name, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, exists := values[name]; !exists {
			order = append(order, name)
		}
		values[name] = value
	}
	goFlags := normalizeGoFlags(values["GOFLAGS"])
	values["GOFLAGS"] = goFlags
	maxProcs := "1"
	if workers > 1 {
		maxProcs = "2"
	}
	values["GOMAXPROCS"] = maxProcs
	if !containsString(order, "GOFLAGS") {
		order = append(order, "GOFLAGS")
	}
	if !containsString(order, "GOMAXPROCS") {
		order = append(order, "GOMAXPROCS")
	}
	env := make([]string, 0, len(order))
	for _, name := range order {
		env = append(env, name+"="+values[name])
	}
	return testCommandEnvPlan{
		Env:        env,
		Applied:    true,
		GoFlags:    goFlags,
		GOMAXPROCS: maxProcs,
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalizeGoFlags(current string) string {
	fields := strings.Fields(current)
	filtered := make([]string, 0, len(fields)+1)
	skipValue := false
	for _, field := range fields {
		if skipValue {
			skipValue = false
			continue
		}
		if field == "-p" {
			skipValue = true
			continue
		}
		if strings.HasPrefix(field, "-p=") {
			continue
		}
		filtered = append(filtered, field)
	}
	filtered = append(filtered, "-p=1")
	return strings.Join(filtered, " ")
}

func (e *Engine) selectTests(mutant Mutant) TestPlan {
	command := append([]string{}, e.cfg.Tests.Command...)
	if len(command) == 0 {
		command = []string{"go", "test", "./..."}
	}
	lineCovered, fileCovered := e.coverageSignal(mutant)
	if e.cfg.Selection.Prefilter && !fileCovered {
		return TestPlan{Command: command, Reason: "coverage prefilter did not match mutant file", CoversMutant: false, CoverageSource: "package-mode-prefilter"}
	}
	if e.cfg.Selection.Mode == "package" && isGoTestCommand(command) && mutant.Package != "" {
		command = packageScopedCommand(command, mutant.Package)
		source := "unknown"
		if e.cfg.Selection.Prefilter {
			source = "package-mode-prefilter"
		}
		return TestPlan{Command: command, Reason: "package selected from mutant file", CoversMutant: true, CoverageSource: source}
	}
	if e.cfg.Selection.Mode == "coverage" {
		if lineCovered && isGoTestCommand(command) && mutant.Package != "" {
			command = packageScopedCommand(command, mutant.Package)
			return TestPlan{Command: command, Reason: "coverage profile matched mutant line", CoversMutant: true, CoverageSource: "coverage-mode"}
		}
		if fileCovered && isGoTestCommand(command) && mutant.Package != "" {
			command = packageScopedCommand(command, mutant.Package)
			return TestPlan{Command: command, Reason: "coverage profile matched mutant file; package fallback selected", CoversMutant: true, CoverageSource: "coverage-mode-file-fallback"}
		}
		return TestPlan{Command: command, Reason: "coverage profile did not match mutant file", CoversMutant: false, CoverageSource: "coverage-mode"}
	}
	return TestPlan{Command: command, Reason: "all tests selected", CoversMutant: true, CoverageSource: "unknown"}
}

func isGoTestCommand(command []string) bool {
	return len(command) >= 2 && command[0] == "go" && command[1] == "test"
}

func packageScopedCommand(command []string, pkg string) []string {
	next := append([]string{}, command[:2]...)
	replacedPackage := false
	for i := 2; i < len(command); i++ {
		arg := command[i]
		if isGoTestFlagWithSeparateValue(arg) && i+1 < len(command) {
			next = append(next, arg, command[i+1])
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			next = append(next, arg)
			continue
		}
		if !replacedPackage {
			next = append(next, pkg)
			replacedPackage = true
		}
	}
	if !replacedPackage {
		next = append(next, pkg)
	}
	return next
}

func isGoTestFlagWithSeparateValue(arg string) bool {
	if strings.Contains(arg, "=") {
		return false
	}
	switch arg {
	case "-run", "-bench", "-count", "-timeout", "-coverprofile", "-covermode", "-coverpkg", "-tags", "-cpu", "-parallel", "-shuffle":
		return true
	default:
		return false
	}
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
	envPlan := effectiveTestCommandEnv(runtime.GOOS, e.cfg.Execution.Isolation, e.workerCount(0), job.TestCommand, os.Environ())
	if envPlan.Applied {
		cmd.Env = envPlan.Env
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Start()
	handle := noopProcessLimitHandle()
	if err == nil {
		handle, err = applyProcessLimits(cmd, e.cfg.Execution.Resources)
		if err != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}
	if err == nil {
		err = cmd.Wait()
	}
	limitStats := handle.Stats()
	handle.Cleanup()
	text := output.String()
	if max := e.cfg.Reports.MaxOutputBytes; max > 0 && len(text) > max {
		text = text[:max]
	}
	status := StatusKilled
	failureKind := ""
	reason := "tests failed with mutant applied"
	if memoryLimitExceeded(err, cmd.ProcessState, e.cfg.Execution.Resources, text) {
		status = StatusMemoryKilled
		failureKind = "memory_limit_exceeded"
		reason = "test process exceeded the configured memory limit"
	} else if runCtx.Err() == context.DeadlineExceeded {
		status = StatusTimedOut
		failureKind = "timeout"
		reason = "test command timed out"
	} else if err == nil {
		status = StatusSurvived
		reason = "tests passed with mutant applied"
	} else if errors.Is(err, errProcessLimitUnsupported) {
		status = StatusSkippedResource
		failureKind = "resource_limit_unsupported"
		reason = "configured process resource limits are not supported on this platform"
	} else if !strings.Contains(text, "FAIL") {
		status = StatusCompileError
		failureKind = classifyFailure(text, err)
		reason = "test command failed before running assertions"
	}
	return MutantResult{
		MutantID:        job.Mutant.ID,
		Status:          status,
		FailureKind:     failureKind,
		MemoryPeakBytes: limitStats.PeakProcessMemoryBytes,
		Duration:        time.Since(start),
		TestCommand:     job.TestCommand,
		StatusReason:    reason,
		Output:          text,
		Mutant:          job.Mutant,
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

func classifyFailure(output string, err error) string {
	text := strings.ToLower(output)
	switch {
	case strings.Contains(text, "panic:"):
		return "test_panic"
	case strings.Contains(text, "build failed"), strings.Contains(text, "compilation failed"), strings.Contains(text, "undefined:"), strings.Contains(text, "syntax error"):
		return "compile_error"
	case strings.Contains(text, "no such file or directory"), strings.Contains(text, "cannot find"):
		return "environment_error"
	case err != nil:
		return "runner_error"
	default:
		return ""
	}
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

func (e *Engine) discoverForTest(targets []string) (discover.Result, error) {
	return discover.Discover(targets)
}

func (e *Engine) cacheKeyForTest(mutant Mutant, plan TestPlan) (string, error) {
	return e.cacheKey(mutant, plan)
}

func (e *Engine) cacheKey(mutant Mutant, plan TestPlan) (string, error) {
	parts := []string{
		"v2",
		mutant.Fingerprint,
		mutant.File,
		mutant.Package,
		e.cfg.Mutators.Profile,
		e.cfg.Selection.Mode,
		strings.Join(plan.Command, "\x00"),
		runtime.Version(),
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		if digest, err := digestFile(filepath.Join(mutant.Module, name)); err == nil {
			parts = append(parts, name+"="+digest)
		}
	}
	sourceDigest, err := digestFile(mutant.File)
	if err != nil {
		return "", err
	}
	parts = append(parts, "source="+sourceDigest)
	testDigests, err := e.testDigests(mutant.Module, mutant.Package)
	if err != nil {
		return "", err
	}
	parts = append(parts, testDigests...)
	configData, _ := json.Marshal(e.cfg)
	parts = append(parts, "config="+digestBytes(configData))
	return digestBytes([]byte(strings.Join(parts, "\x00"))), nil
}

func (e *Engine) testDigests(moduleDir, pkg string) ([]string, error) {
	dir := moduleDir
	if pkg != "." && strings.HasPrefix(pkg, "./") {
		dir = filepath.Join(moduleDir, filepath.FromSlash(strings.TrimPrefix(pkg, "./")))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var digests []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		digest, err := digestFile(path)
		if err != nil {
			return nil, err
		}
		digests = append(digests, "test:"+entry.Name()+"="+digest)
	}
	sort.Strings(digests)
	return digests, nil
}

func digestFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func digestBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (e *Engine) coverageMentions(mutant Mutant) bool {
	lineCovered, _ := e.coverageSignal(mutant)
	return lineCovered
}

func (e *Engine) coverageSignal(mutant Mutant) (lineCovered bool, fileCovered bool) {
	profile := e.cfg.Selection.CoverageProfile
	if !filepath.IsAbs(profile) {
		profile = filepath.Join(mutant.Module, profile)
	}
	data, err := os.ReadFile(profile)
	if err != nil {
		return false, false
	}
	rel, _ := filepath.Rel(mutant.Module, mutant.File)
	rel = filepath.ToSlash(rel)
	base := filepath.Base(mutant.File)
	return coverageDataSignal(string(data), rel, base, mutant.Line)
}

func coverageDataSignal(data, rel, base string, mutantLine int) (lineCovered bool, fileCovered bool) {
	parseable := false
	for _, line := range strings.Split(data, "\n") {
		file, startLine, endLine, count, ok := parseCoverageProfileLine(line)
		if !ok {
			continue
		}
		parseable = true
		if count <= 0 || !coverageFileMatches(file, rel, base) {
			continue
		}
		fileCovered = true
		if mutantLine >= startLine && mutantLine <= endLine {
			lineCovered = true
		}
	}
	if !parseable {
		fileCovered = fallbackCoverageMentions(data, rel, base)
		lineCovered = fileCovered
	}
	return lineCovered, fileCovered
}

func fallbackCoverageMentions(data, rel, base string) bool {
	for _, line := range strings.Split(data, "\n") {
		if strings.Contains(line, rel+":") || strings.Contains(line, base+":") {
			return true
		}
	}
	return false
}

func parseCoverageProfileLine(line string) (string, int, int, int, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "mode:") {
		return "", 0, 0, 0, false
	}
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return "", 0, 0, 0, false
	}
	colon := strings.LastIndex(fields[0], ":")
	if colon < 0 {
		return "", 0, 0, 0, false
	}
	file := filepath.ToSlash(fields[0][:colon])
	span := fields[0][colon+1:]
	comma := strings.Index(span, ",")
	if comma < 0 {
		return "", 0, 0, 0, false
	}
	startLine, ok := parseCoverageLineNumber(span[:comma])
	if !ok {
		return "", 0, 0, 0, false
	}
	endLine, ok := parseCoverageLineNumber(span[comma+1:])
	if !ok {
		return "", 0, 0, 0, false
	}
	count, err := strconv.Atoi(fields[2])
	if err != nil {
		return "", 0, 0, 0, false
	}
	return file, startLine, endLine, count, true
}

func parseCoverageLineNumber(value string) (int, bool) {
	dot := strings.Index(value, ".")
	if dot < 0 {
		return 0, false
	}
	line, err := strconv.Atoi(value[:dot])
	return line, err == nil
}

func coverageFileMatches(profileFile, rel, base string) bool {
	profileFile = filepath.ToSlash(profileFile)
	return profileFile == rel || strings.HasSuffix(profileFile, "/"+rel) || filepath.Base(profileFile) == base
}

func withCoverProfile(command []string, profile string) []string {
	if len(command) < 2 || command[0] != "go" || command[1] != "test" {
		return command
	}
	next := append([]string{}, command...)
	for _, arg := range next[2:] {
		if strings.HasPrefix(arg, "-coverprofile") {
			return next
		}
	}
	return append(next[:2], append([]string{"-coverprofile", profile}, next[2:]...)...)
}

func (e *Engine) recordTiming(mutantID string, duration time.Duration) {
	if !e.cfg.Selection.UseTimings || e.cfg.Selection.TimingsPath == "" || mutantID == "" {
		return
	}
	e.timingMu.Lock()
	defer e.timingMu.Unlock()
	path := e.cfg.Selection.TimingsPath
	if !filepath.IsAbs(path) {
		moduleDir := "."
		path = filepath.Join(moduleDir, path)
	}
	timings := map[string]int64{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &timings)
	}
	timings[mutantID] = duration.Milliseconds()
	if timings[mutantID] == 0 {
		timings[mutantID] = 1
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(timings, "", "  ")
	if err == nil {
		_ = os.WriteFile(path, data, 0o644)
	}
}
