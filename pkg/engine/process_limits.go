package engine

import (
	"errors"

	"github.com/cervantesh/CervoMutants/pkg/config"
)

var errProcessLimitUnsupported = errors.New("process resource limits are not supported on this platform")

type processLimitStats struct {
	PeakProcessMemoryBytes int64
	PeakJobMemoryBytes     int64
}

type processLimitHandle struct {
	cleanup func()
	stats   func() processLimitStats
}

func (h processLimitHandle) Cleanup() {
	if h.cleanup != nil {
		h.cleanup()
	}
}

func (h processLimitHandle) Stats() processLimitStats {
	if h.stats == nil {
		return processLimitStats{}
	}
	return h.stats()
}

func noopProcessLimitHandle() processLimitHandle {
	return processLimitHandle{
		cleanup: noopProcessLimitCleanup,
		stats: func() processLimitStats {
			return processLimitStats{}
		},
	}
}

func hasProcessLimits(resources config.Resources) bool {
	return resources.MaxProcessMemoryMB > 0 || resources.MaxProcesses > 0
}

func noopProcessLimitCleanup() {
	// No process resources were acquired on platforms without process-limit support.
}
