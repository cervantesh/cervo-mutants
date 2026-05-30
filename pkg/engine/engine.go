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
	cfg             config.Config
	timingMu        sync.Mutex
	checkpointMu    sync.Mutex
	checkpointScope []Mutant
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
	e.scheduleMutants(mutants)
	if e.cfg.Limits.MaxMutants > 0 && len(mutants) > e.cfg.Limits.MaxMutants {
		mutants = mutants[:e.cfg.Limits.MaxMutants]
	}
	e.setCheckpointScope(mutants)
	quarantined, expired, err := e.loadQuarantine()
	if err != nil {
		return RunResult{}, err
	}
	result := RunResult{
		SchemaVersion: "1",
		Environment:   e.environment(len(mutants)),
		Checkpoint:    e.checkpoint(mutants, "final"),
		Thresholds:    map[string]any{"fail_under": e.cfg.CI.FailUnder},
		Mutants:       []MutantResult{},
		Quarantine:    QuarantineStats{Active: len(quarantined), Expired: expired},
	}
	if req.DryRun {
		for _, mutant := range mutants {
			result.Mutants = append(result.Mutants, MutantResult{MutantID: mutant.ID, Status: StatusSkipped, StatusReason: "dry-run", Mutant: mutant})
		}
		rankSurvivors(result.Mutants)
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
	result.History = e.applyHistory(result.Mutants)
	rankSurvivors(result.Mutants)
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
	if err := e.writeReports(result); err != nil {
		return RunResult{}, err
	}
	return result, nil
}

func (e *Engine) runMutants(ctx context.Context, mutants []Mutant, quarantined map[string]bool) ([]MutantResult, error) {
	if e.cfg.Execution.Resume {
		return e.runMutantsWithResume(ctx, mutants, quarantined)
	}
	workers := e.workerCount(len(mutants))
	if workers <= 1 {
		return e.runMutantsSerial(ctx, mutants, quarantined)
	}
	return e.runMutantsParallel(ctx, mutants, quarantined, workers)
}

func (e *Engine) runMutantsWithResume(ctx context.Context, mutants []Mutant, quarantined map[string]bool) ([]MutantResult, error) {
	completed, err := e.loadPartialResults(mutants)
	if err != nil {
		return nil, err
	}
	if len(completed) == 0 {
		workers := e.workerCount(len(mutants))
		if workers <= 1 {
			return e.runMutantsSerial(ctx, mutants, quarantined)
		}
		return e.runMutantsParallel(ctx, mutants, quarantined, workers)
	}
	results := make([]MutantResult, 0, len(mutants))
	remaining := make([]Mutant, 0, len(mutants))
	for _, mutant := range mutants {
		if result, ok := completed[mutant.ID]; ok {
			result.PreviousStatus = result.Status
			result.Status = StatusCached
			result.StatusReason = "result reused from partial checkpoint"
			results = append(results, result)
			continue
		}
		remaining = append(remaining, mutant)
	}
	next, err := e.runMutantsSerial(ctx, remaining, quarantined)
	if err != nil {
		return nil, err
	}
	results = append(results, next...)
	return orderResults(mutants, results), nil
}

