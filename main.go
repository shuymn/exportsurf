package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/shuymn/exportsurf/internal/baseline"
	"github.com/shuymn/exportsurf/internal/config"
	"github.com/shuymn/exportsurf/internal/scan"
	"github.com/shuymn/exportsurf/pkg/report"
)

var errFindingsFound = errors.New("candidates found")

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, errFindingsFound) {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return usageError("usage: exportsurf scan ...")
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
	fileCfg, err := loadConfig(cfg.configPath)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if cfg.sarifOutput && cfg.jsonOutput {
		return usageError("--sarif and --json are mutually exclusive")
	}

	candidates, err := scan.Run(scan.Options{
		Patterns:             cfg.patterns,
		WorkingDir:           cwd,
		TreatTestsAsExternal: cfg.treatTestsAsExternal || fileCfg.Rules.TreatTestsAsExternal,
		ExcludePackages:      fileCfg.Exclude.Packages,
		ExcludeSymbols:       fileCfg.Exclude.Symbols,
		Rules:                resolveRules(fileCfg.Rules),
		LowConfidence:        resolveLowConfidence(fileCfg.Rules.MarkLowConfidence),
	})
	if err != nil {
		return err
	}

	candidates, err = filterBaseline(cfg.baselinePath, candidates)
	if err != nil {
		return err
	}

	if err := writeOutput(stdout, candidates, cfg.jsonOutput, cfg.sarifOutput); err != nil {
		return err
	}

	if cfg.failOnFindings && len(candidates) > 0 {
		return errFindingsFound
	}

	return nil
}

func resolveRules(cfg config.RulesConfig) scan.RulesFlags {
	return scan.RulesFlags{
		Funcs:   boolOrTrue(cfg.IncludeFuncs),
		Types:   boolOrTrue(cfg.IncludeTypes),
		Vars:    boolOrTrue(cfg.IncludeVars),
		Consts:  boolOrTrue(cfg.IncludeConsts),
		Methods: boolOrTrue(cfg.IncludeMethods),
		Fields:  boolOrTrue(cfg.IncludeFields),
	}
}

func resolveLowConfidence(cfg config.MarkLowConfidence) scan.LowConfidenceFlags {
	return scan.LowConfidenceFlags{
		PackageMain:           boolOrTrue(cfg.PackageMain),
		PackageUnderCmd:       boolOrTrue(cfg.PackageUnderCmd),
		GeneratedFile:         boolOrTrue(cfg.GeneratedFile),
		ReflectUsage:          boolOrTrue(cfg.ReflectUsage),
		PluginUsage:           boolOrTrue(cfg.PluginUsage),
		CgoExport:             boolOrTrue(cfg.CgoExport),
		Linkname:              boolOrTrue(cfg.Linkname),
		InterfaceSatisfaction: boolOrTrue(cfg.InterfaceSatisfaction),
		EmbeddedField:         boolOrTrue(cfg.EmbeddedField),
		SerializationTag:      boolOrTrue(cfg.SerializationTag),
	}
}

func boolOrTrue(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}

func filterBaseline(
	path string,
	candidates []report.Candidate,
) ([]report.Candidate, error) {
	if path == "" {
		return candidates, nil
	}

	accepted, err := baseline.Load(path)
	if err != nil {
		return nil, err
	}

	filtered := make([]report.Candidate, 0, len(candidates))
	for _, c := range candidates {
		if _, ok := accepted[c.Symbol]; !ok {
			filtered = append(filtered, c)
		}
	}
	return filtered, nil
}

func writeOutput(
	w io.Writer,
	candidates []report.Candidate,
	jsonOutput bool,
	sarifOutput bool,
) error {
	if sarifOutput {
		return report.WriteSARIF(w, candidates)
	}
	if jsonOutput {
		return report.WriteJSON(w, candidates)
	}
	return report.WriteText(w, candidates)
}

type scanConfig struct {
	patterns             []string
	configPath           string
	baselinePath         string
	jsonOutput           bool
	sarifOutput          bool
	failOnFindings       bool
	treatTestsAsExternal bool
}

func parseScanArgs(args []string) (scanConfig, error) {
	cfg := scanConfig{
		patterns: []string{"./..."},
	}

	var patterns []string
	var err error

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]

		switch {
		case arg == "--json":
			cfg.jsonOutput = true
		case arg == "--sarif":
			cfg.sarifOutput = true
		case arg == "--baseline":
			cfg.baselinePath, idx, err = parseRequiredPathFlag(args, idx, "--baseline", "baseline")
			if err != nil {
				return scanConfig{}, err
			}
		case arg == "--config":
			cfg.configPath, idx, err = parseRequiredPathFlag(args, idx, "--config", "config")
			if err != nil {
				return scanConfig{}, err
			}
		case arg == "--fail-on-findings":
			cfg.failOnFindings = true
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

var defaultConfigNames = []string{
	".exportsurf.yaml",
	".exportsurf.yml",
	"exportsurf.yaml",
	"exportsurf.yml",
}

func loadConfig(path string) (config.File, error) {
	if path == "" {
		path = discoverConfig()
		if path == "" {
			return config.File{}, nil
		}
	}

	cfg, err := config.Load(path)
	if err != nil {
		return config.File{}, fmt.Errorf("load config: %w", err)
	}

	return cfg, nil
}

func discoverConfig() string {
	for _, name := range defaultConfigNames {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}
