package selecttest

import (
	"context"
	"strings"
	"testing"

	"github.com/cervantesh/CervoMutants/pkg/engine"
)

func TestSelectorDefaultsToAllTests(t *testing.T) {
	plan, err := (Selector{Mode: "all"}).Select(context.Background(), engine.Mutant{})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if strings.Join(plan.Command, " ") != "go test ./..." || plan.Reason != "all tests selected" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestSelectorUsesPackageFallback(t *testing.T) {
	selector := Selector{Command: []string{"go", "test", "./..."}}
	plan, err := selector.Select(context.Background(), engine.Mutant{Package: "./pkg/config"})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if strings.Join(plan.Command, " ") != "go test ./pkg/config" {
		t.Fatalf("package fallback command mismatch: %+v", plan)
	}
}

func TestSelectorCoverageModeExplainsFallback(t *testing.T) {
	plan, err := (Selector{Mode: "coverage", Command: []string{"go", "test", "./..."}}).Select(context.Background(), engine.Mutant{Package: "./pkg/config"})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if plan.CoverageSource != "coverage-mode" || !strings.Contains(plan.Reason, "package fallback") {
		t.Fatalf("coverage fallback not explained: %+v", plan)
	}
}
