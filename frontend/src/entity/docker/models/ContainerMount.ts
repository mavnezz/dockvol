export interface ContainerMount {
  type: string;
  name: string;
  source: string;
  destination: string;
  isBackupCandidate: boolean;
}
