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

// The configuration entries are taken from testBackupDir/config.yaml
func TestFlow(t *testing.T) {
	excl := makeRcloneExcludePatterns(
		[]string{"ex*dir1", "ex*dir2"},
		[]string{"ex*file1", "ex*file2"},
		[]string{"excludesubdir"},
		[]string{"excludesubfile"})
	slices.Sort(excl)
	assert.Equal(t, []string{
		"ex*dir1/",
		"ex*dir2/",
		"ex*file1",
		"ex*file2",
		"excludesubdir/",
		"excludesubfile"},
		excl)
}
