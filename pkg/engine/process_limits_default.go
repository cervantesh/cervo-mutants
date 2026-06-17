//go:build !windows

package engine

import (
	"os/exec"

	"github.com/cervantesh/cervo-mutants/pkg/config"
)

func applyProcessLimits(cmd *exec.Cmd, resources config.Resources) (processLimitHandle, error) {
	return noopProcessLimitHandle(), nil
}
