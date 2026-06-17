package engine

import (
	"errors"

	"github.com/cervantesh/CervoMutants/pkg/config"
)

var errProcessLimitUnsupported = errors.New("process resource limits are not supported on this platform")

func hasProcessLimits(resources config.Resources) bool {
	return resources.MaxProcessMemoryMB > 0 || resources.MaxProcesses > 0
}

func noopProcessLimitCleanup() {
	// No process resources were acquired on platforms without process-limit support.
}
