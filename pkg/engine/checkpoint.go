package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

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
		fingerprints = append(fingerprints, e.moduleCheckpointFingerprints(module)...)
	}
	sort.Strings(fingerprints)
	return fingerprints
}

func (e *Engine) moduleCheckpointFingerprints(module string) []string {
	var fingerprints []string
	_ = filepath.WalkDir(module, func(path string, entry os.DirEntry, err error) error {
		if skipCheckpointWalkEntry(entry, err) {
			return checkpointDirAction(entry)
		}
		fingerprint, ok := e.checkpointFileFingerprint(module, path, entry.Name())
		if ok {
			fingerprints = append(fingerprints, fingerprint)
		}
		return nil
	})
	return fingerprints
}

func skipCheckpointWalkEntry(entry os.DirEntry, err error) bool {
	return err != nil || entry == nil || entry.IsDir()
}

func checkpointDirAction(entry os.DirEntry) error {
	if entry != nil && entry.IsDir() && shouldSkipCheckpointDir(entry.Name()) {
		return filepath.SkipDir
	}
	return nil
}

func (e *Engine) checkpointFileFingerprint(module, path, name string) (string, bool) {
	if !e.checkpointIncludesFile(module, path, name) {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(module, path)
	if err != nil {
		rel = path
	}
	return filepath.ToSlash(rel) + ":" + digestBytes(data), true
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
	for _, match := range []func(string, string) bool{directGlobMatch, recursivePrefixGlobMatch, recursiveSuffixGlobMatch, recursiveMiddleGlobMatch} {
		if match(pattern, rel) {
			return true
		}
	}
	return false
}

func directGlobMatch(pattern, rel string) bool {
	ok, err := filepath.Match(pattern, rel)
	return err == nil && ok
}

func recursivePrefixGlobMatch(pattern, rel string) bool {
	if !strings.Contains(pattern, "**/") {
		return false
	}
	return directGlobMatch(strings.ReplaceAll(pattern, "**/", ""), rel)
}

func recursiveSuffixGlobMatch(pattern, rel string) bool {
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return rel == prefix || strings.HasPrefix(rel, prefix+"/")
	}
	return false
}

func recursiveMiddleGlobMatch(pattern, rel string) bool {
	if !strings.Contains(pattern, "/**/") {
		return false
	}
	parts := strings.Split(pattern, "/**/")
	if len(parts) != 2 || !strings.HasPrefix(rel, parts[0]+"/") {
		return false
	}
	tail := strings.TrimPrefix(rel, parts[0]+"/")
	return directGlobMatch(parts[1], tail)
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
		Slice:         e.sliceMeta,
		Checkpoint:    e.checkpointFromResults(results, "partial"),
		Thresholds:    map[string]any{"fail_under": e.cfg.CI.FailUnder, "partial": true},
		Mutants:       append([]MutantResult{}, results...),
	}
	run.StoppedReason, run.LastCompletedMutant = runStopMetadata(run.Mutants)
	rankSurvivors(run.Mutants)
	run.Summary = summarize(run.Mutants)
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(e.cfg.Reports.Output, 0o755)
	_ = writeFileAtomic(filepath.Join(e.cfg.Reports.Output, "partial-mutation-report.json"), data, 0o644)
	summary, err := json.MarshalIndent(struct {
		SchemaVersion string         `json:"schema_version"`
		Checkpoint    Checkpoint     `json:"checkpoint"`
		Summary       Summary        `json:"summary"`
		Thresholds    map[string]any `json:"thresholds"`
	}{
		SchemaVersion: "1",
		Checkpoint:    run.Checkpoint,
		Summary:       run.Summary,
		Thresholds:    run.Thresholds,
	}, "", "  ")
	if err == nil {
		_ = writeFileAtomic(filepath.Join(e.cfg.Reports.Output, "partial-summary.json"), summary, 0o644)
	}
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
