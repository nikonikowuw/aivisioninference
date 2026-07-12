import { request } from './api';
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

function buildQuery(params: Record<string, string | number | undefined>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') sp.set(k, String(v));
  }
  return sp.toString() ? `?${sp}` : '';
}

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
