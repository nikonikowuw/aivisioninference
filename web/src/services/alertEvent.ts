import { request } from './api';

// Alert Event types
export interface AlertEvent {
  id: string;
  rule_id: string;
  node_id: string;
  metric_value: number;
  status: string; // firing, resolved, acknowledged
  fired_at: string;
  resolved_at?: string;
  acknowledged_by?: string;
  acknowledged_at?: string;
  notify_sent: boolean;
  notify_sent_at?: string;

  // Joined fields
  rule_name?: string;
  node_name?: string;
  metric_type?: string;
  operator?: string;
  threshold?: number;
}

export interface AlertEventListParams {
  page?: number;
  page_size?: number;
  node_id?: string;
  rule_id?: string;
  status?: string;
  from?: string;
  to?: string;
  [key: string]: string | number | undefined;
}

export interface AcknowledgeAlertEventRequest {
  acknowledged_by: string;
}

function buildQuery(params: Record<string, string | number | undefined>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') sp.set(k, String(v));
  }
  return sp.toString() ? `?${sp}` : '';
}

export const alertEventApi = {
  list: (params?: AlertEventListParams) => {
    const query = buildQuery(params || {});
    return request<{ list: AlertEvent[]; total: number; page: number; page_size: number }>(`/alert-events${query}`);
  },
  acknowledge: (id: string, acknowledgedBy: string) =>
    request<void>(`/alert-events/${id}/acknowledge`, {
      method: 'POST',
      body: JSON.stringify({ acknowledged_by: acknowledgedBy }),
    }),
};