func (e *Engine) runMutantsSerial(ctx context.Context, mutants []Mutant, quarantined map[string]bool) ([]MutantResult, error) {
	results := make([]MutantResult, 0, len(mutants))
	start := time.Now()
	for i, mutant := range mutants {
		if quarantined[mutant.ID] {
			result := MutantResult{MutantID: mutant.ID, Status: StatusQuarantined, StatusReason: "mutant is in active quarantine", Mutant: mutant}
			results = append(results, result)
			e.recordProgress(start, i+1, len(mutants), result)
			e.writePartialResults(results)
			continue
		}
		if result, ok := e.suppressedResult(mutant); ok {
			results = append(results, result)
			e.recordProgress(start, i+1, len(mutants), result)
			e.writePartialResults(results)
			continue
		}
		if e.budgetExhausted(start) {
			result := MutantResult{MutantID: mutant.ID, Status: StatusSkipped, StatusReason: "budget exhausted", Mutant: mutant}
			results = append(results, result)
			e.recordProgress(start, i+1, len(mutants), result)
			e.writePartialResults(results)
			continue
		}
		mutantResult, err := e.runMutant(ctx, mutant)
		if err != nil {
			return nil, err
		}
		results = append(results, mutantResult)
		e.recordProgress(start, i+1, len(mutants), mutantResult)
		e.writePartialResults(results)
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
		if result, ok := e.suppressedResult(mutant); ok {
			results[i] = result
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
	completed := 0
	for item := range done {
		dispatched--
		if item.err != nil && firstErr == nil {
			firstErr = item.err
			cancel()
		}
		results[item.index] = item.result
		completed++
		e.recordProgress(start, completed, len(mutants), item.result)
		e.writePartialResults(compactedResults(results))
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func (e *Engine) budgetExhausted(start time.Time) bool {
	return e.cfg.Execution.Budget > 0 && time.Since(start) >= e.cfg.Execution.Budget
}

func (e *Engine) suppressedResult(mutant Mutant) (MutantResult, bool) {
	rule, ok := strongestSuppression(mutant.SuppressionAudit)
	if !ok || rule.Action != "suppress" {
		return MutantResult{}, false
	}
	return MutantResult{
		MutantID:           mutant.ID,
		Status:             StatusIgnored,
		StatusReason:       fmt.Sprintf("suppressed by audited rule %q: %s", rule.Name, rule.Reason),
		Mutant:             mutant,
		SuggestedTestScope: suggestedTestScope(mutant),
		NearestTests:       mutant.NearbyTests,
	}, true
}

func strongestSuppression(audits []SuppressionAudit) (SuppressionAudit, bool) {
	var best SuppressionAudit
	bestPriority := -1
	for _, audit := range audits {
		priority := suppressionPriority(audit.Action)
		if priority > bestPriority {
			best = audit
			bestPriority = priority
		}
	}
	return best, bestPriority >= 0
}

func suppressionPriority(action string) int {
	switch action {
	case "report-only":
		return 0
	case "lower-priority":
		return 1
	case "quarantine-required":
		return 2
	case "suppress":
		return 3
	default:
		return -1
	}
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

func (e *Engine) environment(mutants int) Environment {
	wd, _ := os.Getwd()
	env := Environment{
		OS:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		GoVersion:       runtime.Version(),
		WorkingDir:      wd,
		TempDir:         os.TempDir(),
		Isolation:       e.cfg.Execution.Isolation,
		Workers:         e.workerCount(mutants),
		TestTimeout:     e.cfg.Tests.Timeout.String(),
		GoFlags:         os.Getenv("GOFLAGS"),
		GoMaxProcs:      os.Getenv("GOMAXPROCS"),
		GoMemLimit:      os.Getenv("GOMEMLIMIT"),
		CI:              os.Getenv("CI"),
		WSL:             isWSL(),
		CGroup:          cgroupSummary(),
		WindowsOneDrive: runtime.GOOS == "windows" && pathMentionsOneDrive(wd),
	}
	if e.cfg.Execution.Resources.MaxProcessMemoryMB > 0 {
		env.Extra = map[string]string{"max_process_memory_mb": strconv.Itoa(e.cfg.Execution.Resources.MaxProcessMemoryMB)}
	}
	if e.cfg.Execution.Budget > 0 {
		env.Budget = e.cfg.Execution.Budget.String()
	}
	return env
}

func isWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	text := strings.ToLower(string(data))
	return strings.Contains(text, "microsoft") || strings.Contains(text, "wsl")
}

func cgroupSummary() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return ""
	}
	line := lines[0]
	if len(line) > 160 {
		line = line[:160]
	}
	return line
}

func pathMentionsOneDrive(path string) bool {
	return strings.Contains(strings.ToLower(path), "onedrive")
}

func (e *Engine) checkpoint(mutants []Mutant, reason string) Checkpoint {
	ids := make([]string, 0, len(mutants))
	for _, mutant := range mutants {
		ids = append(ids, mutant.ID)
	}
	sort.Strings(ids)
	cfg := struct {
		Policy          string
		MutatorProfile  string
		SelectionMode   string
		SelectionFilter bool
		Isolation       string
		TestCommand     []string
		TestTimeout     string
		GoVersion       string
		GOFLAGS         string
		Mutants         []string
		Files           []string
	}{
		Policy:          e.cfg.Policy,
		MutatorProfile:  e.cfg.Mutators.Profile,
		SelectionMode:   e.cfg.Selection.Mode,
		SelectionFilter: e.cfg.Selection.Prefilter,
		Isolation:       e.cfg.Execution.Isolation,
		TestCommand:     e.cfg.Tests.Command,
		TestTimeout:     e.cfg.Tests.Timeout.String(),
		GoVersion:       runtime.Version(),
		GOFLAGS:         os.Getenv("GOFLAGS"),
		Mutants:         ids,
		Files:           e.checkpointFileFingerprints(mutants),
	}
	data, _ := json.Marshal(cfg)
	return Checkpoint{Fingerprint: digestBytes(data), Mutants: len(ids), IncludesFileDigests: true, Reason: reason}
}

func (e *Engine) checkpointFileFingerprints(mutants []Mutant) []string {
	modules := map[string]bool{}
	for _, mutant := range mutants {
		if mutant.Module != "" {
			modules[mutant.Module] = true
		}
	}
	var fingerprints []string
	for module := range modules {
		_ = filepath.WalkDir(module, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				if entry != nil && entry.IsDir() && shouldSkipCheckpointDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if !e.checkpointIncludesFile(module, path, entry.Name()) {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			rel, err := filepath.Rel(module, path)
			if err != nil {
				rel = path
			}
			fingerprints = append(fingerprints, filepath.ToSlash(rel)+":"+digestBytes(data))
			return nil
		})
	}
	sort.Strings(fingerprints)
	return fingerprints
}

func (e *Engine) checkpointIncludesFile(module, path, name string) bool {
	if strings.HasSuffix(name, ".go") || name == "go.mod" || name == "go.sum" {
		return true
	}
	rel, err := filepath.Rel(module, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, pattern := range e.cfg.Execution.CheckpointIncludes {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if globMatch(pattern, rel) {
			return true
		}
	}
	return false
}

func globMatch(pattern, rel string) bool {
	if ok, err := filepath.Match(pattern, rel); err == nil && ok {
		return true
	}
	if strings.Contains(pattern, "**/") {
		withoutRecursive := strings.ReplaceAll(pattern, "**/", "")
		if ok, err := filepath.Match(withoutRecursive, rel); err == nil && ok {
			return true
		}
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return rel == prefix || strings.HasPrefix(rel, prefix+"/")
	}
	if strings.Contains(pattern, "/**/") {
		parts := strings.Split(pattern, "/**/")
		if len(parts) == 2 && strings.HasPrefix(rel, parts[0]+"/") {
			tail := strings.TrimPrefix(rel, parts[0]+"/")
			if ok, err := filepath.Match(parts[1], tail); err == nil && ok {
				return true
			}
		}
	}
	return false
}

func shouldSkipCheckpointDir(name string) bool {
	switch name {
	case ".git", ".cervomut", "vendor", "node_modules", "dist", "build":
		return true
	default:
		return false
	}
}

func (e *Engine) setCheckpointScope(mutants []Mutant) {
	e.checkpointMu.Lock()
	defer e.checkpointMu.Unlock()
	e.checkpointScope = append([]Mutant{}, mutants...)
}

func (e *Engine) currentCheckpointScope() []Mutant {
	e.checkpointMu.Lock()
	defer e.checkpointMu.Unlock()
	return append([]Mutant{}, e.checkpointScope...)
}

func (e *Engine) checkpointFromResults(results []MutantResult, reason string) Checkpoint {
	mutants := e.currentCheckpointScope()
	if len(mutants) > 0 {
		return e.checkpoint(mutants, reason)
	}
	mutants = make([]Mutant, 0, len(results))
	for _, result := range results {
		if result.Mutant.ID == "" {
			continue
		}
		mutants = append(mutants, result.Mutant)
	}
	return e.checkpoint(mutants, reason)
}

func (e *Engine) recordProgress(start time.Time, completed, total int, result MutantResult) {
	if total <= 0 || e.cfg.Reports.Output == "" {
		return
	}
	event := ProgressEvent{
		SchemaVersion: "1",
		Time:          time.Now().UTC(),
		Completed:     completed,
		Total:         total,
		MutantID:      result.MutantID,
		Status:        result.Status,
		Elapsed:       time.Since(start),
		Remaining:     total - completed,
	}
	if completed > 0 {
		perMutant := event.Elapsed / time.Duration(completed)
		event.ETA = (perMutant * time.Duration(event.Remaining)).Round(time.Second).String()
	}
	event.ActiveMutant = result.MutantID
	event.Message = fmt.Sprintf("mutant %d/%d %s %s eta=%s", completed, total, result.MutantID, result.Status, event.ETA)
	_ = os.MkdirAll(e.cfg.Reports.Output, 0o755)
	_ = appendProgressEvent(filepath.Join(e.cfg.Reports.Output, "progress.jsonl"), event)
	fmt.Fprintf(os.Stderr, "progress %d/%d %s %s eta=%s\n", completed, total, result.MutantID, result.Status, event.ETA)
}

func appendProgressEvent(path string, event ProgressEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (e *Engine) writePartialResults(results []MutantResult) {
	if e.cfg.Reports.Output == "" {
		return
	}
	run := RunResult{
		SchemaVersion: "1",
		Environment:   e.environment(len(results)),
		Checkpoint:    e.checkpointFromResults(results, "partial"),
		Thresholds:    map[string]any{"fail_under": e.cfg.CI.FailUnder, "partial": true},
		Mutants:       append([]MutantResult{}, results...),
	}
	rankSurvivors(run.Mutants)
	run.Summary = summarize(run.Mutants)
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(e.cfg.Reports.Output, 0o755)
	_ = writeFileAtomic(filepath.Join(e.cfg.Reports.Output, "partial-mutation-report.json"), data, 0o644)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(path)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func compactedResults(results []MutantResult) []MutantResult {
	compacted := make([]MutantResult, 0, len(results))
	for _, result := range results {
		if result.MutantID == "" {
			continue
		}
		compacted = append(compacted, result)
	}
	return compacted
}

func (e *Engine) loadPartialResults(mutants []Mutant) (map[string]MutantResult, error) {
	results := map[string]MutantResult{}
	if e.cfg.Reports.Output == "" {
		return results, nil
	}
	path := filepath.Join(e.cfg.Reports.Output, "partial-mutation-report.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return results, nil
	}
	if err != nil {
		return nil, err
	}
	var run RunResult
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("load partial checkpoint: %w", err)
	}
	want := e.checkpoint(mutants, "partial")
	if run.Checkpoint.Fingerprint == "" {
		return nil, errors.New("partial checkpoint is missing compatibility fingerprint; rerun without --resume")
	}
	if run.Checkpoint.Fingerprint != want.Fingerprint {
		return nil, fmt.Errorf("partial checkpoint fingerprint mismatch: have %s want %s; rerun without --resume", run.Checkpoint.Fingerprint, want.Fingerprint)
	}
	for _, result := range run.Mutants {
		if result.MutantID == "" {
			continue
		}
		results[result.MutantID] = result
	}
	return results, nil
}

func orderResults(mutants []Mutant, results []MutantResult) []MutantResult {
	byID := map[string]MutantResult{}
	for _, result := range results {
		byID[result.MutantID] = result
	}
	ordered := make([]MutantResult, 0, len(results))
	for _, mutant := range mutants {
		if result, ok := byID[mutant.ID]; ok {
			ordered = append(ordered, result)
		}
	}
	return ordered
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
				ID:               id,
				Module:           generated[i].Module,
				Package:          generated[i].Package,
				File:             generated[i].File,
				Line:             generated[i].Line,
				Function:         generated[i].Function,
				Operator:         generated[i].Operator,
				Original:         generated[i].Original,
				Mutated:          generated[i].Mutated,
				StartOffset:      generated[i].StartOffset,
				EndOffset:        generated[i].EndOffset,
				Diff:             generated[i].Diff,
				Fingerprint:      fingerprint,
				Hint:             generated[i].Hint,
				Description:      generated[i].Description,
				NearbyTests:      nearbyTests(file.ModuleDir, file.Path),
				EquivalentRisk:   generated[i].EquivalentRisk,
				Recommendation:   generated[i].Recommendation,
				CompileErrorRisk: generated[i].CompileErrorRisk,
				SuppressionAudit: e.suppressionAudit(generated[i]),
			})
		}
	}
	return mutants, nil
}

