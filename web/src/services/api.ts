import i18n from '../i18n';

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
    throw new Error(json.message || '请求失败');
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
};

// Types
export interface Menu {
  id: string;
  name: string;
  code: string;
  path: string;
  icon: string;
  sort_order: number;
  children?: Menu[];
}

export interface User {
  id: string;
  username: string;
  display_name: string;
  email: string;
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
  mime_type: string;
  size: number;
  created_at: string;
}

export interface AuditLog {
  id: string;
  user_id: string | null;
  username: string;
  action: string;
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

export interface DashboardStats {
  total_users: number;
  total_roles: number;
  total_files: number;
  active_tasks: number;
}

// Generic CRUD
function crud<T>(resource: string) {
  return {
    list: (params?: {
      page?: number;
      page_size?: number;
      sort_by?: string;
      sort_order?: string;
      [key: string]: string | number | undefined;
    }) => {
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

export const usersApi = crud<User>('users');
export const rolesApi = {
  ...crud<Role>('roles'),
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
  ...crud<FileItem>('files'),
  upload: async (file: File, onProgress?: (pct: number) => void) => {
    // Init upload
    const initRes = await request<{ upload_id: string }>('/files/upload/init', {
      method: 'POST',
      body: JSON.stringify({ name: file.name, size: file.size, mime_type: file.type }),
    });
    const uploadId = initRes.upload_id;
    const chunkSize = 5 * 1024 * 1024; // 5MB
    const totalChunks = Math.ceil(file.size / chunkSize);

    for (let i = 0; i < totalChunks; i++) {
      const start = i * chunkSize;
      const end = Math.min(start + chunkSize, file.size);
      const chunk = file.slice(start, end);
      const formData = new FormData();
      formData.append('chunk', chunk);
      formData.append('index', String(i));
      await fetch(`${API_BASE}/files/upload/${uploadId}/chunk`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
          'Accept-Language': i18n.language || 'en-US',
        },
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
  list: (params?: {
    page?: number;
    page_size?: number;
    sort?: string;
    order?: string;
    user_id?: string;
    action?: string;
    resource_type?: string;
  }) => {
    const query = buildQuery(params || {});
    return request<PaginatedData<AuditLog>>(`/audit-logs${query}`);
  },
};
export const tasksApi = {
  ...crud<Task>('tasks'),
  cancel: (id: string) => request<Task>(`/tasks/${id}/cancel`, { method: 'POST' }),
};

export const dashboardApi = {
  stats: () => request<DashboardStats>('/dashboard/stats'),
};
