package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manifest struct {
	SchemaVersion   string `json:"schema_version"`
	TrackingIssue   string `json:"tracking_issue"`
	SelectionPolicy string `json:"selection_policy"`
	Repos           []Repo `json:"repos"`
}

type Repo struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Target string `json:"target"`
	Lane   string `json:"lane"`
	Domain string `json:"domain"`
	Reason string `json:"reason"`
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func FilterRepos(manifest Manifest, names []string, limit int) []Repo {
	repos := append([]Repo(nil), manifest.Repos...)
	if len(names) > 0 {
		wanted := map[string]bool{}
		for _, name := range names {
			if name = strings.TrimSpace(name); name != "" {
				wanted[name] = true
			}
		}
		filtered := make([]Repo, 0, len(repos))
		for _, repo := range repos {
			if wanted[repo.Name] {
				filtered = append(filtered, repo)
			}
		}
		repos = filtered
	}
	if limit > 0 && limit < len(repos) {
		repos = repos[:limit]
	}
	return repos
}

type CommandSpec struct {
	Path                   string
	Args                   []string
	Dir                    string
	LogPath                string
	Timeout                time.Duration
	MinFreeMemoryMB        int
	MinFreeCommitMB        int
	KillBelowFreeMemoryMB  int
	KillBelowFreeCommitMB  int
	MemoryWait             time.Duration
	MemoryPoll             time.Duration
	MaxProcessTreeMemoryMB int
	Env                    []string
}

type CommandResult struct {
	ExitCode int
}

type CommandRunner interface {
	Run(context.Context, CommandSpec) (CommandResult, error)
}

type MemoryStatus struct {
	TotalMemoryMB int
	FreeMemoryMB  int
	TotalCommitMB int
	FreeCommitMB  int
}

type MemoryMonitor interface {
	Status() (MemoryStatus, error)
}

type RunSummary[T any] struct {
	Results     []T
	SummaryPath string
	Artifacts   map[string]string
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func defaultPath(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func seconds(start time.Time) float64 {
	return float64(time.Since(start).Milliseconds()) / 1000
}

func summaryPath(root string) string {
	return filepath.Join(root, "summary.json")
}

func studyJSONPath(root string) string {
	return filepath.Join(root, "comparison-study.json")
}

func studyMarkdownPath(root string) string {
	return filepath.Join(root, "comparison-summary.md")
}

func requiredBinary(name, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s binary path is required", name)
	}
	return value, nil
}
