package engine

import (
	"context"
	"errors"
	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/mutator"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunUsesParallelWorkers(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Execution.Workers = 2
	cfg.Limits.MaxMutants = 2
	isolateArtifacts(&cfg, dir)

	result, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("parallel Run returned error: %v", err)
	}
	if len(result.Mutants) != 2 {
		t.Fatalf("mutants = %d, want 2", len(result.Mutants))
	}
	for _, mutant := range result.Mutants {
		if mutant.Status == "" || mutant.MutantID == "" {
			t.Fatalf("parallel result missing status/id: %+v", mutant)
		}
	}
}

func TestSuppressionRuleCanIgnoreMutantBeforeExecution(t *testing.T) {
	cfg := config.Defaults()
	cfg.Suppression.Rules = []config.SuppressionRule{{
		Name:      "known-equivalent-conditional",
		Operator:  "conditionals-boundary",
		Action:    "suppress",
		Reason:    "Audited as equivalent in generated comparison wrappers.",
		Evidence:  "confirmed",
		Reviewers: 1,
	}}
	mutant := Mutant{
		ID:               "m-suppressed",
		Module:           t.TempDir(),
		Package:          ".",
		File:             "calc.go",
		Line:             3,
		Operator:         "conditionals-boundary",
		SuppressionAudit: New(cfg).suppressionAudit(mutator.Mutant{Operator: "conditionals-boundary", EquivalentRisk: "medium"}),
	}

	e := New(cfg)
	session := e.newRunSession()
	results, err := session.runMutantsSerial(context.Background(), []Mutant{mutant}, map[string]bool{})
	if err != nil {
		t.Fatalf("runMutantsSerial returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Status != StatusIgnored {
		t.Fatalf("status = %q, want %q", results[0].Status, StatusIgnored)
	}
	if !strings.Contains(results[0].StatusReason, "known-equivalent-conditional") {
		t.Fatalf("status reason does not name suppression rule: %q", results[0].StatusReason)
	}
}

func TestSerialRunnerHandlesQuarantineAndBudgetBranches(t *testing.T) {
	cfg := config.Defaults()
	cfg.Execution.Budget = time.Nanosecond
	e := New(cfg)
	base := time.Unix(0, 0)
	nowCalls := 0
	e.now = func() time.Time {
		nowCalls++
		if nowCalls == 1 {
			return base
		}
		return base.Add(time.Nanosecond)
	}
	mutants := []Mutant{
		{ID: "q", Operator: "conditionals-negation"},
		{ID: "budget", Operator: "conditionals-negation"},
	}
	session := e.newRunSession()
	results, err := session.runMutantsSerial(context.Background(), mutants, map[string]bool{"q": true})
	if err != nil {
		t.Fatalf("runMutantsSerial returned error: %v", err)
	}
	if results[0].Status != StatusQuarantined || results[1].Status != StatusPendingBudget || results[1].FailureKind != "budget_exhausted" {
		t.Fatalf("unexpected serial statuses: %+v", results)
	}
}

func TestRunTestClassifiesPassFailureAndTimeout(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Timeout = 10 * time.Second
	e := New(cfg)
	session := e.newRunSession()
	pass, err := session.runTest(context.Background(), MutantJob{Mutant: Mutant{ID: "pass"}, WorkDir: dir, TestCommand: []string{"go", "test", "."}})
	if err != nil {
		t.Fatalf("pass runTest returned error: %v", err)
	}
	if pass.Status != StatusSurvived {
		t.Fatalf("pass status = %q", pass.Status)
	}

	fail, err := session.runTest(context.Background(), MutantJob{Mutant: Mutant{ID: "fail"}, WorkDir: dir, TestCommand: []string{"go", "test", "./missing"}})
	if err != nil {
		t.Fatalf("fail runTest returned error: %v", err)
	}
	if fail.Status != StatusKilled || !strings.Contains(fail.Output, "missing") {
		t.Fatalf("fail result = %+v", fail)
	}

	cfg.Tests.Timeout = time.Nanosecond
	timeoutEngine := New(cfg)
	timeoutSession := timeoutEngine.newRunSession()
	timeout, err := timeoutSession.runTest(context.Background(), MutantJob{Mutant: Mutant{ID: "timeout"}, WorkDir: dir, TestCommand: []string{"go", "test", "."}})
	if err != nil {
		t.Fatalf("timeout runTest returned error: %v", err)
	}
	if timeout.Status != StatusTimedOut {
		t.Fatalf("timeout status = %q", timeout.Status)
	}

	if runtime.GOOS != "windows" {
		cfg := config.Defaults()
		cfg.Execution.Resources.MaxProcessMemoryMB = 64
		resourceEngine := New(cfg)
		resourceSession := resourceEngine.newRunSession()
		resourceSkipped, err := resourceSession.runTest(context.Background(), MutantJob{Mutant: Mutant{ID: "resource"}, WorkDir: dir, TestCommand: []string{"go", "test", "."}})
		if err != nil {
			t.Fatalf("resource-limited runTest returned error: %v", err)
		}
		if resourceSkipped.Status != StatusSurvived || resourceSkipped.FailureKind != "" {
			t.Fatalf("resource-limited result = %+v", resourceSkipped)
		}
	}
}

func TestEnvironmentWarnsWhenProcessLimitsAreBestEffort(t *testing.T) {
	cfg := config.Defaults()
	cfg.Execution.Resources.MaxProcessMemoryMB = 64
	env := New(cfg).environment(1)
	if runtime.GOOS == "windows" {
		for _, warning := range env.Warnings {
			if strings.Contains(warning, "process resource limits are not enforced on this platform") {
				t.Fatalf("unexpected non-Windows process-limit warning on Windows: %+v", env.Warnings)
			}
		}
		return
	}
	found := false
	for _, warning := range env.Warnings {
		if strings.Contains(warning, "process resource limits are not enforced on this platform") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected best-effort process-limit warning, got %+v", env.Warnings)
	}
}

func TestPrepareMutationTempWorkdirAndOverlayBranches(t *testing.T) {
	dir := writeFixture(t)
	source := filepath.Join(dir, "calc.go")
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(data), ">=")
	mutant := Mutant{
		ID:          "m-prepare",
		Module:      dir,
		Package:     ".",
		File:        source,
		Original:    ">=",
		Mutated:     ">",
		StartOffset: start,
		EndOffset:   start + len(">="),
	}
	cfg := config.Defaults()
	cfg.Execution.Isolation = "temp-workdir"
	cfg.Execution.TempRoot = filepath.Join(dir, ".cervomut", "tmp")
	tempEngine := New(cfg)
	tempSession := tempEngine.newRunSession()
	workdir, command, cleanup, err := tempSession.prepareMutation(mutant, []string{"go", "test", "."})
	if err != nil {
		t.Fatalf("prepareMutation temp-workdir returned error: %v", err)
	}
	defer cleanup()
	if workdir == dir || strings.Join(command, " ") != "go test ." {
		t.Fatalf("unexpected temp workdir/command: %s %#v", workdir, command)
	}
	mutated, err := os.ReadFile(filepath.Join(workdir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mutated), "n > 0") {
		t.Fatalf("mutant was not applied in temp workdir: %s", mutated)
	}

	cfg.Execution.Isolation = "overlay"
	overlayEngine := New(cfg)
	overlaySession := overlayEngine.newRunSession()
	workdir, command, cleanup, err = overlaySession.prepareMutation(mutant, []string{"go", "test", "."})
	if err != nil {
		t.Fatalf("prepareMutation overlay returned error: %v", err)
	}
	defer cleanup()
	if workdir != dir || !containsArg(command, "-overlay") {
		t.Fatalf("unexpected overlay workdir/command: %s %#v", workdir, command)
	}

	bad := mutant
	bad.File = filepath.Join(t.TempDir(), "outside.go")
	badEngine := New(cfg)
	badSession := badEngine.newRunSession()
	if _, _, cleanup, err := badSession.prepareMutation(bad, []string{"go", "test", "."}); err == nil {
		cleanup()
		t.Fatal("prepareMutation accepted outside file")
	}
}

