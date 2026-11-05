package logger

import (
	"backupWithRclone/config"
	"backupWithRclone/tools"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type tLogger interface {
	Verbose() bool
	Printf(format string, v ...any)
	Close()
}

type tFileLog struct {
	verbose bool
	logger  *log.Logger
}

func (l *tFileLog) Close() {
}

func (l *tFileLog) Printf(format string, v ...any) {
	l.logger.Printf(format, v...)
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Printf(format, v...)
}

func (l *tFileLog) Verbose() bool {
	return l.verbose
}

var gLoggerSingleton tLogger

func Verbose() bool {
	return gLoggerSingleton.Verbose()
}

// GetLog initializes the logger
func InitLog(verbose bool) {
	if gLoggerSingleton != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(config.GBackupDir, "exec.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Errorf("Error accessing logger file: %v", err))
	}
	tools.AddDeferredCall("Close log file", func() {
		file.Close()
	})
	logger := log.New(file, "", log.LstdFlags)
	fileLog := &tFileLog{
		verbose: verbose,
		logger:  logger,
	}
	gLoggerSingleton = fileLog
}

func Printf(format string, v ...any) {
	gLoggerSingleton.Printf(format, v...)
}
