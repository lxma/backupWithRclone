package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandHome(t *testing.T) {
	// prerequisite: we need a home directory
	home, err := os.UserHomeDir()
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(home, "test"), ExpandHome("~/test"))

	assert.Equal(t, home, ExpandHome("~/"))
	assert.Equal(t, home, ExpandHome("~"))
}

func TestSanitizeFilename(t *testing.T) {
	assert.Equal(t, EscapeFilenamePart("a"), "a")
	assert.Equal(t, EscapeFilenamePart(""), "_")
	assert.Equal(t, EscapeFilenamePart("."), "_")
	assert.Equal(t, EscapeFilenamePart(".."), "_")
	assert.Equal(t, EscapeFilenamePart("abc/def:ghi/"), "abc_def_ghi_")
}

func TestDeferredCalls(t *testing.T) {
	x := 1
	AddDeferredCall("test", func() {
		x = 2
	})
	RunDeferredCalls(false)
	assert.Equal(t, 2, x)
	assert.Zero(t, gDeferredCalls.Len(), "Deferred calls are only called once")
}