func (e *Engine) scheduleMutants(mutants []Mutant) {
	sort.SliceStable(mutants, func(i, j int) bool {
		if e.cfg.Execution.Budget > 0 {
			left := recommendationPriority(mutants[i].Recommendation)
			right := recommendationPriority(mutants[j].Recommendation)
			if left != right {
				return left < right
			}
			left = timeoutRiskPriority(mutants[i])
			right = timeoutRiskPriority(mutants[j])
			if left != right {
				return left < right
			}
		}
		return mutants[i].ID < mutants[j].ID
	})
}

func recommendationPriority(recommendation string) int {
	switch recommendation {
	case "fast-ci":
		return 0
	case "conservative":
		return 1
	case "default":
		return 2
	case "aggressive":
		return 3
	default:
		return 4
	}
}

func timeoutRiskPriority(mutant Mutant) int {
	switch mutant.Operator {
	case "conditionals-negation", "conditionals-boundary", "boolean-literals", "logical":
		return 0
	case "arithmetic-basic", "string-empty-literals", "nil-checks", "numeric-literals", "return-bool-literals", "assignment-arithmetic", "inc-dec":
		return 1
	case "error-returns":
		return 2
	case "literals", "returns", "loop-control", "slice-map-len-boundary":
		return 3
	default:
		return 2
	}
}

