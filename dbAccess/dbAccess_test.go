package dbAccess

import (
	"testing"
	"time"

	"github.com/lxma/gotools/v2"
	"github.com/stretchr/testify/assert"
)

//goland:noinspection SqlResolve
func TestInMemoryDB(t *testing.T) {
	var filename string
	err := gDB.QueryRow(`
        SELECT file
        FROM pragma_database_list
        WHERE name = 'main'
    `).Scan(&filename)

	assert.Nil(t, err, "check SQLite database location: %v", err)
	assert.Empty(t, filename, "expected in-memory SQLite database, got file %q", filename)
}

func TestDBFlowWorks(t *testing.T) {
	const cTestDir = "TestDir"
	const cSubDir1 = "subdir1"
	const cSubDir2 = "subdir2"

	// ensure the DB flow works and the cutoff timestamp is recorded appropriately
	StoreMainDirectoryStates(gDB, []string{cTestDir})
	MarkUpdateDirStarted(gDB, cTestDir, time.Now())
	MarkSubDirStarted(gDB, cSubDir1)
	MarkSubDirCompleted(gDB, cSubDir1)
	MarkSubDirStarted(gDB, cSubDir2)
	MarkSubDirCompleted(gDB, cSubDir2)
	cutoffTimestamp := time.Now()
	wrongCutoffTimestamp := time.Now().Add(-time.Hour)
	MarkUpdateDirCompleted(gDB, cTestDir, cutoffTimestamp)
	cutoffTimestamps := ReadCutoffTimestamps(gDB)
	assert.True(t, gotools.MapHasKey(cutoffTimestamps, cTestDir))
	assert.Equal(t, DateToDbDate(cutoffTimestamp), DateToDbDate(cutoffTimestamps[cTestDir]))
	assert.NotEqual(t, DateToDbDate(wrongCutoffTimestamp), DateToDbDate(cutoffTimestamps[cTestDir]))
}
