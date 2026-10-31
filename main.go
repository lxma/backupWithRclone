package main

import (
	"backupWithRclone/config"
	"backupWithRclone/dbAccess"
	"backupWithRclone/exclusions"
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

// flags
var gExtraRcloneArgs []string
var gInteractiveMode bool
var gCheckSums bool
var gCheckSizes bool

func main() {
	flag.Func("x", "Extra rclone args", func(arg string) error {
		gExtraRcloneArgs = append(gExtraRcloneArgs, arg)
		return nil
	})
	flag.BoolVar(&gInteractiveMode, "i", false, "Use output for interactive usage")
	flag.BoolVar(&gCheckSums, "cc", false, "Perform a check using true checksums")
	flag.BoolVar(&gCheckSizes, "cs", false, "Perform a check using just file sizes")
	flag.Parse()

	config.InitConfig()
	log := logger.InitLog()
	defer log.Close()

	if gCheckSizes && gCheckSums {
		logger.Printf("Error: Can only use one of --cc, --cs (compare either by sizes or by checksums)\n")
		os.Exit(1)
	}
	if gCheckSums {
		logger.Printf("Don't Copy, just check by comparing checksums\n")
	}
	if gCheckSizes {
		logger.Printf("Don't Copy, just check by comparing sizes\n")
	}

	lock, lockOk := lock.AcquireLock()
	if !lockOk {
		fmt.Println("A parallel instance is already running, quitting")
		return
	}
	defer lock.ReleaseLock()

	db := dbAccess.InitDB(filepath.Join(config.GBackupDir, "backups.sqlite"))
	defer db.Close()

	backupDates := dbAccess.ReadCutoffTimestamps(db)

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
	if !(gCheckSizes || gCheckSums) {
		gotools.SortStable(dirsAndCutoffs, func(a, b tDirAndCutoff) bool {
			return b.cutoff.After(a.cutoff)
		})
	}
	dbAccess.StoreMainDirectoryStates(db, sourceDirs)

	for _, dirAndCutoff := range dirsAndCutoffs {
		updateDir(db, dirAndCutoff.dir)
	}
}

func updateDir(db *sql.DB, dir string) {
	logger.Printf("##############################################################\n")
	logger.Printf("## Updating main dir %s\n", dir)

	cleanDir := filepath.Clean(tools.ExpandHome(dir))
	exclusions := exclusions.MakeExclusions(dir)

	newCutoff := time.Now().Add(-time.Second)
	if !(gCheckSizes || gCheckSums) {
		dbAccess.MarkUpdateDirStarted(db, dir, newCutoff)
	}

	cleanTargetDir := filepath.Clean(config.GConfig.Dirs[dir].TargetDir)
	destination := addPathToRcloneDestination(cleanTargetDir)
	ok := runRclone(cleanDir, destination, cleanTargetDir, exclusions.RclonePatterns, config.GConfig.Dirs[dir].RcloneParams, config.GConfig.Dirs[dir].CopyLinks)
	if ok {
		logger.Printf("Completed updating %s\n", cleanDir)
	} else {
		logger.Printf("Failed updating %s\n", cleanDir)
	}

	if !(gCheckSizes || gCheckSums) {
		dbAccess.MarkUpdateDirCompleted(db, dir, newCutoff)
	}
}

func addPathToRcloneDestination(targetDir string) string {
	if strings.HasSuffix(config.GConfig.RcloneDestinationPrefix, ":") ||
		strings.HasSuffix(config.GConfig.RcloneDestinationPrefix, "/") {
		return config.GConfig.RcloneDestinationPrefix + targetDir
	}
	return filepath.Join(config.GConfig.RcloneDestinationPrefix, targetDir)
}

func runRclone(sourceDir, destination string, cleanTargetDir string, rclonePatterns []string, configParams []string, copyLinks bool) bool {
	extraArgs := []string{sourceDir, destination, "--delete-excluded", "--exclude-from", "-"}
	if gInteractiveMode {
		extraArgs = append(extraArgs, "-P")
	} else {
		extraArgs = append(extraArgs, "-v", "--stats", "0")
	}
	extraArgs = append(extraArgs, configParams...)
	extraArgs = append(extraArgs, gExtraRcloneArgs...)

	// Handing of soft links
	if copyLinks {
		// Copy links as content rather than storing the links
		extraArgs = append(extraArgs, "--copy-links")
	} else {
		// Store only the softlinks
		extraArgs = append(extraArgs, "--links")
	}

	rCloneCommand := "sync"

	if gCheckSizes {
		rCloneCommand = "cryptcheck"
		protocolFilename := filepath.Join(config.GBackupDir, tools.SanitizeFilename(fmt.Sprintf("size-checks-%s.log", cleanTargetDir)))
		extraArgs = append(extraArgs, "--size-only", "--combined", protocolFilename)
	}

	if gCheckSums {
		rCloneCommand = "cryptcheck"
		protocolFilename := filepath.Join(config.GBackupDir, tools.SanitizeFilename(fmt.Sprintf("checksum-checks-%s.log", cleanTargetDir)))
		extraArgs = append(extraArgs, "--combined", protocolFilename)
	}

	return executor.ExecuteRclone(rCloneCommand, extraArgs, rclonePatterns)
}
