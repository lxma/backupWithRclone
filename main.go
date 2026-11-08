package main

import (
	"backupWithRclone/config"
	"backupWithRclone/dbAccess"
	"backupWithRclone/executor"
	"backupWithRclone/lock"
	"backupWithRclone/logger"
	"backupWithRclone/tools"
	"flag"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lxma/gotools/v2"
)

type tDirAndCutoff struct {
	dir    string
	cutoff time.Time
}

type tRunMode string

type tChanges struct {
	dir         string
	changeLines []string
	errorText   string
}

const (
	cRunModeSync       tRunMode = "SYNC"
	cRunModeCheckSums  tRunMode = "CHECK_SUMS"
	cRunModeCheckSizes tRunMode = "CHECK_SIZES"
)

// Regular expressions
var rProtocolLineIsChange = regexp.MustCompile("^[*+-]")

// flags
var gExtraRcloneArgs []string
var gInteractiveMode bool
var gRunMode tRunMode
var gNoSort bool

func main() {
	gRunMode = cRunModeSync

	flag.Func("x", "Extra rclone args", func(arg string) error {
		gExtraRcloneArgs = append(gExtraRcloneArgs, arg)
		return nil
	})
	flag.BoolFunc("cc", "Perform a check using true checksums", func(arg string) error {
		gRunMode = cRunModeCheckSums
		return nil
	})
	flag.BoolFunc("cs", "Perform a check using just file sizes", func(arg string) error {
		gRunMode = cRunModeCheckSizes
		return nil
	})
	flag.BoolVar(&gInteractiveMode, "i", false, "Use output for interactive usage")
	flag.BoolVar(&gNoSort, "ns", false, "No sort; Perform update of directories in the order they appear in the config file")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	defer tools.RunDeferredCalls(*verbose)

	config.InitConfig()
	logger.InitLog(*verbose)
	lockOk := lock.AcquireLock()
	if !lockOk {
		fmt.Println("A parallel instance is already running, quitting")
		return
	}
	dbAccess.InitDB()

	run()
}

func run() {
	backupDates := dbAccess.ReadCutoffTimestamps()

	// sort dirs by recent update time ascending
	sourceDirs := gotools.Map(config.GConfig.DirsConfig, func(dirConfig config.TDirConfig) string {
		return dirConfig.SourceDir
	})
	dirsAndCutoffs := gotools.Map(sourceDirs, func(sourceDir string) tDirAndCutoff {
		date, ok := backupDates[sourceDir]
		if !ok {
			date = time.Time{}
		}
		return tDirAndCutoff{dir: sourceDir, cutoff: date}
	})
	if !gNoSort {
		gotools.SortStable(dirsAndCutoffs, func(a, b tDirAndCutoff) bool {
			return b.cutoff.After(a.cutoff)
		})
	}
	dbAccess.StoreMainDirectoryStates(sourceDirs)

	var allChanges []tChanges
	for _, dirAndCutoff := range dirsAndCutoffs {
		changes := updateDir(dirAndCutoff.dir)
		if changes != nil && (changes.errorText != "" || len(changes.changeLines) > 0) {
			allChanges = append(allChanges, *changes)
		}
	}
	if allChanges != nil {
		logger.Printf("###############################################################################")
		logger.Printf("## All changes:")
		for _, changes := range allChanges {
			logger.Printf("## Dir %s", changes.dir)
			if changes.errorText != "" {
				logger.Printf("## ERROR: %s", changes.errorText)
			}
			for _, changeLine := range changes.changeLines {
				logger.Printf("##    %s", changeLine)
			}
		}
	}
}

