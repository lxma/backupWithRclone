package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	ExcludeDirs             []string    `yaml:"exclude_dirs"`
	ExcludePaths            []string    `yaml:"exclude_paths"`
	ExcludeFiles            []string    `yaml:"exclude_files"`
	RcloneDestinationPrefix []string    `yaml:"rclone_destination_prefix"`
	DirSizeFactor           int         `yaml:"dir_size_factor"`
	Dirs                    []DirConfig `yaml:"dirs"`
}

type DirConfig struct {
	SourceDir string `yaml:"source_dir"`
	TargetDir string `yaml:"target_dir,omitempty"`
}

const cDefaultDirSizeFactor = 200
const cDefaultBackupDir = "~/.backupWithRclone"

var cfg Config
var backupDir string

func LoadConfig() {
	backupDir, _ = os.LookupEnv("BACKUP_WITH_RCLONE_DIR")
	if backupDir == "" {
		backupDir = cDefaultBackupDir
	}
	backupDir = ExpandHome(backupDir)
	filename := filepath.Join(backupDir, "config.yaml")
	f, err := os.Open(filename)
	if err != nil {
		panic(fmt.Errorf("error when opening configuration %s: %w", filename, err))
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	if err := dec.Decode(&cfg); err != nil {
		if err == io.EOF {
			panic(fmt.Errorf("configuration %s is empty", filename))
		}
		panic(fmt.Errorf(
			"error reading configuration %s: %w", filename, err))
	}

	if cfg.DirSizeFactor == 0 {
		cfg.DirSizeFactor = cDefaultDirSizeFactor
	}
}
