package engine

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/mutator"
)

func TestFailureHelpersAndFailureResult(t *testing.T) {
	panicErr := &PanicError{Stage: "baseline", Recovered: "boom"}
	if got := panicErr.Error(); got != "internal_error: panic in baseline: boom" {
		t.Fatalf("PanicError.Error() = %q", got)
	}

	if err := wrapStageError("runner_error", nil); err != nil {
		t.Fatalf("wrapStageError(nil) = %v", err)
	}
	if got := wrapStageError("runner_error", panicErr); got != panicErr {
		t.Fatalf("wrapStageError should preserve PanicError: %v", got)
	}

	prefixed := errors.New("runner_error: failed")
	if got := wrapStageError("runner_error", prefixed); got != prefixed {
		t.Fatalf("wrapStageError should preserve prefixed error: %v", got)
	}

	internal := errors.New("internal_error: panic in run")
	if got := wrapStageError("runner_error", internal); got != internal {
		t.Fatalf("wrapStageError should preserve internal_error: %v", got)
	}

	plain := errors.New("bad baseline")
	wrapped := wrapStageError("runner_error", plain)
	if wrapped == plain || wrapped == nil {
		t.Fatalf("wrapStageError should wrap plain errors: %v", wrapped)
	}
	if !strings.Contains(wrapped.Error(), "runner_error: bad baseline") {
		t.Fatalf("wrapped error = %q", wrapped.Error())
	}

	cfg := config.Defaults()
	cfg.CI.FailUnder = 85
	cfg.Baseline.Enabled = true
	cfg.History.Enabled = true
	cfg.History.Path = filepath.ToSlash(filepath.Join(t.TempDir(), "history.json"))
	failure := Failure{
		Kind:          "runner_error",
		Message:       "baseline failed",
		CorrelationID: "cid-test",
		RunnerResult: &FailureRunnerResult{
			Status:       StatusCompileError,
			StatusReason: "baseline compile failed",
			Command:      []string{"go", "test", "./..."},
			Output:       "go: toolchain mismatch",
		},
	}
	result := FailureResult(cfg, failure)
	if result.SchemaVersion != "1" || result.Failure == nil {
		t.Fatalf("FailureResult() missing schema/failure: %+v", result)
	}
	if result.Failure.Kind != failure.Kind || result.StoppedReason != failure.Kind {
		t.Fatalf("FailureResult() did not preserve failure metadata: %+v", result)
	}
	if result.Failure.RunnerResult == nil || result.Failure.RunnerResult.Status != StatusCompileError {
		t.Fatalf("FailureResult() did not preserve runner result: %+v", result.Failure)
	}
	if failed, _ := result.Thresholds["failed"].(bool); !failed {
		t.Fatalf("FailureResult() thresholds = %+v", result.Thresholds)
	}
	if result.History.Path != cfg.History.Path || !result.History.Enabled {
		t.Fatalf("FailureResult() history = %+v", result.History)
	}
	if result.Baseline.Enabled != cfg.Baseline.Enabled || len(result.Mutants) != 0 {
		t.Fatalf("FailureResult() baseline/mutants = %+v %+v", result.Baseline, result.Mutants)
	}
	if result.Gate.Evaluated {
		t.Fatalf("FailureResult() should not evaluate gates: %+v", result.Gate)
	}
}

func TestHelperCoverageBranches(t *testing.T) {
	handle := processLimitHandle{}
	handle.Cleanup()
	if got := handle.Stats(); got != (processLimitStats{}) {
		t.Fatalf("nil processLimitHandle stats = %+v", got)
	}

	noopProcessLimitCleanup()
	noopCleanup()

	cases := []struct {
		name    string
		sliceBy string
		mutant  Mutant
		want    string
	}{
		{name: "package", sliceBy: "package", mutant: Mutant{Package: "pkg/a", ID: "m1"}, want: "pkg/a"},
		{name: "file", sliceBy: "file", mutant: Mutant{File: filepath.Join("dir", "calc.go"), ID: "m2"}, want: "dir/calc.go"},
		{name: "function", sliceBy: "function", mutant: Mutant{Function: "DoThing", ID: "m3"}, want: "DoThing"},
		{name: "operator", sliceBy: "operator", mutant: Mutant{Operator: "inc-dec", ID: "m4"}, want: "inc-dec"},
		{name: "mutant", sliceBy: "mutant", mutant: Mutant{ID: "m5"}, want: "m5"},
		{name: "fallback-id", sliceBy: "package", mutant: Mutant{ID: "m6"}, want: "m6"},
		{name: "fallback-file-operator", sliceBy: "package", mutant: Mutant{File: filepath.Join("dir", "other.go"), Operator: "logical"}, want: "dir/other.go:logical"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sliceGroupKey(tc.mutant, tc.sliceBy); got != tc.want {
				t.Fatalf("sliceGroupKey(%q) = %q, want %q", tc.sliceBy, got, tc.want)
			}
		})
	}
}

