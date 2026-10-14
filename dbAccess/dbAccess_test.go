package dbAccess

import (
	"testing"
	"time"

	"github.com/lxma/gotools/v2"
	"github.com/stretchr/testify/assert"
)

//goland:noinspection SqlResolve
func TestInMemoryDB(t *testing.T) {
	db := InitTestDB() //"/Users/d048927/GitHub/backupWithRclone/.backupWithRclone/backups.sqlite")
	var filename string
	err := db.QueryRow(`
        SELECT file
        FROM pragma_database_list
        WHERE name = 'main'
    `).Scan(&filename)

	assert.Nil(t, err, "check SQLite database location: %v", err)
	assert.Empty(t, filename, "expected in-memory SQLite database, got file %q", filename)
}

func TestDBFlowWorks(t *testing.T) {
	db := InitTestDB()
	const cTestDir = "TestDir"
	const cSubDir1 = "subdir1"
	const cSubDir2 = "subdir2"

	// ensure the DB flow works and the cutoff timestamp is recorded appropriately
	StoreMainDirectoryStates(db, []string{cTestDir})
	StartStoringMainDir(db, cTestDir, []string{cSubDir1, cSubDir2}, time.Now())
	MarkSubDirStarted(db, cSubDir1)
	MarkSubDirCompleted(db, cSubDir1)
	MarkSubDirStarted(db, cSubDir2)
	MarkSubDirCompleted(db, cSubDir2)
	cutoffTimestamp := time.Now()
	wrongCutoffTimestamp := time.Now().Add(-time.Hour)
	MarkMainDirCompleted(db, cTestDir, cutoffTimestamp)
	cutoffTimestamps := ReadCutoffTimestamps(db)
	assert.True(t, gotools.MapHasKey(cutoffTimestamps, cTestDir))
	assert.Equal(t, DateToDbDate(cutoffTimestamp), DateToDbDate(cutoffTimestamps[cTestDir]))
	assert.NotEqual(t, DateToDbDate(wrongCutoffTimestamp), DateToDbDate(cutoffTimestamps[cTestDir]))
}
