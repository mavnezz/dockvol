import type { ConsistencyMode } from './ConsistencyMode';

export interface CreateBackupRequest {
  containerId: string;
  storageId: string;
  mountPaths: string[];
  consistency: ConsistencyMode;
  isEncrypted: boolean;
}
