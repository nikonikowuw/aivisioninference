import { request } from './api';
import { buildQuery } from '../utils/query';
import type { EdgeNode } from './edgeNode';

// --- Types ---

export interface MetricDataPoint {
  t: string;  // ISO8601 timestamp
  v: number;  // value
}

export interface MetricQueryResponse {
  list: MetricDataPoint[];
  total: number;
  page: number;
  page_size: number;
}

export interface MetricQueryParams {
  metric: string;
  from?: string;
  to?: string;
  aggregation?: 'avg' | 'max' | 'min';
  interval?: string;  // '5m', '1h', '1d'
  page?: number;
  page_size?: number;
}

export interface OverviewStats {
  total: number;
  online: number;
  offline: number;
  error: number;
  alert_count: number;
}

/** Live metrics snapshot from WebSocket heartbeat events */
export interface NodeMetrics {
  cpu_usage: number;
  memory_usage: number;
  cpu_load_1m: number;
  cpu_load_5m: number;
  cpu_load_15m: number;
  net_rx_speed: number;
  net_tx_speed: number;
  process_count: number;
  thread_count: number;
  temperature: number;
}


export const edgeNodeMetricsApi = {

export const edgeNodeMetricsApi = {
  /** Query time-series metrics for a specific edge node */
  queryMetrics: (nodeId: string, params: MetricQueryParams) => {
    const query = buildQuery({ ...params });
    return request<MetricQueryResponse>(`/edge-nodes/${nodeId}/metrics${query}`);
  },

  /** Get overview stats for all edge nodes */
  getOverview: () => {
    return request<OverviewStats>('/edge-nodes/overview');
  },
};
