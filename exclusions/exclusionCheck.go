package exclusions

import (
	"backupWithRclone/config"
	"backupWithRclone/logger"
	"path/filepath"

	"github.com/lxma/gotools/v2"
)

type TExclusions struct {
	ExcludeFiles   []string
	ExcludeDirs    []string
	RclonePatterns []string
}

func MakeExclusions(dir string) TExclusions {
	excludeDirs := gotools.CopySlice(config.GConfig.ExcludeDirs)
	excludeDirs = append(excludeDirs, config.GConfig.Dirs[dir].ExcludeDirs...)
	excludeFiles := gotools.CopySlice(config.GConfig.ExcludeFiles)
	excludeFiles = append(excludeFiles, config.GConfig.Dirs[dir].ExcludeFiles...)
	patterns := gotools.CopySlice(excludeFiles)
	patterns = append(patterns, gotools.Map(excludeDirs, func(pattern string) string {
		return filepath.Clean(pattern) + string(filepath.Separator)
	})...)
	return TExclusions{excludeFiles, excludeDirs, patterns}
}

func isFilenameExcluded(filename string, excludeList []string) bool {
	for _, exclude := range excludeList {
		matched, err := filepath.Match(exclude, filename)
		if err != nil {
			// such errors should have been ruled out when reading the configuration
			logger.Printf("error when matching filename pattern '%s': %v", exclude, err)
		}
		if matched {
			return true
		}
	}
	return false
}

func (e TExclusions) IsEntryExcluded(name string, isDir bool) bool {
	if isDir {
		return isFilenameExcluded(name, e.ExcludeDirs)
	} else {
		return isFilenameExcluded(name, e.ExcludeFiles)
	}
}
