import type { BackupStatus } from './BackupStatus';

export interface VolumeBackup {
  id: string;
  fileName: string;
  containerId: string;
  containerName: string;
  image: string;
  mountPaths: string[];
  storageId: string;
  status: BackupStatus;
  failMessage?: string | null;
  isEncrypted: boolean;
  backupSizeMb: number;
  backupDurationMs: number;
  createdAt: string;
}
