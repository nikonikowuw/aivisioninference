import i18n from '../i18n';
import SparkMD5 from 'spark-md5';

/**
 * 根据错误码获取翻译后的错误消息
 * 后端只返回错误码，前端根据当前语言翻译
 * 前置条件：调用方已确保 code !== 0（成功码不应调用此函数）
 */
export function getErrorMessage(code: number): string {
  if (code === 0) return '';
  const key = `common:message.error.${code}`;
  const msg = i18n.t(key);
  return msg === key ? i18n.t('common:message.serverError') : msg;
}

function resolveApiErrorMessage(code: number, backendMessage?: string): string {
  // 参数校验错误优先展示后端具体提示，避免前端只显示“请求参数错误”。
  if (code === 10001 && backendMessage && backendMessage.trim() !== '') {
    return backendMessage;
  }
  return getErrorMessage(code);
}

export class ApiError extends Error {
  code: number;
  status: number;

  constructor(code: number, message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}

function computeMD5(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunkSize = 2 * 1024 * 1024;
    const chunks = Math.ceil(file.size / chunkSize) || 1;
    const spark = new SparkMD5.ArrayBuffer();
    const reader = new FileReader();
    let current = 0;

    reader.onload = (e) => {
      if (e.target?.result) spark.append(e.target.result as ArrayBuffer);
      current++;
      if (current < chunks) {
        loadNext();
      } else {
        resolve(spark.end());
      }
    };
    reader.onerror = () => reject(reader.error);

    function loadNext() {
      const start = current * chunkSize;
      const end = Math.min(start + chunkSize, file.size);
      reader.readAsArrayBuffer(file.slice(start, end));
    }

    loadNext();
  });
}

const API_BASE = '/api/v1';

interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

interface PaginatedData<T> {
  list: T[];
  total: number;
  page: number;
  page_size: number;
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const token = localStorage.getItem('access_token');
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'Accept-Language': i18n.language || 'en-US',
    ...(options.headers as Record<string, string>),
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  const json: ApiResponse<T> = await response.json();

  if (json.code !== 0) {
    // 非登录页面收到 401：token 失效，清除凭证并跳转登录页
    if (response.status === 401 && !window.location.pathname.startsWith('/auth/')) {
      localStorage.removeItem('access_token');
      window.location.href = '/auth/sign-in';
    }
    // 使用前端翻译的错误消息
    throw new ApiError(
      json.code,
      resolveApiErrorMessage(json.code, json.message),
      response.status,
    );
  }

  return json.data;
}

function buildQuery(params: Record<string, string | number | undefined>): string {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== '',
  );
  return entries.length ? `?${new URLSearchParams(entries.map(([k, v]) => [k, String(v)]))}` : '';
}

// Auth
export const authApi = {
  login: (username: string, password: string) =>
    request<{ access_token: string; user: User }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  logout: () =>
    request('/auth/logout', { method: 'POST' }),
  me: () =>
    request<User>('/auth/me'),
  changePassword: (oldPassword: string, newPassword: string) =>
    request('/auth/password', {
      method: 'PUT',
      body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }),
    }),
  updateProfile: (data: { display_name?: string; email?: string; avatar_url?: string }) =>
    request<User>('/auth/profile', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  uploadAvatar: async (file: File): Promise<{ avatar_url: string }> => {
    const formData = new FormData();
    formData.append('avatar', file);
    const token = localStorage.getItem('access_token');
    const response = await fetch(`${API_BASE}/auth/avatar`, {
      method: 'POST',
      headers: {
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        'Accept-Language': i18n.language || 'en-US',
      },
      body: formData,
    });
    if (!response.ok) {
      let msg = `Upload failed (HTTP ${response.status})`;
      let code = 50001; // Internal server error as default
      try {
        const errJson = await response.json();
        if (errJson.code !== undefined && errJson.code !== 0) {
          code = errJson.code;
          msg = resolveApiErrorMessage(errJson.code, errJson.message);
        } else if (errJson.message) {
          msg = errJson.message;
        }
      } catch { /* ignore */ }
      throw new ApiError(code, msg, response.status);
    }
    const json: ApiResponse<{ avatar_url: string }> = await response.json();
    if (json.code !== 0) {
      throw new ApiError(
        json.code,
        resolveApiErrorMessage(json.code, json.message),
        response.status,
      );
    }
    return json.data;
  },
};

// Types
export interface Menu {
  id: string;
  name: string;
  code: string;
  path: string;
  icon: string;
  sort_order: number;
  hidden?: boolean;
  children?: Menu[];
}

