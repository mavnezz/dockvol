import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { Container } from '../models/Container';
import type { ContainerBackupSummary } from '../models/ContainerBackupSummary';
import type { CreateBackupRequest } from '../models/CreateBackupRequest';
import type { SaveVolumeBackupConfigRequest } from '../models/SaveVolumeBackupConfigRequest';
import type { VolumeBackup } from '../models/VolumeBackup';
import type { VolumeBackupConfig } from '../models/VolumeBackupConfig';

export const dockerApi = {
  async getContainers() {
    const requestOptions = new RequestOptions();
    const response = await apiHelper.fetchGetJson<{ containers: Container[] }>(
      `${getApplicationServer()}/api/v1/docker/containers`,
      requestOptions,
      true,
    );
    return response.containers ?? [];
  },

  async createBackup(request: CreateBackupRequest) {
    const requestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson<VolumeBackup>(
      `${getApplicationServer()}/api/v1/docker/backup`,
      requestOptions,
    );
  },

  async getBackups(containerId?: string) {
    const requestOptions = new RequestOptions();
    const query = containerId ? `?containerId=${encodeURIComponent(containerId)}` : '';
    const response = await apiHelper.fetchGetJson<{ backups: VolumeBackup[] }>(
      `${getApplicationServer()}/api/v1/docker/backups${query}`,
      requestOptions,
      true,
    );
    return response.backups ?? [];
  },

  async getContainerBackupSummaries() {
    const requestOptions = new RequestOptions();
    const response = await apiHelper.fetchGetJson<{ containers: ContainerBackupSummary[] }>(
      `${getApplicationServer()}/api/v1/docker/backed-up-containers`,
      requestOptions,
      true,
    );
    return response.containers ?? [];
  },

  async deleteBackup(id: string) {
    const requestOptions = new RequestOptions();
    return apiHelper.fetchDeleteJson(
      `${getApplicationServer()}/api/v1/docker/backups/${id}`,
      requestOptions,
    );
  },

  async restoreBackup(id: string) {
    const requestOptions = new RequestOptions();
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/docker/backups/${id}/restore`,
      requestOptions,
    );
  },

  async fetchBackupBlob(id: string): Promise<Blob> {
    const requestOptions = new RequestOptions();
    return apiHelper.fetchGetBlob(
      `${getApplicationServer()}/api/v1/docker/backups/${id}/download`,
      requestOptions,
      true,
    );
  },

  async getConfigs() {
    const requestOptions = new RequestOptions();
    const response = await apiHelper.fetchGetJson<{ configs: VolumeBackupConfig[] }>(
      `${getApplicationServer()}/api/v1/docker/configs`,
      requestOptions,
      true,
    );
    return response.configs ?? [];
  },

  async saveConfig(config: SaveVolumeBackupConfigRequest) {
    const requestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(config));
    return apiHelper.fetchPostJson<VolumeBackupConfig>(
      `${getApplicationServer()}/api/v1/docker/configs`,
      requestOptions,
    );
  },

  async deleteConfig(id: string) {
    const requestOptions = new RequestOptions();
    return apiHelper.fetchDeleteJson(
      `${getApplicationServer()}/api/v1/docker/configs/${id}`,
      requestOptions,
    );
  },
};
