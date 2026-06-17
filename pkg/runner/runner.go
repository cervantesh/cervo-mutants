package runner

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

type GoTestRunner struct {
	MaxOutputBytes int
}

func (r GoTestRunner) Run(ctx context.Context, job engine.MutantJob) (engine.MutantResult, error) {
	timeout := 30 * time.Second
	if job.Timeout != "" {
		if parsed, err := time.ParseDuration(job.Timeout); err == nil {
			timeout = parsed
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	if len(job.TestCommand) == 0 {
		return engine.MutantResult{}, errors.New("test command is empty")
	}
	cmd := exec.CommandContext(runCtx, job.TestCommand[0], job.TestCommand[1:]...)
	cmd.Dir = job.WorkDir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	text := trim(output.String(), r.MaxOutputBytes)
	status, reason := classifyTestResult(text, err, runCtx.Err() == context.DeadlineExceeded)
	return engine.MutantResult{
		MutantID:     job.Mutant.ID,
		Status:       status,
		Duration:     time.Since(start),
		TestCommand:  job.TestCommand,
		StatusReason: reason,
		Output:       text,
		Mutant:       job.Mutant,
	}, nil
}

func classifyTestResult(text string, err error, timedOut bool) (engine.Status, string) {
	if timedOut {
		return engine.StatusTimedOut, "test command timed out"
	}
	if err == nil {
		return engine.StatusSurvived, "tests passed with mutant applied"
	}
	if failedBeforeAssertions(text) {
		return engine.StatusCompileError, "test command failed before running assertions"
	}
	if strings.Contains(text, "FAIL") {
		return engine.StatusKilled, "tests failed with mutant applied"
	}
	return engine.StatusCompileError, "test command failed before running assertions"
}

func failedBeforeAssertions(text string) bool {
	return strings.Contains(text, "[build failed]") || strings.Contains(text, "setup failed")
}

func trim(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
