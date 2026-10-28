package executor

import (
	"backupWithRclone/logger"
	"os"
	"os/exec"
	"strings"
)

var ExecuteRclone func(extraArgs []string, rclonePatterns []string)

func init() {
	if ExecuteRclone == nil {
		ExecuteRclone = executeRclone
	}
}

func executeRclone(extraArgs []string, rclonePatterns []string) {
	args := append([]string{"sync"}, extraArgs...)
	cmd := exec.Command("rclone", args...)
	cmd.Stdin = strings.NewReader(strings.Join(rclonePatterns, "\n") + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logger.Printf("error running rclone command: %v\n", err)
	}
}
