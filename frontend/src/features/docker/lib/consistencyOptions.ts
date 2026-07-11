import type { ConsistencyMode } from '../../../entity/docker';

export const consistencyOptions: { label: string; value: ConsistencyMode }[] = [
  { label: 'Keep running', value: 'NONE' },
  { label: 'Pause during backup', value: 'PAUSE' },
  { label: 'Stop during backup', value: 'STOP' },
];
