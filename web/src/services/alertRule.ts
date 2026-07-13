import { request } from './api';

// Alert Rule types
export interface AlertRule {
  id: string;
  name: string;
  node_id?: string | null;
  metric_type: string;
  operator: string;
  threshold: number;
  duration_seconds: number;
  silence_minutes: number;
  enabled: boolean;
  notify_channels: string[];
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAlertRuleRequest {
  name: string;
  node_id?: string;
  metric_type: string;
  operator: string;
  threshold: number;
  duration_seconds?: number;
  silence_minutes?: number;
  enabled?: boolean;
  notify_channels?: string[];
  description?: string;
}

export interface UpdateAlertRuleRequest {
  name?: string;
  node_id?: string;
  metric_type?: string;
  operator?: string;
  threshold?: number;
  duration_seconds?: number;
  silence_minutes?: number;
  enabled?: boolean;
  notify_channels?: string[];
  description?: string;
}

export interface AlertRuleListParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  node_id?: string;
  metric_type?: string;
  enabled?: string;
  [key: string]: string | number | undefined;
}

function buildQuery(params: Record<string, string | number | undefined>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') sp.set(k, String(v));
  }
  return sp.toString() ? `?${sp}` : '';
}

export const alertRuleApi = {
  list: (params?: AlertRuleListParams) => {
    const query = buildQuery(params || {});
    return request<{ list: AlertRule[]; total: number; page: number; page_size: number }>(`/alert-rules${query}`);
  },
  get: (id: string) => request<AlertRule>(`/alert-rules/${id}`),
  create: (data: CreateAlertRuleRequest) =>
    request<AlertRule>('/alert-rules', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: UpdateAlertRuleRequest) =>
    request<void>(`/alert-rules/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    request<void>(`/alert-rules/${id}`, { method: 'DELETE' }),
};
