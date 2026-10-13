package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var gLogFile *File
func
func CreateLog() *log.Logger {
	file, err := os.OpenFile(filepath.Join(gBackupDir, "exec.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Errorf("Error accessing log file: %w", err))
	}
	return log.New(file, "", log.LstdFlags)
}
