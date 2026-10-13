package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
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

// logger
var gLogger *log.Logger

func main() {
	LoadConfig()

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

	db := InitDB(filepath.Join(gBackupDir, "backups.sqlite"))
	defer db.Close()

	backupDates := ReadCutoffTimestamps(db)
	// sort dirs by recent update time ascending

	sourceDirs := gotools.GetKeys(gConfig.Dirs)

	dirsAndCutoffs := make([]tDirAndCutoff, len(sourceDirs))
	for idx, dir := range sourceDirs {
		date, ok := backupDates[dir]
		if !ok {
			date = time.Time{}
		}
		dirsAndCutoffs[idx] = tDirAndCutoff{dir, date}
	}
	gotools.SortBy(dirsAndCutoffs, func(a, b tDirAndCutoff) bool {
		return b.cutoff.After(a.cutoff)
	})
	StoreMainDirectoryStates(db, sourceDirs)

	for _, dirAndCutoff := range dirsAndCutoffs {
		updateDir(db, dirAndCutoff.dir, dirAndCutoff.cutoff)
	}
}

func updateDir(db *sql.DB, dir string, oldCutoff time.Time) {
	fmt.Printf("\nUpdating %s\n", dir)
	cleanDir := filepath.Clean(ExpandHome(dir))
	exclusions := MakeExclusions(dir)

	var updateDirs []string
	var err error
	if gForceRootSync {
		updateDirs = []string{cleanDir}
	} else {
		updateDirs, err = findDirsForUpdate(cleanDir, oldCutoff, exclusions)
		if err != nil {
			fmt.Printf("error finding dirs for update in %s: %v\n", dir, err)
			fmt.Printf("NOT SYNCING %s!\n", dir)
			return
		}
	}
	newCutoff := time.Now().Add(-time.Second)
	StartStoringMainDir(db, dir, updateDirs, newCutoff)
	for _, updateDir := range updateDirs {
		MarkSubDirStarted(db, updateDir)
		fmt.Printf("- updating %s ...\n", updateDir)
		destination := addPathToRcloneDestination(getTargetPath(cleanDir, updateDir, gConfig.Dirs[dir].TargetDir))
		if err := RunRclone(updateDir, destination, exclusions); err != nil {
			panic(err)
		}
		MarkSubDirCompleted(db, updateDir)
	}
	MarkMainDirCompleted(db, dir, newCutoff)
}

func getTargetPath(rootDir, updateDir, targetDir string) string {
	gotools.Assert(strings.HasPrefix(updateDir, rootDir))
	if rootDir == updateDir {
		return filepath.Clean(targetDir)
	}
	return filepath.Join(filepath.Clean(targetDir), updateDir[len(rootDir)+1:])
}

func addPathToRcloneDestination(targetDir string) string {
	if strings.HasSuffix(gConfig.RcloneDestinationPrefix, ":") ||
		strings.HasSuffix(gConfig.RcloneDestinationPrefix, "/") {
		return gConfig.RcloneDestinationPrefix + targetDir
	}
	return filepath.Join(gConfig.RcloneDestinationPrefix, targetDir)
}

func RunRclone(sourceDir, destination string, exclusions TExclusions) error {
	extraArgs := []string{"sync", sourceDir, destination, "--delete-excluded", "--exclude-from", "-"}
	if gInteractiveMode {
		extraArgs = append(extraArgs, "-P")
	} else {
		extraArgs = append(extraArgs, "-v", "--stats", "0")
	}
	extraArgs = append(extraArgs, gExtraRcloneArgs...)
	cmd := exec.Command("rclone", extraArgs...)

	cmd.Stdin = strings.NewReader(strings.Join(exclusions.RclonePatterns, "\n") + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone failed: %w", err)
	}

	return nil
}

// AcquireInstanceLock returns an open file that must remain open
// for the entire protected operation. Close it to release the lock.
func AcquireInstanceLock() (*os.File, bool) {
	lockFile := filepath.Join(gBackupDir, "lock")
	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		panic(fmt.Errorf("failed to open lock file %s: %w", lockFile, err))
	}

	err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		file.Close()

		if errors.Is(err, unix.EWOULDBLOCK) ||
			errors.Is(err, unix.EAGAIN) {
			return nil, false
		}

		panic(fmt.Errorf("error acquiring instance lock %s: %w", lockFile, err))
	}

	return file, true
}
