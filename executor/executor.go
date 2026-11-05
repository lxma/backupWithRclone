package executor

import (
	"backupWithRclone/logger"
	"os"
	"os/exec"
	"strings"
)

var ExecuteRclone func(string, []string, []string) bool

func init() {
	if ExecuteRclone == nil {
		ExecuteRclone = executeRclone
	}
}

func executeRclone(command string, extraArgs []string, rclonePatterns []string) bool {
	if logger.Verbose() {
		logger.Printf("Running %s %s", command, strings.Join(extraArgs, " "))
	}
	args := append([]string{command}, extraArgs...)
	cmd := exec.Command("rclone", args...)
	cmd.Stdin = strings.NewReader(strings.Join(rclonePatterns, "\n") + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logger.Printf("error running rclone command: %v\n", err)
		return false
	}

	return true
}