func updateDir(dir string) *tChanges {
	logger.Printf("##############################################################")
	logger.Printf("## Updating directory %s", dir)

	newCutoff := time.Now().Add(-time.Second)
	if gRunMode == cRunModeSync {
		dbAccess.MarkUpdateDirStarted(dir, newCutoff)
	}

	excludePatterns := getRcloneExcludePatterns(dir)
	ok := executor.ExecuteRclone(getRcloneCommand(gRunMode), getRcloneArgs(dir), excludePatterns)
	if ok {
		logger.Printf("## Completed updating %s", dir)
		if gRunMode == cRunModeSync {
			dbAccess.MarkUpdateDirCompleted(dir, newCutoff)
		}
	} else {
		logger.Printf("## Failed updating %s", dir)
	}
	if gRunMode == cRunModeCheckSizes || gRunMode == cRunModeCheckSums {
		return getChanges(dir)
	}
	return nil
}

// getChanges collects differences from the protocol file of the given directory
func getChanges(dir string) *tChanges {
	protocolFile := getProtocolFilename(dir)
	lines, err := gotools.ReadLines(protocolFile)
	if err != nil {
		return &tChanges{
			dir:       dir,
			errorText: fmt.Sprintf("Error reading protocol file %s: %v\n", protocolFile, err),
		}
	}
	return &tChanges{
		dir: dir,
		changeLines: gotools.Filter(lines, func(line string) bool {
			return rProtocolLineIsChange.MatchString(line)
		}),
	}
}

func getDestination(dir string) string {
	destinationPrefix := config.GConfig.RcloneDestinationPrefix
	dirConfig := config.GConfig.Dirs[dir]
	if strings.HasSuffix(destinationPrefix, ":") ||
		strings.HasSuffix(destinationPrefix, "/") {
		return destinationPrefix + dirConfig.CleanTargetDir
	}
	return filepath.Join(destinationPrefix, dirConfig.CleanTargetDir)
}

func getRcloneCommand(runMode tRunMode) string {
	switch runMode {
	case cRunModeSync:
		return "sync"
	default:
		return "cryptcheck"
	}
}

func getRcloneArgs(dir string) []string {
	dirConfig := config.GConfig.Dirs[dir]
	destination := getDestination(dir)
	args := []string{tools.ExpandHome(dirConfig.CleanSourceDir), destination, "--delete-excluded", "--exclude-from", "-"}
	if gInteractiveMode {
		args = append(args, "-P")
	} else {
		args = append(args, "-v", "--stats", "0")
	}
	args = append(args, dirConfig.RcloneParams...)
	args = append(args, gExtraRcloneArgs...)

	// Handing of soft links
	if dirConfig.CopyLinks {
		// Copy links as content rather than storing the links
		args = append(args, "--copy-links")
	} else {
		// Store only the softlinks
		args = append(args, "--links")
	}

	switch gRunMode {
	case cRunModeCheckSizes:
		args = append(args, "--size-only", "--combined", getProtocolFilename(dir))
	case cRunModeCheckSums:
		args = append(args, "--combined", getProtocolFilename(dir))
	}
	return args
}

func getRcloneExcludePatterns(dir string) []string {
	dirConfig := config.GConfig.Dirs[dir]
	excludeDirs := gotools.CopySlice(config.GConfig.ExcludeDirs)
	excludeDirs = append(excludeDirs, dirConfig.ExcludeDirs...)
	patterns := gotools.CopySlice(config.GConfig.ExcludeFiles)
	patterns = append(patterns, dirConfig.ExcludeFiles...)
	patterns = append(patterns, gotools.Map(excludeDirs, func(pattern string) string {
		return filepath.Clean(pattern) + string(filepath.Separator)
	})...)
	return patterns
}

func getProtocolFilename(dir string) string {
	dirConfig := config.GConfig.Dirs[dir]
	var filePrefix string
	switch gRunMode {
	case cRunModeCheckSizes:
		filePrefix = "size-checks"
	case cRunModeCheckSums:
		filePrefix = "checksum-checks"
	default:
		tools.StopWithErrorMessage("should not call \"getProtocolFilename\" when runmode is %d", gRunMode)
	}
	return filepath.Join(config.GBackupDir, tools.EscapeFilenamePart(fmt.Sprintf("%s-%s.log", filePrefix, dirConfig.CleanTargetDir)))
}
