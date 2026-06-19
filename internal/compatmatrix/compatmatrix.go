package compatmatrix

import "strings"

const (
	SupportedGoVersion   = "1.25.6"
	SupportedGoSeries    = "1.25.x"
	SupportedGoMajor     = 1
	SupportedGoMinor     = 25
	CompatibilityDocPath = "docs/go-toolchain-compatibility.md"
	TestWorkflowPath     = ".github/workflows/test.yml"
	ReleaseWorkflowPath  = ".github/workflows/release.yml"
	GoModPath            = "go.mod"
)

type Target struct {
	DocLabel string
	GOOS     string
	Runner   string
}

func Targets() []Target {
	return []Target{
		{DocLabel: "Linux", GOOS: "linux", Runner: "ubuntu-latest"},
		{DocLabel: "Windows", GOOS: "windows", Runner: "windows-latest"},
		{DocLabel: "macOS", GOOS: "darwin", Runner: "macos-latest"},
	}
}

func CompatibilityMatrixText() string {
	return "supported Go versions: " + SupportedGoSeries + " validated on Linux, Windows, and macOS; newer versions warn until validated"
}

func ValidatedGoOSText() string {
	var goos []string
	for _, target := range Targets() {
		goos = append(goos, target.GOOS)
	}
	return "validated operating systems: " + strings.Join(goos, ", ")
}

func IsSupportedGOOS(goos string) bool {
	goos = strings.TrimSpace(strings.ToLower(goos))
	for _, target := range Targets() {
		if target.GOOS == goos {
			return true
		}
	}
	return false
}
