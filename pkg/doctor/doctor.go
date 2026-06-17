package doctor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	minSupportedGoMinor     = 25
	currentTestedGoMinor    = 25
	compatibilityMatrixText = "supported Go versions: 1.25.x validated on Linux, Windows, and macOS; newer versions warn until validated"
)

type Check struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Severity string `json:"severity,omitempty"`
	Message  string `json:"message"`
}

func Run(ctx context.Context) []Check {
	goVersion := checkCommand(ctx, "go", "version")
	checks := []Check{goVersion, checkCommand(ctx, "git", "--version")}
	if goVersion.OK {
		checks = append(checks, goToolchainChecks(ctx, goVersion.Message)...)
	}
	checks = append(checks, checkRuntimeEnvironment()...)
	return checks
}

func checkCommand(ctx context.Context, name string, args ...string) Check {
	cmd := exec.CommandContext(ctx, name, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	severity := "ok"
	if err != nil {
		severity = "fail"
	}
	return Check{Name: name, OK: err == nil, Severity: severity, Message: output.String()}
}

func goToolchainChecks(ctx context.Context, versionOutput string) []Check {
	checks := []Check{goVersionCompatibilityCheck(versionOutput), goOverlayCompatibilityCheck(versionOutput)}
	checks = append(checks, goEnvChecks(ctx)...)
	return checks
}

func goVersionCompatibilityCheck(output string) Check {
	major, minor, ok := parseGoMajorMinor(output)
	if !ok {
		return warning("go-version-compatibility", "unable to parse Go version; "+compatibilityMatrixText+"\n")
	}
	if major != 1 || minor < minSupportedGoMinor {
		return Check{Name: "go-version-compatibility", OK: false, Severity: "fail", Message: fmt.Sprintf("Go %d.%d is below the supported matrix; %s\n", major, minor, compatibilityMatrixText)}
	}
	if minor > currentTestedGoMinor {
		return warning("go-version-compatibility", fmt.Sprintf("Go %d.%d is newer than the validated matrix; %s\n", major, minor, compatibilityMatrixText))
	}
	return Check{Name: "go-version-compatibility", OK: true, Severity: "ok", Message: fmt.Sprintf("Go %d.%d is within the compatibility matrix; %s\n", major, minor, compatibilityMatrixText)}
}

func goOverlayCompatibilityCheck(output string) Check {
	major, minor, ok := parseGoMajorMinor(output)
	if !ok {
		return warning("go-overlay", "unable to confirm Go overlay support from version output\n")
	}
	if major != 1 || minor < 14 {
		return Check{Name: "go-overlay", OK: false, Severity: "fail", Message: "Go overlay isolation requires Go 1.14 or newer\n"}
	}
	return Check{Name: "go-overlay", OK: true, Severity: "ok", Message: "Go overlay isolation is supported by this toolchain\n"}
}

func goEnvChecks(ctx context.Context) []Check {
	values, err := goEnv(ctx, "GOTOOLCHAIN", "GOFLAGS", "GOMAXPROCS", "GOMEMLIMIT")
	if err != nil {
		return []Check{warning("go-env", fmt.Sprintf("unable to inspect go env: %v\n", err))}
	}
	checks := []Check{{Name: "go-env", OK: true, Severity: "ok", Message: goEnvSummary(values)}}
	if strings.EqualFold(values["GOTOOLCHAIN"], "auto") {
		checks = append(checks, warning("go-toolchain-auto", "GOTOOLCHAIN=auto can download toolchains during CI; pin toolchains for reproducible mutation runs\n"))
	}
	if values["GOFLAGS"] != "" && strings.Contains(values["GOFLAGS"], "-coverprofile") {
		checks = append(checks, warning("go-flags-coverprofile", "GOFLAGS contains -coverprofile; CervoMutants may override or conflict with coverage profile output\n"))
	}
	return checks
}

func goEnv(ctx context.Context, names ...string) (map[string]string, error) {
	args := append([]string{"env"}, names...)
	cmd := exec.CommandContext(ctx, "go", args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(output.String()))
	}
	lines := strings.Split(strings.TrimRight(output.String(), "\r\n"), "\n")
	values := map[string]string{}
	for i, name := range names {
		if i < len(lines) {
			values[name] = strings.TrimSpace(lines[i])
		}
	}
	return values, nil
}

func goEnvSummary(values map[string]string) string {
	return fmt.Sprintf("GOTOOLCHAIN=%s GOFLAGS=%s GOMAXPROCS=%s GOMEMLIMIT=%s\n", values["GOTOOLCHAIN"], values["GOFLAGS"], values["GOMAXPROCS"], values["GOMEMLIMIT"])
}

