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
	StoreMainDirectoryStates([]string{cTestDir})
	MarkUpdateDirStarted(cTestDir, time.Now())
	MarkSubDirStarted(cSubDir1)
	MarkSubDirCompleted(cSubDir1)
	MarkSubDirStarted(cSubDir2)
	MarkSubDirCompleted(cSubDir2)
	cutoffTimestamp := time.Now()
	wrongCutoffTimestamp := time.Now().Add(-time.Hour)
	MarkUpdateDirCompleted(cTestDir, cutoffTimestamp)
	cutoffTimestamps := ReadCutoffTimestamps()
	assert.True(t, gotools.MapHasKey(cutoffTimestamps, cTestDir))
	assert.Equal(t, dateToDbDate(cutoffTimestamp), dateToDbDate(cutoffTimestamps[cTestDir]))
	assert.NotEqual(t, dateToDbDate(wrongCutoffTimestamp), dateToDbDate(cutoffTimestamps[cTestDir]))
}
