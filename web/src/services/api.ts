import i18n from '../i18n';
import SparkMD5 from 'spark-md5';

const SERVER_ERROR_FALLBACK = 'Server error';

function isMissingTranslation(result: string, key: string): boolean {
  const [, keyWithoutNamespace = key] = key.split(':');
  return result === key || result === keyWithoutNamespace;
}

function getServerErrorMessage(): string {
  const key = 'common:message.serverError';
  const msg = i18n.t(key);
  return isMissingTranslation(msg, key) ? SERVER_ERROR_FALLBACK : msg;
}

export function getErrorMessage(code: number): string {
  if (code === 0) return '';
  const key = `common:message.error.${code}`;
  const msg = i18n.t(key);
  return isMissingTranslation(msg, key) ? getServerErrorMessage() : msg;
}

function resolveApiErrorMessage(code: number, backendMessage?: string): string {
  const trimmedBackendMessage = backendMessage?.trim();
  if (!trimmedBackendMessage) return getErrorMessage(code);

  // 参数校验错误和前端未知错误优先展示后端具体文案，避免丢失上下文。
  const key = `common:message.error.${code}`;
  const translatedMessage = i18n.t(key);
  if (code === 10001 || isMissingTranslation(translatedMessage, key)) return trimmedBackendMessage;
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

  isUnauthenticated(): boolean {
    return this.status === 401 || (this.code >= 20001 && this.code <= 20004);
  }
}

function computeMD5(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunkSize = 2 * 1024 * 1024;
    const chunks = Math.ceil(file.size / chunkSize) || 1;
    const spark = new SparkMD5.ArrayBuffer();
    const reader = new FileReader();
    let current = 0;

    const loadNext = () => {
      const start = current * chunkSize;
      const end = Math.min(start + chunkSize, file.size);
      reader.readAsArrayBuffer(file.slice(start, end));
    };

    reader.onload = (e) => {
      if (e.target?.result) spark.append(e.target.result as ArrayBuffer);
      if (++current < chunks) {
        loadNext();
      } else {
        resolve(spark.end());
      }
    };
    reader.onerror = () => reject(reader.error);
    loadNext();
  });
}

const API_BASE = '/api/v1';
const ACCESS_TOKEN_KEY = 'access_token';

export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_KEY) || sessionStorage.getItem(ACCESS_TOKEN_KEY);
}

export function setAccessToken(token: string, rememberMe: boolean): void {
  clearAccessToken();
  const storage = rememberMe ? localStorage : sessionStorage;
  storage.setItem(ACCESS_TOKEN_KEY, token);
}

export function clearAccessToken(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  sessionStorage.removeItem(ACCESS_TOKEN_KEY);
}

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
  const token = getAccessToken();
  const headers = new Headers(options.headers);

  if (!headers.has('Accept-Language')) {
    headers.set('Accept-Language', i18n.language || 'en-US');
  }

  // Only set Content-Type to JSON if not explicitly provided and body is not FormData
  if (!(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers,
    });
  } catch (err) {
    // Network error or fetch failure
    throw new ApiError(50001, i18n.t('common:message.networkError'), 0);
  }

  if (response.status === 401 && !window.location.pathname.startsWith('/auth/')) {
    clearAccessToken();
    window.location.href = '/auth/sign-in';
    throw new ApiError(401, getErrorMessage(401), 401);
  }

  let json: ApiResponse<T>;
  const text = await response.text();
  try {
    json = text ? JSON.parse(text) : { code: 0, message: '', data: null };
  } catch {
    throw new ApiError(50001, getErrorMessage(50001), response.status);
  }

  if (json.code !== 0) {
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
    return request<{ avatar_url: string }>('/auth/avatar', {
      method: 'POST',
      body: formData,
    });
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
  ...crud<User, StatusListParams>('users'),
  resetPassword: (id: string, password: string) =>
    request(`/users/${id}/password`, {
      method: 'PUT',
      body: JSON.stringify({ password }),
    }),
  uploadAvatar: async (userId: string, file: File): Promise<{ avatar_url: string }> => {
    const formData = new FormData();
    formData.append('avatar', file);
    return request<{ avatar_url: string }>(`/users/${userId}/avatar`, {
      method: 'POST',
      body: formData,
    });
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
      
      await request(`/files/upload/${uploadId}/chunk`, {
        method: 'POST',
        body: formData,
      });
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
