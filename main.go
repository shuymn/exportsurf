package main

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/shuymn/exportsurf/internal/baseline"
	"github.com/shuymn/exportsurf/internal/config"
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
		return usageError("usage: exportsurf <scan|diff> ...")
	}

	switch args[0] {
	case "scan":
		return runScan(args[1:], stdout)
	case "diff":
		return runDiff(args[1:], stdout)
	default:
		return usageError("unknown command: %s", args[0])
	}
}

func runScan(args []string, stdout io.Writer) error {
	cfg, err := parseScanArgs(args)
	if err != nil {
		return err
	}
	fileCfg, err := loadConfig(cfg.configPath)
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
		TreatTestsAsExternal: cfg.treatTestsAsExternal || fileCfg.TreatTestsAsExternal,
		ExcludePackages:      fileCfg.ExcludePackages,
		ExcludeSymbols:       fileCfg.ExcludeSymbols,
	})
	if err != nil {
		return err
	}

	return report.WriteJSON(stdout, candidates)
}

func runDiff(args []string, stdout io.Writer) error {
	cfg, err := parseDiffArgs(args)
	if err != nil {
		return err
	}
	fileCfg, err := loadConfig(cfg.configPath)
	if err != nil {
		return err
	}

	accepted, err := baseline.Load(cfg.baselinePath)
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
		TreatTestsAsExternal: cfg.treatTestsAsExternal || fileCfg.TreatTestsAsExternal,
		ExcludePackages:      fileCfg.ExcludePackages,
		ExcludeSymbols:       fileCfg.ExcludeSymbols,
	})
	if err != nil {
		return err
	}

	filtered := make([]report.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := accepted[candidate.Symbol]; ok {
			continue
		}
		filtered = append(filtered, candidate)
	}

	return report.WriteJSON(stdout, filtered)
}

type scanConfig struct {
	patterns             []string
	configPath           string
	treatTestsAsExternal bool
}

type diffConfig struct {
	patterns             []string
	baselinePath         string
	configPath           string
	treatTestsAsExternal bool
}

func parseScanArgs(args []string) (scanConfig, error) {
	cfg := scanConfig{
		patterns: []string{"./..."},
	}

	var patterns []string
	var jsonOutput bool
	var err error

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]

		switch {
		case arg == "--json":
			jsonOutput = true
		case arg == "--config":
			cfg.configPath, idx, err = parseRequiredPathFlag(args, idx, "--config", "config")
			if err != nil {
				return scanConfig{}, err
			}
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

func parseDiffArgs(args []string) (diffConfig, error) {
	cfg := diffConfig{
		patterns: []string{"./..."},
	}

	var patterns []string
	var err error

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]

		switch {
		case arg == "--baseline":
			cfg.baselinePath, idx, err = parseRequiredPathFlag(args, idx, "--baseline", "baseline")
			if err != nil {
				return diffConfig{}, err
			}
		case arg == "--config":
			cfg.configPath, idx, err = parseRequiredPathFlag(args, idx, "--config", "config")
			if err != nil {
				return diffConfig{}, err
			}
		case arg == "--treat-tests-as-external":
			cfg.treatTestsAsExternal = true
		case arg == "":
			return diffConfig{}, usageError("empty argument is not allowed")
		case arg[0] == '-':
			return diffConfig{}, usageError("unknown flag: %s", arg)
		default:
			patterns = append(patterns, arg)
		}
	}

	if cfg.baselinePath == "" {
		return diffConfig{}, usageError("--baseline is required")
	}

	if len(patterns) > 0 {
		cfg.patterns = slices.Clone(patterns)
	}

	return cfg, nil
}

func usageError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func parseRequiredPathFlag(args []string, idx int, flag, label string) (string, int, error) {
	idx++
	if idx >= len(args) {
		return "", idx, usageError("%s requires a path", flag)
	}
	if args[idx] == "" {
		return "", idx, usageError("empty %s path is not allowed", label)
	}

	return args[idx], idx, nil
}

func loadConfig(path string) (config.File, error) {
	if path == "" {
		return config.File{}, nil
	}

	cfg, err := config.Load(path)
	if err != nil {
		return config.File{}, fmt.Errorf("load config: %w", err)
	}

	return cfg, nil
}
