package engine

import (
	"strings"

	internalgotestenv "github.com/cervantesh/cervo-mutants/pkg/internal/gotestenv"
)

func isGoTestCommand(command []string) bool {
	return internalgotestenv.IsGoTestCommand(command)
}

func packageScopedCommand(command []string, pkg string) []string {
	return internalgotestenv.PackageScopedCommand(command, pkg)
}

func withCoverProfile(command []string, profile string) []string {
	return internalgotestenv.WithCoverProfile(command, profile)
}

func withOverlayFlag(command []string, overlayPath string) []string {
	return internalgotestenv.WithOverlayFlag(command, overlayPath)
}

func isGoTestFlagWithSeparateValue(arg string) bool {
	if strings.Contains(arg, "=") {
		return false
	}
	switch arg {
	case "-run", "-bench", "-count", "-timeout", "-coverprofile", "-covermode", "-coverpkg", "-tags", "-cpu", "-parallel", "-shuffle":
		return true
	default:
		return false
	}
}
