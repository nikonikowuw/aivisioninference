import { buildQuery } from '../utils/query';
import { request, type AlgorithmPackage } from './api';

// Edge Node types
export interface EdgeNode {
    id: string;
    name: string;
    description?: string;
    endpoint: string;
    max_load: number;
    current_load: number;
    cpu_usage?: number;
    memory_usage?: number;
    media_decode_capacity: number;
    media_encode_capacity: number;
    media_egress_capacity_bps: number;
    media_metrics_ttl_seconds: number;
    status: string; // online, offline, error, disabled
    hal_platform: string;
    engine_version: string;
    hardware_info?: HardwareInfo;
    uptime: number;
    last_heartbeat: string;
    enabled: boolean;
    remark?: string;
    runtime_error?: string;
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
    cpu_usage?: number;
    memory_usage?: number;
    // Legacy display fields (mapped from EdgeNode flat fields)
    memory?: string; // display string like "8GB"
    platform?: string;
}

export interface CreateEdgeNodeRequest {
    name: string;
    description?: string;
    endpoint: string;
    max_load: number;
    media_decode_capacity?: number;
    media_encode_capacity?: number;
    media_egress_capacity_bps?: number;
    media_metrics_ttl_seconds?: number;
    remark?: string;
}

export interface UpdateEdgeNodeRequest {
    name?: string;
    description?: string;
    endpoint?: string;
    max_load?: number;
    media_decode_capacity?: number;
    media_encode_capacity?: number;
    media_egress_capacity_bps?: number;
    media_metrics_ttl_seconds?: number;
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
    algo_name?: string;
    status: string; // installed, failed
    version?: string;
    install_path?: string;
    runtime_status?: string;
    supports_embedding?: boolean;
    supports_face_library?: boolean;
    embedding_capacity?: number;
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

export interface DiskUsageInfo {
    path: string;
    total: number;
    used: number;
    percent: number;
}

export interface EdgeNodeCardSnapshot extends EdgeNode {
    temp?: number;
    active_stream_count?: number;
    decode_slots_used?: number;
    decode_slots_total?: number;
    decode_usage?: number;
    encode_slots_used?: number;
    encode_slots_total?: number;
    encode_usage?: number;
    egress_bps?: number;
    egress_capacity_bps?: number;
    egress_usage?: number;
    accelerator_utilization?: number;
    accelerator_metrics_valid: boolean;
    metrics_received_at?: string;
    disk_usage?: DiskUsageInfo[];
}

export const edgeNodeApi = {
    // CRUD
    list: (params?: EdgeNodeListParams) => {
        const query = buildQuery(params || {});
        return request<{ list: EdgeNodeCardSnapshot[]; total: number; page: number; page_size: number }>(`/edge-nodes${query}`);
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

    // Node recommendation (part of AI Vision Tasks)
};

export const recommendNodeApi = {
    recommend: (algoPackageId: string) => {
        const query = buildQuery({ algo_package_id: algoPackageId });
        return request<RecommendNodeResponse>(`/edge-nodes/recommend-node${query}`);
    },
};
