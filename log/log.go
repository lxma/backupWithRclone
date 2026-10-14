package log

import (
	"backupWithRclone/config"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var gLogFile *os.File
var gLogger *log.Logger

// InitLog initializes the logger
func InitLog() {
	file, err := os.OpenFile(filepath.Join(config.GBackupDir, "exec.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Errorf("Error accessing log file: %v", err))
	}
	gLogger = log.New(file, "", log.LstdFlags)
}

// Printf prints to Log and to stdout
func Printf(format string, v ...interface{}) {
	gLogger.Printf(format, v...)
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Printf(format, v...)
}

// CloseLog closes the logfile
func CloseLog() {
	gLogFile.Close()
}
