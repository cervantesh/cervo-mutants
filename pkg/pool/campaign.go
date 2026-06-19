package pool

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultCampaignSmokeMaxMutants             = 10
	defaultCampaignSmokeWorkers                = 2
	defaultCampaignSmokeCloneTimeoutSeconds    = 180
	defaultCampaignSmokeTestTimeoutSeconds     = 120
	defaultCampaignSmokeDryRunTimeoutSeconds   = 120
	defaultCampaignSmokeMutationTimeoutSeconds = 300

	defaultCampaignCompareWorkers               = 2
	defaultCampaignCompareGremlinsTimeoutFactor = 1
	defaultCampaignCompareGomuWorkers           = 1
	defaultCampaignCompareGoMutestingWorkers    = 1
	defaultCampaignCompareTimeoutSeconds        = 600
	defaultCampaignCompareMinFreeMemoryMB       = 4096
	defaultCampaignCompareMinFreeCommitMB       = 8192
	defaultCampaignCompareKillFreeMemoryMB      = 2048
	defaultCampaignCompareKillFreeCommitMB      = 4096
	defaultCampaignCompareMemoryWaitSeconds     = 900
	defaultCampaignCompareMemoryPollSeconds     = 5

	defaultCampaignWorkDirName   = "cervomut-pool-campaign"
	defaultCampaignOutputDirName = "cervomut-pool-campaign-results"
)

type CampaignManifest struct {
	SchemaVersion string        `json:"schema_version"`
	TrackingIssue string        `json:"tracking_issue,omitempty"`
	Description   string        `json:"description,omitempty"`
	WorkRoot      string        `json:"work_root,omitempty"`
	OutputRoot    string        `json:"output_root,omitempty"`
	Jobs          []CampaignJob `json:"jobs"`
}

type CampaignJob struct {
	Name                       string   `json:"name"`
	Kind                       string   `json:"kind"`
	Enabled                    *bool    `json:"enabled,omitempty"`
	ManifestPath               string   `json:"manifest_path,omitempty"`
	CorpusPath                 string   `json:"corpus_path,omitempty"`
	WorkRoot                   string   `json:"work_root,omitempty"`
	OutputRoot                 string   `json:"output_root,omitempty"`
	Names                      []string `json:"names,omitempty"`
	Tools                      []string `json:"tools,omitempty"`
	Limit                      int      `json:"limit,omitempty"`
	Resume                     bool     `json:"resume,omitempty"`
	RunMutation                bool     `json:"run_mutation,omitempty"`
	MaxMutants                 int      `json:"max_mutants,omitempty"`
	Workers                    int      `json:"workers,omitempty"`
	CloneTimeoutSeconds        int      `json:"clone_timeout_seconds,omitempty"`
	TestTimeoutSeconds         int      `json:"test_timeout_seconds,omitempty"`
	DryRunTimeoutSeconds       int      `json:"dry_run_timeout_seconds,omitempty"`
	MutationTimeoutSeconds     int      `json:"mutation_timeout_seconds,omitempty"`
	CompareTargetMode          string   `json:"compare_target_mode,omitempty"`
	GremlinsTargetMode         string   `json:"gremlins_target_mode,omitempty"`
	GremlinsTimeoutCoefficient int      `json:"gremlins_timeout_coefficient,omitempty"`
	GomuWorkers                int      `json:"gomu_workers,omitempty"`
	GoMutestingWorkers         int      `json:"go_mutesting_workers,omitempty"`
	TimeoutSeconds             int      `json:"timeout_seconds,omitempty"`
	MinFreeMemoryMB            int      `json:"min_free_memory_mb,omitempty"`
	MinFreeCommitMB            int      `json:"min_free_commit_mb,omitempty"`
	KillBelowFreeMemoryMB      int      `json:"kill_below_free_memory_mb,omitempty"`
	KillBelowFreeCommitMB      int      `json:"kill_below_free_commit_mb,omitempty"`
	MaxUsedMemoryMB            int      `json:"max_used_memory_mb,omitempty"`
	MaxCommittedMemoryMB       int      `json:"max_committed_memory_mb,omitempty"`
	MaxProcessTreeMemoryMB     int      `json:"max_process_tree_memory_mb,omitempty"`
	MemoryWaitSeconds          int      `json:"memory_wait_seconds,omitempty"`
	MemoryPollSeconds          int      `json:"memory_poll_seconds,omitempty"`
	GoMemoryLimit              string   `json:"go_memory_limit,omitempty"`
	GoMaxProcs                 int      `json:"go_max_procs,omitempty"`
	GoFlags                    string   `json:"go_flags,omitempty"`
	Notes                      string   `json:"notes,omitempty"`
}