func (e *Engine) suppressionAudit(mutant mutator.Mutant) []SuppressionAudit {
	if !e.cfg.Suppression.Enabled {
		return nil
	}
	var audits []SuppressionAudit
	for _, rule := range e.cfg.Suppression.Rules {
		if rule.Operator != "" && rule.Operator != mutant.Operator {
			continue
		}
		if rule.EquivalentRisk != "" && rule.EquivalentRisk != mutant.EquivalentRisk {
			continue
		}
		if rule.File != "" && !suppressionFileMatches(rule.File, mutant.File) {
			continue
		}
		if rule.Original != "" && rule.Original != mutant.Original {
			continue
		}
		if rule.Mutated != "" && rule.Mutated != mutant.Mutated {
			continue
		}
		evidenceLevel := "suspected"
		if rule.Evidence != "" {
			evidenceLevel = rule.Evidence
		}
		if rule.Action == "suppress" {
			evidenceLevel = "rule-suppressed"
		}
		audits = append(audits, SuppressionAudit{Name: rule.Name, Action: rule.Action, Reason: rule.Reason, EvidenceLevel: evidenceLevel, ReviewerCount: rule.Reviewers})
	}
	return audits
}

func suppressionFileMatches(pattern, file string) bool {
	file = filepath.ToSlash(file)
	pattern = filepath.ToSlash(pattern)
	if ok, err := filepath.Match(pattern, file); err == nil && ok {
		return true
	}
	return file == pattern || strings.HasSuffix(file, "/"+pattern)
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
	if e.cfg.Selection.Prefilter && !e.coverageMentions(mutant) {
		return TestPlan{Command: command, Reason: "coverage prefilter did not match mutant file", CoversMutant: false, CoverageSource: "package-mode-prefilter"}
	}
	if e.cfg.Selection.Mode == "package" && len(command) >= 3 && command[0] == "go" && command[1] == "test" && mutant.Package != "" {
		command[2] = mutant.Package
		source := "unknown"
		if e.cfg.Selection.Prefilter {
			source = "package-mode-prefilter"
		}
		return TestPlan{Command: command, Reason: "package selected from mutant file", CoversMutant: true, CoverageSource: source}
	}
	if e.cfg.Selection.Mode == "coverage" {
		if e.coverageMentions(mutant) && len(command) >= 3 && command[0] == "go" && command[1] == "test" && mutant.Package != "" {
			command[2] = mutant.Package
			return TestPlan{Command: command, Reason: "coverage profile matched mutant file", CoversMutant: true, CoverageSource: "coverage-mode"}
		}
		return TestPlan{Command: command, Reason: "coverage profile did not match mutant file", CoversMutant: false, CoverageSource: "coverage-mode"}
	}
	return TestPlan{Command: command, Reason: "all tests selected", CoversMutant: true, CoverageSource: "unknown"}
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
	err := cmd.Start()
	var cleanup func()
	if err == nil {
		cleanup, err = applyProcessLimits(cmd, e.cfg.Execution.Resources)
		if err != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}
	if err == nil {
		err = cmd.Wait()
	}
	if cleanup != nil {
		cleanup()
	}
	text := output.String()
	if max := e.cfg.Reports.MaxOutputBytes; max > 0 && len(text) > max {
		text = text[:max]
	}
	status := StatusKilled
	failureKind := ""
	reason := "tests failed with mutant applied"
	if runCtx.Err() == context.DeadlineExceeded {
		status = StatusTimedOut
		failureKind = "timeout"
		reason = "test command timed out"
	} else if err == nil {
		status = StatusSurvived
		reason = "tests passed with mutant applied"
	} else if errors.Is(err, errProcessLimitUnsupported) {
		status = StatusCompileError
		failureKind = "resource_limit_unsupported"
		reason = err.Error()
	} else if !strings.Contains(text, "FAIL") {
		status = StatusCompileError
		failureKind = classifyFailure(text, err)
		reason = "test command failed before running assertions"
	}
	return MutantResult{
		MutantID:     job.Mutant.ID,
		Status:       status,
		FailureKind:  failureKind,
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

func summarize(results []MutantResult) Summary {
	s := Summary{MutatorStats: map[string]MutatorStat{}, EquivalentRiskStats: map[string]int{}}
	s.Total = len(results)
	s.GeneratedMutants = len(results)
	for _, result := range results {
		operator := result.Mutant.Operator
		if operator == "" {
			operator = "unknown"
		}
		stat := s.MutatorStats[operator]
		stat.Total++
		risk := result.Mutant.EquivalentRisk
		if risk == "" {
			risk = "unknown"
		}
		s.EquivalentRiskStats[risk]++
		if stat.Recommendation == "" {
			stat.Recommendation = result.Mutant.Recommendation
		}
		if stat.EquivalentRisk == "" {
			stat.EquivalentRisk = result.Mutant.EquivalentRisk
		}
		switch result.Status {
		case StatusKilled:
			s.Killed++
			stat.Killed++
			s.ExecutedMutants++
			s.CoveredMutants++
		case StatusSurvived:
			s.Survived++
			stat.Survived++
			s.ExecutedMutants++
			s.CoveredMutants++
			if result.Mutant.EquivalentRisk == "high" {
				s.HighRiskSurvivors++
			}
		case StatusNotCovered:
			s.NotCovered++
			stat.NotCovered++
		case StatusTimedOut:
			s.TimedOut++
			stat.TimedOut++
			s.ExecutedMutants++
			s.CoveredMutants++
		case StatusCompileError:
			s.CompileError++
			stat.CompileError++
			s.ExecutedMutants++
			s.CoveredMutants++
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
			switch result.PreviousStatus {
			case StatusKilled:
				s.Killed++
				stat.Killed++
				s.ExecutedMutants++
				s.CoveredMutants++
			case StatusSurvived:
				s.Survived++
				stat.Survived++
				s.ExecutedMutants++
				s.CoveredMutants++
				if result.Mutant.EquivalentRisk == "high" {
					s.HighRiskSurvivors++
				}
			case StatusNotCovered:
				s.NotCovered++
				stat.NotCovered++
			case StatusTimedOut:
				s.TimedOut++
				stat.TimedOut++
				s.ExecutedMutants++
				s.CoveredMutants++
			case StatusCompileError:
				s.CompileError++
				stat.CompileError++
				s.ExecutedMutants++
				s.CoveredMutants++
			}
		}
		for _, audit := range result.Mutant.SuppressionAudit {
			switch audit.Action {
			case "report-only":
				s.SuppressionReportOnly++
			case "lower-priority":
				s.SuppressionLowerPriority++
			case "suppress":
				s.SuppressionSuppressed++
			case "quarantine-required":
				s.SuppressionQuarantineRequired++
			}
		}
		s.MutatorStats[operator] = stat
	}
	eligible := s.Total - s.Ignored - s.Quarantined - s.Skipped - s.NotCovered
	s.EffectiveMutants = s.Killed + s.Survived
	s.ScoreDenominator = eligible
	if eligible > 0 {
		s.Score = float64(s.Killed) / float64(eligible) * 100
	}
	if s.EffectiveMutants > 0 {
		s.EffectiveScore = float64(s.Killed) / float64(s.EffectiveMutants) * 100
		s.TestEfficacy = s.EffectiveScore
	}
	coverable := s.Total - s.Ignored - s.Quarantined - s.Skipped
	if coverable > 0 {
		s.MutationCoverage = float64(coverable-s.NotCovered) / float64(coverable) * 100
	}
	s.DenominatorHealth = denominatorHealth(s)
	return s
}

func denominatorHealth(s Summary) DenominatorHealth {
	health := DenominatorHealth{
		Generated:        s.GeneratedMutants,
		Covered:          s.CoveredMutants,
		Executed:         s.ExecutedMutants,
		Effective:        s.EffectiveMutants,
		ScoreDenominator: s.ScoreDenominator,
		Killed:           s.Killed,
		Survived:         s.Survived,
		NotCovered:       s.NotCovered,
		TimedOut:         s.TimedOut,
		CompileError:     s.CompileError,
		Skipped:          s.Skipped,
		Ignored:          s.Ignored,
		Quarantined:      s.Quarantined,
		Healthy:          true,
	}
	if health.Generated > 0 && health.Effective == 0 {
		health.Warnings = append(health.Warnings, "no_effective_mutants")
	}
	if health.Effective > 0 && health.TimedOut > health.Effective {
		health.Warnings = append(health.Warnings, "timed_out_exceeds_effective")
	}
	if health.Effective > 0 && health.NotCovered > health.Effective {
		health.Warnings = append(health.Warnings, "not_covered_exceeds_effective")
	}
	if health.Effective > 0 && health.ScoreDenominator > health.Effective*2 {
		health.Warnings = append(health.Warnings, "score_denominator_dwarfs_effective")
	}
	if health.Effective > 0 && s.TestEfficacy >= 90 && (health.TimedOut > health.Effective || health.NotCovered > health.Effective) {
		health.Warnings = append(health.Warnings, "high_score_poor_denominator_health")
	}
	health.Healthy = len(health.Warnings) == 0
	return health
}

type historyFile struct {
	SchemaVersion string                  `json:"schema_version"`
	UpdatedAt     string                  `json:"updated_at"`
	Mutants       map[string]historyEntry `json:"mutants"`
}

type historyEntry struct {
	MutantID         string `json:"mutant_id"`
	Operator         string `json:"operator"`
	Status           Status `json:"status"`
	FirstSeen        string `json:"first_seen"`
	LastSeen         string `json:"last_seen"`
	SeenRuns         int    `json:"seen_runs"`
	SurvivedRuns     int    `json:"survived_runs"`
	KilledRuns       int    `json:"killed_runs"`
	NotCoveredRuns   int    `json:"not_covered_runs"`
	CompileErrorRuns int    `json:"compile_error_runs"`
	TimedOutRuns     int    `json:"timed_out_runs"`
}

func (e *Engine) applyHistory(results []MutantResult) HistoryStats {
	stats := HistoryStats{Enabled: e.cfg.History.Enabled, Path: e.cfg.History.Path, OperatorUsefulSurvivor: map[string]float64{}}
	if !e.cfg.History.Enabled {
		return stats
	}
	path := e.cfg.History.Path
	if path == "" {
		path = ".cervomut/history.json"
		stats.Path = path
	}
	store := historyFile{SchemaVersion: "1", Mutants: map[string]historyEntry{}}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &store)
	}
	if store.Mutants == nil {
		store.Mutants = map[string]historyEntry{}
	}
	stats.LoadedMutants = len(store.Mutants)
	now := time.Now().UTC().Format(time.RFC3339)
	operatorSeen := map[string]int{}
	operatorSurvived := map[string]int{}
	for i := range results {
		result := &results[i]
		operator := result.Mutant.Operator
		if operator == "" {
			operator = "unknown"
		}
		previous, existed := store.Mutants[result.MutantID]
		if existed {
			result.PreviousStatus = previous.Status
			result.FirstSeen = previous.FirstSeen
			result.SurvivorAgeRuns = previous.SurvivedRuns
			if result.Status == StatusSurvived {
				result.HistoryStatus = "existing_survivor"
				if previous.SurvivedRuns > 0 {
					stats.LongStandingSurvivors++
					result.HistoryStatus = "long_standing_survivor"
				}
			}
		} else {
			result.FirstSeen = now
			if result.Status == StatusSurvived {
				result.HistoryStatus = "new_survivor"
				stats.NewSurvivors++
			}
		}
		if result.HistoryStatus == "" {
			result.HistoryStatus = "seen"
		}
		result.LastSeen = now
		entry := previous
		entry.MutantID = result.MutantID
		entry.Operator = operator
		entry.Status = result.Status
		if entry.FirstSeen == "" {
			entry.FirstSeen = result.FirstSeen
		}
		entry.LastSeen = now
		entry.SeenRuns++
		switch result.Status {
		case StatusSurvived:
			entry.SurvivedRuns++
		case StatusKilled:
			entry.KilledRuns++
		case StatusNotCovered:
			entry.NotCoveredRuns++
		case StatusCompileError:
			entry.CompileErrorRuns++
		case StatusTimedOut:
			entry.TimedOutRuns++
		}
		result.SurvivorAgeRuns = entry.SurvivedRuns
		store.Mutants[result.MutantID] = entry
		operatorSeen[operator]++
		if result.Status == StatusSurvived {
			operatorSurvived[operator]++
		}
	}
	for operator, seen := range operatorSeen {
		if seen > 0 {
			stats.OperatorUsefulSurvivor[operator] = float64(operatorSurvived[operator]) / float64(seen)
		}
	}
	for i := range results {
		results[i].OperatorYield = stats.OperatorUsefulSurvivor[results[i].Mutant.Operator]
	}
	stats.UpdatedMutants = len(results)
	store.UpdatedAt = now
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
		if data, err := json.MarshalIndent(store, "", "  "); err == nil {
			_ = os.WriteFile(path, data, 0o644)
		}
	}
	return stats
}

