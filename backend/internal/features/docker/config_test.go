package docker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_VolumeBackupConfig_NextRun(t *testing.T) {
	base := time.Date(2026, 7, 11, 10, 30, 0, 0, time.UTC)

	cases := []struct {
		name    string
		config  VolumeBackupConfig
		nextRun time.Time
	}{
		{
			name:    "hourly rounds up to the next hour",
			config:  VolumeBackupConfig{Interval: BackupIntervalHourly},
			nextRun: time.Date(2026, 7, 11, 11, 0, 0, 0, time.UTC),
		},
		{
			name:    "daily later today keeps today",
			config:  VolumeBackupConfig{Interval: BackupIntervalDaily, TimeOfDay: "22:00"},
			nextRun: time.Date(2026, 7, 11, 22, 0, 0, 0, time.UTC),
		},
		{
			name:    "daily already passed rolls to tomorrow",
			config:  VolumeBackupConfig{Interval: BackupIntervalDaily, TimeOfDay: "04:00"},
			nextRun: time.Date(2026, 7, 12, 4, 0, 0, 0, time.UTC),
		},
		{
			name:    "weekly rolls forward a week when passed",
			config:  VolumeBackupConfig{Interval: BackupIntervalWeekly, TimeOfDay: "04:00"},
			nextRun: time.Date(2026, 7, 18, 4, 0, 0, 0, time.UTC),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.nextRun, testCase.config.NextRun(base))
		})
	}
}
