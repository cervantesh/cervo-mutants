package engine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/config"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/discover"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/isolate"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/mutator"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/quarantine"
)

type Engine struct {
	cfg      config.Config
	timingMu sync.Mutex
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
	mutantResults, err := e.runMutants(ctx, mutants, quarantined)
	if err != nil {
		return RunResult{}, err
	}
	result.Mutants = mutantResults
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

func (e *Engine) runMutants(ctx context.Context, mutants []Mutant, quarantined map[string]bool) ([]MutantResult, error) {
	workers := e.workerCount(len(mutants))
	if workers <= 1 {
		return e.runMutantsSerial(ctx, mutants, quarantined)
	}
	return e.runMutantsParallel(ctx, mutants, quarantined, workers)
}

func (e *Engine) runMutantsSerial(ctx context.Context, mutants []Mutant, quarantined map[string]bool) ([]MutantResult, error) {
	results := make([]MutantResult, 0, len(mutants))
	start := time.Now()
	for _, mutant := range mutants {
		if quarantined[mutant.ID] {
			results = append(results, MutantResult{MutantID: mutant.ID, Status: StatusQuarantined, StatusReason: "mutant is in active quarantine", Mutant: mutant})
			continue
		}
		if e.budgetExhausted(start) {
			results = append(results, MutantResult{MutantID: mutant.ID, Status: StatusSkipped, StatusReason: "budget exhausted", Mutant: mutant})
			continue
		}
		mutantResult, err := e.runMutant(ctx, mutant)
		if err != nil {
			return nil, err
		}
		results = append(results, mutantResult)
	}
	return results, nil
}

type indexedMutant struct {
	index  int
	mutant Mutant
}

type indexedResult struct {
	index  int
	result MutantResult
	err    error
}

func (e *Engine) runMutantsParallel(ctx context.Context, mutants []Mutant, quarantined map[string]bool, workers int) ([]MutantResult, error) {
	results := make([]MutantResult, len(mutants))
	jobs := make(chan indexedMutant, len(mutants))
	done := make(chan indexedResult, len(mutants))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					done <- indexedResult{index: job.index, err: ctx.Err()}
					continue
				}
				result, err := e.runMutant(ctx, job.mutant)
				done <- indexedResult{index: job.index, result: result, err: err}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(done)
	}()

	dispatched := 0
	start := time.Now()
	for i, mutant := range mutants {
		if quarantined[mutant.ID] {
			results[i] = MutantResult{MutantID: mutant.ID, Status: StatusQuarantined, StatusReason: "mutant is in active quarantine", Mutant: mutant}
			continue
		}
		if e.budgetExhausted(start) {
			results[i] = MutantResult{MutantID: mutant.ID, Status: StatusSkipped, StatusReason: "budget exhausted", Mutant: mutant}
			continue
		}
		dispatched++
		jobs <- indexedMutant{index: i, mutant: mutant}
	}
	close(jobs)

	var firstErr error
	for item := range done {
		dispatched--
		if item.err != nil && firstErr == nil {
			firstErr = item.err
			cancel()
		}
		results[item.index] = item.result
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func (e *Engine) budgetExhausted(start time.Time) bool {
	return e.cfg.Execution.Budget > 0 && time.Since(start) >= e.cfg.Execution.Budget
}

func (e *Engine) workerCount(mutants int) int {
	workers := e.cfg.Execution.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > mutants && mutants > 0 {
		workers = mutants
	}
	if workers < 1 {
		workers = 1
	}
	return workers
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
			id, fingerprint := e.stableMutantIdentity(generated[i])
			mutants = append(mutants, Mutant{
				ID:          id,
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
				Fingerprint: fingerprint,
				Hint:        generated[i].Hint,
				Description: generated[i].Description,
				NearbyTests: nearbyTests(file.ModuleDir, file.Path),
			})
		}
	}
	return mutants, nil
}

