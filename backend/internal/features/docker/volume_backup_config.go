package docker

import (
	"time"

	"github.com/google/uuid"
)

type BackupInterval string

const (
	BackupIntervalHourly  BackupInterval = "HOURLY"
	BackupIntervalDaily   BackupInterval = "DAILY"
	BackupIntervalWeekly  BackupInterval = "WEEKLY"
	BackupIntervalMonthly BackupInterval = "MONTHLY"
)

// VolumeBackupConfig is a scheduled backup for a container. It references the
// container by name, not id, because a container's id changes every time it is
// recreated (e.g. docker compose up) while the name stays stable.
type VolumeBackupConfig struct {
	ID            uuid.UUID `json:"id"            gorm:"column:id;primaryKey"`
	ContainerName string    `json:"containerName" gorm:"column:container_name;type:text;not null;uniqueIndex"`

	MountPaths []string  `json:"mountPaths" gorm:"column:mount_paths;serializer:json;not null"`
	StorageID  uuid.UUID `json:"storageId"  gorm:"column:storage_id;not null"`

	Interval  BackupInterval `json:"interval"  gorm:"column:interval;type:text;not null"`
	TimeOfDay string         `json:"timeOfDay" gorm:"column:time_of_day;type:text"`

	// RetentionDays of 0 keeps every backup.
	RetentionDays int `json:"retentionDays" gorm:"column:retention_days;default:0"`

	Consistency ConsistencyMode `json:"consistency" gorm:"column:consistency;type:text;default:'NONE'"`

	IsEncrypted bool `json:"isEncrypted" gorm:"column:is_encrypted;default:false"`

	IsEnabled bool `json:"isEnabled" gorm:"column:is_enabled;default:true"`

	LastRunAt *time.Time `json:"lastRunAt" gorm:"column:last_run_at"`
	NextRunAt *time.Time `json:"nextRunAt" gorm:"column:next_run_at"`

	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (VolumeBackupConfig) TableName() string { return "volume_backup_configs" }

// NextRun returns the first scheduled run strictly after `after`, based on the
// config's interval and time-of-day (UTC). Hourly ignores TimeOfDay.
func (c *VolumeBackupConfig) NextRun(after time.Time) time.Time {
	after = after.UTC()

	if c.Interval == BackupIntervalHourly {
		return after.Truncate(time.Hour).Add(time.Hour)
	}

	hour, minute := parseTimeOfDay(c.TimeOfDay)
	candidate := time.Date(after.Year(), after.Month(), after.Day(), hour, minute, 0, 0, time.UTC)

	switch c.Interval {
	case BackupIntervalDaily:
		for !candidate.After(after) {
			candidate = candidate.AddDate(0, 0, 1)
		}
	case BackupIntervalWeekly:
		for !candidate.After(after) {
			candidate = candidate.AddDate(0, 0, 7)
		}
	case BackupIntervalMonthly:
		for !candidate.After(after) {
			candidate = candidate.AddDate(0, 1, 0)
		}
	case BackupIntervalHourly:
	}

	return candidate
}

func parseTimeOfDay(timeOfDay string) (hour, minute int) {
	parsed, err := time.Parse("15:04", timeOfDay)
	if err != nil {
		return 0, 0
	}

	return parsed.Hour(), parsed.Minute()
}
