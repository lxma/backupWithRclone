package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var gLogFile *os.File
var GLogger *log.Logger

func InitLog() {
	file, err := os.OpenFile(filepath.Join(gBackupDir, "exec.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Errorf("Error accessing log file: %v", err))
	}
	GLogger = log.New(file, "", log.LstdFlags)
}

func CloseLog() {
	gLogFile.Close()
}
