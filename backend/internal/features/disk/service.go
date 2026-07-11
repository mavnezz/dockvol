package disk

import (
	"fmt"
	"path/filepath"

	"github.com/shirou/gopsutil/v4/disk"

	"dockvol-backend/internal/config"
)

type DiskService struct{}

func (s *DiskService) GetDiskUsage() (*DiskUsage, error) {
	cfg := config.GetEnv()
	path := filepath.Dir(cfg.DataFolder) // Gets /dockvol-data from /dockvol-data/backups

	diskUsage, err := disk.Usage(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage for path %s: %w", path, err)
	}

	return &DiskUsage{
		TotalSpaceBytes: int64(diskUsage.Total),
		UsedSpaceBytes:  int64(diskUsage.Used),
		FreeSpaceBytes:  int64(diskUsage.Free),
	}, nil
}
