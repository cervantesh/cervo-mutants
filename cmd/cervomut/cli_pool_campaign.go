package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cervantesh/cervo-mutants/pkg/pool"
)

func (app *cliApp) cmdPoolCampaign(args []string) error {
	fs := flag.NewFlagSet("pool campaign", flag.ContinueOnError)
	file := fs.String("file", "docs/evaluations/pool-campaign-example.json", "campaign manifest")
	workRoot := fs.String("work-root", "", "override campaign-level work root")
	outputRoot := fs.String("output-root", "", "override campaign-level output root")
	resume := fs.Bool("resume", false, "resume completed jobs from campaign-summary.json when possible")
	cervoBinary := fs.String("cervomutants", currentExecutable(), "path to the cervomut binary used for nested runs")
	gitBinary := fs.String("git", "git", "path to git")
	gremlinsBinary := fs.String("gremlins", filepath.Join(os.TempDir(), "cervomut-study-cobra", "tools", "gremlins.exe"), "path to Gremlins")
	gomuBinary := fs.String("gomu", filepath.Join(os.TempDir(), "cervomut-study-cobra", "tools", "gomu-patched.exe"), "path to gomu")
	goMutestingBinary := fs.String("go-mutesting", filepath.Join(os.TempDir(), "cervomut-study-cobra", "tools", "go-mutesting-patched.exe"), "path to go-mutesting")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"file": true, "work-root": true, "output-root": true, "cervomutants": true, "git": true, "gremlins": true, "gomu": true, "go-mutesting": true,
	})); err != nil {
		return err
	}
	run, err := app.deps.runPoolCampaign(app.deps.background(), pool.CampaignOptions{
		Path:              *file,
		WorkRoot:          *workRoot,
		OutputRoot:        *outputRoot,
		Resume:            *resume,
		CervoBinary:       *cervoBinary,
		GitBinary:         *gitBinary,
		GremlinsBinary:    *gremlinsBinary,
		GomuBinary:        *gomuBinary,
		GoMutestingBinary: *goMutestingBinary,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Pool campaign summary: %s\n", run.SummaryPath)
	if failed := failedCampaignJobs(run.Results); failed > 0 {
		return fmt.Errorf("pool campaign completed with %d failed jobs", failed)
	}
	return nil
}

func failedCampaignJobs(results []pool.CampaignJobResult) int {
	count := 0
	for _, result := range results {
		if result.Status == "failed" {
			count++
		}
	}
	return count
}