func TestRunMutantCacheAndErrorBranches(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "."}
	cfg.Cache.Path = filepath.Join(dir, ".cervomut", "cache")
	cfg.Selection.Mode = "package"
	cfg.Reports.Output = filepath.Join(dir, ".cervomut", "reports")
	mutant := Mutant{
		ID:          "m-cache",
		Module:      dir,
		Package:     ".",
		File:        filepath.Join(dir, "calc.go"),
		Line:        3,
		Operator:    "conditionals-boundary",
		Original:    ">=",
		Mutated:     ">",
		StartOffset: 0,
		EndOffset:   1,
		Fingerprint: "fp",
		Ownership:   &OwnershipRoute{Owner: "fresh-owner", Rule: "current"},
	}
	e := New(cfg)
	session := e.newRunSession()
	plan := session.selectTests(mutant)
	key, err := session.cacheKey(mutant, plan)
	if err != nil {
		t.Fatal(err)
	}
	stale := mutant
	stale.Ownership = &OwnershipRoute{Owner: "stale-owner", Rule: "stale"}
	if err := session.putCached(key, MutantResult{MutantID: mutant.ID, Status: StatusKilled, Mutant: stale}); err != nil {
		t.Fatal(err)
	}
	cached, err := session.runMutant(context.Background(), mutant)
	if err != nil {
		t.Fatalf("runMutant cached returned error: %v", err)
	}
	if cached.Status != StatusCached || cached.PreviousStatus != StatusKilled {
		t.Fatalf("cached result not reused: %+v", cached)
	}
	if cached.Mutant.Ownership == nil || cached.Mutant.Ownership.Owner != "fresh-owner" {
		t.Fatalf("cached result did not refresh current mutant metadata: %+v", cached.Mutant.Ownership)
	}

	missing := mutant
	missing.File = filepath.Join(dir, "missing.go")
	if _, err := session.runMutant(context.Background(), missing); err == nil {
		t.Fatal("runMutant accepted missing source file")
	}
}

