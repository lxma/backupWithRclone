package main

import (
	"backupWithRclone/executor"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMainStraight(t *testing.T) {
	main()
	assert.NotEmpty(t, executor.GTestExecutions)
}
