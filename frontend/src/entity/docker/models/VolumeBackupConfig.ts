import type { BackupInterval } from './BackupInterval';
import type { ConsistencyMode } from './ConsistencyMode';

export interface VolumeBackupConfig {
  id: string;
  containerName: string;
  mountPaths: string[];
  storageId: string;
  interval: BackupInterval;
  timeOfDay: string;
  retentionDays: number;
  consistency: ConsistencyMode;
  isEncrypted: boolean;
  isEnabled: boolean;
  lastRunAt?: string | null;
  nextRunAt?: string | null;
  createdAt: string;
}