func rankSurvivors(results []MutantResult) {
	survivors := make([]int, 0)
	for i := range results {
		if results[i].Status == StatusSurvived {
			survivors = append(survivors, i)
		}
	}
	sort.SliceStable(survivors, func(i, j int) bool {
		left := results[survivors[i]]
		right := results[survivors[j]]
		leftScore, _ := survivorRankScore(left)
		rightScore, _ := survivorRankScore(right)
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		if riskPriority(left.Mutant.EquivalentRisk) != riskPriority(right.Mutant.EquivalentRisk) {
			return riskPriority(left.Mutant.EquivalentRisk) < riskPriority(right.Mutant.EquivalentRisk)
		}
		if recommendationPriority(left.Mutant.Recommendation) != recommendationPriority(right.Mutant.Recommendation) {
			return recommendationPriority(left.Mutant.Recommendation) < recommendationPriority(right.Mutant.Recommendation)
		}
		if len(left.Mutant.NearbyTests) != len(right.Mutant.NearbyTests) {
			return len(left.Mutant.NearbyTests) > len(right.Mutant.NearbyTests)
		}
		return left.MutantID < right.MutantID
	})
	for rank, index := range survivors {
		score, reason := survivorRankScore(results[index])
		results[index].SurvivorRank = rank + 1
		results[index].RankScore = score
		results[index].RankReason = reason
		results[index].Actionability = actionability(score)
		results[index].SuggestedTestScope = suggestedTestScope(results[index].Mutant)
		results[index].NearestTests = results[index].Mutant.NearbyTests
	}
}

