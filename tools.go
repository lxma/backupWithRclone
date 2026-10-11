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
			panic(fmt.Errorf("error getiting home directory: %w", err))
		}
		return home
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(fmt.Errorf("error getiting home directory: %w", err))
		}
		return filepath.Join(home, path[2:])
	}

	return path
}
