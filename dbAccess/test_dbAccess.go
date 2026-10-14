//go:build for_testing

package dbAccess

import "database/sql"

const cMemoryDb = ":memory:"

func InitTestDB() *sql.DB {
	var err error
	gDB, err = sql.Open("sqlite", cMemoryDb)
	if err != nil {
		panic(err)
	}
	setupInitialDbStructure(gDB)
	return gDB
}
