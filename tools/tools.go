package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ExpandHome(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			panic("cannot get home dir")
		}
		return home
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			panic("cannot get home dir")
		}
		return filepath.Join(home, path[2:])
	}

	return path
}

func StopWithErrorMessage(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
