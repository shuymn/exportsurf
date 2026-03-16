package main

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/shuymn/exportsurf/internal/scan"
	"github.com/shuymn/exportsurf/pkg/report"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return usageError("usage: exportsurf scan [patterns...] --json [--treat-tests-as-external]")
	}

	switch args[0] {
	case "scan":
		return runScan(args[1:], stdout)
	default:
		return usageError("unknown command: %s", args[0])
	}
}

func runScan(args []string, stdout io.Writer) error {
	cfg, err := parseScanArgs(args)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	candidates, err := scan.Run(scan.Options{
		Patterns:             cfg.patterns,
		WorkingDir:           cwd,
		TreatTestsAsExternal: cfg.treatTestsAsExternal,
	})
	if err != nil {
		return err
	}

	return report.WriteJSON(stdout, candidates)
}

type scanConfig struct {
	patterns             []string
	treatTestsAsExternal bool
}

func parseScanArgs(args []string) (scanConfig, error) {
	cfg := scanConfig{
		patterns: []string{"./..."},
	}

	var patterns []string
	var jsonOutput bool

	for _, arg := range args {
		switch {
		case arg == "--json":
			jsonOutput = true
		case arg == "--treat-tests-as-external":
			cfg.treatTestsAsExternal = true
		case arg == "":
			return scanConfig{}, usageError("empty argument is not allowed")
		case arg[0] == '-':
			return scanConfig{}, usageError("unknown flag: %s", arg)
		default:
			patterns = append(patterns, arg)
		}
	}

	if !jsonOutput {
		return scanConfig{}, usageError("--json is required")
	}

	if len(patterns) > 0 {
		cfg.patterns = slices.Clone(patterns)
	}

	return cfg, nil
}

func usageError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
