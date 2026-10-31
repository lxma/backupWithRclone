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

func main() {
	config.InitConfig()
	log := logger.InitLog()
	defer log.Close()

	flag.Func("x", "Extra rclone args", func(arg string) error {
		gExtraRcloneArgs = append(gExtraRcloneArgs, arg)
		return nil
	})
	flag.BoolVar(&gInteractiveMode, "i", false, "Use output for interactive usage")
	flag.Parse()

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
	gotools.SortStable(dirsAndCutoffs, func(a, b tDirAndCutoff) bool {
		return b.cutoff.After(a.cutoff)
	})
	dbAccess.StoreMainDirectoryStates(db, sourceDirs)

	for _, dirAndCutoff := range dirsAndCutoffs {
		updateDir(db, dirAndCutoff.dir, dirAndCutoff.cutoff)
	}
}

func updateDir(db *sql.DB, dir string, oldCutoff time.Time) {
	logger.Printf("##############################################################\n")
	logger.Printf("## Updating main dir %s\n", dir)

	cleanDir := filepath.Clean(tools.ExpandHome(dir))
	exclusions := exclusions.MakeExclusions(dir)

	newCutoff := time.Now().Add(-time.Second)
	dbAccess.MarkUpdateDirStarted(db, dir, newCutoff)

	destination := addPathToRcloneDestination(filepath.Clean(config.GConfig.Dirs[dir].TargetDir))
	ok := runRclone(cleanDir, destination, exclusions.RclonePatterns, config.GConfig.Dirs[dir].RcloneParams, config.GConfig.Dirs[dir].CopyLinks)
	if ok {
		logger.Printf("Completed updating %s\n", cleanDir)
	} else {
		logger.Printf("Failed updating %s\n", cleanDir)
	}

	dbAccess.MarkUpdateDirCompleted(db, dir, newCutoff)
}

func addPathToRcloneDestination(targetDir string) string {
	if strings.HasSuffix(config.GConfig.RcloneDestinationPrefix, ":") ||
		strings.HasSuffix(config.GConfig.RcloneDestinationPrefix, "/") {
		return config.GConfig.RcloneDestinationPrefix + targetDir
	}
	return filepath.Join(config.GConfig.RcloneDestinationPrefix, targetDir)
}

func runRclone(sourceDir, destination string, rclonePatterns []string, configParams []string, copyLinks bool) bool {
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

	return executor.ExecuteRclone(extraArgs, rclonePatterns)
}
