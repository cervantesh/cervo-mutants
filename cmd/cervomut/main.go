package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/daemon"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
	evalpkg "github.com/cervantesh/cervo-mutants/pkg/eval"
	"github.com/cervantesh/cervo-mutants/pkg/pool"
)

const (
	configFileName           = "cervomut.yaml"
	mutationReportFileName   = "mutation-report.json"
	failureDebugFileName     = "failure-debug.json"
	flagTestTimeout          = "test-timeout"
	flagMaxMutants           = "max-mutants"
	flagMaxProcessMemoryMB   = "max-process-memory-mb"
	reportOutputDirectoryDoc = "report output directory"
)

var (
	runEngineFn = func(cfg config.Config, req engine.RunRequest) (engine.RunResult, error) {
		return engine.New(cfg).Run(context.Background(), req)
	}
	writeRunResultFn = writeRunResult
	writeEvalFn      = evalpkg.Write
	buildEvalFn      = evalpkg.Build
	runPoolSmokeFn   = pool.RunSmoke
	runPoolCompareFn = pool.RunCompare
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

func run(args []string) (err error) {
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
		return cmdRun(args[1:])
	case "fast":
		return cmdFast(args[1:])
	case "eval":
		return cmdEval(args[1:])
	case "compare":
		return cmdCompare(args[1:])
	case "pool":
		return cmdPool(args[1:])
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
	case "daemon", "worker":
		return daemon.ServeJSONLines(context.Background(), os.Stdin, os.Stdout, daemon.WorkerRunner{MaxOutputBytes: 12000})
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Println("usage: cervomut <init|doctor|affected|run|fast|eval|compare|pool|baseline|report|show|explain|list-mutators|daemon|worker>")
}
