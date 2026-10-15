//go:build for_testing

package config

import (
	"os"
	"path/filepath"
)

const cTestBackupDir = "./testBackupDir"

var gInitCalled = false

func findTestBackupDir() string {
	for _, path := range []string{cTestBackupDir, filepath.Join("..", cTestBackupDir)} {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}

	panic("directory x not found in current or parent directory")
}

func InitTestConfig() {
	if gInitCalled {
		return
	}
	gInitCalled = true
	GBackupDir = findTestBackupDir()
	InitConfig()
}

// Init Test Config
var _ = func() struct{} {
	InitTestConfig()
	return struct{}{}
}()