func TestWriteReportsAndTimingNoopBranches(t *testing.T) {
	cfg := config.Defaults()
	cfg.Reports.Output = ""
	reportEngine := New(cfg)
	if err := reportEngine.writeReports(RunResult{}); err != nil {
		t.Fatalf("writeReports with empty output returned error: %v", err)
	}
	cfg.Selection.UseTimings = false
	disabledTimingsEngine := New(cfg)
	disabledTimingsSession := disabledTimingsEngine.newRunSession()
	disabledTimingsSession.recordTiming("m", time.Millisecond)
	cfg.Selection.UseTimings = true
	cfg.Selection.TimingsPath = ""
	emptyPathEngine := New(cfg)
	emptyPathSession := emptyPathEngine.newRunSession()
	emptyPathSession.recordTiming("m", time.Millisecond)
}

func TestRunStopMetadata(t *testing.T) {
	reason, last := runStopMetadata([]MutantResult{
		{MutantID: "done", Status: StatusKilled},
		{MutantID: "later", Status: StatusPendingBudget},
	})
	if reason != "budget_exhausted" || last != "done" {
		t.Fatalf("budget stop metadata = %q %q", reason, last)
	}
	reason, last = runStopMetadata([]MutantResult{
		{MutantID: "a", Status: StatusSkippedResource},
		{MutantID: "b", Status: StatusSkippedResource},
	})
	if reason != "resource_limits_unavailable" || last != "" {
		t.Fatalf("resource stop metadata = %q %q", reason, last)
	}
}

