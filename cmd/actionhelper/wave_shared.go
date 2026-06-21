package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
	reportpkg "github.com/cervantesh/cervo-mutants/pkg/report"
)

func selectWaveReport(reportDir string) (string, string) {
	full := filepath.Join(reportDir, "mutation-report.json")
	if fileExists(full) {
		return full, "full"
	}
	partial := filepath.Join(reportDir, "partial-mutation-report.json")
	if fileExists(partial) {
		return partial, "partial"
	}
	return "", "missing"
}

func readRunResult(path string) (engine.RunResult, error) {
	var result engine.RunResult
	err := readJSONFile(path, &result)
	return result, err
}

func readTriageLedger(path string) (reportpkg.TriageLedger, error) {
	var ledger reportpkg.TriageLedger
	if !fileExists(path) {
		return reportpkg.TriageLedger{Entries: []reportpkg.TriageLedgerEntry{}}, nil
	}
	if err := readJSONFile(path, &ledger); err != nil {
		return reportpkg.TriageLedger{}, err
	}
	if ledger.Entries == nil {
		ledger.Entries = []reportpkg.TriageLedgerEntry{}
	}
	return ledger, nil
}

func readGovernanceReview(path string) (reportpkg.GovernanceReview, error) {
	var review reportpkg.GovernanceReview
	if !fileExists(path) {
		return reportpkg.GovernanceReview{
			QuarantineTemplates:  []reportpkg.GovernanceQuarantineTemplate{},
			SuppressionTemplates: []reportpkg.GovernanceSuppressionTemplate{},
		}, nil
	}
	if err := readJSONFile(path, &review); err != nil {
		return reportpkg.GovernanceReview{}, err
	}
	if review.QuarantineTemplates == nil {
		review.QuarantineTemplates = []reportpkg.GovernanceQuarantineTemplate{}
	}
	if review.SuppressionTemplates == nil {
		review.SuppressionTemplates = []reportpkg.GovernanceSuppressionTemplate{}
	}
	return review, nil
}

func readJSONFile(path string, target any) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path must not be empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func nullableFloat64(value float64) *float64 {
	return &value
}

func cloneDenominatorHealth(src engine.DenominatorHealth) *engine.DenominatorHealth {
	dst := src
	dst.Warnings = append([]string{}, src.Warnings...)
	return &dst
}

func cloneEnvironment(src engine.Environment) *engine.Environment {
	dst := src
	dst.Warnings = append([]string{}, src.Warnings...)
	if src.Extra != nil {
		dst.Extra = map[string]string{}
		for key, value := range src.Extra {
			dst.Extra[key] = value
		}
	}
	return &dst
}

func cloneFailure(src engine.Failure) *engine.Failure {
	dst := src
	dst.Command = append([]string{}, src.Command...)
	dst.Targets = append([]string{}, src.Targets...)
	if src.RunnerResult != nil {
		runner := *src.RunnerResult
		runner.Command = append([]string{}, src.RunnerResult.Command...)
		dst.RunnerResult = &runner
	}
	return &dst
}

func denominatorField(health *engine.DenominatorHealth, selector func(engine.DenominatorHealth) int) int {
	if health == nil {
		return 0
	}
	return selector(*health)
}
