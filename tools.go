package main

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
			GLogger.Printf("error getting home directory: %v\n", err)
			panic(fmt.Errorf("error getting home directory: %v", err))
		}
		return home
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			GLogger.Printf("error getting home directory: %v\n", err)
			panic(fmt.Errorf("error getting home directory: %v", err))
		}
		return filepath.Join(home, path[2:])
	}

	return path
}
