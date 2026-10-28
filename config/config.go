package config

import (
	"backupWithRclone/tools"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
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
	SourceDir      string   `yaml:"source_dir"`
	CleanSourceDir string   `yaml:"-"`
	TargetDir      string   `yaml:"target_dir,omitempty"`
	ExcludeDirs    []string `yaml:"exclude_dirs"`
	ExcludeFiles   []string `yaml:"exclude_files"`
	RcloneParams   []string `yaml:"rclone_params"`
}

const cBackupWithRcloneDirEnv = "BACKUP_WITH_RCLONE_DIR"
const cDefaultDirSizeFactor = 200
const cDefaultBackupDir = "~/.backupWithRclone"

var GConfig TConfig
var GBackupDir string

func getConfigFilename() string {
	return filepath.Join(GBackupDir, "config.yaml")
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
func errorInConfigFile(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error in config file %s:\n", getConfigFilename())
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

func determineBackupDir() {
	if GBackupDir != "" {
		return
	}
	GBackupDir, _ = os.LookupEnv(cBackupWithRcloneDirEnv)
	if GBackupDir == "" {
		GBackupDir = cDefaultBackupDir
	}
}

func cleanDirsConfig() {
	GConfig.DirsConfig = gotools.Map(GConfig.DirsConfig, func(dc TDirConfig) TDirConfig {
		dc.CleanSourceDir = filepath.Clean(dc.SourceDir)
		dc.TargetDir = getTargetDir(dc)
		return dc
	})
	GConfig.Dirs = map[string]TDirConfig{}
	for _, dc := range GConfig.DirsConfig {
		GConfig.Dirs[dc.SourceDir] = dc
	}
}

func InitConfig() {
	if !reflect.ValueOf(GConfig).IsZero() {
		return
	}
	determineBackupDir()
	var err error
	GBackupDir = tools.ExpandHome(GBackupDir)
	configFilename := getConfigFilename()
	f, err := os.Open(configFilename)
	gotools.Assert(err == nil, "error when opening configuration %s: %v", configFilename, err)

	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	if err := dec.Decode(&GConfig); err != nil {
		if err == io.EOF {
			errorInConfigFile("Configuration file is empty")
		} else {
			errorInConfigFile("error reading configuration %s: %v", configFilename, err)
		}
	}

	if GConfig.DirSizeFactor == 0 {
		GConfig.DirSizeFactor = cDefaultDirSizeFactor
	}

	cleanDirsConfig()

	// Ensure no relevant directory is a prefix of another
	for i := 0; i < len(GConfig.DirsConfig)-1; i++ {
		for k := i + 1; k < len(GConfig.DirsConfig); k++ {
			if arePathsRelated(GConfig.DirsConfig[i].SourceDir, GConfig.DirsConfig[k].SourceDir) {
				errorInConfigFile("overlapping source dirs in config %s found: %s and %s",
					configFilename, GConfig.DirsConfig[i].SourceDir, GConfig.DirsConfig[k].SourceDir)
			}

			if arePathsRelated(getTargetDir(GConfig.DirsConfig[i]), getTargetDir(GConfig.DirsConfig[k])) {
				errorInConfigFile("overlapping target dirs in config %s found: %s and %s",
					configFilename, GConfig.DirsConfig[i].TargetDir, GConfig.DirsConfig[k].TargetDir)

			}
		}
	}

	// Ensure all filename patterns are ok
	checkFilenamePatterns(GConfig.ExcludeFiles, "in global exclude files")
	checkFilenamePatterns(GConfig.ExcludeDirs, "in global exclude dirs")
	for _, dirConfig := range GConfig.DirsConfig {
		checkFilenamePatterns(dirConfig.ExcludeFiles, "in exclude files of "+dirConfig.SourceDir)
		checkFilenamePatterns(dirConfig.ExcludeDirs, "in exclude dirs of "+dirConfig.SourceDir)
	}
}

func checkFilenamePatterns(patterns []string, context string) {
	for _, pattern := range patterns {
		if _, err := filepath.Match(pattern, "abc"); err != nil {
			errorInConfigFile("Invalid filename pattern %s %s: %v", pattern, context, err)
		}
	}
}