func TestNewWithOptionsAndCheckpointHelpers(t *testing.T) {
	cfg := config.Defaults()
	generatorCalled := false
	customGenerator := mutator.GeneratorFunc(func(pkg, filename string, src []byte, profile string) ([]mutator.Mutant, error) {
		generatorCalled = true
		return []mutator.Mutant{{ID: "generated"}}, nil
	})
	customSuppression := SuppressionEvaluatorFunc(func(mutator.Mutant) []SuppressionAudit {
		return []SuppressionAudit{{Name: "manual", Action: "report-only"}}
	})
	customRanker := SurvivorRankerFunc(func(goos string, results []MutantResult) []SurvivorRanking {
		return []SurvivorRanking{{MutantID: "m1", SurvivorRank: 1, RankReason: goos}}
	})
	e := NewWithOptions(cfg,
		WithMutantGenerator(customGenerator),
		WithSuppressionEvaluator(customSuppression),
		WithSurvivorRanker(customRanker),
	)
	if e == nil {
		t.Fatal("NewWithOptions returned nil")
	}
	if _, err := e.mutantGenerator.Generate("fixture", "fixture.go", []byte("package fixture"), ""); err != nil || !generatorCalled {
		t.Fatalf("custom generator did not run: called=%v err=%v", generatorCalled, err)
	}
	if audits := e.suppressionEvaluator.Evaluate(mutator.Mutant{}); len(audits) != 1 || audits[0].Action != "report-only" {
		t.Fatalf("custom suppression evaluator = %+v", audits)
	}
	if rankings := e.survivorRanker.Rank(runtime.GOOS, []MutantResult{{MutantID: "m1"}}); len(rankings) != 1 || rankings[0].SurvivorRank != 1 {
		t.Fatalf("custom survivor ranker = %+v", rankings)
	}

	start := time.Now().Add(-3 * time.Second)
	if got := e.elapsedSince(start); got <= 0 {
		t.Fatalf("elapsedSince() = %v, want positive duration", got)
	}
	if now := (&Engine{}).clockNow(); now.IsZero() {
		t.Fatal("clockNow should fall back to time.Now when no custom clock is set")
	}

	moduleDir := writeFixture(t)
	mustWriteFile := func(rel, data string) {
		t.Helper()
		path := filepath.Join(moduleDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", rel, err)
		}
	}
	mustWriteFile("config.yaml", "enabled: true\n")
	mustWriteFile("docs/guide/note.txt", "hello\n")
	mustWriteFile("vendor/ignored.go", "package ignored\n")
	mustWriteFile("assets/release.json", "{}\n")

	cfg.Execution.CheckpointIncludes = []string{"**/*.yaml", "docs/**/note.txt", "assets/**"}
	e = New(cfg)
	mutants := []Mutant{{ID: "m2", Module: moduleDir}, {ID: "m1", Module: moduleDir}}
	cp := e.checkpoint(mutants, "partial")
	if cp.Fingerprint == "" || cp.Mutants != 2 || !cp.IncludesFileDigests || cp.Reason != "partial" {
		t.Fatalf("checkpoint() = %+v", cp)
	}
	fingerprints := e.checkpointFileFingerprints(mutants)
	joined := strings.Join(fingerprints, "\n")
	for _, want := range []string{"calc.go:", "calc_test.go:", "go.mod:", "config.yaml:", "docs/guide/note.txt:", "assets/release.json:"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("checkpointFileFingerprints missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "vendor/ignored.go:") {
		t.Fatalf("checkpointFileFingerprints should skip vendor paths:\n%s", joined)
	}

	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		t.Fatalf("ReadDir(moduleDir) error = %v", err)
	}
	var dirEntry, fileEntry os.DirEntry
	for _, entry := range entries {
		switch entry.Name() {
		case "vendor":
			dirEntry = entry
		case "go.mod":
			fileEntry = entry
		}
	}
	if !skipCheckpointWalkEntry(nil, nil) || !skipCheckpointWalkEntry(fileEntry, errors.New("boom")) || skipCheckpointWalkEntry(fileEntry, nil) {
		t.Fatal("skipCheckpointWalkEntry branches changed")
	}
	if checkpointDirAction(dirEntry) != filepath.SkipDir || checkpointDirAction(fileEntry) != nil {
		t.Fatal("checkpointDirAction classification changed")
	}
	if !globMatch("*.go", "calc.go") || !globMatch("**/*.yaml", "config.yaml") || !globMatch("docs/**/note.txt", "docs/guide/note.txt") || !globMatch("assets/**", "assets/release.json") {
		t.Fatal("globMatch did not cover expected include patterns")
	}
	if globMatch("docs/**/note.txt", "docs/guide/other.txt") {
		t.Fatal("globMatch should not match unrelated files")
	}
	if !shouldSkipCheckpointDir("vendor") || shouldSkipCheckpointDir("pkg") {
		t.Fatal("shouldSkipCheckpointDir branches changed")
	}
	if fingerprint, ok := e.checkpointFileFingerprint(moduleDir, filepath.Join(moduleDir, "config.yaml"), "config.yaml"); !ok || !strings.Contains(fingerprint, "config.yaml:") {
		t.Fatalf("checkpointFileFingerprint include = %q %v", fingerprint, ok)
	}
	if fingerprint, ok := e.checkpointFileFingerprint(moduleDir, filepath.Join(moduleDir, "missing.go"), "missing.go"); ok || fingerprint != "" {
		t.Fatalf("checkpointFileFingerprint missing file = %q %v", fingerprint, ok)
	}

	session := e.newRunSession()
	session.setCheckpointScope(mutants)
	scope := session.currentCheckpointScope()
	scope[0].ID = "mutated-copy"
	if session.currentCheckpointScope()[0].ID != "m2" {
		t.Fatal("currentCheckpointScope should return a copy")
	}
	if scoped := session.checkpointFromResults([]MutantResult{{MutantID: "ignored", Mutant: Mutant{ID: "ignored", Module: moduleDir}}}, "resume"); scoped.Mutants != len(mutants) {
		t.Fatalf("checkpointFromResults with scope = %+v", scoped)
	}
	session.setCheckpointScope(nil)
	fallback := session.checkpointFromResults([]MutantResult{
		{MutantID: "result-one", Mutant: Mutant{ID: "result-one", Module: moduleDir}},
		{MutantID: "", Mutant: Mutant{ID: "", Module: moduleDir}},
	}, "resume")
	if fallback.Mutants != 1 || fallback.Fingerprint == "" {
		t.Fatalf("checkpointFromResults fallback = %+v", fallback)
	}

	compacted := compactedResults([]MutantResult{{MutantID: "a"}, {MutantID: ""}, {MutantID: "b"}})
	if len(compacted) != 2 || compacted[0].MutantID != "a" || compacted[1].MutantID != "b" {
		t.Fatalf("compactedResults = %+v", compacted)
	}
	ordered := orderResults([]Mutant{{ID: "b"}, {ID: "a"}}, []MutantResult{{MutantID: "a"}, {MutantID: "b"}, {MutantID: "extra"}})
	if len(ordered) != 2 || ordered[0].MutantID != "b" || ordered[1].MutantID != "a" {
		t.Fatalf("orderResults = %+v", ordered)
	}
}

