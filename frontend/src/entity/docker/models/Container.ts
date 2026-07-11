import type { ContainerMount } from './ContainerMount';

export interface Container {
  id: string;
  name: string;
  image: string;
  state: string;
  mounts: ContainerMount[];
}
