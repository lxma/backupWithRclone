package dbAccess

import (
	"backupWithRclone/config"
	"backupWithRclone/logger"
	"backupWithRclone/tools"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var MinDate = time.Time{} // // 0001-01-01T00:00:00Z
var gDB *sql.DB

func dateToDbDate(date time.Time) string {
	return date.UTC().Format(time.RFC3339)
}

func dbDateToDate(dbDate string) (time.Time, error) {
	return time.Parse(time.RFC3339, dbDate)
}

func InitDB() {
	filename := filepath.Join(config.GBackupDir, "backups.sqlite")
	if gDB != nil {
		// DB is already initialized, e.g. for test cases
		return
	}
	var err error
	gDB, err = sql.Open("sqlite", filename)
	if err != nil {
		logger.Printf("error opening database file %s: %v\n", filename, err)
		panic(fmt.Errorf("error opening DB file %s: %v", filename, err))
	}
	tools.AddDeferredCall("Close DB", func() {
		gDB.Close()
	})
	setupInitialDbStructure()
}

func setupInitialDbStructure() {
	gDB.SetMaxOpenConns(1)
	gDB.SetMaxIdleConns(1)
	if _, err := gDB.Exec(`CREATE TABLE IF NOT EXISTS backup_states (
		directory              TEXT PRIMARY KEY NOT NULL,
		cutoff_timestamp       TEXT NOT NULL,
		completed_at_timestamp TEXT NOT NULL
	);`); err != nil {
		logger.Printf("error creating backup_states table: %v\n", err)
		panic(fmt.Errorf("error creating backup_states: %v", err))
	}

	if _, err := gDB.Exec(`CREATE TABLE IF NOT EXISTS syncing_main_dirs (
		directory        TEXT PRIMARY KEY NOT NULL,
		started          INTEGER NOT NULL DEFAULT 0 CHECK (started IN (0, 1)),
	completed        INTEGER NOT NULL DEFAULT 0 CHECK (completed IN (0, 1)),
	cutoff_timestamp TEXT
	);`); err != nil {
		logger.Printf("error creating syncing_main_dirs table: %v\n", err)
		panic(fmt.Errorf("error creating syncing_main_dirs: %v", err))
	}

	if _, err := gDB.Exec(`CREATE TABLE IF NOT EXISTS syncing_subdirs (
		directory TEXT PRIMARY KEY NOT NULL,
		started   INTEGER NOT NULL DEFAULT 0 CHECK (started IN (0, 1)),
	completed INTEGER NOT NULL DEFAULT 0 CHECK (completed IN (0, 1))
);`); err != nil {
		logger.Printf("error creating syncing_subdirs table: %v\n", err)
		panic(fmt.Errorf("error creating syncing_subdirs: %v", err))
	}
}

func ReadCutoffTimestamps() map[string]time.Time {
	rows, err := gDB.Query(`SELECT directory, cutoff_timestamp FROM backup_states`)
	if err != nil {
		logger.Printf("error reading cutoff timestamps: %v\n", err)
		panic(fmt.Errorf("query backup states: %v", err))
	}
	defer rows.Close()

	cutoffs := make(map[string]time.Time)

	for rows.Next() {
		var path, timestamp string

		if err := rows.Scan(&path, &timestamp); err != nil {
			logger.Printf("error reading cutoff timestamps: %v\n", err)
			panic(fmt.Errorf("error reading backup state: %v", err))
		}

		cutoff, err := dbDateToDate(timestamp)
		if err != nil {
			println("invalid cutoff timestamp %s for path %s", timestamp, path)
			continue
		}

		cutoffs[path] = cutoff
	}

	return cutoffs
}

func beginTransaction() *sql.Tx {
	tx, err := gDB.Begin()
	if err != nil {
		logger.Printf("error beginning transaction: %v", err)
		panic(fmt.Errorf("error beginning transaction: %v", err))
	}
	return tx
}

func StoreMainDirectoryStates(mainDirs []string) {
	tx := beginTransaction()
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM syncing_main_dirs`); err != nil {
		logger.Printf("error deleting syncing_main_dirs: %v", err)
		panic(fmt.Errorf("error clearing syncinc_dirs: %v", err))
	}

	if _, err := tx.Exec(`DELETE FROM syncing_subdirs`); err != nil {
		logger.Printf("error deleting syncing_subdirs: %v", err)
		panic(fmt.Errorf("error clearing handling_dir: %v", err))
	}

	stmt, err := tx.Prepare(`INSERT INTO syncing_main_dirs
	 (directory, started, completed, cutoff_timestamp)
	 VALUES (?, 0, 0, ?)`)
	if err != nil {
		logger.Printf("error preparing statement: %v", err)
		panic(fmt.Errorf("error preparing handling_dir insert: %v", err))
	}
	defer stmt.Close()

	for _, mainDir := range mainDirs {
		if _, err := stmt.Exec(mainDir, dateToDbDate(MinDate)); err != nil {
			logger.Printf("error executing statement: %v", err)
			panic(fmt.Errorf("error inserting directory %q: %v", mainDir, err))
		}
	}
	err = tx.Commit()
	if err != nil {
		logger.Printf("error committing transaction: %v", err)
		panic(fmt.Errorf("commit handling_dir transaction: %v", err))
	}
}

func MarkUpdateDirStarted(syncDir string, cutoff time.Time) {
	if _, err := gDB.Exec(`UPDATE syncing_main_dirs SET started = 1, cutoff_timestamp = ? WHERE directory = ?`, dateToDbDate(cutoff), syncDir); err != nil {
		logger.Printf("error updating syncing main_dirs: %v", err)
		panic(fmt.Errorf("error updating syncinc_dirs for %s: %v", syncDir, err))
	}
}

func MarkUpdateDirCompleted(mainDir string, newCutoff time.Time) {
	if _, err := gDB.Exec(`UPDATE syncing_main_dirs SET completed = 1 WHERE directory = ?`, mainDir); err != nil {
		logger.Printf("error updating syncing main_dirs: %v", err)
		panic(fmt.Errorf("error updating syncinc_dirs for %s: %v", mainDir, err))
	}
	if _, err := gDB.Exec(`INSERT INTO backup_states
		(directory, cutoff_timestamp, completed_at_timestamp)
		VALUES (?, ?, ?)
		ON CONFLICT(directory) DO UPDATE
		    SET cutoff_timestamp = excluded.cutoff_timestamp, completed_at_timestamp = excluded.completed_at_timestamp`,
		mainDir, dateToDbDate(newCutoff), dateToDbDate(time.Now())); err != nil {
		logger.Printf("error updating backup_states: %v", err)
		panic(fmt.Errorf("error updating backup_states for %s: %v", mainDir, err))
	}
}

func MarkSubDirStarted(syncDir string) {
	if _, err := gDB.Exec(`UPDATE syncing_subdirs SET started = 1 WHERE directory = ?`, syncDir); err != nil {
		logger.Printf("error updating syncing subdirs: %v", err)
		panic(fmt.Errorf("error updating syncing subdirs for %s: %v", syncDir, err))
	}
}

func MarkSubDirCompleted(syncDir string) {
	if _, err := gDB.Exec(`UPDATE syncing_subdirs SET completed = 1 WHERE directory = ?`, syncDir); err != nil {
		logger.Printf("error updating syncing subdirs: %v", err)
		panic(fmt.Errorf("error updating syncing subdirs for %s: %v", syncDir, err))
	}
}
