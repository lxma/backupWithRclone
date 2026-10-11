package main

import (
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	ExcludeFiles  []string `yaml:"exclude_files"`
	ExcludeDirs   []string `yaml:"exclude_dirs"`
	ExcludePaths  []string `yaml:"exclude_paths"`
	DirSizeFactor int      `yaml:"dir_size_factor"`
	Dirs          []string `yaml:"dirs"`
}

func LoadConfig(filename string) (Config, error) {
	var cfg Config

	f, err := os.Open(filename)
	if err != nil {
		return Config{}, fmt.Errorf(
			"error when opening configuration %s: %w", filename, err,
		)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	if err := dec.Decode(&cfg); err != nil {
		if err == io.EOF {
			return Config{}, fmt.Errorf(
				"configuration %s is empty", filename,
			)
		}
		return Config{}, fmt.Errorf(
			"error reading configuration %s: %w", filename, err,
		)
	}

	return cfg, nil
}
