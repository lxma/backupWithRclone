package main

import (
	"backupWithRclone/executor"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func executionsContainSource(executions []executor.TTestExecution, sourceDir string) bool {
	return slices.ContainsFunc(executions, func(execution executor.TTestExecution) bool {
		return execution.Source == sourceDir
	})
}

func TestMainStraight(t *testing.T) {
	main()
	assert.NotEmpty(t, executor.GTestExecutions)
	assert.True(t, executionsContainSource(executor.GTestExecutions, "testdirs/changed_subpaths"))
}
