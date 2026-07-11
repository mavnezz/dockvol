package testutil

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"dockvol-backend/internal/schema"
	"dockvol-backend/internal/storage"
)

// SetupDb points the global store at a fresh, migrated SQLite database for the
// lifetime of the test, so every repository runs against isolated data. Tests
// must not run in parallel: they share the one global database handle.
func SetupDb(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)"

	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, schema.AutoMigrate(database))

	storage.SetDb(database)

	t.Cleanup(func() {
		if sqlDB, err := database.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	return database
}