export interface User {
  id: string;
  username: string;
  display_name: string;
  email: string;
  email_verified: boolean;
  avatar_url: string;
  status: number;
  roles: Role[];
  menus: Menu[];
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  sort_order: number;
  status: number;
  level: number;
  permissions: Permission[];
  created_at: string;
  updated_at: string;
}

export interface Permission {
  id: string;
  name: string;
  code: string;
  path: string;
  method: string;
  type: string;
  parent_id: string | null;
  sort_order: number;
  children?: Permission[];
  created_at: string;
  updated_at: string;
}

export interface FileItem {
  id: string;
  name: string;
  original_name: string;
  mime_type: string;
  size: number;
  created_at: string;
}

export interface AuditLog {
  id: string;
  user_id: string | null;
  username: string;
  action_type: string;
  resource_type: string;
  resource_id: string;
  request_path: string;
  request_method: string;
  request_ip: string;
  user_agent: string;
  response_status: number;
  duration_ms: number;
  result_summary: string;
  error_summary: string;
  created_at: string;
}

export interface Task {
  id: string;
  type: string;
  status: string;
  payload: string;
  result: string;
  error: string;
  created_at: string;
  updated_at: string;
}

export interface MailConfig {
  id: string;
  enabled: boolean;
  from_name: string;
  from_address: string;
  reply_to: string;
  smtp_enabled: boolean;
  smtp_host: string;
  smtp_port: number;
  smtp_username: string;
  smtp_password_configured: boolean;
  smtp_encryption: string;
  smtp_timeout_sec: number;
  imap_enabled: boolean;
  imap_host: string;
  imap_port: number;
  imap_username: string;
  imap_password_configured: boolean;
  imap_encryption: string;
  imap_mailbox: string;
  imap_sync_minutes: number;
  created_at: string;
  updated_at: string;
}

export interface Feedback {
  id: string;
  source: string;
  category: string;
  title: string;
  content: string;
  email: string;
  status: string;
  created_at: string;
  updated_at: string;
  handled_at?: string | null;
}

export interface DashboardStats {
  total_users: number;
  total_roles: number;
  total_files: number;
  active_tasks: number;
}

interface CrudListParams {
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: string;
  keyword?: string;
  [key: string]: string | number | undefined;
}

// Generic CRUD
type CrudApi<T, ListParams extends CrudListParams = CrudListParams> = {
  list: (params?: ListParams) => Promise<PaginatedData<T>>;
  get: (id: string) => Promise<T>;
  create: (data: Partial<T>) => Promise<T>;
  update: (id: string, data: Partial<T>) => Promise<T>;
  delete: (id: string) => Promise<void>;
};

type StatusListParams = CrudListParams & { status?: number };
type FileListParams = CrudListParams & { storage_type?: string; start_time?: string; end_time?: string };
type AuditLogListParams = CrudListParams & { sort?: string; order?: string; user_id?: string; resource_type?: string; result?: string; start_time?: string; end_time?: string };
type TaskListParams = CrudListParams & { type?: string; status?: string; start_time?: string; end_time?: string };
type FeedbackListParams = CrudListParams & { source?: string; status?: string; start_time?: string; end_time?: string };

