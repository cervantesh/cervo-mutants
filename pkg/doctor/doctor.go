package doctor

import (
	"bytes"
	"context"
	"os/exec"
)

type Check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func Run(ctx context.Context) []Check {
	return []Check{checkCommand(ctx, "go", "version"), checkCommand(ctx, "git", "--version")}
}

func checkCommand(ctx context.Context, name string, args ...string) Check {
	cmd := exec.CommandContext(ctx, name, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return Check{Name: name, OK: err == nil, Message: output.String()}
}
