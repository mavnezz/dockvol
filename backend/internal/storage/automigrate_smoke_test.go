package storage_test

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"dockvol-backend/internal/schema"
)

func openSmokeTestDb(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "smoke.db")
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)"

	smokeDb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	return smokeDb
}

func Test_AutoMigrateAllModels_OnSqlite_Succeeds(t *testing.T) {
	smokeDb := openSmokeTestDb(t)

	require.NoError(t, schema.AutoMigrate(smokeDb))

	for _, tableName := range []string{"users", "workspaces", "storages", "notifiers", "volume_backups", "volume_backup_configs"} {
		assert.Truef(
			t,
			smokeDb.Migrator().HasTable(tableName),
			"table %q missing after AutoMigrate",
			tableName,
		)
	}
}
