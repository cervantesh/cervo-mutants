package engine

import (
	"encoding/json"
	"errors"
	"github.com/cervantesh/cervo-mutants/internal/testharness"
	"github.com/cervantesh/cervo-mutants/pkg/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFixture(t *testing.T) string {
	t.Helper()
	return testharness.WriteGoModuleTempDir(t, "fixture", fixtureFiles())
}

func writeFixtureFiles(t *testing.T, dir string) {
	t.Helper()
	testharness.WriteGoModuleFixture(t, dir, "fixture", fixtureFiles())
}

func fixtureFiles() map[string]string {
	return map[string]string{
		"calc.go": `package fixture

func IsPositiveOrZero(n int) bool {
	return n >= 0
}
`,
		"calc_test.go": `package fixture

import "testing"

func TestIsPositiveOrZero(t *testing.T) {
	if !IsPositiveOrZero(1) {
		t.Fatal("want positive")
	}
}
`,
	}
}

func isolateArtifacts(cfg *config.Config, dir string) {
	cfg.Reports.Output = filepath.Join(dir, ".cervomut", "reports")
	cfg.Cache.Path = filepath.Join(dir, ".cervomut", "cache")
	cfg.Selection.CoverageProfile = filepath.Join(dir, ".cervomut", "coverage.out")
	cfg.Selection.TimingsPath = filepath.Join(dir, ".cervomut", "timings.json")
	cfg.Baseline.Path = filepath.Join(dir, ".cervomut", "baseline.json")
	cfg.Quarantine.Path = filepath.Join(dir, ".cervomut", "quarantine.json")
	cfg.History.Path = filepath.Join(dir, ".cervomut", "history.json")
	cfg.Execution.TempRoot = filepath.Join(dir, ".cervomut", "tmp")
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func assertGlobBranches(t *testing.T) {
	t.Helper()
	if !globMatch("testdata/**", "testdata/case.json") {
		t.Fatal("globMatch should match recursive suffix pattern")
	}
	if !globMatch("testdata/**", "testdata") {
		t.Fatal("globMatch should match recursive suffix root")
	}
	if !globMatch("pkg/**/*.go", "pkg/deep/calc.go") {
		t.Fatal("globMatch should match middle recursive pattern")
	}
	if globMatch("pkg/**/case.go", "cmd/case.go") {
		t.Fatal("globMatch middle pattern matched wrong prefix")
	}
	if globMatch("pkg/**/case.go", "pkg/deep/case.go/extra") {
		t.Fatal("globMatch middle pattern matched wrong tail")
	}
	if !globMatch("**/*.go", "calc.go") {
		t.Fatal("globMatch should match recursive prefix pattern")
	}
	if globMatch("pkg/*.go", "cmd/main.go") {
		t.Fatal("globMatch matched unrelated path")
	}
	if !suppressionFileMatches("pkg/*.go", "pkg/calc.go") || !suppressionFileMatches("calc.go", "pkg/calc.go") {
		t.Fatal("suppressionFileMatches should support glob and suffix matches")
	}
}

func assertClassificationBranches(t *testing.T) {
	t.Helper()
	if classifyFailure("panic: boom", nil) != "test_panic" {
		t.Fatal("panic output should classify as test_panic")
	}
	if classifyFailure("undefined: Symbol", nil) != "compile_error" {
		t.Fatal("undefined output should classify as compile_error")
	}
	if classifyFailure("cannot find go", nil) != "environment_error" {
		t.Fatal("missing binary output should classify as environment_error")
	}
	if classifyFailure("", errors.New("runner failed")) != "runner_error" {
		t.Fatal("plain runner error should classify as runner_error")
	}
	noopCleanup()
	noopProcessLimitCleanup()
	if !fallbackCoverageMentions("calc.go:1.1,2.1 1 1", "calc.go", "calc.go") {
		t.Fatal("fallbackCoverageMentions should detect raw coverage line")
	}
	if compacted := compactedResults([]MutantResult{{}, {MutantID: "m1"}}); len(compacted) != 1 || compacted[0].MutantID != "m1" {
		t.Fatalf("compactedResults = %+v", compacted)
	}
}

func assertEnvironmentBranches(t *testing.T) {
	t.Helper()
	cfg := config.Defaults()
	cfg.Execution.Resources.MaxProcessMemoryMB = 64
	cfg.Execution.Budget = time.Minute
	env := New(cfg).environment(2)
	if env.Extra["max_process_memory_mb"] != "64" || env.Budget != "1m0s" {
		t.Fatalf("environment did not expose limits: %+v", env)
	}
	if New(config.Defaults()).workerCount(1) != 1 {
		t.Fatal("workerCount should cap workers to mutant count")
	}
}

func assertEngineTargetBranches(t *testing.T) {
	t.Helper()
	if targets := New(config.Defaults()).runTargets(nil); len(targets) == 0 {
		t.Fatal("runTargets should fall back to configured scope")
	}
	if _, err := New(config.Defaults()).discoverMutants([]string{filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("discoverMutants accepted missing target")
	}
	wd, err := moduleForTargets(nil)
	if err != nil || wd == "" {
		t.Fatalf("moduleForTargets(nil) = %q err=%v", wd, err)
	}
	if got := effectiveWorkerCount("windows", config.IsolationTempWorkdir, 4, 10); got != 2 {
		t.Fatalf("effectiveWorkerCount windows temp-workdir = %d, want 2", got)
	}
	if got := effectiveWorkerCount("windows", "overlay", 4, 10); got != 4 {
		t.Fatalf("effectiveWorkerCount windows overlay = %d, want 4", got)
	}
	if got := effectiveWorkerCount("linux", config.IsolationTempWorkdir, 4, 1); got != 1 {
		t.Fatalf("effectiveWorkerCount linux mutant cap = %d, want 1", got)
	}
	plan := effectiveTestCommandEnv("windows", config.IsolationTempWorkdir, 2, []string{"go", "test", "./..."}, []string{"PATH=C:\\Windows\\System32"})
	if !plan.Applied || plan.GOMAXPROCS != "2" || !strings.Contains(plan.GoFlags, "-p=1") {
		t.Fatalf("effectiveTestCommandEnv did not apply conservative settings: %+v", plan)
	}
	plan = effectiveTestCommandEnv("windows", "overlay", 1, []string{"go", "test", "./..."}, []string{"PATH=C:\\Windows\\System32"})
	if plan.Applied {
		t.Fatalf("effectiveTestCommandEnv should not apply for already-conservative overlay run: %+v", plan)
	}
	plan = effectiveTestCommandEnv("windows", config.IsolationTempWorkdir, 2, []string{"echo", "ok"}, []string{"PATH=C:\\Windows\\System32"})
	if plan.Applied {
		t.Fatalf("effectiveTestCommandEnv should ignore non-go-test commands: %+v", plan)
	}
}

type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Name() string { return f.name }

func (f fakeDirEntry) IsDir() bool { return f.dir }

func (f fakeDirEntry) Type() os.FileMode { return 0 }

func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, errors.New("no info") }

func assertCacheStore(t *testing.T, e *Engine) {
	t.Helper()
	session := e.newRunSession()
	if _, ok, err := session.getCached("missing"); err != nil || ok {
		t.Fatalf("missing cache = ok %t err %v", ok, err)
	}
	if err := session.putCached("hit", MutantResult{MutantID: "m1", Status: StatusKilled}); err != nil {
		t.Fatalf("putCached returned error: %v", err)
	}
	if cached, ok, err := session.getCached("hit"); err != nil || !ok || cached.MutantID != "m1" {
		t.Fatalf("cached = %+v ok=%t err=%v", cached, ok, err)
	}
}

func assertBaselineStore(t *testing.T, e *Engine, path string) {
	t.Helper()
	session := e.newRunSession()
	if _, ok, err := session.loadBaseline(); err != nil || ok {
		t.Fatalf("missing baseline = ok %t err %v", ok, err)
	}
	baseline := RunResult{SchemaVersion: "1", Summary: Summary{Score: 90}}
	data, _ := json.Marshal(baseline)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if loaded, ok, err := session.loadBaseline(); err != nil || !ok || loaded.Summary.Score != 90 {
		t.Fatalf("loaded baseline = %+v ok=%t err=%v", loaded, ok, err)
	}
}

func assertQuarantineLoad(t *testing.T, e *Engine, path string) {
	t.Helper()
	session := e.newRunSession()
	entries := []map[string]any{{
		"mutant_id":  "m-active",
		"reason":     "temporary",
		"owner":      "qa",
		"issue":      "cervantesh/CervoMutants#31",
		"created_at": time.Now().Add(-time.Hour).Format(time.RFC3339),
		"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
	}}
	quarantineData, _ := json.Marshal(entries)
	if err := os.WriteFile(path, quarantineData, 0o600); err != nil {
		t.Fatal(err)
	}
	active, expired, err := session.loadQuarantine()
	if err != nil {
		t.Fatalf("loadQuarantine returned error: %v", err)
	}
	if !active["m-active"] || expired != 0 {
		t.Fatalf("unexpected quarantine state active=%+v expired=%d", active, expired)
	}
}

func assertPriorityHelpers(t *testing.T) {
	t.Helper()
	for action, want := range map[string]int{"report-only": 0, "lower-priority": 1, "quarantine-required": 2, "suppress": 3, "none": -1} {
		if got := suppressionPriority(action); got != want {
			t.Fatalf("suppressionPriority(%q) = %d, want %d", action, got, want)
		}
	}
	if hasProcessLimits(config.Resources{}) {
		t.Fatal("empty resource limits should not enable process limits")
	}
	if !hasProcessLimits(config.Resources{MaxProcessMemoryMB: 1}) || !hasProcessLimits(config.Resources{MaxProcesses: 1}) {
		t.Fatal("configured memory or process cap should enable process limits")
	}
	noopProcessLimitCleanup()
}

func statusStrings(statuses []Status) []string {
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, string(status))
	}
	return values
}

func assertPanicError(t *testing.T, err error, recovered string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected panic error")
	}
	var panicErr *PanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("error = %T %v, want PanicError", err, err)
	}
	if panicErr.Stage != "run" {
		t.Fatalf("panic stage = %q, want run", panicErr.Stage)
	}
	if !strings.Contains(panicErr.Recovered, recovered) {
		t.Fatalf("recovered = %q, want substring %q", panicErr.Recovered, recovered)
	}
	if panicErr.Stack == "" {
		t.Fatal("panic stack trace was empty")
	}
}
