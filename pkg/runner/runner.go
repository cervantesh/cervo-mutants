package runner

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/engine"
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
	status := engine.StatusKilled
	reason := "tests failed with mutant applied"
	if runCtx.Err() == context.DeadlineExceeded {
		status = engine.StatusTimedOut
		reason = "test command timed out"
	} else if err == nil {
		status = engine.StatusSurvived
		reason = "tests passed with mutant applied"
	} else if strings.Contains(text, "FAIL") {
		status = engine.StatusKilled
	} else {
		status = engine.StatusCompileError
		reason = "test command failed before running assertions"
	}
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

func trim(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
