package executor

import (
	"backupWithRclone/log"
	"os"
	"os/exec"
	"strings"
)

var ExecuteRclone func(extraArgs []string, rclonePatterns []string)

func init() {
	ExecuteRclone = executeRclone
}

func executeRclone(extraArgs []string, rclonePatterns []string) {
	args := append([]string{"sync"}, extraArgs...)
	cmd := exec.Command("rclone", args...)
	cmd.Stdin = strings.NewReader(strings.Join(rclonePatterns, "\n") + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("error running rclone command: %v\n", err)
	}
}
