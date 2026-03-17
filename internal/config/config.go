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
	ExcludePackages      []string `yaml:"exclude_packages"`
	ExcludeSymbols       []string `yaml:"exclude_symbols"`
	TreatTestsAsExternal bool     `yaml:"treat_tests_as_external"`
	IncludeMethods       bool     `yaml:"include_methods"`
	IncludeFields        bool     `yaml:"include_fields"`
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
	if err := validateList("exclude_packages", cfg.ExcludePackages); err != nil {
		return err
	}
	if err := validateList("exclude_symbols", cfg.ExcludeSymbols); err != nil {
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