func TestSelectionPatchAndRunTestErrorBranches(t *testing.T) {
	cfg := config.Defaults()
	cfg.Tests.Command = nil
	e := New(cfg)
	session := e.newRunSession()
	plan := session.selectTests(Mutant{ID: "m1"})
	if len(plan.Command) != 3 || plan.Command[0] != "go" || plan.Reason != "all tests selected" {
		t.Fatalf("default selectTests plan = %+v", plan)
	}
	if _, err := session.runTest(context.Background(), MutantJob{}); err == nil {
		t.Fatal("runTest accepted empty command")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(path, []byte("package p\nconst n = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mutant := Mutant{File: path, StartOffset: -1, EndOffset: 2, Original: "1", Mutated: "2"}
	if err := applyDiffReplacement(path, mutant); err == nil {
		t.Fatal("applyDiffReplacement accepted invalid offsets")
	}
	mutant = Mutant{File: path, StartOffset: 0, EndOffset: len("package p"), Original: "missing", Mutated: "2"}
	if err := applyDiffReplacement(path, mutant); err == nil {
		t.Fatal("applyDiffReplacement accepted missing original token")
	}
	if err := applyDiffReplacement(filepath.Join(dir, "missing.go"), mutant); err == nil {
		t.Fatal("applyDiffReplacement accepted missing file")
	}
	if got := withOverlayFlag([]string{"echo", "ok"}, "overlay.json"); strings.Join(got, " ") != "echo ok" {
		t.Fatalf("withOverlayFlag changed non-go command: %v", got)
	}
}

func TestParallelWorkerAndCollectorErrorBranches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	jobs := make(chan indexedMutant, 1)
	done := make(chan indexedResult, 1)
	startParallelWorkers(ctx, 1, jobs, done, func(context.Context, Mutant) (MutantResult, error) {
		return MutantResult{MutantID: "unexpected"}, nil
	})
	jobs <- indexedMutant{index: 0, mutant: Mutant{ID: "m1"}}
	close(jobs)
	item := <-done
	if !errors.Is(item.err, context.Canceled) {
		t.Fatalf("worker err = %v, want canceled", item.err)
	}

	cfg := config.Defaults()
	cfg.Reports.Output = t.TempDir()
	e := New(cfg)
	session := e.newRunSession()
	failed := make(chan indexedResult, 1)
	failed <- indexedResult{index: 0, err: errors.New("boom")}
	close(failed)
	_, err := session.collectParallelResults(failed, []MutantResult{{MutantID: "m1"}}, 1, time.Now(), func() {})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("collectParallelResults err = %v, want boom", err)
	}
}

func TestParallelRunnerHandlesPreExecutionOutcomes(t *testing.T) {
	cfg := config.Defaults()
	cfg.Suppression.Rules = []config.SuppressionRule{{
		Name:      "confirmed-equivalent",
		Operator:  "logical",
		Action:    "suppress",
		Reason:    "confirmed equivalent",
		Evidence:  "confirmed",
		Reviewers: 1,
	}}
	e := New(cfg)
	mutants := []Mutant{
		{ID: "quarantined"},
		{ID: "suppressed", Operator: "logical", SuppressionAudit: []SuppressionAudit{{Name: "confirmed-equivalent", Action: "suppress", Reason: "confirmed equivalent"}}},
		{ID: "also-quarantined"},
	}

	session := e.newRunSession()
	results, err := session.runMutantsParallel(context.Background(), mutants, map[string]bool{"quarantined": true, "also-quarantined": true}, 2)
	if err != nil {
		t.Fatalf("runMutantsParallel returned error: %v", err)
	}
	statuses := []Status{results[0].Status, results[1].Status, results[2].Status}
	want := []Status{StatusQuarantined, StatusIgnored, StatusQuarantined}
	if strings.Join(statusStrings(statuses), ",") != strings.Join(statusStrings(want), ",") {
		t.Fatalf("statuses = %+v, want %+v", statuses, want)
	}
}
