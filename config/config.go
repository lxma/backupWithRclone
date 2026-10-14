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

var GConfig TConfig
var GBackupDir string

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
	GBackupDir, _ = os.LookupEnv(cBackupWithRcloneDirEnv)
	if GBackupDir == "" {
		GBackupDir = cDefaultBackupDir
	}
	var err error
	GBackupDir, err = tools.ExpandHome(GBackupDir)
	if err != nil {
		panic("Cannot expand home dir: " + GBackupDir)
	}
	configFilename := filepath.Join(GBackupDir, "config.yaml")
	f, err := os.Open(configFilename)
	if err != nil {
		panic(fmt.Errorf("error when opening configuration %s: %v", configFilename, err))
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	if err := dec.Decode(&GConfig); err != nil {
		if err == io.EOF {
			panic(fmt.Errorf("configuration %s is empty", configFilename))
		}
		panic(fmt.Errorf(
			"error reading configuration %s: %v", configFilename, err))
	}

	if GConfig.DirSizeFactor == 0 {
		GConfig.DirSizeFactor = cDefaultDirSizeFactor
	}

	GConfig.Dirs = map[string]TDirConfig{}
	for _, dirConfig := range GConfig.DirsConfig {
		GConfig.Dirs[dirConfig.SourceDir] = TDirConfig{
			SourceDir:    filepath.Clean(dirConfig.SourceDir),
			TargetDir:    getTargetDir(dirConfig),
			ExcludeDirs:  dirConfig.ExcludeDirs,
			ExcludePaths: dirConfig.ExcludePaths,
			ExcludeFiles: dirConfig.ExcludeFiles,
		}
	}

	// Ensure no relevant directory is a prefix of another
	for i := 0; i < len(GConfig.DirsConfig)-1; i++ {
		for k := i + 1; k < len(GConfig.DirsConfig); k++ {
			if arePathsRelated(GConfig.DirsConfig[i].SourceDir, GConfig.DirsConfig[k].SourceDir) {
				panic(fmt.Errorf("overlapping source dirs in config %s found: %s and %s",
					configFilename, GConfig.DirsConfig[i].SourceDir, GConfig.DirsConfig[k].SourceDir))
			}

			if arePathsRelated(getTargetDir(GConfig.DirsConfig[i]), getTargetDir(GConfig.DirsConfig[k])) {
				panic(fmt.Errorf("overlapping target dirs in config %s found: %s and %s",
					configFilename, GConfig.DirsConfig[i].TargetDir, GConfig.DirsConfig[k].TargetDir))

			}
		}
	}

	// Ensure all filename patterns are ok
	checkFilenamePatterns(GConfig.ExcludeFiles)
	checkFilenamePatterns(GConfig.ExcludeDirs)
	for _, dirConfig := range GConfig.DirsConfig {
		checkFilenamePatterns(dirConfig.ExcludeFiles)
		checkFilenamePatterns(dirConfig.ExcludeDirs)
	}
}

func checkFilenamePatterns(patterns []string) {
	for _, pattern := range patterns {
		if _, err := filepath.Match(pattern, "abc"); err != nil {
			panic(fmt.Errorf("invalid filename pattern %s: %v", pattern, err))
		}
	}
}
