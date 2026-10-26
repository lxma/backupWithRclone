//go:build for_testing

package executor

type TTestExecution struct {
	Source         string
	Target         string
	ExtraArgs      []string
	RclonePatterns []string
}

var GTestExecutions []TTestExecution

func executeRcloneForTests(extraArgs []string, rclonePatterns []string) {
	GTestExecutions = append(GTestExecutions, TTestExecution{
		Source:         extraArgs[0],
		Target:         extraArgs[1],
		ExtraArgs:      extraArgs[2:],
		RclonePatterns: rclonePatterns,
	})
}

var _ = func() struct{} {
	ExecuteRclone = executeRcloneForTests
	return struct{}{}
}()
