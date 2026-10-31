//go:build for_testing

package executor

type TTestExecution struct {
	Command        string
	Source         string
	Target         string
	ExtraArgs      []string
	RclonePatterns []string
}

var GTestExecutions []TTestExecution
var GTestExecutionRunsOk = true

func executeRcloneForTests(command string, extraArgs []string, rclonePatterns []string) bool {
	GTestExecutions = append(GTestExecutions, TTestExecution{
		Command:        command,
		Source:         extraArgs[0],
		Target:         extraArgs[1],
		ExtraArgs:      extraArgs[2:],
		RclonePatterns: rclonePatterns,
	})
	return GTestExecutionRunsOk
}

var _ = func() struct{} {
	ExecuteRclone = executeRcloneForTests
	return struct{}{}
}()
