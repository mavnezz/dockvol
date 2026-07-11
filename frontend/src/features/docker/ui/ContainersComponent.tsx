import {
  ClockCircleOutlined,
  DeleteOutlined,
  DownloadOutlined,
  LockOutlined,
  RollbackOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import {
  App,
  Badge,
  Button,
  Checkbox,
  Divider,
  Popconfirm,
  Select,
  Spin,
  Switch,
  Table,
  Tabs,
  Tag,
} from 'antd';
import { useEffect, useState } from 'react';

import {
  type ConsistencyMode,
  type Container,
  type ContainerBackupSummary,
  type VolumeBackup,
  type VolumeBackupConfig,
  dockerApi,
} from '../../../entity/docker';
import { type Storage, storageApi } from '../../../entity/storages';
import type { WorkspaceResponse } from '../../../entity/workspaces';
import { consistencyOptions } from '../lib/consistencyOptions';
import { ScheduleComponent } from './ScheduleComponent';

interface Props {
  contentHeight: number;
  workspace: WorkspaceResponse;
  canManageBackups: boolean;
}

const statusColor = (status: VolumeBackup['status']): string => {
  if (status === 'COMPLETED') return 'green';
  if (status === 'FAILED') return 'red';
  return 'blue';
};

const formatSize = (sizeMb: number): string => {
  if (sizeMb < 1024) return `${sizeMb.toFixed(2)} MB`;
  return `${(sizeMb / 1024).toFixed(2)} GB`;
};

const formatDuration = (durationMs: number): string => {
  const totalSeconds = Math.round(durationMs / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}m ${seconds}s`;
};

// The health grid always shows a fixed number of cells so the history reads at a
// glance; slots with no backup yet stay grey.
const HEALTH_GRID_SIZE = 100;

const healthCellClass = (backup?: VolumeBackup): string => {
  if (!backup) return 'bg-gray-200 dark:bg-gray-700';
  if (backup.status === 'COMPLETED') return 'bg-green-500';
  if (backup.status === 'FAILED') return 'bg-red-500';
  return 'bg-blue-500';
};

const relativeTimeFrom = (iso: string): string => {
  const seconds = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (seconds < 60) return 'just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
};

const stateBadgeStatus = (state: string): 'success' | 'warning' | 'default' => {
  if (state === 'running') return 'success';
  if (state === 'paused') return 'warning';
  return 'default';
};

export const ContainersComponent = ({ contentHeight, workspace, canManageBackups }: Props) => {
  const { message } = App.useApp();

  const [isLoading, setIsLoading] = useState(true);
  const [containers, setContainers] = useState<Container[]>([]);
  const [storages, setStorages] = useState<Storage[]>([]);
  const [selectedContainerId, setSelectedContainerId] = useState<string | undefined>(undefined);
  const [selectedMountPaths, setSelectedMountPaths] = useState<string[]>([]);
  const [selectedStorageId, setSelectedStorageId] = useState<string | undefined>(undefined);
  const [consistency, setConsistency] = useState<ConsistencyMode>('NONE');
  const [isEncrypted, setIsEncrypted] = useState(false);
  const [backups, setBackups] = useState<VolumeBackup[]>([]);
  const [isBackingUp, setIsBackingUp] = useState(false);
  const [restoringBackupId, setRestoringBackupId] = useState<string | undefined>(undefined);
  const [configs, setConfigs] = useState<VolumeBackupConfig[]>([]);
  const [backupSummaries, setBackupSummaries] = useState<Map<string, ContainerBackupSummary>>(
    new Map(),
  );

  const loadBackups = async (containerId: string) => {
    try {
      setBackups(await dockerApi.getBackups(containerId));
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  const loadBackupSummaries = async () => {
    try {
      const summaries = await dockerApi.getContainerBackupSummaries();
      setBackupSummaries(new Map(summaries.map((summary) => [summary.containerName, summary])));
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  const selectContainer = (container: Container) => {
    setSelectedContainerId(container.id);
    setSelectedMountPaths(
      container.mounts.filter((mount) => mount.isBackupCandidate).map((mount) => mount.destination),
    );
    void loadBackups(container.id);
  };

  const loadData = async () => {
    setIsLoading(true);
    try {
      const [discoveredContainers, workspaceStorages, backupConfigs, summaries] = await Promise.all(
        [
          dockerApi.getContainers(),
          storageApi.getStorages(workspace.id),
          dockerApi.getConfigs(),
          dockerApi.getContainerBackupSummaries(),
        ],
      );
      setContainers(discoveredContainers);
      setStorages(workspaceStorages);
      setConfigs(backupConfigs);
      setBackupSummaries(new Map(summaries.map((summary) => [summary.containerName, summary])));
      setSelectedStorageId((current) => current ?? workspaceStorages[0]?.id);
      if (discoveredContainers.length > 0) {
        selectContainer(discoveredContainers[0]);
      }
    } catch (e) {
      message.error((e as Error).message);
    }
    setIsLoading(false);
  };

  const toggleMountPath = (destination: string) => {
    setSelectedMountPaths((current) =>
      current.includes(destination)
        ? current.filter((path) => path !== destination)
        : [...current, destination],
    );
  };

  const backupNow = async () => {
    if (!selectedContainerId || !selectedStorageId || selectedMountPaths.length === 0) {
      return;
    }

    setIsBackingUp(true);
    try {
      await dockerApi.createBackup({
        containerId: selectedContainerId,
        storageId: selectedStorageId,
        mountPaths: selectedMountPaths,
        consistency,
        isEncrypted,
      });
      message.success('Backup created');
      await loadBackups(selectedContainerId);
      void loadBackupSummaries();
    } catch (e) {
      message.error((e as Error).message);
    }
    setIsBackingUp(false);
  };

  const downloadBackup = async (backup: VolumeBackup) => {
    try {
      const blob = await dockerApi.fetchBackupBlob(backup.id);
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = backup.fileName;
      link.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  const deleteBackup = async (backup: VolumeBackup) => {
    try {
      await dockerApi.deleteBackup(backup.id);
      if (selectedContainerId) {
        await loadBackups(selectedContainerId);
      }
      void loadBackupSummaries();
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  const restoreBackup = async (backup: VolumeBackup) => {
    setRestoringBackupId(backup.id);
    try {
      await dockerApi.restoreBackup(backup.id);
      message.success('Restore completed');
    } catch (e) {
      message.error((e as Error).message);
    }
    setRestoringBackupId(undefined);
  };

  const reloadConfigs = async () => {
    try {
      setConfigs(await dockerApi.getConfigs());
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  useEffect(() => {
    void loadData();
  }, []);

  useEffect(() => {
    if (!selectedContainerId || !backups.some((backup) => backup.status === 'RUNNING')) {
      return;
    }

    const timer = window.setInterval(() => void loadBackups(selectedContainerId), 3000);

    return () => window.clearInterval(timer);
  }, [backups, selectedContainerId]);

  const selectedContainer = containers.find((container) => container.id === selectedContainerId);
  const selectedConfig = configs.find((config) => config.containerName === selectedContainer?.name);
  const scheduledContainerNames = new Set(configs.map((config) => config.containerName));
  const sortedContainers = [...containers].sort((a, b) => a.name.localeCompare(b.name));
  const containerGroups = [
    {
      label: 'Running',
      items: sortedContainers.filter((container) => container.state === 'running'),
    },
    {
      label: 'Paused',
      items: sortedContainers.filter((container) => container.state === 'paused'),
    },
    {
      label: 'Stopped',
      items: sortedContainers.filter(
        (container) => container.state !== 'running' && container.state !== 'paused',
      ),
    },
  ].filter((group) => group.items.length > 0);

  const recentBackups = backups.slice(0, HEALTH_GRID_SIZE).reverse();
  const healthCells: (VolumeBackup | undefined)[] = [
    ...Array<VolumeBackup | undefined>(Math.max(0, HEALTH_GRID_SIZE - recentBackups.length)).fill(
      undefined,
    ),
    ...recentBackups,
  ];

  const backupColumns = [
    {
      title: 'Created at',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (createdAt: string) => new Date(createdAt).toLocaleString(),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: VolumeBackup['status']) => <Tag color={statusColor(status)}>{status}</Tag>,
    },
    {
      title: 'Encryption',
      key: 'isEncrypted',
      render: (_: unknown, backup: VolumeBackup) =>
        backup.isEncrypted ? (
          <Tag icon={<LockOutlined />} color="gold">
            Encrypted
          </Tag>
        ) : (
          <span className="text-gray-400">-</span>
        ),
    },
    {
      title: 'Size',
      dataIndex: 'backupSizeMb',
      key: 'backupSizeMb',
      render: (sizeMb: number) => formatSize(sizeMb),
    },
    {
      title: 'Duration',
      dataIndex: 'backupDurationMs',
      key: 'backupDurationMs',
      render: (durationMs: number) => formatDuration(durationMs),
    },
    {
      title: 'Mounts',
      dataIndex: 'mountPaths',
      key: 'mountPaths',
      render: (mountPaths: string[]) => mountPaths.join(', '),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: unknown, backup: VolumeBackup) => (
        <div className="flex gap-1">
          <Button
            type="text"
            icon={<DownloadOutlined />}
            disabled={backup.status !== 'COMPLETED'}
            onClick={() => downloadBackup(backup)}
          />
          <Popconfirm
            title="Restore this backup?"
            description="This overwrites the current data in the container's mounts. This cannot be undone."
            okText="Restore"
            okButtonProps={{ danger: true }}
            cancelText="Cancel"
            onConfirm={() => restoreBackup(backup)}
          >
            <Button
              type="text"
              icon={<RollbackOutlined />}
              loading={restoringBackupId === backup.id}
              disabled={backup.status !== 'COMPLETED' || !canManageBackups}
            />
          </Popconfirm>
          <Popconfirm
            title="Delete this backup?"
            okText="Delete"
            cancelText="Cancel"
            onConfirm={() => deleteBackup(backup)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} disabled={!canManageBackups} />
          </Popconfirm>
        </div>
      ),
    },
  ];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center" style={{ height: contentHeight }}>
        <Spin />
      </div>
    );
  }

  return (
    <div className="flex gap-3" style={{ height: contentHeight }}>
      <div className="w-[260px] shrink-0 overflow-y-auto rounded bg-white p-2 shadow dark:bg-gray-800">
        {containers.length === 0 ? (
          <div className="p-3 text-sm text-gray-500 dark:text-gray-400">No containers found.</div>
        ) : (
          containerGroups.map((group) => (
            <div key={group.label}>
              <Divider orientation="left" plain className="!my-2 !text-xs !text-gray-400">
                {group.label} ({group.items.length})
              </Divider>
              {group.items.map((container) => {
                const summary = backupSummaries.get(container.name);

                return (
                  <button
                    type="button"
                    key={container.id}
                    onClick={() => selectContainer(container)}
                    className={`mb-2 w-full rounded p-2 text-left ${
                      container.id === selectedContainerId
                        ? 'bg-blue-50 dark:bg-gray-700'
                        : 'hover:bg-gray-50 dark:hover:bg-gray-700'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="truncate font-medium text-black dark:text-gray-100">
                          {container.name}
                        </div>
                        <div className="truncate text-xs text-gray-500 dark:text-gray-400">
                          {container.image}
                        </div>
                        <div className="mt-1 flex items-center gap-2 text-xs text-gray-400">
                          <Badge
                            status={stateBadgeStatus(container.state)}
                            text={container.state}
                          />
                          <span>{container.mounts.length} mounts</span>
                        </div>
                        {summary && (
                          <div className="mt-0.5 truncate text-xs text-gray-400">
                            {summary.storageName ? `${summary.storageName} - ` : ''}last backup{' '}
                            {relativeTimeFrom(summary.lastBackupAt)}
                          </div>
                        )}
                      </div>
                      <div className="flex shrink-0 items-center gap-1.5 pt-0.5">
                        {summary && (
                          <SaveOutlined className="text-green-500" title="Backups available" />
                        )}
                        {scheduledContainerNames.has(container.name) && (
                          <ClockCircleOutlined className="text-blue-500" title="Scheduled backup" />
                        )}
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>
          ))
        )}
      </div>

      <div className="grow overflow-y-auto rounded bg-white p-4 shadow dark:bg-gray-800">
        {!selectedContainer ? (
          <div className="text-gray-500 dark:text-gray-400">Select a container to back up.</div>
        ) : (
          <>
            <div className="mb-1 text-lg font-medium text-black dark:text-gray-100">
              {selectedContainer.name}
            </div>
            <div className="mb-3 text-xs text-gray-500 dark:text-gray-400">
              {selectedContainer.image}
            </div>

            <Tabs
              items={[
                {
                  key: 'config',
                  label: 'Config',
                  children: (
                    <>
                      <div className="mb-2 font-medium text-black dark:text-gray-200">Mounts</div>
                      <div className="mb-4">
                        {selectedContainer.mounts.map((mount) => (
                          <div key={mount.destination} className="mb-1">
                            <Checkbox
                              disabled={!mount.isBackupCandidate || !canManageBackups}
                              checked={selectedMountPaths.includes(mount.destination)}
                              onChange={() => toggleMountPath(mount.destination)}
                            >
                              <span className="text-black dark:text-gray-200">
                                {mount.destination}
                              </span>
                              <span className="ml-2 text-xs text-gray-400">
                                ({mount.type}
                                {mount.isBackupCandidate ? '' : ', infrastructure'})
                              </span>
                            </Checkbox>
                          </div>
                        ))}
                      </div>

                      <div className="mb-4 flex flex-wrap items-center gap-3">
                        <span className="text-black dark:text-gray-200">Target storage</span>
                        <Select
                          value={selectedStorageId}
                          onChange={setSelectedStorageId}
                          className="min-w-[200px]"
                          placeholder="Select storage"
                          options={storages.map((storage) => ({
                            label: storage.name,
                            value: storage.id,
                          }))}
                        />
                        <span className="text-black dark:text-gray-200">Consistency</span>
                        <Select
                          value={consistency}
                          onChange={setConsistency}
                          className="min-w-[180px]"
                          options={consistencyOptions}
                        />
                        <span className="text-black dark:text-gray-200">Encrypt</span>
                        <Switch checked={isEncrypted} onChange={setIsEncrypted} />
                        <Button
                          type="primary"
                          loading={isBackingUp}
                          disabled={
                            !canManageBackups ||
                            !selectedStorageId ||
                            selectedMountPaths.length === 0
                          }
                          onClick={backupNow}
                        >
                          Back up now
                        </Button>
                      </div>

                      <ScheduleComponent
                        key={selectedContainer.name}
                        containerName={selectedContainer.name}
                        availableMounts={selectedContainer.mounts
                          .filter((mount) => mount.isBackupCandidate)
                          .map((mount) => mount.destination)}
                        storageId={selectedStorageId}
                        config={selectedConfig}
                        canManage={canManageBackups}
                        onChanged={reloadConfigs}
                      />
                    </>
                  ),
                },
                {
                  key: 'backups',
                  label: 'Backups',
                  children: (
                    <>
                      <div className="mb-4">
                        <div className="mb-1 font-medium text-black dark:text-gray-200">
                          Backup health
                        </div>
                        <div className="flex flex-wrap gap-1">
                          {healthCells.map((backup, index) => (
                            <div
                              key={backup ? backup.id : `empty-${index}`}
                              title={
                                backup
                                  ? `${new Date(backup.createdAt).toLocaleString()} - ${backup.status}`
                                  : 'No backup yet'
                              }
                              className={`h-3 w-3 rounded-sm ${healthCellClass(backup)}`}
                            />
                          ))}
                        </div>
                      </div>

                      <Table
                        rowKey="id"
                        size="small"
                        pagination={false}
                        dataSource={backups}
                        columns={backupColumns}
                        locale={{ emptyText: 'No backups yet' }}
                      />
                    </>
                  ),
                },
              ]}
            />
          </>
        )}
      </div>
    </div>
  );
};
