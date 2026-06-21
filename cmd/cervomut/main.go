package main

import (
	"fmt"
	"os"
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

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

func run(args []string) error {
	return newCLIApp().run(args)
}

func usage() {
	fmt.Println("usage: cervomut <init|doctor|affected|run|fast|eval|compare|pool|baseline|report|show|explain|list-mutators|daemon|worker>")
	fmt.Println("note: daemon and worker are experimental protocol modes and require explicit opt-in")
}
