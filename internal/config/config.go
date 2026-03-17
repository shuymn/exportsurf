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
	Exclude              ExcludeConfig       `yaml:"exclude"`
	Include              IncludeConfig       `yaml:"include"`
	TreatTestsAsExternal bool                `yaml:"treat_tests_as_external"`
	LowConfidence        LowConfidenceConfig `yaml:"low_confidence"`
}

type ExcludeConfig struct {
	Packages []string `yaml:"packages"`
	Symbols  []string `yaml:"symbols"`
}

type IncludeConfig struct {
	Methods bool `yaml:"methods"`
	Fields  bool `yaml:"fields"`
}

type LowConfidenceConfig struct {
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
