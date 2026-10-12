package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArePathsRelated(t *testing.T) {
	assert.True(t, arePathsRelated(".", "."))
	assert.True(t, arePathsRelated("a", "a"))
	assert.True(t, arePathsRelated("a/b", "a"))
	assert.True(t, arePathsRelated("a", "a/b"))
	assert.True(t, arePathsRelated("a", "a/b/c"))
	assert.False(t, arePathsRelated("a", "b"))
	assert.False(t, arePathsRelated("a", "b/a"))
}
