//go:build for_testing

package logger

import (
	"fmt"

	"github.com/lxma/gotools/v2"
)

type TMemLog struct {
	Active    bool
	Entries   []string
	IsVerbose bool
}

func (m TMemLog) Verbose() bool {
	return m.IsVerbose
}

var GTestLog *TMemLog

func (m TMemLog) Printf(format string, v ...any) {
	gotools.Assert(m.Active)
	m.Entries = append(m.Entries, fmt.Sprintf(format, v...))
}

func (m TMemLog) Close() {
	m.Active = false
}

var _ = func() struct{} {
	GTestLog = &TMemLog{Active: true, Entries: make([]string, 0)}
	gLoggerSingleton = GTestLog
	return struct{}{}
}()
