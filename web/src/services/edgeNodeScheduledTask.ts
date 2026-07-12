import { request } from './api';

export interface EdgeNodeScheduledTask {
  id: string;
  node_id: string;
  name: string;
  command: string;
  cron_expr?: string;
  status: string; // active, disabled, running
  timeout_seconds: number;
  last_run_at?: string;
  created_at: string;
  updated_at: string;
}

export interface EdgeNodeTaskExecution {
  id: string;
  task_id: string;
  status: string; // running, success, failed, timeout
  stdout?: string;
  stderr?: string;
  exit_code?: number;
  started_at?: string;
  finished_at?: string;
  created_at: string;
}

export interface ScheduledTaskRequest {
  name: string;
  command: string;
  cron_expr?: string;
  timeout_seconds?: number;
}

export interface ScheduledTaskUpdateRequest {
  name?: string;
  command?: string;
  cron_expr?: string;
  timeout_seconds?: number;
  status?: string;
}

function buildQuery(params: Record<string, any>): string {
  const qs = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== null && v !== '')
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
    .join('&');
  return qs ? `?${qs}` : '';
}

// Scheduled Tasks API
export const edgeNodeScheduledTaskApi = {
  /** List scheduled tasks for a node */
  list: (nodeId: string, params?: { page?: number; page_size?: number; status?: string }) =>
    request<{ list: EdgeNodeScheduledTask[]; total: number; page: number; page_size: number }>(
      `/edge-nodes/${nodeId}/scheduled-tasks${buildQuery(params || {})}`,
    ),

  /** Create a new scheduled task */
  create: (nodeId: string, data: ScheduledTaskRequest) =>
    request<EdgeNodeScheduledTask>(`/edge-nodes/${nodeId}/scheduled-tasks/create`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  /** Get a scheduled task by ID */
  get: (nodeId: string, taskId: string) =>
    request<EdgeNodeScheduledTask>(`/edge-nodes/${nodeId}/scheduled-tasks/${taskId}`),

  /** Update a scheduled task */
  update: (nodeId: string, taskId: string, data: ScheduledTaskUpdateRequest) =>
    request<EdgeNodeScheduledTask>(`/edge-nodes/${nodeId}/scheduled-tasks/${taskId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  /** Delete a scheduled task */
  delete: (nodeId: string, taskId: string) =>
    request<void>(`/edge-nodes/${nodeId}/scheduled-tasks/${taskId}`, { method: 'DELETE' }),

  /** List execution records for a task */
  listExecutions: (nodeId: string, params: { task_id: string; page?: number; page_size?: number; status?: string }) =>
    request<{ list: EdgeNodeTaskExecution[]; total: number; page: number; page_size: number }>(
      `/edge-nodes/${nodeId}/task-executions${buildQuery(params)}`,
    ),
};
