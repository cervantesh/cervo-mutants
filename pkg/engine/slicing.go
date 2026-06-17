package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func (e *Engine) applySlicing(mutants []Mutant) ([]Mutant, SliceMetadata) {
	meta := SliceMetadata{
		SliceBy:              effectiveSliceBy(e.cfg.Scope.SliceBy, e.cfg.Scope.ShardCount),
		ShardIndex:           e.cfg.Scope.ShardIndex,
		ShardCount:           e.cfg.Scope.ShardCount,
		MaxFilesPerRun:       e.cfg.Limits.MaxFilesPerRun,
		MaxMutantsPerPackage: e.cfg.Limits.MaxMutantsPerPackage,
	}
	if len(mutants) == 0 {
		return mutants, meta
	}
	ordered := append([]Mutant{}, mutants...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })

	if meta.ShardCount > 0 {
		totalGroups := map[string]struct{}{}
		selectedGroups := map[string]struct{}{}
		filtered := make([]Mutant, 0, len(ordered))
		for _, mutant := range ordered {
			key := sliceGroupKey(mutant, meta.SliceBy)
			totalGroups[key] = struct{}{}
			if shardForKey(key, meta.ShardCount) != meta.ShardIndex {
				continue
			}
			selectedGroups[key] = struct{}{}
			filtered = append(filtered, mutant)
		}
		ordered = filtered
		meta.GroupCount = len(totalGroups)
		meta.SelectedGroups = len(selectedGroups)
		meta.Enabled = true
	}

	if meta.MaxFilesPerRun > 0 {
		seenFiles := map[string]struct{}{}
		filtered := make([]Mutant, 0, len(ordered))
		for _, mutant := range ordered {
			fileKey := filepath.ToSlash(mutant.File)
			if _, ok := seenFiles[fileKey]; !ok {
				if len(seenFiles) >= meta.MaxFilesPerRun {
					continue
				}
				seenFiles[fileKey] = struct{}{}
			}
			filtered = append(filtered, mutant)
		}
		ordered = filtered
		meta.SelectedFiles = len(seenFiles)
		meta.Enabled = true
	} else {
		meta.SelectedFiles = uniqueFileCount(ordered)
	}

	if meta.MaxMutantsPerPackage > 0 {
		perPackage := map[string]int{}
		filtered := make([]Mutant, 0, len(ordered))
		for _, mutant := range ordered {
			if perPackage[mutant.Package] >= meta.MaxMutantsPerPackage {
				continue
			}
			perPackage[mutant.Package]++
			filtered = append(filtered, mutant)
		}
		ordered = filtered
		meta.Enabled = true
	}

	meta.SelectedMutants = len(ordered)
	if !meta.Enabled {
		return ordered, SliceMetadata{}
	}
	return ordered, meta
}

func effectiveSliceBy(sliceBy string, shardCount int) string {
	if strings.TrimSpace(sliceBy) != "" {
		return sliceBy
	}
	if shardCount > 0 {
		return "mutant"
	}
	return ""
}

func sliceGroupKey(mutant Mutant, sliceBy string) string {
	switch sliceBy {
	case "package":
		if mutant.Package != "" {
			return mutant.Package
		}
	case "file":
		if mutant.File != "" {
			return filepath.ToSlash(mutant.File)
		}
	case "function":
		if mutant.Function != "" {
			return mutant.Function
		}
	case "operator":
		if mutant.Operator != "" {
			return mutant.Operator
		}
	case "mutant":
		if mutant.ID != "" {
			return mutant.ID
		}
	}
	if mutant.ID != "" {
		return mutant.ID
	}
	return filepath.ToSlash(mutant.File) + ":" + mutant.Operator
}

func shardForKey(key string, shardCount int) int {
	if shardCount <= 1 {
		return 1
	}
	sum := sha256.Sum256([]byte(key))
	slot := binary.BigEndian.Uint32(sum[:4]) % uint32(shardCount)
	return int(slot) + 1
}

func uniqueFileCount(mutants []Mutant) int {
	files := map[string]struct{}{}
	for _, mutant := range mutants {
		files[filepath.ToSlash(mutant.File)] = struct{}{}
	}
	return len(files)
}

func isWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	text := strings.ToLower(string(data))
	return strings.Contains(text, "microsoft") || strings.Contains(text, "wsl")
}

func cgroupSummary() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return ""
	}
	line := lines[0]
	if len(line) > 160 {
		line = line[:160]
	}
	return line
}

func pathMentionsOneDrive(path string) bool {
	return strings.Contains(strings.ToLower(path), "onedrive")
}
