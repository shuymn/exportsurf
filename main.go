package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"

	"github.com/shuymn/exportsurf/internal/baseline"
	"github.com/shuymn/exportsurf/internal/config"
	"github.com/shuymn/exportsurf/internal/scan"
	"github.com/shuymn/exportsurf/pkg/report"
)

var errFindingsFound = errors.New("candidates found")

type cli struct {
	Patterns             []string `arg:"" optional:"" help:"Package patterns to scan (default: ./...)." name:"patterns"`
	JSON                 bool     `name:"json" help:"Output JSON array of candidates." xor:"format"`
	SARIF                bool     `name:"sarif" help:"Output SARIF v2.1.0 format." xor:"format"`
	Baseline             string   `name:"baseline" help:"Path to baseline JSON file to filter accepted symbols." placeholder:"PATH"`
	Config               string   `name:"config" help:"Path to config YAML file." placeholder:"PATH"`
	FailOnFindings       bool     `name:"fail-on-findings" help:"Exit with code 1 if candidates are found."`
	TreatTestsAsExternal bool     `name:"treat-tests-as-external" help:"Count _test.go references as external uses."`
}

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
	var c cli
	var exited bool
	parser, err := kong.New(&c,
		kong.Name("exportsurf"),
		kong.Description("Report exported Go symbols with no external references."),
		kong.Writers(stdout, os.Stderr),
		kong.Exit(func(int) {
			exited = true
		}),
	)
	if err != nil {
		return err
	}

	_, err = parser.Parse(args)
	if exited {
		return nil
	}
	if err != nil {
		return err
	}

	return executeScan(&c, stdout)
}

func executeScan(cmd *cli, stdout io.Writer) error {
	patterns := cmd.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	fileCfg, err := loadConfig(cmd.Config)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	opts := newScanOptions(
		patterns,
		cwd,
		cmd.TreatTestsAsExternal || fileCfg.Rules.TreatTestsAsExternal,
		fileCfg.Exclude.Packages,
		fileCfg.Exclude.Symbols,
		resolveRules(fileCfg.Rules),
		resolveLowConfidence(fileCfg.Rules.MarkLowConfidence),
	)

	candidates, err := scan.Run(opts)
	if err != nil {
		return err
	}

	candidates, err = filterBaseline(cmd.Baseline, candidates)
	if err != nil {
		return err
	}

	if err := writeOutput(stdout, candidates, cmd.JSON, cmd.SARIF); err != nil {
		return err
	}

	if cmd.FailOnFindings && len(candidates) > 0 {
		return errFindingsFound
	}

	return nil
}

func newScanOptions(
	patterns []string,
	workingDir string,
	treatTestsAsExternal bool,
	excludePackages []string,
	excludeSymbols []string,
	rules scan.RulesFlags,
	lowConfidence scan.LowConfidenceFlags,
) scan.Options {
	return scan.NewOptions(
		patterns,
		workingDir,
		treatTestsAsExternal,
		excludePackages,
		excludeSymbols,
		rules,
		lowConfidence,
	)
}

func resolveRules(cfg config.RulesConfig) scan.RulesFlags {
	return scan.NewRulesFlags(
		boolOrTrue(cfg.IncludeFuncs),
		boolOrTrue(cfg.IncludeTypes),
		boolOrTrue(cfg.IncludeVars),
		boolOrTrue(cfg.IncludeConsts),
		boolOrTrue(cfg.IncludeMethods),
		boolOrTrue(cfg.IncludeFields),
	)
}

func resolveLowConfidence(cfg config.MarkLowConfidence) scan.LowConfidenceFlags {
	return scan.NewLowConfidenceFlags(
		boolOrTrue(cfg.PackageMain),
		boolOrTrue(cfg.PackageUnderCmd),
		boolOrTrue(cfg.GeneratedFile),
		boolOrTrue(cfg.ReflectUsage),
		boolOrTrue(cfg.PluginUsage),
		boolOrTrue(cfg.CgoExport),
		boolOrTrue(cfg.Linkname),
		boolOrTrue(cfg.InterfaceSatisfaction),
		boolOrTrue(cfg.EmbeddedField),
		boolOrTrue(cfg.SerializationTag),
	)
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
