package exclusions

import (
	"backupWithRclone/config"
	"backupWithRclone/log"
	"os"
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
			log.Printf("error when matching filename pattern '%s': %v", exclude, err)
		}
		if matched {
			return true
		}
	}
	return false
}

func (e TExclusions) IsEntryExcluded(dirEntry os.FileInfo) bool {
	if dirEntry.IsDir() {
		return isFilenameExcluded(dirEntry.Name(), e.ExcludeDirs)
	} else {
		return isFilenameExcluded(dirEntry.Name(), e.ExcludeFiles)
	}
}
