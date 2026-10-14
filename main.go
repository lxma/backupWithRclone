package main

import (
	"backupWithRclone/config"
	"backupWithRclone/dbAccess"
	"backupWithRclone/exclusions"
	"backupWithRclone/log"
	"backupWithRclone/tools"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxma/gotools/v2"
	"golang.org/x/sys/unix"
)

type tDirAndCutoff struct {
	dir    string
	cutoff time.Time
}

// flags
var gExtraRcloneArgs []string
var gInteractiveMode bool
var gForceRootSync bool

func main() {
	config.LoadConfig()
	log.InitLog()
	defer log.CloseLog()

	flag.Func("x", "Extra rclone args", func(arg string) error {
		gExtraRcloneArgs = append(gExtraRcloneArgs, arg)
		return nil
	})
	flag.BoolVar(&gForceRootSync, "f", false, "Force sync each directory on root level")
	flag.BoolVar(&gInteractiveMode, "i", false, "Use output for interactive usage")
	flag.Parse()

	lock, ok := AcquireInstanceLock()
	if !ok {
		fmt.Println("A parallel instance is already running, quitting")
		return
	}
	defer lock.Close()

	db := dbAccess.InitDB(filepath.Join(config.GBackupDir, "backups.sqlite"))
	defer db.Close()

	backupDates := dbAccess.ReadCutoffTimestamps(db)
	// sort dirs by recent update time ascending

	sourceDirs := gotools.GetKeys(config.GConfig.Dirs)

	dirsAndCutoffs := make([]tDirAndCutoff, len(sourceDirs))
	for idx, dir := range sourceDirs {
		date, ok := backupDates[dir]
		if !ok {
			date = time.Time{}
		}
		dirsAndCutoffs[idx] = tDirAndCutoff{dir, date}
	}
	gotools.SortStable(dirsAndCutoffs, func(a, b tDirAndCutoff) bool {
		return b.cutoff.After(a.cutoff)
	})
	dbAccess.StoreMainDirectoryStates(db, sourceDirs)

	for _, dirAndCutoff := range dirsAndCutoffs {
		updateDir(db, dirAndCutoff.dir, dirAndCutoff.cutoff)
	}
}

func updateDir(db *sql.DB, dir string, oldCutoff time.Time) {
	log.Printf("\nUpdating main dir %s\n", dir)
	expandedDir, err := tools.ExpandHome(dir)
	if err != nil {
		log.Printf("Error expanding backup dir %s: %s", dir, err)
		panic(fmt.Errorf("Error expanding backup dir %s: %s", dir, err))
	}
	cleanDir := filepath.Clean(expandedDir)
	exclusions := exclusions.MakeExclusions(dir)

	var updateDirs []string
	if gForceRootSync {
		updateDirs = []string{cleanDir}
	} else {
		updateDirs, err = findDirsForUpdate(cleanDir, oldCutoff, exclusions)
		if err != nil {
			log.Printf("error finding dirs for update in %s: %v\n", dir, err)
			return
		}
	}
	newCutoff := time.Now().Add(-time.Second)
	dbAccess.StartStoringMainDir(db, dir, updateDirs, newCutoff)
	for _, updateDir := range updateDirs {
		dbAccess.MarkSubDirStarted(db, updateDir)
		log.Printf("Start updating subdir %s\n", updateDir)
		destination := addPathToRcloneDestination(getTargetPath(cleanDir, updateDir, config.GConfig.Dirs[dir].TargetDir))
		if err := RunRclone(updateDir, destination, exclusions, config.GConfig.Dirs[dir].RcloneParams); err != nil {
			panic(err)
		}
		dbAccess.MarkSubDirCompleted(db, updateDir)
		log.Printf("Completed updating subdir %s\n", updateDir)
	}
	dbAccess.MarkMainDirCompleted(db, dir, newCutoff)
}

func getTargetPath(rootDir, updateDir, targetDir string) string {
	gotools.Assert(strings.HasPrefix(updateDir, rootDir))
	if rootDir == updateDir {
		return filepath.Clean(targetDir)
	}
	return filepath.Join(filepath.Clean(targetDir), updateDir[len(rootDir)+1:])
}

func addPathToRcloneDestination(targetDir string) string {
	if strings.HasSuffix(config.GConfig.RcloneDestinationPrefix, ":") ||
		strings.HasSuffix(config.GConfig.RcloneDestinationPrefix, "/") {
		return config.GConfig.RcloneDestinationPrefix + targetDir
	}
	return filepath.Join(config.GConfig.RcloneDestinationPrefix, targetDir)
}

func RunRclone(sourceDir, destination string, exclusions exclusions.TExclusions, configParams []string) error {
	extraArgs := []string{"sync", sourceDir, destination, "--delete-excluded", "--exclude-from", "-"}
	if gInteractiveMode {
		extraArgs = append(extraArgs, "-P")
	} else {
		extraArgs = append(extraArgs, "-v", "--stats", "0")
	}
	extraArgs = append(extraArgs, configParams...)
	extraArgs = append(extraArgs, gExtraRcloneArgs...)

	cmd := exec.Command("rclone", extraArgs...)

	cmd.Stdin = strings.NewReader(strings.Join(exclusions.RclonePatterns, "\n") + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("error running rclone command: %v\n", err)
		return fmt.Errorf("rclone failed: %v", err)
	}

	return nil
}

// AcquireInstanceLock returns an open file that must remain open
// for the entire protected operation. Close it to release the lock.
func AcquireInstanceLock() (*os.File, bool) {
	lockFile := filepath.Join(config.GBackupDir, "lock")
	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Printf("error opening lock file %s: %v\n", lockFile, err)
		panic(fmt.Errorf("failed to open lock file %s: %v", lockFile, err))
	}

	err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		file.Close()

		if errors.Is(err, unix.EWOULDBLOCK) ||
			errors.Is(err, unix.EAGAIN) {
			return nil, false
		}
		log.Printf("error acquiring instance lock %s: %v\n", lockFile, err)
		panic(fmt.Errorf("error acquiring instance lock %s: %v", lockFile, err))
	}

	return file, true
}