function crud<T, ListParams extends CrudListParams = CrudListParams>(resource: string): CrudApi<T, ListParams> {
  return {
    list: (params?: ListParams) => {
      const query = buildQuery(params || {});
      return request<PaginatedData<T>>(`/${resource}${query}`);
    },
    get: (id: string) => request<T>(`/${resource}/${id}`),
    create: (data: Partial<T>) =>
      request<T>(`/${resource}`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: Partial<T>) =>
      request<T>(`/${resource}/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      request(`/${resource}/${id}`, { method: 'DELETE' }),
  };
}

export const usersApi = {
  list: (params?: StatusListParams) => {
    const query = buildQuery(params || {});
    return request<PaginatedData<User>>(`/users${query}`);
  },
  get: (id: string) => request<User>(`/users/${id}`),
  create: (data: Partial<User>) =>
    request<User>(`/users`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: Partial<User>) =>
    request<User>(`/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    request(`/users/${id}`, { method: 'DELETE' }),
  resetPassword: (id: string, password: string) =>
    request(`/users/${id}/password`, {
      method: 'PUT',
      body: JSON.stringify({ password }),
    }),
  uploadAvatar: async (userId: string, file: File): Promise<{ avatar_url: string }> => {
    const formData = new FormData();
    formData.append('avatar', file);
    const token = localStorage.getItem('access_token');
    const response = await fetch(`${API_BASE}/users/${userId}/avatar`, {
      method: 'POST',
      headers: {
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        'Accept-Language': i18n.language || 'en-US',
      },
      body: formData,
    });
    if (!response.ok) {
      let msg = `Upload failed (HTTP ${response.status})`;
      let code = 50001; // Internal server error as default
      try {
        const errJson = await response.json();
        if (errJson.code !== undefined && errJson.code !== 0) {
          code = errJson.code;
          msg = resolveApiErrorMessage(errJson.code, errJson.message);
        } else if (errJson.message) {
          msg = errJson.message;
        }
      } catch { /* ignore */ }
      throw new ApiError(code, msg, response.status);
    }
    const json: ApiResponse<{ avatar_url: string }> = await response.json();
    if (json.code !== 0) {
      throw new ApiError(
        json.code,
        resolveApiErrorMessage(json.code, json.message),
        response.status,
      );
    }
    return json.data;
  },
};
export const rolesApi = {
  ...crud<Role, StatusListParams>('roles'),
  getPermissions: (id: string) => request<Permission[]>(`/roles/${id}/permissions`),
  assignPermissions: (id: string, permissionIds: string[]) =>
    request(`/roles/${id}/permissions`, {
      method: 'PUT',
      body: JSON.stringify({ permission_ids: permissionIds }),
    }),
};
export const permissionsApi = {
  tree: () => request<Permission[]>('/permissions/tree'),
  create: (data: Partial<Permission>) =>
    request<Permission>('/permissions', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: Partial<Permission>) =>
    request(`/permissions/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    request(`/permissions/${id}`, {
      method: 'DELETE',
    }),
};
export const filesApi = {
  ...crud<FileItem, FileListParams>('files'),
  upload: async (file: File, onProgress?: (pct: number) => void) => {
    const chunkSize = 5 * 1024 * 1024;
    const totalChunks = Math.ceil(file.size / chunkSize) || 1;

    const md5 = await computeMD5(file);

    const initRes = await request<{ upload_id: string }>('/files/upload/init', {
      method: 'POST',
      body: JSON.stringify({
        file_name: file.name,
        file_size: file.size,
        md5,
        total_chunks: totalChunks,
      }),
    });
    const uploadId = initRes.upload_id;

    for (let i = 0; i < totalChunks; i++) {
      const start = i * chunkSize;
      const end = Math.min(start + chunkSize, file.size);
      const chunk = file.slice(start, end);
      const formData = new FormData();
      formData.append('chunk', chunk);
      formData.append('index', String(i));
      const res = await fetch(`${API_BASE}/files/upload/${uploadId}/chunk`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
          'Accept-Language': i18n.language || 'en-US',
        },
        body: formData,
      });
      const errBody = await res.json().catch(() => null) as ApiResponse | null;
      if (!res.ok || (errBody && errBody.code !== 0)) {
        const code = errBody?.code || 50001;
        const msg = errBody?.code
          ? resolveApiErrorMessage(errBody.code, errBody.message)
          : `Chunk upload failed (chunk ${i}, HTTP ${res.status})`;
        throw new ApiError(code, msg, res.status);
      }
      onProgress?.(Math.round(((i + 1) / totalChunks) * 100));
    }

    return request<FileItem>(`/files/upload/${uploadId}/complete`, {
      method: 'POST',
    });
  },
};
export const auditLogsApi = {
  list: (params?: AuditLogListParams) => {
    const query = buildQuery(params || {});
    return request<PaginatedData<AuditLog>>(`/audit-logs${query}`);
  },
};
export const tasksApi = {
  ...crud<Task, TaskListParams>('tasks'),
  cancel: (id: string) => request<Task>(`/tasks/${id}/cancel`, { method: 'POST' }),
};

export const mailConfigApi = {
  get: () => request<MailConfig>('/system/mail-config'),
  save: (data: Partial<MailConfig> & { smtp_password?: string; imap_password?: string }) =>
    request<MailConfig>('/system/mail-config', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  testSMTP: (to: string) =>
    request('/system/mail-config/test-smtp', {
      method: 'POST',
      body: JSON.stringify({ to }),
    }),
  testIMAP: () => request('/system/mail-config/test-imap', { method: 'POST' }),
  syncIMAP: () => request<{ synced: number }>('/system/mail-config/sync-imap', { method: 'POST' }),
};

export const feedbackApi = {
  list: (params?: FeedbackListParams) => {
    const query = buildQuery(params || {});
    return request<PaginatedData<Feedback>>(`/feedback${query}`);
  },
  create: (data: { category?: string; title: string; content: string }) =>
    request<Feedback>('/feedback', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  updateStatus: (id: string, status: string) =>
    request<Feedback>(`/feedback/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    }),
};

export const dashboardApi = {
  stats: () => request<DashboardStats>('/dashboard/stats'),
};
