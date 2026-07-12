import { request, getAccessToken } from './api';

export interface TerminalSession {
  id: string;
  node_id: string;
  user_id: string;
  user_name: string;
  status: 'active' | 'paused' | 'closed';
  reason?: string;
  error_message?: string;
  started_at: string;
  ended_at?: string;
  duration_seconds?: number;
  paused_at?: string;
}

export interface TerminalSessionListParams {
  status?: string;
  page?: number;
  page_size?: number;
}

export const terminalApi = {
  listSessions: (nodeId: string, params?: TerminalSessionListParams) => {
    const sp = new URLSearchParams();
    if (params?.status) sp.set('status', params.status);
    if (params?.page) sp.set('page', String(params.page));
    if (params?.page_size) sp.set('page_size', String(params.page_size));
    const qs = sp.toString() ? `?${sp}` : '';
    return request<TerminalSession[]>(`/edge-nodes/${nodeId}/sessions${qs}`);
  },

  getSession: (nodeId: string, sessionId: string) =>
    request<TerminalSession>(`/edge-nodes/${nodeId}/sessions/${sessionId}`),

  closeSession: (nodeId: string, sessionId: string) =>
    request<void>(`/edge-nodes/${nodeId}/sessions/${sessionId}`, { method: 'DELETE' }),

  getRecording: (nodeId: string, sessionId: string) =>
    request<unknown>(`/edge-nodes/${nodeId}/sessions/${sessionId}/recording`),

  getWebSocketURL: (nodeId: string, sessionId?: string) => {
    const token = getAccessToken();
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    let url = `${protocol}//${window.location.host}/api/v1/ws/terminal?token=${encodeURIComponent(token || '')}&node_id=${encodeURIComponent(nodeId)}`;
    if (sessionId) {
      url += `&session_id=${encodeURIComponent(sessionId)}`;
    }
    return url;
  },
};
