import { Button, Modal, Select, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { type Notifier, notifierApi } from '../../../entity/notifiers';
import { type WorkspaceResponse, workspaceApi } from '../../../entity/workspaces';

interface Props {
  notifier: Notifier;
  onClose: () => void;
  onTransferred: () => void;
}

export const NotifierTransferDialogComponent = ({ notifier, onClose, onTransferred }: Props) => {
  const [isLoading, setIsLoading] = useState(true);
  const [workspaces, setWorkspaces] = useState<WorkspaceResponse[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | undefined>();
  const [isTransferring, setIsTransferring] = useState(false);

  const loadData = async () => {
    setIsLoading(true);

    try {
      const response = await workspaceApi.getWorkspaces();
      const filteredWorkspaces = response.workspaces.filter((w) => w.id !== notifier.workspaceId);
      setWorkspaces(filteredWorkspaces);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsLoading(false);
  };

  const transferNotifier = async () => {
    if (!selectedWorkspaceId) return;

    setIsTransferring(true);

    try {
      await notifierApi.transferNotifier(notifier.id, selectedWorkspaceId);
      onTransferred();
    } catch (e) {
      alert((e as Error).message);
    }

    setIsTransferring(false);
  };

  useEffect(() => {
    loadData();
  }, [notifier.id]);

  return (
    <Modal
      title="Transfer notifier to another workspace"
      footer={null}
      open={true}
      onCancel={onClose}
      maskClosable={false}
    >
      {isLoading ? (
        <div className="flex justify-center py-5">
          <Spin />
        </div>
      ) : (
        <div className="py-3">
          <div className="mb-3 text-gray-500 dark:text-gray-400">
            Select a workspace to transfer this notifier to.
          </div>

          <div className="mb-5 flex items-center">
            <div className="min-w-[120px]">Target workspace</div>

            <Select
              value={selectedWorkspaceId}
              onChange={setSelectedWorkspaceId}
              className="min-w-[200px] grow"
              placeholder="Select workspace"
              options={workspaces.map((w) => ({ label: w.name, value: w.id }))}
            />
          </div>

          <div className="flex gap-2">
            <Button type="default" onClick={onClose}>
              Cancel
            </Button>

            <Button
              type="primary"
              onClick={transferNotifier}
              loading={isTransferring}
              disabled={!selectedWorkspaceId || isTransferring}
            >
              Transfer
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
};
