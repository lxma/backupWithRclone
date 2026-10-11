package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/lxma/gotools/v2"
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

	db := InitDB(filepath.Join(backupDir, "backups.sqlite"))
	defer db.Close()

	backupDates := ReadCutoffTimestamps(db)
	// sort dirs by recent update time ascending

	dirsAndCutoffs := make([]tDirAndCutoff, len(cfg.Dirs))
	for idx, dir := range cfg.Dirs {
		date, ok := backupDates[dir]
		if !ok {
			date = time.Time{}
		}
		dirsAndCutoffs[idx] = tDirAndCutoff{dir, date}
	}
	gotools.SortBy(dirsAndCutoffs, func(a, b tDirAndCutoff) bool {
		return b.cutoff.After(a.cutoff)
	})
	StoreMainDirectoryStates(db, cfg.Dirs)

	for _, dirAndCutoff := range dirsAndCutoffs {
		updateDir(db, dirAndCutoff.dir, dirAndCutoff.cutoff)
	}
}

func updateDir(db *sql.DB, dir string, oldCutoff time.Time) {
	fmt.Printf("Updating %s\n", dir)
	updateDirs, err := findDirsForUpdate(dir, oldCutoff)
	if err != nil {
		fmt.Printf("error finding dirs for update in %s: %v\n", dir, err)
		fmt.Printf("NOT SYNCING %s!\n", dir)
	}
	newCutoff := time.Now().Add(-time.Second)
	StartStoringMainDir(db, dir, updateDirs, newCutoff)
	for _, updateDir := range updateDirs {
		MarkSubDirStarted(db, updateDir)
		fmt.Printf("- %s\n", updateDir)
		MarkSubDirCompleted(db, updateDir)
	}
	MarkMainDirCompleted(db, dir, newCutoff)
}
