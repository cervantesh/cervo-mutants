package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
	"github.com/cervantesh/cervo-mutants/pkg/isolate"
	"github.com/cervantesh/cervo-mutants/pkg/runner"
)

type Message struct {
	Type   string              `json:"type"`
	Job    engine.MutantJob    `json:"job,omitempty"`
	Result engine.MutantResult `json:"result,omitempty"`
	Error  string              `json:"error,omitempty"`
}

func ServeJSONLines(ctx context.Context, in io.Reader, out io.Writer, runner engine.Runner) error {
	scanner := bufio.NewScanner(in)
	enc := json.NewEncoder(out)
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			_ = enc.Encode(Message{Type: "error", Error: err.Error()})
			continue
		}
		if msg.Type != "job" {
			_ = enc.Encode(Message{Type: "error", Error: "unsupported message type"})
			continue
		}
		result, err := runner.Run(ctx, msg.Job)
		if err != nil {
			_ = enc.Encode(Message{Type: "error", Error: err.Error()})
			continue
		}
		if err := enc.Encode(Message{Type: "result", Result: result}); err != nil {
			return err
		}
	}
	return scanner.Err()
}

type WorkerRunner struct {
	MaxOutputBytes int
}

func (r WorkerRunner) Run(ctx context.Context, job engine.MutantJob) (engine.MutantResult, error) {
	moduleDir := job.Mutant.Module
	if moduleDir == "" {
		moduleDir = job.WorkDir
	}
	workdir, err := isolate.CopyModule(moduleDir)
	if err != nil {
		return engine.MutantResult{}, err
	}
	defer isolate.Cleanup(workdir)
	targetFile, err := isolate.ContainedTargetPath(moduleDir, workdir, job.Mutant.File)
	if err != nil {
		return engine.MutantResult{}, err
	}
	if err := applyPatch(targetFile, job.Mutant); err != nil {
		return engine.MutantResult{}, err
	}
	job.WorkDir = workdir
	return runner.GoTestRunner{MaxOutputBytes: r.MaxOutputBytes}.Run(ctx, job)
}

func applyPatch(path string, mutant engine.Mutant) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if mutant.StartOffset < 0 || mutant.EndOffset > len(data) || mutant.StartOffset >= mutant.EndOffset {
		return os.ErrInvalid
	}
	segment := string(data[mutant.StartOffset:mutant.EndOffset])
	if !strings.Contains(segment, mutant.Original) {
		return os.ErrInvalid
	}
	replaced := strings.Replace(segment, mutant.Original, mutant.Mutated, 1)
	next := string(data[:mutant.StartOffset]) + replaced + string(data[mutant.EndOffset:])
	return os.WriteFile(path, []byte(next), 0o644)
}
