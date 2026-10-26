package logger

import (
	"backupWithRclone/config"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type TLog interface {
	Printf(format string, v ...any)
	Close()
}

type tFileLog struct {
	logFile *os.File
	logger  *log.Logger
}

func (l *tFileLog) Close() {
	// CloseLog closes the logfile
	l.logFile.Close()
}

func (l *tFileLog) Printf(format string, v ...any) {
	l.logger.Printf(format, v...)
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Printf(format, v...)
}

var gLog TLog

// InitLog initializes the logger
func InitLog() TLog {
	if gLog != nil {
		return gLog
	}
	file, err := os.OpenFile(filepath.Join(config.GBackupDir, "exec.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Errorf("Error accessing logger file: %v", err))
	}
	logger := log.New(file, "", log.LstdFlags)
	fileLog := &tFileLog{
		logFile: file,
		logger:  logger,
	}
	gLog = fileLog
	return fileLog
}

func Printf(format string, v ...any) {
	gLog.Printf(format, v...)
}
