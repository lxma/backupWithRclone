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
