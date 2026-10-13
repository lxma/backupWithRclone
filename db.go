package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var MinDate = time.Time{} // // 0001-01-01T00:00:00Z

func dateToDbDate(date time.Time) string {
	return date.UTC().Format(time.RFC3339)
}

func dbDateToDate(dbDate string) (time.Time, error) {
	return time.Parse(time.RFC3339, dbDate)
}

func InitDB(filename string) *sql.DB {
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		GLogger.Printf("error opening database file %s: %v\n", filename, err)
		panic(fmt.Errorf("error opening DB file %s: %v", filename, err))
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS backup_states (
		directory              TEXT PRIMARY KEY NOT NULL,
		cutoff_timestamp       TEXT NOT NULL,
		completed_at_timestamp TEXT NOT NULL
	);`); err != nil {
		GLogger.Printf("error creating backup_states table: %v\n", err)
		panic(fmt.Errorf("error creating backup_states: %v", err))
	}

	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS syncing_main_dirs (
		directory        TEXT PRIMARY KEY NOT NULL,
		started          INTEGER NOT NULL DEFAULT 0 CHECK (started IN (0, 1)),
	completed        INTEGER NOT NULL DEFAULT 0 CHECK (completed IN (0, 1)),
	cutoff_timestamp TEXT
	);`); err != nil {
		GLogger.Printf("error creating syncing_main_dirs table: %v\n", err)
		panic(fmt.Errorf("error creating syncing_main_dirs: %v", err))
	}

	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS syncing_subdirs (
		directory TEXT PRIMARY KEY NOT NULL,
		started   INTEGER NOT NULL DEFAULT 0 CHECK (started IN (0, 1)),
	completed INTEGER NOT NULL DEFAULT 0 CHECK (completed IN (0, 1))
);`); err != nil {
		GLogger.Printf("error creating syncing_subdirs table: %v\n", err)
		panic(fmt.Errorf("error creating syncing_subdirs: %v", err))
	}

	return db
}

func ReadCutoffTimestamps(db *sql.DB) map[string]time.Time {
	rows, err := db.Query(`SELECT directory, cutoff_timestamp FROM backup_states`)
	if err != nil {
		GLogger.Printf("error reading cutoff timestamps: %v\n", err)
		panic(fmt.Errorf("query backup states: %v", err))
	}
	defer rows.Close()

	cutoffs := make(map[string]time.Time)

	for rows.Next() {
		var path, timestamp string

		if err := rows.Scan(&path, &timestamp); err != nil {
			GLogger.Printf("error reading cutoff timestamps: %v\n", err)
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

func UpdateBackupState(
	db *sql.DB,
	path string,
	cutoff time.Time,
	completedAt time.Time,
) error {
	if _, err := db.Exec(`
INSERT INTO backup_states (
    path,
    cutoff_timestamp,
    completed_at_timestamp
)
VALUES (?, ?, ?)
ON CONFLICT(path) DO UPDATE SET
    cutoff_timestamp       = excluded.cutoff_timestamp,
    completed_at_timestamp = excluded.completed_at_timestamp;
`, path, dateToDbDate(cutoff), dateToDbDate(completedAt)); err != nil {
		GLogger.Printf("error updating backup state: %v", err)
		return fmt.Errorf("upsert backup state for %q: %v", path, err)
	}

	return nil
}

func beginTransaction(db *sql.DB) *sql.Tx {
	tx, err := db.Begin()
	if err != nil {
		GLogger.Printf("error beginning transaction: %v", err)
		panic(fmt.Errorf("error beginning transaction: %v", err))
	}
	return tx
}

func StoreMainDirectoryStates(db *sql.DB, mainDirs []string) {
	tx := beginTransaction(db)
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM syncing_main_dirs`); err != nil {
		GLogger.Printf("error deleting syncing_main_dirs: %v", err)
		panic(fmt.Errorf("error clearing syncinc_dirs: %v", err))
	}

	if _, err := tx.Exec(`DELETE FROM syncing_subdirs`); err != nil {
		GLogger.Printf("error deleting syncing_subdirs: %v", err)
		panic(fmt.Errorf("error clearing handling_dir: %v", err))
	}

	stmt, err := tx.Prepare(`INSERT INTO syncing_main_dirs
	 (directory, started, completed, cutoff_timestamp)
	 VALUES (?, 0, 0, ?)`)
	if err != nil {
		GLogger.Printf("error preparing statement: %v", err)
		panic(fmt.Errorf("error preparing handling_dir insert: %v", err))
	}
	defer stmt.Close()

	for _, mainDir := range mainDirs {
		if _, err := stmt.Exec(mainDir, dateToDbDate(MinDate)); err != nil {
			GLogger.Printf("error executing statement: %v", err)
			panic(fmt.Errorf("error inserting directory %q: %v", mainDir, err))
		}
	}
	err = tx.Commit()
	if err != nil {
		GLogger.Printf("error committing transaction: %v", err)
		panic(fmt.Errorf("commit handling_dir transaction: %v", err))
	}
}