type CampaignJobResult struct {
	Name           string            `json:"name"`
	Kind           string            `json:"kind"`
	Status         string            `json:"status"`
	ResumeKey      string            `json:"resume_key,omitempty"`
	ManifestPath   string            `json:"manifest_path,omitempty"`
	CorpusPath     string            `json:"corpus_path,omitempty"`
	WorkRoot       string            `json:"work_root,omitempty"`
	OutputRoot     string            `json:"output_root,omitempty"`
	SummaryPath    string            `json:"summary_path,omitempty"`
	Artifacts      map[string]string `json:"artifacts,omitempty"`
	ResultCount    int               `json:"result_count,omitempty"`
	Resumed        bool              `json:"resumed,omitempty"`
	ElapsedSeconds float64           `json:"elapsed_seconds"`
	Notes          []string          `json:"notes,omitempty"`
	Error          string            `json:"error,omitempty"`
}

type CampaignTotals struct {
	Jobs      int `json:"jobs"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
	Resumed   int `json:"resumed"`
}

type CampaignSummaryFile struct {
	SchemaVersion string              `json:"schema_version"`
	CampaignPath  string              `json:"campaign_path"`
	TrackingIssue string              `json:"tracking_issue,omitempty"`
	Description   string              `json:"description,omitempty"`
	GeneratedAt   time.Time           `json:"generated_at"`
	Totals        CampaignTotals      `json:"totals"`
	Results       []CampaignJobResult `json:"results"`
}

type CampaignOptions struct {
	Path              string
	WorkRoot          string
	OutputRoot        string
	Resume            bool
	CervoBinary       string
	GitBinary         string
	GremlinsBinary    string
	GomuBinary        string
	GoMutestingBinary string
	Runner            CommandRunner
	Monitor           MemoryMonitor
	SmokeRunner       func(context.Context, SmokeOptions) (RunSummary[SmokeResult], error)
	CompareRunner     func(context.Context, CompareOptions) (RunSummary[CompareResult], error)
	BenchmarkRunner   func(context.Context, BenchmarkOptions) (RunSummary[BenchmarkResult], error)
}

func LoadCampaignManifest(path string) (CampaignManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CampaignManifest{}, err
	}
	var manifest CampaignManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return CampaignManifest{}, err
	}
	return manifest, nil
}

func RunCampaign(ctx context.Context, opts CampaignOptions) (RunSummary[CampaignJobResult], error) {
	campaignPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return RunSummary[CampaignJobResult]{}, err
	}
	manifest, err := LoadCampaignManifest(campaignPath)
	if err != nil {
		return RunSummary[CampaignJobResult]{}, err
	}
	baseDir := filepath.Dir(campaignPath)
	defaultWorkRoot := campaignRootPath(opts.WorkRoot, manifest.WorkRoot, baseDir, filepath.Join(os.TempDir(), defaultCampaignWorkDirName))
	defaultOutputRoot := campaignRootPath(opts.OutputRoot, manifest.OutputRoot, baseDir, filepath.Join(os.TempDir(), defaultCampaignOutputDirName))
	if err := os.MkdirAll(defaultOutputRoot, 0o755); err != nil {
		return RunSummary[CampaignJobResult]{}, err
	}

	summaryFilePath := campaignSummaryPath(defaultOutputRoot)
	resumed := map[string]CampaignJobResult{}
	if opts.Resume {
		if summary, ok, err := loadCampaignSummary(summaryFilePath); err != nil {
			return RunSummary[CampaignJobResult]{}, err
		} else if ok {
			for _, result := range summary.Results {
				resumed[result.Name] = result
			}
		}
	}

	smokeRunner := opts.SmokeRunner
	if smokeRunner == nil {
		smokeRunner = RunSmoke
	}
	compareRunner := opts.CompareRunner
	if compareRunner == nil {
		compareRunner = RunCompare
	}
	benchmarkRunner := opts.BenchmarkRunner
	if benchmarkRunner == nil {
		benchmarkRunner = RunBenchmark
	}

	results := make([]CampaignJobResult, 0, len(manifest.Jobs))
	for idx, job := range manifest.Jobs {
		jobName := strings.TrimSpace(job.Name)
		if jobName == "" {
			jobName = fmt.Sprintf("%s-%d", strings.TrimSpace(job.Kind), idx+1)
		}
		jobKind := strings.TrimSpace(job.Kind)
		jobWorkRoot := campaignJobWorkRoot(defaultWorkRoot, baseDir, job, jobName)
		jobOutputRoot := campaignJobOutputRoot(jobKind, defaultOutputRoot, baseDir, job, jobName, jobWorkRoot)
		resumeKey := campaignJobResumeKey(baseDir, defaultWorkRoot, defaultOutputRoot, job, jobName)
		result := CampaignJobResult{
			Name:         jobName,
			Kind:         jobKind,
			ResumeKey:    resumeKey,
			ManifestPath: resolveCampaignPath(baseDir, job.ManifestPath),
			CorpusPath:   resolveCampaignPath(baseDir, job.CorpusPath),
			WorkRoot:     jobWorkRoot,
			OutputRoot:   jobOutputRoot,
		}
		if note := strings.TrimSpace(job.Notes); note != "" {
			result.Notes = append(result.Notes, note)
		}

		started := time.Now()
		if !campaignJobEnabled(job) {
			result.Status = "skipped"
			result.Notes = append(result.Notes, "job disabled")
			result.ElapsedSeconds = seconds(started)
			results = append(results, result)
			if err := writeCampaignSummary(summaryFilePath, campaignPath, manifest, results); err != nil {
				return RunSummary[CampaignJobResult]{Results: results, SummaryPath: summaryFilePath}, err
			}
			continue
		}
		if resumedResult, ok := resumed[jobName]; ok && canResumeCampaignJob(resumedResult, resumeKey) {
			resumedResult.Resumed = true
			resumedResult.ResumeKey = resumeKey
			if !containsNote(resumedResult.Notes, "resumed from existing campaign summary") {
				resumedResult.Notes = append(resumedResult.Notes, "resumed from existing campaign summary")
			}
			results = append(results, resumedResult)
			if err := writeCampaignSummary(summaryFilePath, campaignPath, manifest, results); err != nil {
				return RunSummary[CampaignJobResult]{Results: results, SummaryPath: summaryFilePath}, err
			}
			continue
		}

		switch jobKind {
		case "smoke":
			if result.ManifestPath == "" {
				result.Status = "failed"
				result.Error = "manifest_path is required for smoke jobs"
				break
			}
			run, runErr := smokeRunner(ctx, SmokeOptions{
				ManifestPath:           result.ManifestPath,
				WorkRoot:               jobWorkRoot,
				Names:                  append([]string(nil), job.Names...),
				Limit:                  job.Limit,
				RunMutation:            job.RunMutation,
				MaxMutants:             defaultInt(job.MaxMutants, defaultCampaignSmokeMaxMutants),
				Workers:                defaultInt(job.Workers, defaultCampaignSmokeWorkers),
				CloneTimeoutSeconds:    defaultInt(job.CloneTimeoutSeconds, defaultCampaignSmokeCloneTimeoutSeconds),
				TestTimeoutSeconds:     defaultInt(job.TestTimeoutSeconds, defaultCampaignSmokeTestTimeoutSeconds),
				DryRunTimeoutSeconds:   defaultInt(job.DryRunTimeoutSeconds, defaultCampaignSmokeDryRunTimeoutSeconds),
				MutationTimeoutSeconds: defaultInt(job.MutationTimeoutSeconds, defaultCampaignSmokeMutationTimeoutSeconds),
				CervoBinary:            opts.CervoBinary,
				GitBinary:              opts.GitBinary,
				Runner:                 opts.Runner,
			})
			applyCampaignRunResult(&result, run.Results, run.SummaryPath, run.Artifacts, runErr)
		case "compare":
			if result.ManifestPath == "" {
				result.Status = "failed"
				result.Error = "manifest_path is required for compare jobs"
				break
			}
			run, runErr := compareRunner(ctx, CompareOptions{
				ManifestPath:               result.ManifestPath,
				WorkRoot:                   jobWorkRoot,
				OutputRoot:                 jobOutputRoot,
				Names:                      append([]string(nil), job.Names...),
				Tools:                      append([]string(nil), job.Tools...),
				Workers:                    defaultInt(job.Workers, defaultCampaignCompareWorkers),
				CompareTargetMode:          defaultString(job.CompareTargetMode, "manifest"),
				GremlinsTargetMode:         defaultString(job.GremlinsTargetMode, "manifest"),
				GremlinsTimeoutCoefficient: defaultInt(job.GremlinsTimeoutCoefficient, defaultCampaignCompareGremlinsTimeoutFactor),
				GomuWorkers:                defaultInt(job.GomuWorkers, defaultCampaignCompareGomuWorkers),
				GoMutestingWorkers:         defaultInt(job.GoMutestingWorkers, defaultCampaignCompareGoMutestingWorkers),
				TimeoutSeconds:             defaultInt(job.TimeoutSeconds, defaultCampaignCompareTimeoutSeconds),
				MinFreeMemoryMB:            defaultInt(job.MinFreeMemoryMB, defaultCampaignCompareMinFreeMemoryMB),
				MinFreeCommitMB:            defaultInt(job.MinFreeCommitMB, defaultCampaignCompareMinFreeCommitMB),
				KillBelowFreeMemoryMB:      defaultInt(job.KillBelowFreeMemoryMB, defaultCampaignCompareKillFreeMemoryMB),
				KillBelowFreeCommitMB:      defaultInt(job.KillBelowFreeCommitMB, defaultCampaignCompareKillFreeCommitMB),
				MaxUsedMemoryMB:            job.MaxUsedMemoryMB,
				MaxCommittedMemoryMB:       job.MaxCommittedMemoryMB,
				MaxProcessTreeMemoryMB:     job.MaxProcessTreeMemoryMB,
				MemoryWaitSeconds:          defaultInt(job.MemoryWaitSeconds, defaultCampaignCompareMemoryWaitSeconds),
				MemoryPollSeconds:          defaultInt(job.MemoryPollSeconds, defaultCampaignCompareMemoryPollSeconds),
				GoMemoryLimit:              job.GoMemoryLimit,
				GoMaxProcs:                 job.GoMaxProcs,
				GoFlags:                    job.GoFlags,
				Resume:                     opts.Resume || job.Resume,
				CervoBinary:                opts.CervoBinary,
				GremlinsBinary:             opts.GremlinsBinary,
				GomuBinary:                 opts.GomuBinary,
				GoMutestingBinary:          opts.GoMutestingBinary,
				Runner:                     opts.Runner,
				Monitor:                    opts.Monitor,
			})
			applyCampaignRunResult(&result, run.Results, run.SummaryPath, run.Artifacts, runErr)
		case "benchmark":
			if result.CorpusPath == "" {
				result.Status = "failed"
				result.Error = "corpus_path is required for benchmark jobs"
				break
			}
			run, runErr := benchmarkRunner(ctx, BenchmarkOptions{
				CorpusPath:  result.CorpusPath,
				WorkRoot:    jobWorkRoot,
				OutputRoot:  jobOutputRoot,
				Names:       append([]string(nil), job.Names...),
				Limit:       job.Limit,
				Resume:      opts.Resume || job.Resume,
				CervoBinary: opts.CervoBinary,
				GitBinary:   opts.GitBinary,
				Runner:      opts.Runner,
			})
			applyCampaignRunResult(&result, run.Results, run.SummaryPath, run.Artifacts, runErr)
		default:
			result.Status = "failed"
			result.Error = fmt.Sprintf("unsupported campaign job kind %q", jobKind)
		}

		result.ElapsedSeconds = seconds(started)
		results = append(results, result)
		if err := writeCampaignSummary(summaryFilePath, campaignPath, manifest, results); err != nil {
			return RunSummary[CampaignJobResult]{Results: results, SummaryPath: summaryFilePath}, err
		}
	}

	return RunSummary[CampaignJobResult]{
		Results:     results,
		SummaryPath: summaryFilePath,
		Artifacts: map[string]string{
			"campaign_summary": summaryFilePath,
		},
	}, nil
}

func applyCampaignRunResult[T any](result *CampaignJobResult, nested []T, summaryPath string, artifacts map[string]string, runErr error) {
	result.ResultCount = len(nested)
	result.SummaryPath = summaryPath
	if len(artifacts) > 0 {
		result.Artifacts = copyStringMap(artifacts)
	}
	if runErr != nil {
		result.Status = "failed"
		result.Error = runErr.Error()
		return
	}
	result.Status = "ok"
}

func campaignJobEnabled(job CampaignJob) bool {
	return job.Enabled == nil || *job.Enabled
}

func canResumeCampaignJob(result CampaignJobResult, wantResumeKey string) bool {
	if result.Status != "ok" && result.Status != "skipped" {
		return false
	}
	return strings.TrimSpace(result.ResumeKey) != "" && result.ResumeKey == strings.TrimSpace(wantResumeKey)
}

func campaignRootPath(cliValue, manifestValue, baseDir, fallback string) string {
	if value := strings.TrimSpace(cliValue); value != "" {
		return filepath.Clean(value)
	}
	if value := strings.TrimSpace(manifestValue); value != "" {
		return resolveCampaignPath(baseDir, value)
	}
	return fallback
}

func campaignJobWorkRoot(defaultRoot, baseDir string, job CampaignJob, jobName string) string {
	if value := strings.TrimSpace(job.WorkRoot); value != "" {
		return resolveCampaignPath(baseDir, value)
	}
	return filepath.Join(defaultRoot, jobName)
}

func campaignJobOutputRoot(kind, defaultRoot, baseDir string, job CampaignJob, jobName, workRoot string) string {
	if kind == "smoke" {
		return workRoot
	}
	if value := strings.TrimSpace(job.OutputRoot); value != "" {
		return resolveCampaignPath(baseDir, value)
	}
	return filepath.Join(defaultRoot, jobName)
}

func resolveCampaignPath(baseDir, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(baseDir, value))
}

func defaultInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func campaignJobResumeKey(baseDir, defaultWorkRoot, defaultOutputRoot string, job CampaignJob, jobName string) string {
	jobKind := strings.TrimSpace(job.Kind)
	descriptor := struct {
		Name                       string   `json:"name"`
		Kind                       string   `json:"kind"`
		Enabled                    bool     `json:"enabled"`
		ManifestPath               string   `json:"manifest_path,omitempty"`
		CorpusPath                 string   `json:"corpus_path,omitempty"`
		WorkRoot                   string   `json:"work_root,omitempty"`
		OutputRoot                 string   `json:"output_root,omitempty"`
		Names                      []string `json:"names,omitempty"`
		Tools                      []string `json:"tools,omitempty"`
		Limit                      int      `json:"limit,omitempty"`
		RunMutation                bool     `json:"run_mutation,omitempty"`
		MaxMutants                 int      `json:"max_mutants,omitempty"`
		Workers                    int      `json:"workers,omitempty"`
		CloneTimeoutSeconds        int      `json:"clone_timeout_seconds,omitempty"`
		TestTimeoutSeconds         int      `json:"test_timeout_seconds,omitempty"`
		DryRunTimeoutSeconds       int      `json:"dry_run_timeout_seconds,omitempty"`
		MutationTimeoutSeconds     int      `json:"mutation_timeout_seconds,omitempty"`
		CompareTargetMode          string   `json:"compare_target_mode,omitempty"`
		GremlinsTargetMode         string   `json:"gremlins_target_mode,omitempty"`
		GremlinsTimeoutCoefficient int      `json:"gremlins_timeout_coefficient,omitempty"`
		GomuWorkers                int      `json:"gomu_workers,omitempty"`
		GoMutestingWorkers         int      `json:"go_mutesting_workers,omitempty"`
		TimeoutSeconds             int      `json:"timeout_seconds,omitempty"`
		MinFreeMemoryMB            int      `json:"min_free_memory_mb,omitempty"`
		MinFreeCommitMB            int      `json:"min_free_commit_mb,omitempty"`
		KillBelowFreeMemoryMB      int      `json:"kill_below_free_memory_mb,omitempty"`
		KillBelowFreeCommitMB      int      `json:"kill_below_free_commit_mb,omitempty"`
		MaxUsedMemoryMB            int      `json:"max_used_memory_mb,omitempty"`
		MaxCommittedMemoryMB       int      `json:"max_committed_memory_mb,omitempty"`
		MaxProcessTreeMemoryMB     int      `json:"max_process_tree_memory_mb,omitempty"`
		MemoryWaitSeconds          int      `json:"memory_wait_seconds,omitempty"`
		MemoryPollSeconds          int      `json:"memory_poll_seconds,omitempty"`
		GoMemoryLimit              string   `json:"go_memory_limit,omitempty"`
		GoMaxProcs                 int      `json:"go_max_procs,omitempty"`
		GoFlags                    string   `json:"go_flags,omitempty"`
	}{
		Name:                       jobName,
		Kind:                       jobKind,
		Enabled:                    campaignJobEnabled(job),
		ManifestPath:               resolveCampaignPath(baseDir, job.ManifestPath),
		CorpusPath:                 resolveCampaignPath(baseDir, job.CorpusPath),
		WorkRoot:                   campaignJobWorkRoot(defaultWorkRoot, baseDir, job, jobName),
		OutputRoot:                 campaignJobOutputRoot(jobKind, defaultOutputRoot, baseDir, job, jobName, campaignJobWorkRoot(defaultWorkRoot, baseDir, job, jobName)),
		Names:                      append([]string(nil), job.Names...),
		Tools:                      append([]string(nil), job.Tools...),
		Limit:                      job.Limit,
		RunMutation:                job.RunMutation,
		MaxMutants:                 job.MaxMutants,
		Workers:                    job.Workers,
		CloneTimeoutSeconds:        job.CloneTimeoutSeconds,
		TestTimeoutSeconds:         job.TestTimeoutSeconds,
		DryRunTimeoutSeconds:       job.DryRunTimeoutSeconds,
		MutationTimeoutSeconds:     job.MutationTimeoutSeconds,
		CompareTargetMode:          job.CompareTargetMode,
		GremlinsTargetMode:         job.GremlinsTargetMode,
		GremlinsTimeoutCoefficient: job.GremlinsTimeoutCoefficient,
		GomuWorkers:                job.GomuWorkers,
		GoMutestingWorkers:         job.GoMutestingWorkers,
		TimeoutSeconds:             job.TimeoutSeconds,
		MinFreeMemoryMB:            job.MinFreeMemoryMB,
		MinFreeCommitMB:            job.MinFreeCommitMB,
		KillBelowFreeMemoryMB:      job.KillBelowFreeMemoryMB,
		KillBelowFreeCommitMB:      job.KillBelowFreeCommitMB,
		MaxUsedMemoryMB:            job.MaxUsedMemoryMB,
		MaxCommittedMemoryMB:       job.MaxCommittedMemoryMB,
		MaxProcessTreeMemoryMB:     job.MaxProcessTreeMemoryMB,
		MemoryWaitSeconds:          job.MemoryWaitSeconds,
		MemoryPollSeconds:          job.MemoryPollSeconds,
		GoMemoryLimit:              job.GoMemoryLimit,
		GoMaxProcs:                 job.GoMaxProcs,
		GoFlags:                    job.GoFlags,
	}
	data, _ := json.Marshal(descriptor)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func buildCampaignSummary(campaignPath string, manifest CampaignManifest, results []CampaignJobResult) CampaignSummaryFile {
	totals := CampaignTotals{Jobs: len(results)}
	for _, result := range results {
		if result.Resumed {
			totals.Resumed++
		}
		switch result.Status {
		case "ok":
			totals.Succeeded++
		case "failed":
			totals.Failed++
		case "skipped":
			totals.Skipped++
		}
	}
	return CampaignSummaryFile{
		SchemaVersion: "1",
		CampaignPath:  campaignPath,
		TrackingIssue: manifest.TrackingIssue,
		Description:   manifest.Description,
		GeneratedAt:   time.Now().UTC(),
		Totals:        totals,
		Results:       append([]CampaignJobResult(nil), results...),
	}
}

func writeCampaignSummary(path, campaignPath string, manifest CampaignManifest, results []CampaignJobResult) error {
	return writeJSON(path, buildCampaignSummary(campaignPath, manifest, results))
}

func loadCampaignSummary(path string) (CampaignSummaryFile, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return CampaignSummaryFile{}, false, nil
		}
		return CampaignSummaryFile{}, false, err
	}
	var summary CampaignSummaryFile
	if err := json.Unmarshal(data, &summary); err != nil {
		return CampaignSummaryFile{}, false, err
	}
	return summary, true, nil
}

func campaignSummaryPath(root string) string {
	return filepath.Join(root, "campaign-summary.json")
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
