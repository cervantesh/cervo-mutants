package engine

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/config"
)

type PanicError struct {
	Stage     string
	Recovered string
	Stack     string
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("internal_error: panic in %s: %s", e.Stage, e.Recovered)
}

func wrapStageError(kind string, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*PanicError); ok {
		return err
	}
	if strings.HasPrefix(err.Error(), kind+":") || strings.HasPrefix(err.Error(), "internal_error:") {
		return err
	}
	return fmt.Errorf("%s: %w", kind, err)
}

func recoverEnginePanic(stage string, err *error) {
	if recovered := recover(); recovered != nil {
		*err = &PanicError{
			Stage:     stage,
			Recovered: fmt.Sprint(recovered),
			Stack:     trimStack(string(debug.Stack())),
		}
	}
}

func trimStack(stack string) string {
	const maxBytes = 8192
	if len(stack) <= maxBytes {
		return stack
	}
	return stack[:maxBytes]
}

type BaselineFailureError struct {
	Result MutantResult
}

func (e *BaselineFailureError) Error() string {
	return "baseline tests failed before mutation"
}

func FailureResult(cfg config.Config, failure Failure) RunResult {
	result := RunResult{
		SchemaVersion: "1",
		Environment:   New(cfg).environment(0),
		Thresholds:    map[string]any{"fail_under": cfg.CI.FailUnder, "failed": true},
		Baseline:      BaselineComparison{Enabled: cfg.Baseline.Enabled},
		Quarantine:    QuarantineStats{},
		Cache:         CacheStats{},
		History:       HistoryStats{Enabled: cfg.History.Enabled, Path: cfg.History.Path},
		Mutants:       []MutantResult{},
		Failure:       &failure,
		StoppedReason: failure.Kind,
	}
	return result
}
