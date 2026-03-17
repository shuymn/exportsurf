package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

type File struct {
	Exclude ExcludeConfig `yaml:"exclude"`
	Rules   RulesConfig   `yaml:"rules"`
}

type ExcludeConfig struct {
	Packages []string `yaml:"packages"`
	Symbols  []string `yaml:"symbols"`
}

type RulesConfig struct {
	IncludeFuncs         *bool             `yaml:"include_funcs"`
	IncludeTypes         *bool             `yaml:"include_types"`
	IncludeVars          *bool             `yaml:"include_vars"`
	IncludeConsts        *bool             `yaml:"include_consts"`
	IncludeMethods       *bool             `yaml:"include_methods"`
	IncludeFields        *bool             `yaml:"include_fields"`
	TreatTestsAsExternal bool              `yaml:"treat_tests_as_external"`
	MarkLowConfidence    MarkLowConfidence `yaml:"mark_low_confidence"`
}

type MarkLowConfidence struct {
	PackageMain           *bool `yaml:"package_main"`
	PackageUnderCmd       *bool `yaml:"package_under_cmd"`
	GeneratedFile         *bool `yaml:"generated_file"`
	ReflectUsage          *bool `yaml:"reflect_usage"`
	PluginUsage           *bool `yaml:"plugin_usage"`
	CgoExport             *bool `yaml:"cgo_export"`
	Linkname              *bool `yaml:"linkname"`
	InterfaceSatisfaction *bool `yaml:"interface_satisfaction"`
	EmbeddedField         *bool `yaml:"embedded_field"`
	SerializationTag      *bool `yaml:"serialization_tag"`
}

func Load(path string) (File, error) {
	file, err := os.Open(path)
	if err != nil {
		return File{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var cfg File
	decoder := yaml.NewDecoder(file, yaml.DisallowUnknownField())
	if err := decoder.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return File{}, nil
		}
		return File{}, fmt.Errorf("decode config: %w", err)
	}

	if err := validate(cfg); err != nil {
		return File{}, err
	}

	return cfg, nil
}

func validate(cfg File) error {
	if err := validateList("exclude.packages", cfg.Exclude.Packages); err != nil {
		return err
	}
	if err := validateList("exclude.symbols", cfg.Exclude.Symbols); err != nil {
		return err
	}

	return nil
}

func validateList(field string, values []string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("validate config: %s must not contain empty values", field)
		}
	}

	return nil
}
