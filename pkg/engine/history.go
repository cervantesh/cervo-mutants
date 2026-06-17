package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type historyFile struct {
	SchemaVersion string                  `json:"schema_version"`
	UpdatedAt     string                  `json:"updated_at"`
	Mutants       map[string]historyEntry `json:"mutants"`
}

type historyEntry struct {
	MutantID         string `json:"mutant_id"`
	Operator         string `json:"operator"`
	Status           Status `json:"status"`
	FirstSeen        string `json:"first_seen"`
	LastSeen         string `json:"last_seen"`
	SeenRuns         int    `json:"seen_runs"`
	SurvivedRuns     int    `json:"survived_runs"`
	KilledRuns       int    `json:"killed_runs"`
	NotCoveredRuns   int    `json:"not_covered_runs"`
	CompileErrorRuns int    `json:"compile_error_runs"`
	TimedOutRuns     int    `json:"timed_out_runs"`
}

func (e *Engine) applyHistory(results []MutantResult) HistoryStats {
	stats := HistoryStats{Enabled: e.cfg.History.Enabled, Path: e.cfg.History.Path, OperatorUsefulSurvivor: map[string]float64{}}
	if !e.cfg.History.Enabled {
		return stats
	}
	path := e.cfg.History.Path
	if path == "" {
		path = ".cervomut/history.json"
		stats.Path = path
	}
	store := historyFile{SchemaVersion: "1", Mutants: map[string]historyEntry{}}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &store)
	}
	if store.Mutants == nil {
		store.Mutants = map[string]historyEntry{}
	}
	stats.LoadedMutants = len(store.Mutants)
	now := time.Now().UTC().Format(time.RFC3339)
	operatorSeen := map[string]int{}
	operatorSurvived := map[string]int{}
	for i := range results {
		result := &results[i]
		operator := historyOperator(result.Mutant.Operator)
		entry := updateHistoryResult(result, store.Mutants[result.MutantID], now, &stats)
		store.Mutants[result.MutantID] = entry
		operatorSeen[operator]++
		if result.Status == StatusSurvived {
			operatorSurvived[operator]++
		}
	}
	for operator, seen := range operatorSeen {
		if seen > 0 {
			stats.OperatorUsefulSurvivor[operator] = float64(operatorSurvived[operator]) / float64(seen)
		}
	}
	for i := range results {
		results[i].OperatorYield = stats.OperatorUsefulSurvivor[results[i].Mutant.Operator]
	}
	stats.UpdatedMutants = len(results)
	store.UpdatedAt = now
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
		if data, err := json.MarshalIndent(store, "", "  "); err == nil {
			_ = os.WriteFile(path, data, 0o644)
		}
	}
	return stats
}

func historyOperator(operator string) string {
	if operator == "" {
		return "unknown"
	}
	return operator
}

func updateHistoryResult(result *MutantResult, previous historyEntry, now string, stats *HistoryStats) historyEntry {
	if previous.MutantID == "" {
		result.FirstSeen = now
		if result.Status == StatusSurvived {
			result.HistoryStatus = "new_survivor"
			stats.NewSurvivors++
		}
	} else {
		result.PreviousStatus = previous.Status
		result.FirstSeen = previous.FirstSeen
		result.SurvivorAgeRuns = previous.SurvivedRuns
		markExistingSurvivor(result, previous, stats)
	}
	if result.HistoryStatus == "" {
		result.HistoryStatus = "seen"
	}
	result.LastSeen = now
	entry := updateHistoryEntry(previous, *result, now)
	result.SurvivorAgeRuns = entry.SurvivedRuns
	return entry
}

func markExistingSurvivor(result *MutantResult, previous historyEntry, stats *HistoryStats) {
	if result.Status != StatusSurvived {
		return
	}
	result.HistoryStatus = "existing_survivor"
	if previous.SurvivedRuns > 0 {
		stats.LongStandingSurvivors++
		result.HistoryStatus = "long_standing_survivor"
	}
}

func updateHistoryEntry(entry historyEntry, result MutantResult, now string) historyEntry {
	entry.MutantID = result.MutantID
	entry.Operator = historyOperator(result.Mutant.Operator)
	entry.Status = result.Status
	if entry.FirstSeen == "" {
		entry.FirstSeen = result.FirstSeen
	}
	entry.LastSeen = now
	entry.SeenRuns++
	incrementHistoryStatus(&entry, result.Status)
	return entry
}

func incrementHistoryStatus(entry *historyEntry, status Status) {
	switch status {
	case StatusSurvived:
		entry.SurvivedRuns++
	case StatusKilled:
		entry.KilledRuns++
	case StatusNotCovered:
		entry.NotCoveredRuns++
	case StatusCompileError:
		entry.CompileErrorRuns++
	case StatusTimedOut:
		entry.TimedOutRuns++
	}
}
