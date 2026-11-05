//go:build for_testing

package dbAccess

import (
	"backupWithRclone/tools"
	"database/sql"
)

const cMemoryDb = ":memory:"

func InitTestDB() {
	var err error
	gDB, err = sql.Open("sqlite", cMemoryDb)
	if err != nil {
		panic(err)
	}
	setupInitialDbStructure()
}

// Init Test DB
var _ = func() struct{} {
	InitTestDB()
	tools.AddTestSetupFunc(InitTestDB)
	return struct{}{}
}()
