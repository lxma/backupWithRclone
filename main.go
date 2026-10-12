package main

import (
	"database/sql"
	"errors"
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

func main() {
	LoadConfig()

	//ds, err := findDirsForUpdate(ExpandHome("~/Fotos"), time.Now())
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Printf("found %d directories\n", len(ds))
	//for _, d := range ds {
	//	fmt.Printf("- %s\n", d)
	//}
	//if 1 == 1 {
	//	return
	//}

	lock, ok := AcquireInstanceLock()
	if !ok {
		fmt.Println("A parallel instance is already running, quitting")
		return
	}
	defer lock.Close()

	db := InitDB(filepath.Join(backupDir, "backups.sqlite"))
	defer db.Close()

	backupDates := ReadCutoffTimestamps(db)
	// sort dirs by recent update time ascending

	sourceDirs := gotools.GetKeys(cfg.Dirs)

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
	fmt.Printf("Updating %s\n", dir)
	baseDir := filepath.Clean(ExpandHome(dir))
	updateDirs, err := findDirsForUpdate(baseDir, oldCutoff)
	if err != nil {
		fmt.Printf("error finding dirs for update in %s: %v\n", dir, err)
		fmt.Printf("NOT SYNCING %s!\n", dir)
	}
	newCutoff := time.Now().Add(-time.Second)
	StartStoringMainDir(db, dir, updateDirs, newCutoff)
	for _, updateDir := range updateDirs {
		MarkSubDirStarted(db, updateDir)
		fmt.Printf("- updating %s ...\n", updateDir)
		destination := addPathToRcloneDestination(getTargetPath(baseDir, updateDir, cfg.Dirs[dir].TargetDir))
		RunRclone(updateDir, destination)
		MarkSubDirCompleted(db, updateDir)
	}
	MarkMainDirCompleted(db, dir, newCutoff)
}

func getTargetPath(baseDir, updateDir, targetDir string) string {
	gotools.Assert(strings.HasPrefix(updateDir, baseDir))
	if baseDir == updateDir {
		return filepath.Clean(targetDir)
	}
	return filepath.Join(filepath.Clean(targetDir), updateDir[len(baseDir)+1:])
}

func addPathToRcloneDestination(targetDir string) string {
	if strings.HasSuffix(cfg.RcloneDestinationPrefix, ":") ||
		strings.HasSuffix(cfg.RcloneDestinationPrefix, "/") {
		return cfg.RcloneDestinationPrefix + targetDir
	}
	return filepath.Join(cfg.RcloneDestinationPrefix, targetDir)
}

func RunRclone(sourceDir, destination string) error {
	//fmt.Printf("rclone %s -> %s\n", sourceDir, destination)
	cmd := exec.Command("rclone", "sync", sourceDir, destination, "-P", "--delete-excluded", "--exclude-from", ExpandHome("~/.backupWithRclone/excluded"))

	cmd.Stdin = os.Stdin
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
	lockFile := filepath.Join(backupDir, "lock")
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
