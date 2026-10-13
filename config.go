package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lxma/gotools/v2"
	"go.yaml.in/yaml/v3"
)

type TConfig struct {
	ExcludeDirs             []string              `yaml:"exclude_dirs"`
	ExcludeFiles            []string              `yaml:"exclude_files"`
	RcloneDestinationPrefix string                `yaml:"rclone_destination_prefix"`
	DirSizeFactor           int                   `yaml:"dir_size_factor"`
	DirsConfig              []TDirConfig          `yaml:"dirs"`
	Dirs                    map[string]TDirConfig `yaml:"-"`
}

type TDirConfig struct {
	SourceDir    string   `yaml:"source_dir"`
	TargetDir    string   `yaml:"target_dir,omitempty"`
	ExcludeDirs  []string `yaml:"exclude_dirs"`
	ExcludePaths []string `yaml:"exclude_paths"`
	ExcludeFiles []string `yaml:"exclude_files"`
}

const cBackupWithRcloneDirEnv = "BACKUP_WITH_RCLONE_DIR"
const cDefaultDirSizeFactor = 200
const cDefaultBackupDir = "~/.backupWithRclone"

var gConfig TConfig
var gBackupDir string

func getTargetDir(dirConfig TDirConfig) string {
	if dirConfig.TargetDir == "" {
		return filepath.Base(dirConfig.SourceDir)
	}
	return dirConfig.TargetDir
}

func isPathPrefix(base, candidate string) bool {
	gotools.Assert(base != "" && candidate != "")
	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return false
	}

	return rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func arePathsRelated(p1, p2 string) bool {
	return isPathPrefix(p1, p2) || isPathPrefix(p2, p1)
}

func LoadConfig() {
	gBackupDir, _ = os.LookupEnv(cBackupWithRcloneDirEnv)
	if gBackupDir == "" {
		gBackupDir = cDefaultBackupDir
	}
	gBackupDir = ExpandHome(gBackupDir)
	configFilename := filepath.Join(gBackupDir, "config.yaml")
	f, err := os.Open(configFilename)
	if err != nil {
		panic(fmt.Errorf("error when opening configuration %s: %w", configFilename, err))
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	if err := dec.Decode(&gConfig); err != nil {
		if err == io.EOF {
			panic(fmt.Errorf("configuration %s is empty", configFilename))
		}
		panic(fmt.Errorf(
			"error reading configuration %s: %w", configFilename, err))
	}

	if gConfig.DirSizeFactor == 0 {
		gConfig.DirSizeFactor = cDefaultDirSizeFactor
	}

	gConfig.Dirs = map[string]TDirConfig{}
	for _, dirConfig := range gConfig.DirsConfig {
		gConfig.Dirs[dirConfig.SourceDir] = TDirConfig{
			SourceDir:    filepath.Clean(dirConfig.SourceDir),
			TargetDir:    getTargetDir(dirConfig),
			ExcludeDirs:  dirConfig.ExcludeDirs,
			ExcludePaths: dirConfig.ExcludePaths,
			ExcludeFiles: dirConfig.ExcludeFiles,
		}
	}

	// Ensure no relevant directory is a prefix of another
	for i := 0; i < len(gConfig.DirsConfig)-1; i++ {
		for k := i + 1; k < len(gConfig.DirsConfig); k++ {
			if arePathsRelated(gConfig.DirsConfig[i].SourceDir, gConfig.DirsConfig[k].SourceDir) {
				panic(fmt.Errorf("overlapping source dirs in config %s found: %s and %s",
					configFilename, gConfig.DirsConfig[i].SourceDir, gConfig.DirsConfig[k].SourceDir))
			}

			if arePathsRelated(getTargetDir(gConfig.DirsConfig[i]), getTargetDir(gConfig.DirsConfig[k])) {
				panic(fmt.Errorf("overlapping target dirs in config %s found: %s and %s",
					configFilename, gConfig.DirsConfig[i].TargetDir, gConfig.DirsConfig[k].TargetDir))

			}
		}
	}

	// Ensure all filename patterns are ok
	checkFilenamePatterns(gConfig.ExcludeFiles)
	checkFilenamePatterns(gConfig.ExcludeDirs)
	for _, dirConfig := range gConfig.DirsConfig {
		checkFilenamePatterns(dirConfig.ExcludeFiles)
		checkFilenamePatterns(dirConfig.ExcludeDirs)
	}
}

func checkFilenamePatterns(patterns []string) {
	for _, pattern := range patterns {
		if _, err := filepath.Match(pattern, "abc"); err != nil {
			panic(fmt.Errorf("invalid filename pattern %s: %w", pattern, err))
		}
	}
}
