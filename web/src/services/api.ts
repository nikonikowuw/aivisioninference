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

/**
 * 转换后端返回的文件路径为可访问的完整 URL
 */
export function getFileUrl(path?: string): string {
  if (!path) return '';
  if (path.startsWith('http')) return path;
  
  const baseUrl = import.meta.env.VITE_API_URL || '';
  // 确保路径以 / 开头
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${baseUrl}${normalizedPath}`;
}

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
  return typeof value === 'object' && value !== null && typeof (value as Record<string, unknown>)?.code === 'number';
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

export async function request<T>(
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
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') sp.set(k, String(v));
  }
  return sp.toString() ? `?${sp}` : '';
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

/** 处理文件下载响应：校验错误、创建 Blob 并触发浏览器下载 */
async function handleDownloadResponse(response: Response, filename: string): Promise<void> {
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

async function downloadFile(path: string, filename: string): Promise<void> {
  const headers = authHeaders({ Accept: 'text/csv, application/octet-stream, application/json' });
  const response = await fetchApi(path, { headers });
  return handleDownloadResponse(response, filename);
}

/** POST 下载文件（用于带 body 的导出场景） */
async function downloadFilePost(path: string, filename: string, body: unknown): Promise<void> {
  const headers = authHeaders({
    Accept: 'text/csv, application/octet-stream, application/json',
    'Content-Type': 'application/json',
  });
  const response = await fetchApi(path, { headers, method: 'POST', body: JSON.stringify(body) });
  return handleDownloadResponse(response, filename);
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
  permission_codes: string[];
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
  path?: string;
  mime_type: string;
  size: number;
  created_at: string;
}

export interface AuditLog {
  id: string;
  user_id: string | null;
  username: string;
  action_type: string;
  action_type_label: string;
  resource_type: string;
  resource_type_label: string;
  resource_id: string;
  request_path: string;
  request_method: string;
  request_method_label: string;
  request_ip: string;
  user_agent: string;
  response_status: number;
  duration_ms: number;
  result_summary: string;
  result_summary_label: string;
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

export type SmartRecordType = 'recognition' | 'alarm' | 'capture';

export interface SmartRecord {
  record_id: string;
  record_type: SmartRecordType;
  capture_time: string;
  task_id?: string | null;
  task_name?: string;
  device_id?: string | null;
  device_name?: string;
  algorithm_name?: string;
  algorithm_version?: string;
  category_code?: number | null;
  category_name?: string;
  confidence?: number | null;
  person_record_id?: string | null;
  person_name?: string;
  similarity?: number | null;
  identity_id?: string;
  alarm_type?: string;
  alarm_level?: string;
  alarm_status?: 'unhandled' | 'handled';
  alarm_major?: string;
  snapshot_image_url?: string;
  target_crop_url?: string;
  background_image_url?: string;
  person_image_url?: string;
  raw_result?: unknown;
  created_at: string;
}

export interface SmartRecordListParams extends CrudListParams {
  type?: SmartRecordType;
  device_id?: string;
  group_id?: string;
  task_id?: string;
  device_name?: string;
  task_name?: string;
  alarm_type?: string;
  alarm_level?: string;
  alarm_status?: string;
  person_name?: string;
  business_tag?: string;
  category_code?: string | number;
  min_confidence?: string | number;
  max_confidence?: string | number;
  min_similarity?: string | number;
  max_similarity?: string | number;
  start_time?: string;
  end_time?: string;
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

type StatusListParams = CrudListParams & { status?: number; ids?: string };
type FileListParams = CrudListParams & { storage_type?: string; start_time?: string; end_time?: string; ids?: string };
type AuditLogListParams = CrudListParams & { sort?: string; order?: string; user_id?: string; resource_type?: string; result?: string; start_time?: string; end_time?: string; ids?: string };
type TaskListParams = CrudListParams & { type?: string; status?: string; start_time?: string; end_time?: string; ids?: string };
type FeedbackListParams = CrudListParams & { source?: string; status?: string; start_time?: string; end_time?: string; ids?: string };

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

export const smartRecordsApi = {
  list: (params?: SmartRecordListParams) =>
    request<PaginatedData<SmartRecord>>(`/smart-records${buildQuery(params || {})}`),
  exportCsv: (params?: SmartRecordListParams) =>
    downloadFile(`/smart-records/export${buildQuery(params || {})}`, 'smart-records.csv'),
  batchDelete: (ids: string[]) =>
    request<BatchResult>('/smart-records/batch-delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  exportSelected: (ids: string[]) =>
    downloadFilePost('/smart-records/export-selected', 'smart-records-selected.csv', { ids }),
  updateAlarmStatus: (id: string, status: 'unhandled' | 'handled') =>
    request<void>(`/smart-records/${id}/alarm-status`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    }),
  listCategoryCodes: () =>
    request<CategoryCodeOption[]>('/smart-records/category-codes'),
};

export interface CategoryCodeOption {
  value: number;
  label: string;
}

// Algorithm Package types
export interface AlgorithmPackage {
  id: string;
  algorithm_name: string;
  algorithm_alias?: string;
  version: string;
  domain: string;
  result_schema: string;
  capabilities_image?: string[];
  capabilities_data?: string[];
  hardware?: string[];
  description?: string;
  package_path: string;
  extract_path: string;
  package_size?: number;
  package_md5?: string;
  so_path: string;
  ai_params_schema?: unknown;
  self_check_status: string;
  self_check_result?: unknown;
  self_check_at?: string;
  status: string;
  ref_count: number;
  is_current: boolean;
  remark?: string;
  created_at: string;
  updated_at: string;
  created_by?: string;
  updated_by?: string;
}

export interface AlgorithmPackageListParams extends CrudListParams {
  status?: string;
  domain?: string;
  self_check_status?: string;
}

export const algorithmPackagesApi = {
  ...crud<AlgorithmPackage, AlgorithmPackageListParams>('algorithmpackages'),
  upload: async (file: File, onProgress?: (pct: number) => void): Promise<AlgorithmPackage> => {
    void onProgress;
    const headers = authHeaders();
    const formData = new FormData();
    formData.append('file', file);
    const response = await fetch(`${API_BASE}/algorithmpackages/upload`, {
      method: 'POST',
      headers,
      body: formData,
    });
    redirectOnUnauthorized(response);
    const json = await parseApiResponse<AlgorithmPackage>(response);
    if (json.code !== 0) {
      throw new ApiError(
        json.code,
        resolveApiErrorMessage(json.code, json.message),
        response.status,
      );
    }
    return json.data;
  },
  batchDelete: (ids: string[]) =>
    request<BatchResult>('/algorithmpackages/batch-delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  exportCsv: (params?: AlgorithmPackageListParams) =>
    downloadFile(`/algorithmpackages/export${buildQuery(params || {})}`, 'algorithm-packages.csv'),
};

// algoPackagesApi 别名（兼容旧导入）
export const algoPackagesApi = algorithmPackagesApi;

// Region types for AI vision tasks
export interface ROIRegion {
  id: string;
  type: 'polygon' | 'rect';
  label?: string;
  points?: number[][]; // normalized 0-1 coordinates [[x, y], ...]
  x?: number;
  y?: number;
  width?: number;
  height?: number;
}

export type MarkRegion = ROIRegion;

export interface LineRegion {
  id: string;
  label?: string;
  start: [number, number]; // normalized [x, y]
  end: [number, number]; // normalized [x, y]
  direction?: 'both' | 'in' | 'out';
}

// AI Time Schedule types
export interface AITimeSchedule {
  id: string;
  name: string;
  description?: string;
  start_date: string;
  end_date: string;
  time_windows: { start: string; end: string }[];
  created_at: string;
  updated_at: string;
}

export interface AITimeScheduleListParams extends CrudListParams {}

export const aiTimeSchedulesApi = {
  ...crud<AITimeSchedule, AITimeScheduleListParams>('ai-time-schedules'),
  listAll: () =>
    request<AITimeSchedule[]>('/ai-time-schedules/all'),
  batchDelete: (ids: string[]) =>
    request<BatchResult>('/ai-time-schedules/batch-delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
};

// AI Vision Task types
export interface AIVisionTask {
  id: string;
  name: string;
  status: string;
  schedule_id: string;
  device_channel_id: string;
  algo_package_id: string;
  target_node_id: string;
  start_date: string;
  end_date: string;
  time_windows: { start: string; end: string }[];
  ai_params?: Record<string, unknown>;
  roi_regions?: ROIRegion[];
  mark_regions?: MarkRegion[];
  line_regions?: LineRegion[];
  error_reason?: string;
  created_at: string;
  updated_at: string;
}

export interface AIVisionTaskListParams extends CrudListParams {
  status?: string;
}

export interface AIVisionTaskConflictCheck {
  target_node_id: string;
  start_date: string;
  end_date: string;
  time_windows: { start: string; end: string }[];
  exclude_task_id?: string;
}

export const aiVisionTasksApi = {
  ...crud<AIVisionTask, AIVisionTaskListParams>('ai-tasks'),
  checkConflict: (data: AIVisionTaskConflictCheck) =>
    request<{ conflict: boolean; message?: string }>('/ai-tasks/check-conflict', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  restart: (id: string) =>
    request<null>(`/ai-tasks/${id}/restart`, {
      method: 'POST',
    }),
  batchDelete: (ids: string[]) =>
    request<BatchResult>('/ai-tasks/batch-delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
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

export interface SystemInfo {
  device_model: string;
  deploy_location: string;
  description?: string;
}

export const systemInfoApi = {
  get: () => request<SystemInfo>('/system/info'),
  save: (data: SystemInfo) =>
    request<null>('/system/info', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
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

// Stream types
export interface ConsumerInfo {
  reason: string;
  ref_at: string;
  metadata?: Record<string, string>;
  last_alive: string;
}

export interface StreamState {
  device_id: string;
  app: string;
  stream: string;
  vhost: string;
  schema: string;
  ref_count: number;
  status: string; // inactive, pulling, active, error
  source_url: string;
  retry_count: number;
  retry_at?: string;
  started_at?: string;
  consumers?: Record<string, ConsumerInfo>;
  play_url_rtsp?: string;
  play_url_rtmp?: string;
  play_url_flv?: string;
  play_url_webrtc?: string;
  play_url_hls?: string;
}

export const mediaApi = {
  listStreams: () => request<StreamState[]>('/media/streams'),
  getPlayUrl: (params: { device_id: string; protocol?: string; stream_type?: string }) => {
    const query = buildQuery(params);
    return request<{ url: string; protocol: string; stream_type: string; expires: number }>(`/media/play${query}`);
  },
  stopPlay: (deviceId: string) =>
    request(`/media/stop?device_id=${deviceId}`, { method: 'POST' }),
  getSnapshot: (deviceId: string) => `${API_BASE}/media/snapshot?device_id=${deviceId}&token=${getAccessToken()}`,
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

// Device types
export interface Device {
  id: string;
  device_name: string;
  access_type: string;
  rtsp_url?: string;
  gb28181_device_id?: string;
  gb28181_channel_id?: string;
  username?: string;
  manufacturer?: string;
  model?: string;
  firmware_version?: string;
  status: string;
  enabled: boolean;
  latitude?: number;
  longitude?: number;
  location_desc?: string;
  last_online_at?: string;
  last_offline_at?: string;
  last_error_code?: string;
  last_error_message?: string;
  external_key?: string;
  remark?: string;
  version: number;
  groups?: DeviceGroup[];
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface DeviceGroup {
  id: string;
  group_name: string;
  description?: string;
  parent_id?: string;
  sort_order: number;
  device_count: number;
  created_at: string;
  updated_at: string;
}

export interface DeviceTestResult {
  success: boolean;
  message: string;
  tested_at: string;
}

type DeviceListParams = CrudListParams & { status?: string; access_type?: string; group_id?: string };

export const devicesApi = {
  ...crud<Device, DeviceListParams>('devices'),
  batchDelete: (ids: string[]) =>
    request<BatchResult>('/devices/batch-delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  exportCsv: (params?: DeviceListParams) =>
    downloadFile(`/devices/export${buildQuery(params || {})}`, 'devices.csv'),
  importCsv: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return request<BatchResult>('/devices/import', {
      method: 'POST',
      body: formData,
    });
  },
  test: (id: string) =>
    request<DeviceTestResult>(`/devices/${id}/test`, { method: 'POST' }),
};

export const deviceGroupsApi = {
  ...crud<DeviceGroup>('device-groups'),
};

// DiscoveredDevice types
export interface DiscoveredDevice {
  id: string;
  source: string; // onvif/gb28181/nvr/scan
  device_name: string;
  device_ip: string;
  device_mac: string;
  manufacturer: string;
  model: string;
  firmware_version: string;
  access_type: string; // rtsp/gb28181/nvr_channel
  access_url: string;
  gb28181_code: string;
  nvr_device_id?: string;
  extra_info?: Record<string, unknown>;
  status: string; // pending/imported/ignored/expired
  imported_at?: string;
  ignored_at?: string;
  matched_device_id?: string;
  created_at: string;
  updated_at: string;
}

export interface DiscoveredDeviceListParams extends CrudListParams {
  source?: string;
  status?: string;
}

export const deviceStagingApi = {
  list: (params?: DiscoveredDeviceListParams) => {
    const query = buildQuery(params || {});
    return request<PaginatedData<DiscoveredDevice>>(`/device-staging${query}`);
  },
  import: (id: string, data: { username: string; password: string; device_name?: string; enable_infer?: boolean }) =>
    request(`/device-staging/${id}/import`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  ignore: (id: string) =>
    request(`/device-staging/${id}/ignore`, { method: 'POST' }),
  batchImport: (ids: string[], data: { username: string; password: string; enable_infer?: boolean }) =>
    request('/device-staging/batch-import', {
      method: 'POST',
      body: JSON.stringify({ ids, ...data }),
    }),
  batchIgnore: (ids: string[]) =>
    request('/device-staging/batch-ignore', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  scanONVIF: (networkInterface?: string) =>
    request('/device-staging/scan-onvif', {
      method: 'POST',
      body: JSON.stringify({ interface: networkInterface || 'eth0' }),
    }),
};

export interface FingerprintResponse {
  device_sn: string;
  fingerprint: string;
  hash_algorithm: string;
}

export interface LicenseInfo {
  id: string;
  license_id: string;
  license_type: string;
  device_sn: string;
  device_fingerprint: string;
  algorithms: string[];
  max_streams: number;
  features: string[];
  not_before: string;
  not_after: string | null;
  expire_action: string;
  status: string;
  remaining_days: number | null;
  created_at: string;
  updated_at: string;
}

export interface LicenseListParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  status?: string;
  [key: string]: string | number | undefined;
}

export const licenseApi = {
  getFingerprint: () =>
    request<FingerprintResponse>('/license/fingerprint'),
  upload: async (file: File): Promise<LicenseInfo> => {
    const formData = new FormData();
    formData.append('file', file);
    return request<LicenseInfo>('/license/upload', {
      method: 'POST',
      body: formData,
    });
  },
  getActive: () =>
    request<LicenseInfo>('/license/active'),
  list: (params?: LicenseListParams) => {
    const query = buildQuery(params || {});
    return request<PaginatedData<LicenseInfo>>(`/license${query}`);
  },
  check: (algorithm: string) =>
    request<{ authorized: boolean; algorithm: string }>(`/license/check${buildQuery({ algorithm })}`),
};

// Person Management
export interface Person {
  id: string;
  person_code: string;
  person_name: string;
  gender: string;
  phone?: string;
  id_number?: string;
  image_url: string;
  image_md5?: string;
  face_quality_score?: number;
  embedding_status: string;
  embedding_error_code?: string;
  embedding_error_message_key?: string;
  embedding_retryable: boolean;
  enabled: boolean;
  remark?: string;
  groups?: PersonGroup[];
  created_at: string;
  updated_at: string;
}

export interface PersonGroup {
  id: string;
  group_name: string;
  description?: string;
  parent_id?: string;
  sort_order: number;
  person_count: number;
  children?: PersonGroup[];
  created_at: string;
  updated_at: string;
}

export interface PersonImportTask {
  id: string;
  task_type: string;
  file_name: string;
  file_url?: string;
  total_rows: number;
  success_rows: number;
  failed_rows: number;
  fail_detail_url?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

type PersonListParams = CrudListParams & {
  group_id?: string;
  embedding_status?: string;
  enabled?: string;
  start_time?: string;
  end_time?: string;
};

export const personsApi = {
  list: (params?: PersonListParams) => {
    const query = buildQuery(params || {});
    return request<PaginatedData<Person>>(`/persons${query}`);
  },
  get: (id: string) => request<Person>(`/persons/${id}`),
  /** 创建人员。支持 FormData（直传）或 JSON with image_url（分片上传后关联） */
  create: (data: FormData | Record<string, unknown>) => {
    if (data instanceof FormData) {
      return request<Person>('/persons', { method: 'POST', body: data });
    }
    return request<Person>('/persons', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
  },
  /** 更新人员。支持 FormData（直传）或 JSON with image_url（分片上传后关联） */
  update: (id: string, data: FormData | Record<string, unknown>) => {
    if (data instanceof FormData) {
      return request<Person>(`/persons/${id}`, { method: 'PUT', body: data });
    }
    return request<Person>(`/persons/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
  },
  delete: (id: string) =>
    request(`/persons/${id}`, { method: 'DELETE' }),
  batchDelete: (ids: string[]) =>
    request('/persons/batch-delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  batchToggle: (ids: string[], enabled: boolean) =>
    request('/persons/batch-toggle', {
      method: 'POST',
      body: JSON.stringify({ ids, enabled }),
    }),
  retryEmbedding: (id: string) =>
    request(`/persons/${id}/retry-embedding`, { method: 'POST' }),
  batchRetryEmbedding: (ids: string[]) =>
    request('/persons/batch-retry-embedding', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  exportExcel: (params?: PersonListParams) =>
    downloadFile(`/persons/export${buildQuery(params || {})}`, 'persons.xlsx'),
};

export const personGroupsApi = {
  list: () => request<PersonGroup[]>('/person-groups'),
  create: (data: Partial<PersonGroup>) =>
    request<PersonGroup>('/person-groups', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: Partial<PersonGroup>) =>
    request<PersonGroup>(`/person-groups/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    request(`/person-groups/${id}`, { method: 'DELETE' }),
};

// PersonTag types and API
export interface PersonTag {
  id: string;
  tag_name: string;
  color: string;
  sort_order: number;
  person_count: number;
  created_at: string;
  updated_at: string;
}

export const personTagsApi = {
  list: () => request<PersonTag[]>('/person-tags'),
  create: (data: Partial<PersonTag>) =>
    request<PersonTag>('/person-tags', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: Partial<PersonTag>) =>
    request<PersonTag>(`/person-tags/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    request(`/person-tags/${id}`, { method: 'DELETE' }),
};

export const personImportsApi = {
  list: (params?: CrudListParams) => {
    const query = buildQuery(params || {});
    return request<PaginatedData<PersonImportTask>>(`/person-import-tasks${query}`);
  },
  get: (id: string) => request<PersonImportTask>(`/person-import-tasks/${id}`),
  create: (file: File, overwrite?: boolean) => {
    const formData = new FormData();
    formData.append('file', file);
    if (overwrite) formData.append('overwrite_on_duplicate', 'true');
    return request<PersonImportTask>('/person-import-tasks', {
      method: 'POST',
      body: formData,
    });
  },
  /** 通过已上传文件 URL 创建导入任务（分片上传后调用） */
  createByUrl: (fileUrl: string, overwrite?: boolean) =>
    request<PersonImportTask>('/person-import-tasks/by-url', {
      method: 'POST',
      body: JSON.stringify({ file_url: fileUrl, overwrite_on_duplicate: overwrite ?? false }),
    }),
};

/**
 * 并行分片上传文件，返回后端文件路径。
 * 复用 /files/upload/init → /chunk → /complete 基础设施。
 *
 * @param file        待上传文件
 * @param concurrency 并行上传分片数，默认 3
 * @param onProgress  进度回调 (0-100)
 */
export async function chunkedUpload(
  file: File,
  concurrency = 3,
  onProgress?: (pct: number) => void,
): Promise<string> {
  const chunkSize = 5 * 1024 * 1024; // 5MB
  const totalChunks = Math.ceil(file.size / chunkSize) || 1;
  const md5 = await computeMD5(file);

  // 1. 初始化上传会话
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

  // 2. 并行上传分片（分批 Promise.all 实现并发控制）
  let completedChunks = 0;
  const chunks = Array.from({ length: totalChunks }, (_, i) => i);

  const uploadChunk = async (index: number) => {
    const start = index * chunkSize;
    const end = Math.min(start + chunkSize, file.size);
    const chunk = file.slice(start, end);
    const formData = new FormData();
    formData.append('chunk', chunk);
    formData.append('index', String(index));
    await request(`/files/upload/${uploadId}/chunk`, {
      method: 'POST',
      body: formData,
    });
    completedChunks++;
    onProgress?.(Math.round((completedChunks / totalChunks) * 100));
  };

  for (let i = 0; i < chunks.length; i += concurrency) {
    const batch = chunks.slice(i, i + concurrency);
    await Promise.all(batch.map(uploadChunk));
  }

  // 3. 合并分片并获取文件记录
  const fileRecord = await request<FileItem>(`/files/upload/${uploadId}/complete`, {
    method: 'POST',
  });

  // 返回文件存储路径
  return fileRecord.path || `uploads/${fileRecord.name}`;
}
