package config

import (
	"backupWithRclone/tools"
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
	BackupDir               string                `yaml:"-"`
}

type TDirConfig struct {
	SourceDir      string   `yaml:"source_dir"`
	CleanSourceDir string   `yaml:"-"`
	TargetDir      string   `yaml:"target_dir,omitempty"`
	ExcludeDirs    []string `yaml:"exclude_dirs"`
	ExcludeFiles   []string `yaml:"exclude_files"`
	RcloneParams   []string `yaml:"rclone_params"`
	CopyLinks      bool     `yaml:"copy_links"`
}

const cBackupWithRcloneDirEnv = "BACKUP_WITH_RCLONE_DIR"
const cDefaultDirSizeFactor = 200
const cDefaultBackupDir = "~/.backupWithRclone"

var gBackupDir string

func getConfigFilename(backupDir string) string {
	return filepath.Join(backupDir, "config.yaml")
}

func getTargetDir(dirConfig TDirConfig) string {
	gotools.Assert(dirConfig.CleanSourceDir != "", "CleanSourceDir must be set before getTargetDir is called")
	if dirConfig.TargetDir == "" {
		return filepath.Base(dirConfig.CleanSourceDir)
	}
	return dirConfig.TargetDir
}

// errorInConfigFile stops the execution gracefully (but with error code) printing
// the error message
func errorInConfigFile(configFileName string, format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error in config file %s:\n", configFileName)
	tools.StopWithErrorMessage(format, args...)
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

func getBackupDir() string {
	if gBackupDir != "" {
		return gBackupDir
	}
	gBackupDir, _ = os.LookupEnv(cBackupWithRcloneDirEnv)
	if gBackupDir == "" {
		gBackupDir = cDefaultBackupDir
	}
	gBackupDir = tools.ExpandHome(gBackupDir)
	return gBackupDir
}

func cleanDirsConfig(cfg *TConfig) {
	cfg.DirsConfig = gotools.Map(cfg.DirsConfig, func(dc TDirConfig) TDirConfig {
		dc.CleanSourceDir = filepath.Clean(dc.SourceDir)
		dc.TargetDir = getTargetDir(dc)
		return dc
	})
	cfg.Dirs = map[string]TDirConfig{}
	for _, dc := range cfg.DirsConfig {
		cfg.Dirs[dc.SourceDir] = dc
	}
}

func GetConfig() *TConfig {
	var resultConfig TConfig

	backupDir := getBackupDir()
	var err error
	configFilename := getConfigFilename(backupDir)
	f, err := os.Open(configFilename)
	gotools.Assert(err == nil, "error when opening configuration %s: %v", configFilename, err)
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	if err := dec.Decode(&resultConfig); err != nil {
		if err == io.EOF {
			errorInConfigFile(configFilename, "Configuration file is empty")
		} else {
			errorInConfigFile(configFilename, "error reading configuration %s: %v", configFilename, err)
		}
	}

	if resultConfig.DirSizeFactor == 0 {
		resultConfig.DirSizeFactor = cDefaultDirSizeFactor
	}

	resultConfig.BackupDir = backupDir

	cleanDirsConfig(&resultConfig)

	// Ensure no relevant directory is a prefix of another
	for i := 0; i < len(resultConfig.DirsConfig)-1; i++ {
		for k := i + 1; k < len(resultConfig.DirsConfig); k++ {
			if arePathsRelated(resultConfig.DirsConfig[i].SourceDir, resultConfig.DirsConfig[k].SourceDir) {
				errorInConfigFile(configFilename, "overlapping source dirs in config %s found: %s and %s",
					configFilename, resultConfig.DirsConfig[i].SourceDir, resultConfig.DirsConfig[k].SourceDir)
			}

			if arePathsRelated(getTargetDir(resultConfig.DirsConfig[i]), getTargetDir(resultConfig.DirsConfig[k])) {
				errorInConfigFile(configFilename, "overlapping target dirs in config %s found: %s and %s",
					configFilename, resultConfig.DirsConfig[i].TargetDir, resultConfig.DirsConfig[k].TargetDir)

			}
		}
	}

	// Ensure all filename patterns are ok
	checkFilenamePatterns(resultConfig.ExcludeFiles, "in global exclude files")
	checkFilenamePatterns(resultConfig.ExcludeDirs, "in global exclude dirs")
	for _, dirConfig := range resultConfig.DirsConfig {
		checkFilenamePatterns(dirConfig.ExcludeFiles, "in exclude files of "+dirConfig.SourceDir)
		checkFilenamePatterns(dirConfig.ExcludeDirs, "in exclude dirs of "+dirConfig.SourceDir)
	}

	return &resultConfig
}

func checkFilenamePatterns(patterns []string, context string) {
	for _, pattern := range patterns {
		if _, err := filepath.Match(pattern, "abc"); err != nil {
			errorInConfigFile("Invalid filename pattern %s %s: %v", pattern, context, err)
		}
	}
}
