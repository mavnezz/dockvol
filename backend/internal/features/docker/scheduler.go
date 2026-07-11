package docker

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const backupSchedulerJobName = "volume_backup_scheduler"

type Scheduler struct {
	configRepository *VolumeBackupConfigRepository
	backuper         *Backuper
	logger           *slog.Logger
	hasRun           atomic.Bool
}

func (s *Scheduler) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runDueBackups(ctx)
		}
	}
}

func (s *Scheduler) runDueBackups(ctx context.Context) {
	logger := s.logger.With("job_id", uuid.New(), "job_name", backupSchedulerJobName)

	now := time.Now().UTC()

	dueConfigs, err := s.configRepository.FindDue(now)
	if err != nil {
		logger.Error("failed to list due backup configs", "error", err)

		return
	}

	for i := range dueConfigs {
		config := &dueConfigs[i]
		configLogger := logger.With("container_name", config.ContainerName)

		if _, err := s.backuper.CreateBackupForConfig(ctx, config); err != nil {
			configLogger.Error("scheduled backup failed to start", "error", err)
		}

		// Advance the schedule even on failure so a broken config isn't retried
		// every tick; the next window gets a fresh attempt.
		nextRun := config.NextRun(now)
		config.LastRunAt = &now
		config.NextRunAt = &nextRun

		if err := s.configRepository.Save(config); err != nil {
			configLogger.Error("failed to advance backup config schedule", "error", err)
		}
	}
}
