package schema

import (
	"gorm.io/gorm"

	"dockvol-backend/internal/features/audit_logs"
	"dockvol-backend/internal/features/docker"
	"dockvol-backend/internal/features/notifiers"
	discord_notifier "dockvol-backend/internal/features/notifiers/models/discord"
	email_notifier "dockvol-backend/internal/features/notifiers/models/email_notifier"
	slack_notifier "dockvol-backend/internal/features/notifiers/models/slack"
	teams_notifier "dockvol-backend/internal/features/notifiers/models/teams"
	telegram_notifier "dockvol-backend/internal/features/notifiers/models/telegram"
	webhook_notifier "dockvol-backend/internal/features/notifiers/models/webhook"
	"dockvol-backend/internal/features/storages"
	azure_blob_storage "dockvol-backend/internal/features/storages/models/azure_blob"
	ftp_storage "dockvol-backend/internal/features/storages/models/ftp"
	local_storage "dockvol-backend/internal/features/storages/models/local"
	nas_storage "dockvol-backend/internal/features/storages/models/nas"
	rclone_storage "dockvol-backend/internal/features/storages/models/rclone"
	s3_storage "dockvol-backend/internal/features/storages/models/s3"
	sftp_storage "dockvol-backend/internal/features/storages/models/sftp"
	users_models "dockvol-backend/internal/features/users/models"
	workspaces_models "dockvol-backend/internal/features/workspaces/models"
)

// Models is the full set of persisted entities, in dependency order. It is the
// single source of truth for the schema — production startup, the AutoMigrate
// smoke test and the test harness all migrate from this list.
func Models() []any {
	return []any{
		&users_models.User{},
		&users_models.UsersSettings{},
		&users_models.SecretKey{},
		&users_models.PasswordResetCode{},
		&workspaces_models.Workspace{},
		&workspaces_models.WorkspaceMembership{},
		&audit_logs.AuditLog{},

		&storages.Storage{},
		&local_storage.LocalStorage{},
		&s3_storage.S3Storage{},
		&nas_storage.NASStorage{},
		&azure_blob_storage.AzureBlobStorage{},
		&ftp_storage.FTPStorage{},
		&sftp_storage.SFTPStorage{},
		&rclone_storage.RcloneStorage{},

		&notifiers.Notifier{},
		&telegram_notifier.TelegramNotifier{},
		&email_notifier.EmailNotifier{},
		&webhook_notifier.WebhookNotifier{},
		&slack_notifier.SlackNotifier{},
		&discord_notifier.DiscordNotifier{},
		&teams_notifier.TeamsNotifier{},

		&docker.VolumeBackup{},
		&docker.VolumeBackupConfig{},
	}
}

func AutoMigrate(database *gorm.DB) error {
	return database.AutoMigrate(Models()...)
}
