import { App, Button, Input, Popconfirm, Select, Switch } from 'antd';
import { useState } from 'react';

import {
  type BackupInterval,
  type ConsistencyMode,
  type VolumeBackupConfig,
  dockerApi,
} from '../../../entity/docker';
import { consistencyOptions } from '../lib/consistencyOptions';

interface Props {
  containerName: string;
  availableMounts: string[];
  storageId: string | undefined;
  config: VolumeBackupConfig | undefined;
  canManage: boolean;
  onChanged: () => void;
}

const intervalOptions: { label: string; value: BackupInterval }[] = [
  { label: 'Hourly', value: 'HOURLY' },
  { label: 'Daily', value: 'DAILY' },
  { label: 'Weekly', value: 'WEEKLY' },
  { label: 'Monthly', value: 'MONTHLY' },
];

const retentionOptions = [
  { label: 'Keep forever', value: 0 },
  { label: '1 week', value: 7 },
  { label: '1 month', value: 30 },
  { label: '3 months', value: 90 },
  { label: '1 year', value: 365 },
];

export const ScheduleComponent = ({
  containerName,
  availableMounts,
  storageId,
  config,
  canManage,
  onChanged,
}: Props) => {
  const { message } = App.useApp();

  const [scheduleMountPaths, setScheduleMountPaths] = useState<string[]>(
    config?.mountPaths ?? availableMounts,
  );
  const [backupInterval, setBackupInterval] = useState<BackupInterval>(config?.interval ?? 'DAILY');
  const [timeOfDay, setTimeOfDay] = useState(config?.timeOfDay ?? '04:00');
  const [retentionDays, setRetentionDays] = useState(config?.retentionDays ?? 30);
  const [consistency, setConsistency] = useState<ConsistencyMode>(config?.consistency ?? 'NONE');
  const [isEncrypted, setIsEncrypted] = useState(config?.isEncrypted ?? false);
  const [isEnabled, setIsEnabled] = useState(config?.isEnabled ?? true);
  const [isSaving, setIsSaving] = useState(false);

  const save = async () => {
    if (!storageId) {
      message.error('Select a target storage first');
      return;
    }
    if (scheduleMountPaths.length === 0) {
      message.error('Select at least one mount first');
      return;
    }

    setIsSaving(true);
    try {
      await dockerApi.saveConfig({
        id: config?.id,
        containerName,
        mountPaths: scheduleMountPaths,
        storageId,
        interval: backupInterval,
        timeOfDay,
        retentionDays,
        consistency,
        isEncrypted,
        isEnabled,
      });
      message.success('Schedule saved');
      onChanged();
    } catch (e) {
      message.error((e as Error).message);
    }
    setIsSaving(false);
  };

  const remove = async () => {
    if (!config) {
      return;
    }

    try {
      await dockerApi.deleteConfig(config.id);
      message.success('Schedule removed');
      onChanged();
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  return (
    <div className="mb-4 rounded border border-gray-200 p-3 dark:border-gray-700">
      <div className="mb-2 font-medium text-black dark:text-gray-200">Schedule</div>
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2">
          <span className="text-gray-500 dark:text-gray-400">Mounts</span>
          <Select
            mode="multiple"
            value={scheduleMountPaths}
            onChange={setScheduleMountPaths}
            options={availableMounts.map((path) => ({ label: path, value: path }))}
            placeholder="Select mounts"
            className="min-w-[200px]"
          />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-gray-500 dark:text-gray-400">Interval</span>
          <Select
            value={backupInterval}
            onChange={setBackupInterval}
            options={intervalOptions}
            className="min-w-[120px]"
          />
        </div>
        {backupInterval !== 'HOURLY' && (
          <div className="flex items-center gap-2">
            <span className="text-gray-500 dark:text-gray-400">Time (UTC)</span>
            <Input
              value={timeOfDay}
              onChange={(event) => setTimeOfDay(event.target.value)}
              placeholder="HH:MM"
              className="w-[90px]"
            />
          </div>
        )}
        <div className="flex items-center gap-2">
          <span className="text-gray-500 dark:text-gray-400">Store period</span>
          <Select
            value={retentionDays}
            onChange={setRetentionDays}
            options={retentionOptions}
            className="min-w-[130px]"
          />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-gray-500 dark:text-gray-400">Consistency</span>
          <Select
            value={consistency}
            onChange={setConsistency}
            options={consistencyOptions}
            className="min-w-[180px]"
          />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-gray-500 dark:text-gray-400">Encrypt</span>
          <Switch checked={isEncrypted} onChange={setIsEncrypted} />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-gray-500 dark:text-gray-400">Enabled</span>
          <Switch checked={isEnabled} onChange={setIsEnabled} />
        </div>
        <Button type="primary" loading={isSaving} disabled={!canManage} onClick={save}>
          {config ? 'Update schedule' : 'Save schedule'}
        </Button>
        {config && (
          <Popconfirm
            title="Remove this schedule?"
            okText="Remove"
            cancelText="Cancel"
            onConfirm={remove}
          >
            <Button danger disabled={!canManage}>
              Remove
            </Button>
          </Popconfirm>
        )}
      </div>
      {config?.nextRunAt && config.isEnabled && (
        <div className="mt-2 text-xs text-gray-400">
          Next run: {new Date(config.nextRunAt).toLocaleString()}
        </div>
      )}
    </div>
  );
};
