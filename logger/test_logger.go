//go:build for_testing

package logger

import (
	"fmt"

	"github.com/lxma/gotools/v2"
)

type TMemLog struct {
	active  bool
	entries []string
}

var GTestLog *TMemLog

func (m TMemLog) Printf(format string, v ...any) {
	gotools.Assert(m.active)
	m.entries = append(m.entries, fmt.Sprintf(format, v...))
}

func (m TMemLog) Close() {
	m.active = false
}

var _ = func() struct{} {
	GTestLog = &TMemLog{active: true, entries: make([]string, 0)}
	gLog = GTestLog
	return struct{}{}
}()