func survivorRankScore(result MutantResult) (float64, string) {
	score := 100.0
	risk := result.Mutant.EquivalentRisk
	switch risk {
	case "low":
		score += 20
	case "medium":
		score += 5
	case "high":
		score -= 25
	default:
		score -= 10
	}
	switch result.Mutant.Recommendation {
	case "fast-ci":
		score += 20
	case "conservative":
		score += 12
	case "default":
		score += 4
	case "aggressive":
		score -= 12
	}
	if len(result.Mutant.NearbyTests) > 0 {
		score += 12
	}
	if result.Mutant.Function != "" && strings.HasPrefix(result.Mutant.Function, strings.ToUpper(result.Mutant.Function[:1])) {
		score += 5
	}
	if result.CoverageSource != "" && result.CoverageSource != "unknown" {
		score += 6
	}
	switch result.HistoryStatus {
	case "new_survivor":
		score += 18
	case "long_standing_survivor":
		score += 10
	case "existing_survivor":
		score += 4
	}
	if result.OperatorYield > 0 {
		score += result.OperatorYield * 10
	}
	for _, audit := range result.Mutant.SuppressionAudit {
		if audit.Action == "lower-priority" || audit.Action == "report-only" {
			score -= 8
		}
	}
	reason := fmt.Sprintf("score=%.1f risk=%s recommendation=%s coverage_source=%s nearby_tests=%d history=%s survivor_age_runs=%d operator_yield=%.2f", score, risk, result.Mutant.Recommendation, result.CoverageSource, len(result.Mutant.NearbyTests), result.HistoryStatus, result.SurvivorAgeRuns, result.OperatorYield)
	return score, reason
}

func actionability(score float64) string {
	switch {
	case score >= 125:
		return "high"
	case score >= 95:
		return "medium"
	default:
		return "low"
	}
}

func suggestedTestScope(mutant Mutant) string {
	if mutant.Package != "" && mutant.Package != "." {
		return mutant.Package
	}
	if len(mutant.NearbyTests) > 0 {
		return filepath.ToSlash(filepath.Dir(mutant.NearbyTests[0]))
	}
	return "."
}

func riskPriority(risk string) int {
	switch risk {
	case "low":
		return 0
	case "medium":
		return 1
	case "high":
		return 2
	default:
		return 3
	}
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
	parseable := false
	for _, line := range strings.Split(string(data), "\n") {
		file, startLine, endLine, count, ok := parseCoverageProfileLine(line)
		if !ok {
			continue
		}
		parseable = true
		if count <= 0 || !coverageFileMatches(file, rel, base) {
			continue
		}
		if mutant.Line >= startLine && mutant.Line <= endLine {
			return true
		}
	}
	if !parseable {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, rel+":") || strings.Contains(line, base+":") {
				return true
			}
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
