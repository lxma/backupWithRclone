//go:build for_testing

package dbAccess

import (
	"backupWithRclone/config"
	"database/sql"
)

const cMemoryDb = ":memory:"

var gInitCalled = false

func InitTestDB() {
	if gInitCalled {
		return
	}
	gInitCalled = true
	config.InitTestConfig()

	var err error
	gDB, err = sql.Open("sqlite", cMemoryDb)
	if err != nil {
		panic(err)
	}
	setupInitialDbStructure(gDB)
}

// Init Test DB
var _ = func() struct{} {
	InitTestDB()
	return struct{}{}
}()
