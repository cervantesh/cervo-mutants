package baseline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cervantesh/CervoMutants/pkg/engine"
)

func TestLoadMissingBaseline(t *testing.T) {
	result, ok, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if ok {
		t.Fatalf("Load ok = true for missing file: %+v", result)
	}
}

func TestLoadRejectsMalformedBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(path); err == nil {
		t.Fatal("Load accepted malformed baseline JSON")
	}
}

func TestSaveLoadAndCompareBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "baseline.json")
	previous := engine.RunResult{
		Summary: engine.Summary{Score: 80},
		Mutants: []engine.MutantResult{
			{MutantID: "old-survivor", Status: engine.StatusSurvived},
			{MutantID: "killed", Status: engine.StatusKilled},
		},
	}
	if err := Save(path, previous); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, ok, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !ok || loaded.Summary.Score != previous.Summary.Score || len(loaded.Mutants) != 2 {
		t.Fatalf("loaded baseline mismatch: ok=%t result=%+v", ok, loaded)
	}

	current := engine.RunResult{
		Summary: engine.Summary{Score: 70},
		Mutants: []engine.MutantResult{
			{MutantID: "old-survivor", Status: engine.StatusSurvived},
			{MutantID: "new-survivor", Status: engine.StatusSurvived},
		},
	}
	comparison := Compare(previous, current)
	if !comparison.Enabled || !comparison.Regression {
		t.Fatalf("comparison flags missing: %+v", comparison)
	}
	if len(comparison.NewSurvivors) != 1 || comparison.NewSurvivors[0] != "new-survivor" {
		t.Fatalf("new survivors mismatch: %+v", comparison)
	}
}
