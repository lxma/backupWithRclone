package main

import (
	"backupWithRclone/config"
	"backupWithRclone/dbAccess"
	"backupWithRclone/executor"
	"backupWithRclone/lock"
	"backupWithRclone/logger"
	"backupWithRclone/tools"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxma/gotools/v2"
)

type tDirAndCutoff struct {
	dir    string
	cutoff time.Time
}

type tRunMode string

const (
	cRunModeSync       tRunMode = "SYNC"
	cRunModeCheckSums  tRunMode = "CHECK_SUMS"
	cRunModeCheckSizes tRunMode = "CHECK_SIZES"
)

// flags
var gExtraRcloneArgs []string
var gInteractiveMode bool
var gCheckSums bool
var gCheckSizes bool
var gNoSort bool
var gVerbose bool

func main() {
	flag.Func("x", "Extra rclone args", func(arg string) error {
		gExtraRcloneArgs = append(gExtraRcloneArgs, arg)
		return nil
	})
	flag.BoolVar(&gInteractiveMode, "i", false, "Use output for interactive usage")
	flag.BoolVar(&gCheckSums, "cc", false, "Perform a check using true checksums")
	flag.BoolVar(&gCheckSizes, "cs", false, "Perform a check using just file sizes")
	flag.BoolVar(&gNoSort, "ns", false, "No sort; Perform update of directories in the order they appear in the config file")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	cfg := config.GetConfig()
	log := logger.InitLog(cfg.BackupDir)
	defer log.Close()

	config.GVerbose = *verbose

	if gCheckSizes && gCheckSums {
		logger.Printf("Error: Can only use one of --cc, --cs (compare either by sizes or by checksums)\n")
		os.Exit(1)
	}
	runMode := cRunModeSync
	if gCheckSums {
		runMode = cRunModeCheckSums
		logger.Printf("Don't Copy, just check by comparing checksums\n")
	}
	if gCheckSizes {
		runMode = cRunModeCheckSizes
		logger.Printf("Don't Copy, just check by comparing sizes\n")
	}

	lock, lockOk := lock.AcquireLock(cfg.BackupDir)
	if !lockOk {
		fmt.Println("A parallel instance is already running, quitting")
		return
	}
	defer lock.ReleaseLock()

	db := dbAccess.InitDB(filepath.Join(cfg.BackupDir, "backups.sqlite"))
	defer db.Close()

	backupDates := dbAccess.ReadCutoffTimestamps(db)

	// sort dirs by recent update time ascending
	sourceDirs := gotools.Map(cfg.DirsConfig, func(dirConfig config.TDirConfig) string {
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
	dbAccess.StoreMainDirectoryStates(db, sourceDirs)

	for _, dirAndCutoff := range dirsAndCutoffs {
		updateDir(db, dirAndCutoff.dir, cfg, runMode)
	}
}

func updateDir(db *sql.DB, dir string, cfg *config.TConfig, runMode tRunMode) {
	logger.Printf("##############################################################\n")
	logger.Printf("## Updating main dir %s\n", dir)

	cleanDir := filepath.Clean(tools.ExpandHome(dir))

	newCutoff := time.Now().Add(-time.Second)
	if !(gCheckSizes || gCheckSums) {
		dbAccess.MarkUpdateDirStarted(db, dir, newCutoff)
	}

	cleanTargetDir := filepath.Clean(cfg.Dirs[dir].TargetDir)
	destination := addPathToRcloneDestination(cleanTargetDir, cfg.RcloneDestinationPrefix)
	dirConfig := cfg.Dirs[dir]
	args := getArgs(cleanDir, cleanTargetDir, destination, cfg.BackupDir, runMode, gInteractiveMode, dirConfig.CopyLinks, dirConfig.RcloneParams, gExtraRcloneArgs)
	excludePatterns := makeRcloneExcludePatterns(cfg.ExcludeDirs, cfg.ExcludeFiles, dirConfig.ExcludeDirs, dirConfig.ExcludeFiles)
	ok := executor.ExecuteRclone(getCommand(runMode), args, excludePatterns)
	//	ok := runRclone(cleanDir, destination, cleanTargetDir, exclusions.RclonePatterns, cnf.Dirs[dir].RcloneParams, cnf.Dirs[dir].CopyLinks)
	if ok {
		logger.Printf("Completed updating %s\n", cleanDir)
	} else {
		logger.Printf("Failed updating %s\n", cleanDir)
	}

	if !(gCheckSizes || gCheckSums) {
		dbAccess.MarkUpdateDirCompleted(db, dir, newCutoff)
	}
}

func addPathToRcloneDestination(targetDir string, destinationPrefix string) string {
	if strings.HasSuffix(destinationPrefix, ":") ||
		strings.HasSuffix(destinationPrefix, "/") {
		return destinationPrefix + targetDir
	}
	return filepath.Join(destinationPrefix, targetDir)
}

func getCommand(runMode tRunMode) string {
	switch runMode {
	case cRunModeSync:
		return "sync"
	default:
		return "cryptcheck"
	}
}

func getArgs(
	sourceDir string,
	cleanTargetDir string,
	destination string,
	backupDir string,
	runMode tRunMode,
	interactiveMode bool,
	copyLinks bool,
	argsFromConfig []string,
	argsFromCommandLine []string) []string {
	args := []string{sourceDir, destination, "--delete-excluded", "--exclude-from", "-"}
	if interactiveMode {
		args = append(args, "-P")
	} else {
		args = append(args, "-v", "--stats", "0")
	}
	args = append(args, argsFromConfig...)
	args = append(args, argsFromCommandLine...)

	// Handing of soft links
	if copyLinks {
		// Copy links as content rather than storing the links
		args = append(args, "--copy-links")
	} else {
		// Store only the softlinks
		args = append(args, "--links")
	}

	switch runMode {
	case cRunModeCheckSizes:
		protocolFilename := filepath.Join(backupDir, tools.SanitizeFilename(fmt.Sprintf("size-checks-%s.log", cleanTargetDir)))
		args = append(args, "--size-only", "--combined", protocolFilename)
	case cRunModeCheckSums:
		protocolFilename := filepath.Join(backupDir, tools.SanitizeFilename(fmt.Sprintf("checksum-checks-%s.log", cleanTargetDir)))
		args = append(args, "--combined", protocolFilename)
	}
	return args
}

func makeRcloneExcludePatterns(globalExcludeDirs, globalExcludeFiles, dirExcludeDirs, dirExcludeFiles []string) []string {
	excludeDirs := gotools.CopySlice(globalExcludeDirs)
	excludeDirs = append(excludeDirs, dirExcludeDirs...)
	patterns := gotools.CopySlice(globalExcludeFiles)
	patterns = append(patterns, dirExcludeFiles...)
	patterns = append(patterns, gotools.Map(excludeDirs, func(pattern string) string {
		return filepath.Clean(pattern) + string(filepath.Separator)
	})...)
	return patterns
}