func TestMemoryLimitExceededBranches(t *testing.T) {
	state := exitStateForTest(t, 2)
	resources := config.Resources{MaxProcessMemoryMB: 64}
	errBoom := errors.New("boom")

	if memoryLimitExceeded(nil, state, resources, "out of memory") {
		t.Fatal("nil error should not count as memory limit exceeded")
	}
	if memoryLimitExceeded(errBoom, state, config.Resources{}, "out of memory") {
		t.Fatal("zero memory limit should not count as memory limit exceeded")
	}
	if !memoryLimitExceeded(errBoom, state, resources, "fatal: out of memory") {
		t.Fatal("explicit out-of-memory output should be detected")
	}
	if memoryLimitExceeded(errBoom, nil, resources, "plain failure") {
		t.Fatal("nil process state should not count as memory limit exceeded")
	}
	for _, output := range []string{"panic: boom", "build failed", "setup failed", "undefined: x", "syntax error", "FAIL fixture"} {
		if memoryLimitExceeded(errBoom, state, resources, output) {
			t.Fatalf("output %q should not count as memory limit exceeded", output)
		}
	}
	if !memoryLimitExceeded(errBoom, state, resources, "plain process exit") {
		t.Fatal("exit code 2 without conflicting output should count as memory limit exceeded")
	}
}

