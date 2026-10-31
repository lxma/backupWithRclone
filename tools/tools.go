package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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

func SanitizeFilename(filename string) string {
	filename = strings.Map(func(r rune) rune {
		switch {
		case r == '/', r == ':', unicode.IsControl(r):
			return '_'
		default:
			return r
		}
	}, filename)

	// Special Cases to avoid strange filenames
	switch filename {
	case "", ".", "..":
		return "_"
	default:
		return filename
	}
}
