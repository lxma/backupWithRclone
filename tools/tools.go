package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/lxma/golist/v2"
)

type TStandaloneFunction func()

type tDeferredCall struct {
	name     string
	function TStandaloneFunction
}

var gDeferredCalls = golist.New[tDeferredCall]()

// ExpandHome returns a version of the input path in which a
// prefix ~ is replaced with the home directory
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

// EscapeFilenamePart returns a sanitized version of that string. It
// replaces all characters that are obscure for single file names and
// replaces them with underscore. Also, it prevents empty filenames
// or filenames that conflict with the standard directory names "." and
// "..". Those are replaced with a single underscore.
func EscapeFilenamePart(filename string) string {
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

// AddDeferredCall registers a function as a deferred call. The main function must
// call RunDeferredCalls to finally run all those registered calls.
func AddDeferredCall(name string, callFunction TStandaloneFunction) {
	gDeferredCalls.PushBack(tDeferredCall{name, callFunction})
}

// RunDeferredCalls runs all functions previously registered with AddDeferredCall().
func RunDeferredCalls(verbose bool) {
	if verbose {
		fmt.Printf("Running %d deferred calls:\n", gDeferredCalls.Len())
	}
	for gDeferredCalls.Len() > 0 {
		deferredCall := gDeferredCalls.PopFront()
		if verbose {
			fmt.Printf(" - %s\n", deferredCall.name)
		}
		deferredCall.function()
	}
}
