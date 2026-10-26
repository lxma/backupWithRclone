//go:build for_testing

package exclusions

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The configuration entries are taken from testBackupDir/config.yaml
func TestFlow(t *testing.T) {
	excl := MakeExclusions("./testdirs/exclude")
	assert.True(t, excl.IsEntryExcluded("excludeddir1", true))
	assert.True(t, excl.IsEntryExcluded("excludeddir2", true))
	assert.True(t, excl.IsEntryExcluded("excludedfile1", false))
	assert.True(t, excl.IsEntryExcluded("excludedfile2", false))
	assert.True(t, excl.IsEntryExcluded("excludesubdir", true))
	assert.True(t, excl.IsEntryExcluded("excludesubfile", false))

	assert.False(t, excl.IsEntryExcluded("not_excluded", false))
	assert.False(t, excl.IsEntryExcluded("not_excluded", true))
	assert.False(t, excl.IsEntryExcluded("excludeddir1", false))
	assert.False(t, excl.IsEntryExcluded("excludedfile1", true))

	slices.Sort(excl.RclonePatterns)
	assert.Equal(t, []string{
		"ex*dir1/",
		"ex*dir2/",
		"ex*file1",
		"ex*file2",
		"excludesubdir/",
		"excludesubfile"},
		excl.RclonePatterns)
}
