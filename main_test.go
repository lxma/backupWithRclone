package main

import (
	"backupWithRclone/executor"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

const cStandardDir = "./testdirs/unchanged_path"

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
func TestMakeRcloneExcludePatterns(t *testing.T) {
	excl := makeRcloneExcludePatterns("./testdirs/exclude")
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

func TestArgsCopyLinks(t *testing.T) {
	assert.Contains(t, getArgs("./testdirs/changed_subpaths"), "--links")
	assert.Contains(t, getArgs("./testdirs/copy_links"), "--copy-links")
}

func TestArgsExtraArgs(t *testing.T) {
	const cCommandLineArg = "Command Line Arg"
	const cDirConfigArg = "--help" // set in config.yaml for dir: ./testdirs/extra_args
	gExtraRcloneArgs = []string{cCommandLineArg}
	args := getArgs("./testdirs/extra_args")
	assert.Contains(t, args, cCommandLineArg)
	assert.Contains(t, args, cDirConfigArg)
}

func TestArgsRunMode(t *testing.T) {
	gRunMode = cRunModeSync
	assert.NotContains(t, getArgs(cStandardDir), "--combined")
	gRunMode = cRunModeCheckSums
	assert.Contains(t, getArgs(cStandardDir), "--combined")
	assert.NotContains(t, getArgs(cStandardDir), "--size-only")
	gRunMode = cRunModeCheckSizes
	assert.Contains(t, getArgs(cStandardDir), "--combined")
	assert.Contains(t, getArgs(cStandardDir), "--size-only")
}