func nearbyTests(moduleDir, sourceFile string) []string {
	dir := filepath.Dir(sourceFile)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var tests []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		rel, err := filepath.Rel(moduleDir, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			rel = path
		}
		tests = append(tests, filepath.ToSlash(rel))
	}
	sort.Strings(tests)
	return tests
}

func (e *Engine) stableMutantIdentity(mutant mutator.Mutant) (string, string) {
	rel, err := filepath.Rel(mutant.Module, mutant.File)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		rel = filepath.Base(mutant.File)
	}
	rel = filepath.ToSlash(rel)
	fp := digestBytes([]byte(strings.Join([]string{
		rel,
		strconv.Itoa(mutant.Line),
		strconv.Itoa(mutant.StartOffset),
		strconv.Itoa(mutant.EndOffset),
		mutant.Operator,
		mutant.Original,
		mutant.Mutated,
		mutant.Diff,
	}, "\x00")))
	return fmt.Sprintf("%s:%d:%s:%s", rel, mutant.Line, mutant.Operator, fp[:12]), fp
}

func (e *Engine) runBaseline(ctx context.Context, targets []string) (MutantResult, error) {
	moduleDir, err := moduleForTargets(targets)
	if err != nil {
		return MutantResult{}, err
	}
	command := append([]string{}, e.cfg.Tests.Command...)
	if e.cfg.Selection.Mode == "coverage" {
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
	if e.cfg.Selection.Mode == "coverage" && !plan.CoversMutant {
		return MutantResult{
			MutantID:     mutant.ID,
			Status:       StatusNotCovered,
			TestCommand:  plan.Command,
			StatusReason: "coverage profile did not execute mutant file",
			Mutant:       mutant,
		}, nil
	}
	key, err := e.cacheKey(mutant, plan)
	if err != nil {
		return MutantResult{}, err
	}
	if e.cfg.Cache.Enabled && e.cfg.Cache.Mode != "off" {
		if cached, ok, err := e.getCached(key); err == nil && ok {
			result := cached
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
	e.recordTiming(mutant.ID, result.Duration)
	if err == nil && e.cfg.Cache.Enabled && e.cfg.Cache.Mode == "incremental" {
		_ = e.putCached(key, result)
	}
	return result, err
}

func (e *Engine) prepareMutation(mutant Mutant, command []string) (string, []string, func(), error) {
	if e.cfg.Execution.Isolation == "overlay" {
		return prepareOverlayMutation(mutant, command)
	}
	workdir, err := isolate.CopyModule(mutant.Module)
	if err != nil {
		return "", nil, func() {}, err
	}
	cleanup := func() { _ = isolate.Cleanup(workdir) }
	targetFile, err := isolate.ContainedTargetPath(mutant.Module, workdir, mutant.File)
	if err != nil {
		cleanup()
		return "", nil, func() {}, err
	}
	if err := applyDiffReplacement(targetFile, mutant); err != nil {
		cleanup()
		return "", nil, func() {}, err
	}
	return workdir, command, cleanup, nil
}

func prepareOverlayMutation(mutant Mutant, command []string) (string, []string, func(), error) {
	tmp, err := os.MkdirTemp("", "cervomut-overlay-*")
	if err != nil {
		return "", nil, func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	rel, err := filepath.Rel(mutant.Module, mutant.File)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		cleanup()
		return "", nil, func() {}, errors.New("mutant file is outside module")
	}
	mutatedPath := filepath.Join(tmp, rel)
	if err := os.MkdirAll(filepath.Dir(mutatedPath), 0o755); err != nil {
		cleanup()
		return "", nil, func() {}, err
	}
	data, err := os.ReadFile(mutant.File)
	if err != nil {
		cleanup()
		return "", nil, func() {}, err
	}
	if err := os.WriteFile(mutatedPath, data, 0o644); err != nil {
		cleanup()
		return "", nil, func() {}, err
	}
	if err := applyDiffReplacement(mutatedPath, mutant); err != nil {
		cleanup()
		return "", nil, func() {}, err
	}
	overlayPath := filepath.Join(tmp, "overlay.json")
	overlay := struct {
		Replace map[string]string `json:"Replace"`
	}{Replace: map[string]string{mutant.File: mutatedPath}}
	overlayData, err := json.MarshalIndent(overlay, "", "  ")
	if err != nil {
		cleanup()
		return "", nil, func() {}, err
	}
	if err := os.WriteFile(overlayPath, overlayData, 0o644); err != nil {
		cleanup()
		return "", nil, func() {}, err
	}
	return mutant.Module, withOverlayFlag(command, overlayPath), cleanup, nil
}

func withOverlayFlag(command []string, overlayPath string) []string {
	next := append([]string{}, command...)
	if len(next) >= 2 && next[0] == "go" && next[1] == "test" {
		return append(append([]string{}, next[:2]...), append([]string{"-overlay", overlayPath}, next[2:]...)...)
	}
	return next
}

func (e *Engine) selectTests(mutant Mutant) TestPlan {
	command := append([]string{}, e.cfg.Tests.Command...)
	if len(command) == 0 {
		command = []string{"go", "test", "./..."}
	}
	if e.cfg.Selection.Mode == "package" && len(command) >= 3 && command[0] == "go" && command[1] == "test" && mutant.Package != "" {
		command[2] = mutant.Package
		return TestPlan{Command: command, Reason: "package selected from mutant file", CoversMutant: true}
	}
	if e.cfg.Selection.Mode == "coverage" {
		if e.coverageMentions(mutant) && len(command) >= 3 && command[0] == "go" && command[1] == "test" && mutant.Package != "" {
			command[2] = mutant.Package
			return TestPlan{Command: command, Reason: "coverage profile matched mutant file", CoversMutant: true}
		}
		return TestPlan{Command: command, Reason: "coverage profile did not match mutant file", CoversMutant: false}
	}
	return TestPlan{Command: command, Reason: "all tests selected", CoversMutant: true}
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
	s := Summary{MutatorStats: map[string]MutatorStat{}}
	s.Total = len(results)
	for _, result := range results {
		operator := result.Mutant.Operator
		if operator == "" {
			operator = "unknown"
		}
		stat := s.MutatorStats[operator]
		stat.Total++
		switch result.Status {
		case StatusKilled:
			s.Killed++
			stat.Killed++
		case StatusSurvived:
			s.Survived++
			stat.Survived++
		case StatusNotCovered:
			s.NotCovered++
			stat.NotCovered++
		case StatusTimedOut:
			s.TimedOut++
			stat.TimedOut++
		case StatusCompileError:
			s.CompileError++
			stat.CompileError++
		case StatusSkipped:
			s.Skipped++
			stat.Skipped++
		case StatusIgnored:
			s.Ignored++
			stat.Ignored++
		case StatusQuarantined:
			s.Quarantined++
			stat.Quarantined++
		case StatusCached:
			s.Cached++
			stat.Cached++
		}
		s.MutatorStats[operator] = stat
	}
	eligible := s.Total - s.Ignored - s.Quarantined - s.Skipped - s.NotCovered
	if eligible > 0 {
		s.Score = float64(s.Killed) / float64(eligible) * 100
		s.EffectiveScore = s.Score
		s.TestEfficacy = s.Score
	}
	coverable := s.Total - s.Ignored - s.Quarantined - s.Skipped
	if coverable > 0 {
		s.MutationCoverage = float64(coverable-s.NotCovered) / float64(coverable) * 100
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
	profile := e.cfg.Selection.CoverageProfile
	if !filepath.IsAbs(profile) {
		profile = filepath.Join(mutant.Module, profile)
	}
	data, err := os.ReadFile(profile)
	if err != nil {
		return false
	}
	rel, _ := filepath.Rel(mutant.Module, mutant.File)
	rel = filepath.ToSlash(rel)
	base := filepath.Base(mutant.File)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, rel+":") || strings.Contains(line, base+":") {
			return true
		}
	}
	return false
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
