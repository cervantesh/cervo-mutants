//go:build !windows

package engine

import (
	"os/exec"

	"github.com/cervantesh/CervoMutants/pkg/config"
)

func applyProcessLimits(cmd *exec.Cmd, resources config.Resources) (processLimitHandle, error) {
	if hasProcessLimits(resources) {
		return noopProcessLimitHandle(), errProcessLimitUnsupported
	}
	return noopProcessLimitHandle(), nil
}
