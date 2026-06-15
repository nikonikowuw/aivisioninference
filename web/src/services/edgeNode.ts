import { request, type AlgorithmPackage } from './api';

// Edge Node types
export interface EdgeNode {
  id: string;
  name: string;
  description?: string;
  endpoint: string;
  max_load: number;
  current_load: number;
  status: string; // online, offline, error, disabled
  platform: string;
  engine_version: string;
  hardware_info?: HardwareInfo;
  uptime: number;
  last_heartbeat: string;
  enabled: boolean;
  remark?: string;
  created_at: string;
  updated_at: string;
}

// HardwareInfo matches the backend dto.HardwareInfo (total_memory, cpu_cores)
// and also carries legacy display fields used by the frontend pages.
export interface HardwareInfo {
  cpu_model: string;
  gpu_model: string;
  total_memory?: number; // bytes (backend heartbeat field)
  cpu_cores?: number;
  // Legacy display fields (mapped from EdgeNode flat fields)
  memory?: string; // display string like "8GB"
  platform?: string;
}

export interface CreateEdgeNodeRequest {
  name: string;
  description?: string;
  endpoint: string;
  max_load: number;
  remark?: string;
}

export interface UpdateEdgeNodeRequest {
  name?: string;
  description?: string;
  endpoint?: string;
  max_load?: number;
  enabled?: boolean;
  remark?: string;
}

export interface EdgeNodeListParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  status?: string;
  enabled?: string;
  [key: string]: string | number | undefined;
}

export interface HeartbeatRequest {
  uptime: number;
  current_load: number;
  hardware_info: HardwareInfo;
  installed_algorithms: InstalledAlgorithmInfo[];
  engine_version: string;
}

export interface PendingDeployment {
  algo_package_id: string;
  algo_name: string;
  version: string;
  download_url: string;
  md5: string;
}

export interface InstalledAlgorithmInfo {
  algo_package_id: string;
  status: string; // installed, failed
  version?: string;
  install_path?: string;
  error_message?: string;
}

export interface NodeAlgorithm {
  id: string;
  node_id: string;
  algo_package_id: string;
  algo_name: string;
  algo_version: string;
  status: string; // pending, downloading, installed, failed
  download_url: string;
  install_path: string;
  retry_count: number;
  max_retry_count: number;
  last_retry_at: string;
  error_message: string;
  created_at: string;
  updated_at: string;
  algo_package?: AlgorithmPackage;
}

export interface DeployAlgorithmRequest {
  algo_package_id: string;
}

export interface CreateNodeResponse {
  node: EdgeNode;
  token: string;
}

export interface HeartbeatResponse {
  pending_deployments: PendingDeployment[];
}

export interface RecommendNodeResponse {
  recommended_node_id: string;
  node_name: string;
  current_load: number;
  max_load: number;
  load_rate: number;
}

function buildQuery(params: Record<string, string | number | undefined>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') sp.set(k, String(v));
  }
  return sp.toString() ? `?${sp}` : '';
}

export const edgeNodeApi = {
  // CRUD
  list: (params?: EdgeNodeListParams) => {
    const query = buildQuery(params || {});
    return request<{ list: EdgeNode[]; total: number; page: number; page_size: number }>(`/edge-nodes${query}`);
  },
  get: (id: string) => request<EdgeNode>(`/edge-nodes/${id}`),
  create: (data: CreateEdgeNodeRequest) =>
    request<CreateNodeResponse>('/edge-nodes', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: UpdateEdgeNodeRequest) =>
    request<EdgeNode>(`/edge-nodes/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    request<void>(`/edge-nodes/${id}`, { method: 'DELETE' }),

  // Algorithms
  deployAlgorithm: (nodeId: string, algoPackageId: string) =>
    request<NodeAlgorithm>(`/edge-nodes/${nodeId}/deploy-algo`, {
      method: 'POST',
      body: JSON.stringify({ algo_package_id: algoPackageId }),
    }),
  getNodeAlgorithms: (nodeId: string) =>
    request<NodeAlgorithm[]>(`/edge-nodes/${nodeId}/algorithms`),
  deleteAlgorithm: (nodeId: string, algoPackageId: string) =>
    request<void>(`/edge-nodes/${nodeId}/algorithms/${algoPackageId}`, { method: 'DELETE' }),

  // Heartbeat (used by engine, not admin UI)
  heartbeat: (nodeId: string, token: string, data: HeartbeatRequest) =>
    request<HeartbeatResponse>(`/edge-nodes/${nodeId}/heartbeat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    }),
};

// Node recommendation (part of AI Vision Tasks)
export const recommendNodeApi = {
  recommend: (algoPackageId: string) => {
    const query = buildQuery({ algo_package_id: algoPackageId });
    return request<RecommendNodeResponse>(`/edge-nodes/recommend-node${query}`);
  },
};
