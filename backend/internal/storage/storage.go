package storage

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"dockvol-backend/internal/config"
	"dockvol-backend/internal/util/logger"
)

var log = logger.GetLogger()

var db *gorm.DB

var initDb = sync.OnceFunc(loadDbs)

func GetDb() *gorm.DB {
	if db == nil {
		initDb()
	}

	return db
}

// SetDb lets tests swap in an isolated SQLite database so repositories don't
// touch the on-disk store.
func SetDb(database *gorm.DB) {
	db = database
}

func loadDbs() {
	LoadMainDb()
}

func LoadMainDb() {
	dbPath := config.GetEnv().DatabasePath

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Error("error creating database directory", "error", err)
		os.Exit(1)
	}

	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)"

	log.Info("Connecting to database...")

	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		log.Error("error on connecting to database", "error", err)
		os.Exit(1)
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.Error("error getting underlying sql.DB", "error", err)
		os.Exit(1)
	}

	// SQLite is a single-writer engine; capping to one connection avoids
	// "database is locked" under concurrent writes.
	sqlDB.SetMaxOpenConns(1)

	db = database

	log.Info("Main database connected successfully!")
}
