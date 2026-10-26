package findDirs

import (
	"backupWithRclone/exclusions"
	"strings"
	"testing"
	"time"

	"github.com/lxma/gotools/v2"
	"github.com/stretchr/testify/assert"
)

func TestFindDirsForUpdate(t *testing.T) {
	keyDate := time.Date(2024, time.February, 22, 12, 0, 0, 0, time.UTC)
	dirs, err := FindDirsForUpdate("../testdirs", keyDate, exclusions.TExclusions{
		ExcludeFiles:   nil,
		ExcludeDirs:    nil,
		RclonePatterns: nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEmpty(t, dirs, "At least one directory should be found")
	dirsNamedUnchanged := gotools.Filter(dirs, func(s string) bool {
		return strings.Contains(s, "unchanged")
	})
	assert.Empty(t, dirsNamedUnchanged, "No returned directory should be named 'unchanged'")
}
