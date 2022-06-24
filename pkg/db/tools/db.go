package tools

import (
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Get database handler
var load_issue_db_once sync.Once
var db *gorm.DB

func GetDBConnection() *gorm.DB {
	load_issue_db_once.Do(func() {
		var err error
		if db, err = gorm.Open(sqlite.Open("db.sqlite"), &gorm.Config{}); err != nil {
			panic(err)
		}
	})
	return db
}
