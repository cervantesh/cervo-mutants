package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cervantesh/cervo-mutants/pkg/baseline"
	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

func cmdBaseline(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("baseline requires update or compare")
	}
	cfg := loadConfigIfPresent()
	switch args[0] {
	case "update":
		return updateBaseline(cfg)
	case "compare":
		return compareBaselineCommand(cfg)
	default:
		return fmt.Errorf("unknown baseline command %q", args[0])
	}
}

func updateBaseline(cfg config.Config) error {
	result, err := readRunReport(filepath.Join(cfg.Reports.Output, mutationReportFileName))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Baseline.Path), 0o755); err != nil {
		return err
	}
	return baseline.Save(cfg.Baseline.Path, result)
}

func compareBaselineCommand(cfg config.Config) error {
	prev, ok, err := baseline.Load(cfg.Baseline.Path)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("baseline not found")
	}
	current, err := readRunReport(filepath.Join(cfg.Reports.Output, mutationReportFileName))
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(baseline.Compare(prev, current))
}

func readRunReport(path string) (engine.RunResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return engine.RunResult{}, err
	}
	var result engine.RunResult
	return result, json.Unmarshal(data, &result)
}
