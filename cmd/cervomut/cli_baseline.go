package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cervantesh/cervo-mutants/pkg/baseline"
	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

func cmdBaseline(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("baseline requires update, compare, diff, accept, or promote")
	}
	cfg := loadConfigIfPresent()
	switch args[0] {
	case "update":
		return updateBaseline(cfg)
	case "compare":
		return compareBaselineCommand(cfg)
	case "diff":
		return diffBaselineCommand(cfg, args[1:])
	case "accept":
		return acceptBaselineCommand(cfg)
	case "promote":
		return promoteBaselineCommand(cfg)
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

func diffBaselineCommand(cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("baseline diff", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "print baseline diff as JSON")
	useCandidate := fs.Bool("candidate", false, "compare the accepted candidate baseline instead of the current report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	prev, ok, err := baseline.Load(cfg.Baseline.Path)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("baseline not found")
	}
	currentPath := filepath.Join(cfg.Reports.Output, mutationReportFileName)
	current, err := readRunReport(currentPath)
	if err != nil {
		if *useCandidate {
			current, ok, err = baseline.Load(baseline.CandidatePath(cfg.Baseline.Path))
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("candidate baseline not found")
			}
		} else {
			return err
		}
	} else if *useCandidate {
		current, ok, err = baseline.Load(baseline.CandidatePath(cfg.Baseline.Path))
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("candidate baseline not found")
		}
	}
	diff := baseline.BuildDiff(prev, current)
	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(diff)
	}
	_, err = fmt.Fprint(os.Stdout, baseline.FormatDiff(diff))
	return err
}

func acceptBaselineCommand(cfg config.Config) error {
	current, err := readRunReport(filepath.Join(cfg.Reports.Output, mutationReportFileName))
	if err != nil {
		return err
	}
	candidatePath, err := baseline.Accept(cfg.Baseline.Path, current)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Accepted current report as candidate baseline: %s\n", filepath.ToSlash(candidatePath))
	if prev, ok, err := baseline.Load(cfg.Baseline.Path); err == nil && ok {
		_, err = fmt.Fprint(os.Stdout, baseline.FormatDiff(baseline.BuildDiff(prev, current)))
		return err
	} else if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "No existing baseline was found. Run `cervomut baseline promote` to publish this first accepted baseline.")
	return err
}

func promoteBaselineCommand(cfg config.Config) error {
	candidatePath, err := baseline.Promote(cfg.Baseline.Path)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "Promoted candidate baseline into %s and removed %s\n", filepath.ToSlash(cfg.Baseline.Path), filepath.ToSlash(candidatePath))
	return err
}

func readRunReport(path string) (engine.RunResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return engine.RunResult{}, err
	}
	var result engine.RunResult
	return result, json.Unmarshal(data, &result)
}