func TestPlatformAndEnvironmentHelperCoverage(t *testing.T) {
	plain := platformSensitivityPriority(Mutant{})
	sensitive := platformSensitivityPriority(Mutant{PlatformSensitive: true})
	if runtime.GOOS == "windows" {
		if sensitive <= plain {
			t.Fatalf("platformSensitivityPriority sensitive=%d plain=%d", sensitive, plain)
		}
	} else if sensitive != plain {
		t.Fatalf("platformSensitivityPriority should stay neutral off Windows: sensitive=%d plain=%d", sensitive, plain)
	}
	if !pathMentionsOneDrive(filepath.Join("C:\\Users", "user", "OneDrive - Personal", "repo")) {
		t.Fatal("pathMentionsOneDrive should detect OneDrive paths")
	}
	if pathMentionsOneDrive(filepath.Join("C:\\dev", "repo")) {
		t.Fatal("pathMentionsOneDrive should ignore non-OneDrive paths")
	}
	if runtime.GOOS == "windows" {
		if isWSL() {
			t.Fatal("isWSL should be false on native Windows")
		}
		if got := cgroupSummary(); got != "" {
			t.Fatalf("cgroupSummary on Windows = %q, want empty", got)
		}
	}
}

func TestCommandAndReportHelperCoverage(t *testing.T) {
	if got := normalizeGoFlags("-count=1 -p 4 ./..."); got != "-count=1 ./... -p=1" {
		t.Fatalf("normalizeGoFlags = %q", got)
	}
	if got := normalizeGoFlags("-count=1 -p=4 ./..."); got != "-count=1 ./... -p=1" {
		t.Fatalf("normalizeGoFlags inline = %q", got)
	}
	if !isGoTestFlagWithSeparateValue("-run") || isGoTestFlagWithSeparateValue("-run=TestOne") {
		t.Fatal("isGoTestFlagWithSeparateValue classification changed")
	}

	scoped := packageScopedCommand([]string{"go", "test", "-run", "TestOne", "./...", "-count=1"}, "./pkg/sample")
	if strings.Join(scoped, " ") != "go test -run TestOne ./pkg/sample -count=1" {
		t.Fatalf("packageScopedCommand replace = %q", strings.Join(scoped, " "))
	}
	scoped = packageScopedCommand([]string{"go", "test", "-count=1"}, "./pkg/sample")
	if strings.Join(scoped, " ") != "go test -count=1 ./pkg/sample" {
		t.Fatalf("packageScopedCommand append = %q", strings.Join(scoped, " "))
	}

	withProfile := withCoverProfile([]string{"go", "test", "./..."}, "cover.out")
	if strings.Join(withProfile, " ") != "go test -coverprofile cover.out ./..." {
		t.Fatalf("withCoverProfile add = %q", strings.Join(withProfile, " "))
	}
	preserved := withCoverProfile([]string{"go", "test", "-coverprofile", "existing.out", "./..."}, "cover.out")
	if strings.Join(preserved, " ") != "go test -coverprofile existing.out ./..." {
		t.Fatalf("withCoverProfile preserve = %q", strings.Join(preserved, " "))
	}
	if got := withCoverProfile([]string{"bash", "script.sh"}, "cover.out"); strings.Join(got, " ") != "bash script.sh" {
		t.Fatalf("withCoverProfile non-go = %q", strings.Join(got, " "))
	}

	longStack := strings.Repeat("stack-frame\n", 2000)
	if got := trimStack(longStack); len(got) != 8192 {
		t.Fatalf("trimStack length = %d, want 8192", len(got))
	}

	if !allStatus([]MutantResult{{Status: StatusKilled}, {Status: StatusKilled}}, StatusKilled) {
		t.Fatal("allStatus should accept uniform statuses")
	}
	if allStatus([]MutantResult{{Status: StatusKilled}, {Status: StatusSurvived}}, StatusKilled) {
		t.Fatal("allStatus should reject mixed statuses")
	}
	results := []MutantResult{
		{MutantID: "skip", Status: StatusPendingBudget},
		{MutantID: "later", Status: StatusSurvived},
	}
	if got := lastCompletedMutant(results); got != "later" {
		t.Fatalf("lastCompletedMutant = %q, want later", got)
	}

	dir := t.TempDir()
	eventPath := filepath.Join(dir, "progress.jsonl")
	if err := appendProgressEvent(eventPath, ProgressEvent{SchemaVersion: "1", Message: "ok"}); err != nil {
		t.Fatalf("appendProgressEvent returned error: %v", err)
	}
	if text, err := os.ReadFile(eventPath); err != nil || !strings.Contains(string(text), `"schema_version":"1"`) {
		t.Fatalf("appendProgressEvent data mismatch: %v %s", err, text)
	}

	cfg := config.Defaults()
	cfg.Reports.Output = filepath.Join(dir, "reports")
	result := RunResult{SchemaVersion: "1", Summary: Summary{Total: 1}}
	if err := New(cfg).writeReports(result); err != nil {
		t.Fatalf("writeReports returned error: %v", err)
	}
	for _, name := range []string{"mutation-report.json", "summary.txt"} {
		if _, err := os.Stat(filepath.Join(cfg.Reports.Output, name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}
}

func TestWriteFileAtomicAndPrepareOverlayMutationBranches(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "report.json")
	if err := writeFileAtomic(target, []byte("first"), 0o600); err != nil {
		t.Fatalf("writeFileAtomic(first) error = %v", err)
	}
	if err := writeFileAtomic(target, []byte("second"), 0o644); err != nil {
		t.Fatalf("writeFileAtomic(second) error = %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("writeFileAtomic data = %q", data)
	}
	if err := writeFileAtomic(filepath.Join(dir, "missing", "report.json"), []byte("x"), 0o644); err == nil {
		t.Fatal("writeFileAtomic accepted a path whose parent directory does not exist")
	}

	moduleDir := writeFixture(t)
	source := filepath.Join(moduleDir, "calc.go")
	sourceData, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read fixture source: %v", err)
	}
	start := strings.Index(string(sourceData), ">=")
	if start < 0 {
		t.Fatal("fixture source missing comparison token")
	}
	mutant := Mutant{
		ID:          "overlay",
		Module:      moduleDir,
		File:        source,
		Original:    ">=",
		Mutated:     ">",
		StartOffset: start,
		EndOffset:   start + len(">="),
	}

	missingSource := mutant
	missingSource.File = filepath.Join(moduleDir, "missing.go")
	if _, _, cleanup, err := prepareOverlayMutation(missingSource, []string{"go", "test", "."}, ""); err == nil {
		cleanup()
		t.Fatal("prepareOverlayMutation accepted a missing source file")
	}

	badOffsets := mutant
	badOffsets.StartOffset = len(sourceData) + 10
	badOffsets.EndOffset = badOffsets.StartOffset + 1
	if _, _, cleanup, err := prepareOverlayMutation(badOffsets, []string{"go", "test", "."}, ""); err == nil {
		cleanup()
		t.Fatal("prepareOverlayMutation accepted invalid mutant offsets")
	}
}

func exitStateForTest(t *testing.T, code int) *os.ProcessState {
	t.Helper()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", fmt.Sprintf("exit %d", code))
	} else {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("exit %d", code))
	}
	err := cmd.Run()
	if err == nil {
		t.Fatalf("exitStateForTest expected non-zero exit for code %d", code)
	}
	if cmd.ProcessState == nil {
		t.Fatalf("exitStateForTest missing process state for code %d", code)
	}
	return cmd.ProcessState
}