func parseGoMajorMinor(output string) (int, int, bool) {
	fields := strings.Fields(output)
	for _, field := range fields {
		if !strings.HasPrefix(field, "go") {
			continue
		}
		version := strings.TrimPrefix(field, "go")
		parts := strings.Split(version, ".")
		if len(parts) < 2 {
			continue
		}
		var major, minor int
		if _, err := fmt.Sscanf(parts[0]+"."+parts[1], "%d.%d", &major, &minor); err != nil {
			continue
		}
		return major, minor, true
	}
	return 0, 0, false
}

func checkRuntimeEnvironment() []Check {
	var checks []Check
	wd, _ := os.Getwd()
	temp := os.TempDir()
	checks = append(checks, Check{
		Name:     "runtime",
		OK:       true,
		Severity: "ok",
		Message:  fmt.Sprintf("%s/%s temp=%s\n", runtime.GOOS, runtime.GOARCH, temp),
	})
	if runtime.GOOS == "windows" {
		checks = append(checks, windowsChecks(wd, temp)...)
	}
	if runtime.GOOS == "linux" {
		checks = append(checks, linuxChecks()...)
	}
	return checks
}

func windowsChecks(wd, temp string) []Check {
	var checks []Check
	if mentionsOneDrive(wd) {
		checks = append(checks, warning("windows-onedrive", "workspace is under OneDrive; large mutation runs should use a short local temp/workdir outside synced folders\n"))
	}
	if mentionsOneDrive(temp) {
		checks = append(checks, warning("windows-temp-onedrive", "TEMP appears to be under OneDrive; set execution.temp_root or --temp-root to a short local path such as %LOCALAPPDATA%\\CervoMutants\\tmp\n"))
	}
	if len(wd) > 120 {
		checks = append(checks, warning("windows-long-path", fmt.Sprintf("workspace path is %d characters; long paths increase risk for external tools and temp workdirs\n", len(wd))))
	}
	if len(temp) > 120 {
		checks = append(checks, warning("windows-temp-long-path", fmt.Sprintf("TEMP path is %d characters; use a shorter local temp root for mutation runs\n", len(temp))))
	}
	volume := filepath.VolumeName(wd)
	if strings.HasPrefix(wd, `\\`) || strings.HasPrefix(volume, `\\`) {
		checks = append(checks, warning("windows-network-path", "workspace appears to be on a network/UNC path; local disk is recommended for mutation runs\n"))
	}
	if pathContainsOther(wd, temp) {
		checks = append(checks, warning("windows-temp-workspace", "TEMP shares the workspace tree; set execution.temp_root or --temp-root to a short local path outside the repo tree\n"))
	}
	if strings.Contains(wd, " ") {
		checks = append(checks, warning("windows-path-spaces", "workspace path contains spaces; keep temp roots short and local for easier Windows tool interoperability\n"))
	}
	checks = append(checks, warning("windows-resource-control", "for large Windows-native runs, use conservative workers, prefer Job Object/process-tree limits when available, and avoid very short per-mutant timeouts\n"))
	return checks
}

func linuxChecks() []Check {
	var checks []Check
	if isWSL() {
		checks = append(checks, Check{Name: "wsl", OK: true, Severity: "ok", Message: "running under WSL; Linux filesystem paths such as /tmp or $HOME are recommended over /mnt/c/OneDrive for large mutation runs\n"})
		if _, err := exec.LookPath("systemd-run"); err == nil {
			checks = append(checks, Check{Name: "cgroup-scope", OK: true, Severity: "ok", Message: "systemd-run is available for bounded local experiments\n"})
		} else {
			checks = append(checks, warning("cgroup-scope", "systemd-run not found; hard local cgroup limits may not be available\n"))
		}
	}
	return checks
}

func warning(name, message string) Check {
	return Check{Name: name, OK: true, Severity: "warn", Message: message}
}

func mentionsOneDrive(path string) bool {
	return strings.Contains(strings.ToLower(path), "onedrive")
}

func pathContainsOther(a, b string) bool {
	left := comparableWindowsPath(a)
	right := comparableWindowsPath(b)
	if left == "" || right == "" {
		return false
	}
	return left == right || strings.HasPrefix(left, right+`\`) || strings.HasPrefix(right, left+`\`)
}

func comparableWindowsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.Clean(path)
	path = strings.ReplaceAll(path, "/", `\`)
	path = strings.TrimRight(path, `\`)
	return strings.ToLower(path)
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	text := strings.ToLower(string(data))
	return strings.Contains(text, "microsoft") || strings.Contains(text, "wsl")
}
