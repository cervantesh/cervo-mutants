package examples

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/pkg/config"
)

func TestExampleWorkspacesStayRunnableAndDocumented(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	cases := []struct {
		name           string
		dir            string
		expectedPolicy string
	}{
		{name: "small-library", dir: filepath.Join(repoRoot, "examples", "small-library"), expectedPolicy: "ci-fast"},
		{name: "medium-service", dir: filepath.Join(repoRoot, "examples", "medium-service"), expectedPolicy: "ci-balanced"},
		{name: "large-repo-ci", dir: filepath.Join(repoRoot, "examples", "large-repo-ci"), expectedPolicy: "nightly"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(tc.dir, "cervomut.yaml")
			readmePath := filepath.Join(tc.dir, "README.md")
			workflowPath := filepath.Join(tc.dir, ".github", "workflows", "cervomut.yml")

			cfg, err := config.Load(configPath)
			if err != nil {
				t.Fatalf("config.Load(%s) returned error: %v", configPath, err)
			}
			if cfg.Policy != tc.expectedPolicy {
				t.Fatalf("config policy = %q, want %q", cfg.Policy, tc.expectedPolicy)
			}
			if !cfg.History.Enabled || !cfg.Baseline.Enabled {
				t.Fatalf("history/baseline should be enabled in %s: %+v", tc.name, cfg)
			}

			readmeData, err := os.ReadFile(readmePath)
			if err != nil {
				t.Fatalf("example README missing: %v", err)
			}
			if !strings.Contains(string(readmeData), "cervomut run") {
				t.Fatalf("example README should document cervomut run: %s", readmePath)
			}

			workflowData, err := os.ReadFile(workflowPath)
			if err != nil {
				t.Fatalf("example workflow missing: %v", err)
			}
			if !strings.Contains(string(workflowData), "cervomut run") {
				t.Fatalf("example workflow should invoke cervomut run: %s", workflowPath)
			}

			cmd := exec.Command("go", "test", "./...")
			cmd.Dir = tc.dir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go test ./... failed in %s: %v\n%s", tc.dir, err, output)
			}
		})
	}
}
