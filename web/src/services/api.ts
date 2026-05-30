import SparkMD5 from 'spark-md5';
import i18n from '../i18n';

const SERVER_ERROR_FALLBACK = 'Server error';

function isMissingTranslation(result: string, key: string): boolean {
  const [, keyWithoutNamespace = key] = key.split(':');
  return result === key || result === keyWithoutNamespace;
}

function getServerErrorMessage(): string {
  const key = 'common:message.serverError';
  const message = i18n.t(key);
  return isMissingTranslation(message, key) ? SERVER_ERROR_FALLBACK : message;
}

export function getErrorMessage(code: number): string {
  if (code === 0) return '';
  const key = `common:message.error.${code}`;
  const message = i18n.t(key);
  return isMissingTranslation(message, key) ? getServerErrorMessage() : message;
}

function resolveApiErrorMessage(code: number, backendMessage?: string): string {
  const trimmedBackendMessage = backendMessage?.trim();
  if (!trimmedBackendMessage) return getErrorMessage(code);

  // 参数校验错误和未知错误优先展示后端具体文案，避免丢失上下文
  const key = `common:message.error.${code}`;
  const translated = i18n.t(key);
  if (code === 10001 || isMissingTranslation(translated, key)) return trimmedBackendMessage;
  return translated;
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

function isApiResponseLike(value: unknown): value is ApiResponse<unknown> {
  return typeof value === 'object' && value !== null && typeof (value as { code?: unknown }).code === 'number';
}

function isJsonResponse(response: Response): boolean {
  return (response.headers.get('Content-Type') || '').toLowerCase().includes('json');
}

interface PaginatedData<T> {
  list: T[];
  total: number;
  page: number;
  page_size: number;
}

function authHeaders(base?: HeadersInit): Headers {
  const headers = new Headers(base);
  if (!headers.has('Accept-Language')) {
    headers.set('Accept-Language', i18n.language || 'en-US');
  }
  const token = getAccessToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  return headers;
}

async function fetchApi(path: string, options: RequestInit = {}): Promise<Response> {
  try {
    return await fetch(`${API_BASE}${path}`, options);
  } catch {
    throw new ApiError(50001, i18n.t('common:message.networkError'), 0);
  }
}

function redirectOnUnauthorized(response: Response): void {
  if (response.status !== 401 || window.location.pathname.startsWith('/auth/')) return;
  clearAccessToken();
  window.location.href = '/auth/sign-in';
  throw new ApiError(401, getErrorMessage(401), 401);
}

async function parseApiResponse<T>(response: Response): Promise<ApiResponse<T>> {
  const text = await response.text();
  try {
    const json = text ? JSON.parse(text) : { code: 0, message: '', data: null };
    if (!isApiResponseLike(json)) throw new Error('invalid api response');
    return json as ApiResponse<T>;
  } catch {
    throw new ApiError(50001, getErrorMessage(50001), response.status);
  }
}

async function parseOptionalApiResponse(response: Response): Promise<ApiResponse<unknown> | null> {
  if (!isJsonResponse(response)) return null;
  const text = await response.text();
  try {
    const json = text ? JSON.parse(text) : null;
    return isApiResponseLike(json) ? json : null;
  } catch (err) {
    console.warn('Failed to parse API error response', err);
    return null;
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers = authHeaders(options.headers);

  if (!(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetchApi(path, { ...options, headers });
  redirectOnUnauthorized(response);

  const json = await parseApiResponse<T>(response);
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

function filenameFromContentDisposition(header: string | null): string | null {
  if (!header) return null;
  const utf8Match = header.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) return sanitizeDownloadFilename(decodeURIComponent(utf8Match[1]));
  const asciiMatch = header.match(/filename="?([^";]+)"?/i);
  if (asciiMatch?.[1]) return sanitizeDownloadFilename(asciiMatch[1]);
  return null;
}

function sanitizeDownloadFilename(filename: string): string {
  const cleaned = filename.replace(/[\\/\r\n]/g, '_').trim();
  return cleaned || 'download.csv';
}

async function downloadFile(path: string, filename: string): Promise<void> {
  const headers = authHeaders({ Accept: 'text/csv, application/octet-stream, application/json' });
  const response = await fetchApi(path, { headers });
  redirectOnUnauthorized(response);

  const json = await parseOptionalApiResponse(response.clone());
  if (json) {
    const code = json.code === 0 ? 50001 : json.code;
    throw new ApiError(code, resolveApiErrorMessage(code, json.message), response.status);
  }

  if (!response.ok) {
    throw new ApiError(response.status, getServerErrorMessage(), response.status);
  }

  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filenameFromContentDisposition(response.headers.get('Content-Disposition')) || filename;
  document.body.appendChild(link);
  try {
    link.click();
  } finally {
    link.remove();
    window.URL.revokeObjectURL(url);
  }
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

export interface BrandConfig {
  id: string;
  system_name: string;
  logo_url: string;
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

export interface DashboardUserStat {
  date: string;
  new: number;
  active: number;
}

export interface DashboardAuditLog {
  username: string;
  action: string;
  method: string;
  created_at: string;
}

export interface DashboardStats {
  total_users: number;
  total_roles: number;
  total_files: number;
  total_tasks: number;
  active_tasks: number;
  user_stats: DashboardUserStat[];
  audit_logs: DashboardAuditLog[];
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

export interface BatchItemResult {
  id: string;
  success: boolean;
  code?: number;
  message?: string;
}

export interface BatchResult {
  total: number;
  success: number;
  failed: number;
  items: BatchItemResult[];
}

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
  batchDelete: (ids: string[]) =>
    request<BatchResult>('/users/batch-delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  batchUpdateStatus: (ids: string[], status: number) =>
    request<BatchResult>('/users/batch-status', {
      method: 'PUT',
      body: JSON.stringify({ ids, status }),
    }),
  exportCsv: (params?: StatusListParams) =>
    downloadFile(`/users/export${buildQuery(params || {})}`, 'users.csv'),
  importCsv: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return request<BatchResult>('/users/import', {
      method: 'POST',
      body: formData,
    });
  },
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
  batchDelete: (ids: string[]) =>
    request<BatchResult>('/roles/batch-delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  exportCsv: (params?: StatusListParams) =>
    downloadFile(`/roles/export${buildQuery(params || {})}`, 'roles.csv'),
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
  batchDelete: (ids: string[]) =>
    request<BatchResult>('/files/batch-delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  exportCsv: (params?: FileListParams) =>
    downloadFile(`/files/export${buildQuery(params || {})}`, 'files.csv'),
  download: (id: string, filename: string) =>
    downloadFile(`/files/${id}/download`, filename),
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
  exportCsv: (params?: AuditLogListParams) =>
    downloadFile(`/audit-logs/export${buildQuery(params || {})}`, 'audit-logs.csv'),
};
export const tasksApi = {
  ...crud<Task, TaskListParams>('tasks'),
  batchCancel: (ids: string[]) =>
    request<BatchResult>('/tasks/batch-cancel', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  exportCsv: (params?: TaskListParams) =>
    downloadFile(`/tasks/export${buildQuery(params || {})}`, 'tasks.csv'),
  cancel: (id: string) => request<Task>(`/tasks/${id}/cancel`, { method: 'POST' }),
};

export const brandConfigApi = {
  get: () => request<BrandConfig>('/system/brand-config'),
  save: (data: Pick<BrandConfig, 'system_name' | 'logo_url'>) =>
    request<BrandConfig>('/system/brand-config', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  uploadLogo: async (file: File): Promise<{ logo_url: string }> => {
    const formData = new FormData();
    formData.append('logo', file);
    return request<{ logo_url: string }>('/system/brand-config/logo', {
      method: 'POST',
      body: formData,
    });
  },
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
  batchUpdateStatus: (ids: string[], status: string) =>
    request<BatchResult>('/feedback/batch-status', {
      method: 'PUT',
      body: JSON.stringify({ ids, status }),
    }),
  exportCsv: (params?: FeedbackListParams) =>
    downloadFile(`/feedback/export${buildQuery(params || {})}`, 'feedback.csv'),
};

export const dashboardApi = {
  stats: () => request<DashboardStats>('/dashboard/stats'),
};
