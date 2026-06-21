package main

import (
	"context"
	"fmt"
	"io"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/daemon"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
	evalpkg "github.com/cervantesh/cervo-mutants/pkg/eval"
	"github.com/cervantesh/cervo-mutants/pkg/pool"
)

type cliDeps struct {
	runEngine        func(config.Config, engine.RunRequest) (engine.RunResult, error)
	writeRunResult   func(config.Config, engine.RunResult, bool) error
	writeEval        func(string, evalpkg.Evaluation) error
	buildEval        func(evalpkg.BuildRequest) evalpkg.Evaluation
	runPoolSmoke     func(context.Context, pool.SmokeOptions) (pool.RunSummary[pool.SmokeResult], error)
	runPoolCompare   func(context.Context, pool.CompareOptions) (pool.RunSummary[pool.CompareResult], error)
	runPoolBenchmark func(context.Context, pool.BenchmarkOptions) (pool.RunSummary[pool.BenchmarkResult], error)
	runPoolCampaign  func(context.Context, pool.CampaignOptions) (pool.RunSummary[pool.CampaignJobResult], error)
	serveDaemon      func(context.Context, io.Reader, io.Writer, engine.Runner) error
	background       func() context.Context
}

type cliApp struct {
	deps cliDeps
}

func newCLIApp() *cliApp {
	return &cliApp{
		deps: cliDeps{
			runEngine: func(cfg config.Config, req engine.RunRequest) (engine.RunResult, error) {
				return engine.New(cfg).Run(context.Background(), req)
			},
			writeRunResult:   writeRunResult,
			writeEval:        evalpkg.Write,
			buildEval:        evalpkg.Build,
			runPoolSmoke:     pool.RunSmoke,
			runPoolCompare:   pool.RunCompare,
			runPoolBenchmark: pool.RunBenchmark,
			runPoolCampaign:  pool.RunCampaign,
			serveDaemon:      daemon.ServeJSONLines,
			background:       context.Background,
		},
	}
}

func (app *cliApp) run(args []string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("internal_error: unexpected panic: %v", recovered)
		}
	}()
	if len(args) == 0 {
		usage()
		return nil
	}
	switch args[0] {
	case "help", "--help", "-h":
		usage()
		return nil
	case "init":
		return cmdInit()
	case "doctor":
		return cmdDoctor()
	case "affected":
		return cmdAffected(args[1:])
	case "run":
		return app.cmdRun(args[1:])
	case "fast":
		return app.cmdFast(args[1:])
	case "eval":
		return app.cmdEval(args[1:])
	case "compare":
		return cmdCompare(args[1:])
	case "pool":
		return app.cmdPool(args[1:])
	case "baseline":
		return cmdBaseline(args[1:])
	case "report":
		return cmdReport(args[1:])
	case "show":
		return cmdShow(args[1:])
	case "explain":
		return cmdExplain(args[1:])
	case "list-mutators":
		return cmdListMutators()
	case "daemon":
		return app.cmdDaemonMode("daemon", args[1:])
	case "worker":
		return app.cmdDaemonMode("worker", args[1:])
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}
