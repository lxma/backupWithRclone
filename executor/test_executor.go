package executor

func executeRcloneForTests(extraArgs []string, rclonePatterns []string) {}

var _ = func() struct{} {
	ExecuteRclone = executeRcloneForTests
	return struct{}{}
}
