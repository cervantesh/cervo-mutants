package doctor

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestWarningChecksDoNotFailDoctor(t *testing.T) {
	check := warning("windows-onedrive", "workspace is under OneDrive")
	if !check.OK {
		t.Fatal("warning checks should not fail doctor")
	}
	if check.Severity != "warn" {
		t.Fatalf("severity = %q, want warn", check.Severity)
	}
}

func TestRunIncludesCommandAndRuntimeChecks(t *testing.T) {
	checks := Run(context.Background())
	if len(checks) < 2 {
		t.Fatalf("checks = %d, want at least command checks", len(checks))
	}
	if !containsCheck(checks, "go") || !containsCheck(checks, "git") || !containsCheck(checks, "runtime") || !containsCheck(checks, "go-version-compatibility") {
		t.Fatalf("expected go/git/runtime checks: %+v", checks)
	}
}

func TestCheckCommandClassifiesMissingCommand(t *testing.T) {
	check := checkCommand(context.Background(), "cervomut-command-that-does-not-exist")
	if check.OK || check.Severity != "fail" {
		t.Fatalf("missing command check = %+v, want failed", check)
	}
}

func TestRuntimeEnvironmentHelpers(t *testing.T) {
	if !mentionsOneDrive(`C:\Users\me\OneDrive\Documents`) {
		t.Fatal("mentionsOneDrive should be case-insensitive")
	}
	if mentionsOneDrive(`/tmp/work`) {
		t.Fatal("mentionsOneDrive should ignore unrelated paths")
	}

	checks := windowsChecks(strings.Repeat("x", 121), `C:\Temp`)
	if len(checks) == 0 {
		t.Fatal("windowsChecks should report conservative Windows guidance")
	}
	checks = windowsChecks(`C:\Users\me\OneDrive\Documents\project`, `C:\Users\me\OneDrive\Temp`)
	if !containsCheck(checks, "windows-onedrive") || !containsCheck(checks, "windows-temp-onedrive") {
		t.Fatalf("windowsChecks missing OneDrive warnings: %+v", checks)
	}
	checks = windowsChecks(`\\server\share\project`, `C:\Temp`)
	if !containsCheck(checks, "windows-network-path") {
		t.Fatalf("windowsChecks missing network path warning: %+v", checks)
	}
	if runtime.GOOS != "linux" && isWSL() {
		t.Fatal("isWSL should be false outside Linux")
	}
	_ = linuxChecks()
}

func TestGoVersionCompatibilityChecks(t *testing.T) {
	ok := goVersionCompatibilityCheck("go version go1.25.6 windows/amd64")
	if !ok.OK || ok.Severity != "ok" {
		t.Fatalf("go1.25 should be supported: %+v", ok)
	}
	old := goVersionCompatibilityCheck("go version go1.23.9 linux/amd64")
	if old.OK || old.Severity != "fail" {
		t.Fatalf("old Go should fail compatibility: %+v", old)
	}
	future := goVersionCompatibilityCheck("go version go1.26.0 linux/amd64")
	if !future.OK || future.Severity != "warn" {
		t.Fatalf("future Go should warn: %+v", future)
	}
	unknown := goVersionCompatibilityCheck("not a go version")
	if !unknown.OK || unknown.Severity != "warn" {
		t.Fatalf("unknown Go version should warn: %+v", unknown)
	}
}

func TestGoToolchainEnvHelpers(t *testing.T) {
	if major, minor, ok := parseGoMajorMinor("go version go1.25.6 windows/amd64"); !ok || major != 1 || minor != 25 {
		t.Fatalf("parseGoMajorMinor = %d.%d ok=%t", major, minor, ok)
	}
	if _, _, ok := parseGoMajorMinor("garbage"); ok {
		t.Fatal("parseGoMajorMinor accepted garbage")
	}
	overlay := goOverlayCompatibilityCheck("go version go1.13.15 linux/amd64")
	if overlay.OK {
		t.Fatalf("old Go should not support overlay: %+v", overlay)
	}
	summary := goEnvSummary(map[string]string{"GOTOOLCHAIN": "auto", "GOFLAGS": "-p=1", "GOMAXPROCS": "1", "GOMEMLIMIT": "3GiB"})
	for _, want := range []string{"GOTOOLCHAIN=auto", "GOFLAGS=-p=1", "GOMAXPROCS=1", "GOMEMLIMIT=3GiB"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("go env summary missing %q: %s", want, summary)
		}
	}
}

func containsCheck(checks []Check, name string) bool {
	for _, check := range checks {
		if check.Name == name {
			return true
		}
	}
	return false
}