func StartStoringMainDir(db *sql.DB, mainDir string, subDirs []string, cutoff time.Time) {
	if len(subDirs) == 0 {
		return
	}

	tx := beginTransaction(db)
	defer tx.Rollback()

	MarkMainDirStarted(tx, mainDir, cutoff)

	if _, err := tx.Exec(`DELETE FROM syncing_subdirs`); err != nil {
		panic(fmt.Errorf("error clearing handling_dir: %v", err))
	}
	stmt, err := tx.Prepare(`INSERT INTO syncing_subdirs (directory,completed,started) VALUES (?,0,0)`)
	if err != nil {
		panic(fmt.Errorf("error preparing handling_dir insert: %v", err))
	}
	defer stmt.Close()

	for _, subDir := range subDirs {
		if _, err := stmt.Exec(subDir); err != nil {
			GLogger.Printf("error inserting directory %s: %v", subDir, err)
			panic(fmt.Errorf("insert directory %s: %v", subDir, err))
		}
	}

	err = tx.Commit()
	if err != nil {
		GLogger.Printf("error committing transaction: %v", err)
		panic(fmt.Errorf("commit handling_dir transaction: %v", err))
	}
}

func MarkMainDirStarted(tx *sql.Tx, syncDir string, cutoff time.Time) {
	if _, err := tx.Exec(`UPDATE syncing_main_dirs SET started = 1, cutoff_timestamp = ? WHERE directory = ?`, dateToDbDate(cutoff), syncDir); err != nil {
		GLogger.Printf("error updating syncing main_dirs: %v", err)
		panic(fmt.Errorf("error updating syncinc_dirs for %s: %v", syncDir, err))
	}
}

func MarkMainDirCompleted(db *sql.DB, mainDir string, newCutoff time.Time) {
	if _, err := db.Exec(`UPDATE syncing_main_dirs SET completed = 1 WHERE directory = ?`, mainDir); err != nil {
		GLogger.Printf("error updating syncing main_dirs: %v", err)
		panic(fmt.Errorf("error updating syncinc_dirs for %s: %v", mainDir, err))
	}
	if _, err := db.Exec(`INSERT INTO backup_states
		(directory, cutoff_timestamp, completed_at_timestamp)
		VALUES (?, ?, ?)
		ON CONFLICT(directory) DO UPDATE
		    SET cutoff_timestamp = excluded.cutoff_timestamp, completed_at_timestamp = excluded.completed_at_timestamp`,
		mainDir, dateToDbDate(newCutoff), dateToDbDate(time.Now())); err != nil {
		GLogger.Printf("error updating backup_states: %v", err)
		panic(fmt.Errorf("error updating backup_states for %s: %v", mainDir, err))
	}
}

func MarkSubDirStarted(db *sql.DB, syncDir string) {
	if _, err := db.Exec(`UPDATE syncing_subdirs SET started = 1 WHERE directory = ?`, syncDir); err != nil {
		GLogger.Printf("error updating syncing subdirs: %v", err)
		panic(fmt.Errorf("error updating syncing subdirs for %s: %v", syncDir, err))
	}
}

func MarkSubDirCompleted(db *sql.DB, syncDir string) {
	if _, err := db.Exec(`UPDATE syncing_subdirs SET completed = 1 WHERE directory = ?`, syncDir); err != nil {
		GLogger.Printf("error updating syncing subdirs: %v", err)
		panic(fmt.Errorf("error updating syncing subdirs for %s: %v", syncDir, err))
	}
}
