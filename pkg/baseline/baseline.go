package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

type StatusChange struct {
	MutantID       string        `json:"mutant_id"`
	PreviousStatus engine.Status `json:"previous_status,omitempty"`
	CurrentStatus  engine.Status `json:"current_status,omitempty"`
}

type Diff struct {
	BaselineFound        bool           `json:"baseline_found"`
	PreviousScore        float64        `json:"previous_score"`
	CurrentScore         float64        `json:"current_score"`
	ScoreDelta           float64        `json:"score_delta"`
	PreviousActionable   float64        `json:"previous_actionable_score"`
	CurrentActionable    float64        `json:"current_actionable_score"`
	ActionableScoreDelta float64        `json:"actionable_score_delta"`
	NewSurvivors         []string       `json:"new_survivors,omitempty"`
	ResolvedSurvivors    []string       `json:"resolved_survivors,omitempty"`
	StatusChanges        []StatusChange `json:"status_changes,omitempty"`
}

func Load(path string) (engine.RunResult, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return engine.RunResult{}, false, nil
	}
	if err != nil {
		return engine.RunResult{}, false, err
	}
	var result engine.RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return engine.RunResult{}, false, err
	}
	return result, true, nil
}

func Save(path string, result engine.RunResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func CandidatePath(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + ".candidate"
	}
	base := strings.TrimSuffix(path, ext)
	return base + ".candidate" + ext
}

func Accept(path string, result engine.RunResult) (string, error) {
	candidatePath := CandidatePath(path)
	return candidatePath, Save(candidatePath, result)
}

func Promote(path string) (string, error) {
	candidatePath := CandidatePath(path)
	result, ok, err := Load(candidatePath)
	if err != nil {
		return candidatePath, err
	}
	if !ok {
		return candidatePath, fmt.Errorf("candidate baseline not found")
	}
	if err := Save(path, result); err != nil {
		return candidatePath, err
	}
	if err := os.Remove(candidatePath); err != nil && !os.IsNotExist(err) {
		return candidatePath, err
	}
	return candidatePath, nil
}

func Compare(previous, current engine.RunResult) engine.BaselineComparison {
	seen := map[string]engine.Status{}
	for _, mutant := range previous.Mutants {
		seen[mutant.MutantID] = mutant.Status
	}
	comparison := engine.BaselineComparison{
		Enabled:       true,
		PreviousScore: previous.Summary.Score,
		CurrentScore:  current.Summary.Score,
		Regression:    current.Summary.Score < previous.Summary.Score,
	}
	for _, mutant := range current.Mutants {
		if mutant.Status == engine.StatusSurvived && seen[mutant.MutantID] != engine.StatusSurvived {
			comparison.NewSurvivors = append(comparison.NewSurvivors, mutant.MutantID)
		}
	}
	return comparison
}

func BuildDiff(previous, current engine.RunResult) Diff {
	previousByID := map[string]engine.Status{}
	currentByID := map[string]engine.Status{}
	for _, mutant := range previous.Mutants {
		previousByID[mutant.MutantID] = mutant.Status
	}
	for _, mutant := range current.Mutants {
		currentByID[mutant.MutantID] = mutant.Status
	}
	diff := Diff{
		BaselineFound:        true,
		PreviousScore:        previous.Summary.Score,
		CurrentScore:         current.Summary.Score,
		ScoreDelta:           current.Summary.Score - previous.Summary.Score,
		PreviousActionable:   previous.Summary.Actionable.ActionableScore,
		CurrentActionable:    current.Summary.Actionable.ActionableScore,
		ActionableScoreDelta: current.Summary.Actionable.ActionableScore - previous.Summary.Actionable.ActionableScore,
	}
	seen := map[string]bool{}
	for id, previousStatus := range previousByID {
		currentStatus, ok := currentByID[id]
		if !ok {
			if previousStatus == engine.StatusSurvived {
				diff.ResolvedSurvivors = append(diff.ResolvedSurvivors, id)
			}
			diff.StatusChanges = append(diff.StatusChanges, StatusChange{
				MutantID:       id,
				PreviousStatus: previousStatus,
			})
			continue
		}
		seen[id] = true
		if currentStatus == engine.StatusSurvived && previousStatus != engine.StatusSurvived {
			diff.NewSurvivors = append(diff.NewSurvivors, id)
		}
		if previousStatus == engine.StatusSurvived && currentStatus != engine.StatusSurvived {
			diff.ResolvedSurvivors = append(diff.ResolvedSurvivors, id)
		}
		if previousStatus != currentStatus {
			diff.StatusChanges = append(diff.StatusChanges, StatusChange{
				MutantID:       id,
				PreviousStatus: previousStatus,
				CurrentStatus:  currentStatus,
			})
		}
	}
	for id, currentStatus := range currentByID {
		if seen[id] {
			continue
		}
		if currentStatus == engine.StatusSurvived {
			diff.NewSurvivors = append(diff.NewSurvivors, id)
		}
		diff.StatusChanges = append(diff.StatusChanges, StatusChange{
			MutantID:      id,
			CurrentStatus: currentStatus,
		})
	}
	sort.Strings(diff.NewSurvivors)
	sort.Strings(diff.ResolvedSurvivors)
	sort.Slice(diff.StatusChanges, func(i, j int) bool {
		return diff.StatusChanges[i].MutantID < diff.StatusChanges[j].MutantID
	})
	return diff
}

func FormatDiff(diff Diff) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Baseline found: %t\n", diff.BaselineFound)
	fmt.Fprintf(&b, "Raw score: %.2f%% -> %.2f%% (%+.2f)\n", diff.PreviousScore, diff.CurrentScore, diff.ScoreDelta)
	fmt.Fprintf(&b, "Actionable score: %.2f%% -> %.2f%% (%+.2f)\n", diff.PreviousActionable, diff.CurrentActionable, diff.ActionableScoreDelta)
	fmt.Fprintf(&b, "New survivors: %d\n", len(diff.NewSurvivors))
	fmt.Fprintf(&b, "Resolved survivors: %d\n", len(diff.ResolvedSurvivors))
	fmt.Fprintf(&b, "Status changes: %d\n", len(diff.StatusChanges))
	if len(diff.NewSurvivors) > 0 {
		b.WriteString("New survivors:\n")
		for _, mutantID := range diff.NewSurvivors {
			fmt.Fprintf(&b, "- %s\n", mutantID)
		}
	}
	if len(diff.ResolvedSurvivors) > 0 {
		b.WriteString("Resolved survivors:\n")
		for _, mutantID := range diff.ResolvedSurvivors {
			fmt.Fprintf(&b, "- %s\n", mutantID)
		}
	}
	if len(diff.StatusChanges) > 0 {
		b.WriteString("Status changes:\n")
		for _, change := range diff.StatusChanges {
			fmt.Fprintf(&b, "- %s: %s -> %s\n", change.MutantID, displayStatus(change.PreviousStatus), displayStatus(change.CurrentStatus))
		}
	}
	return b.String()
}

func displayStatus(status engine.Status) string {
	if status == "" {
		return "missing"
	}
	return string(status)
}
