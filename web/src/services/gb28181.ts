import { request } from './api';

// ============= GB28181 设备管理 =============

export interface GB28181Device {
  id: string;
  device_code: string;
  register_address: string;
  register_port: number;
  sip_id: string;
  sip_domain: string;
  last_register_at: string | null;
  last_heartbeat_at: string | null;
  last_catalog_at: string | null;
  heartbeat_interval: number;
  status: string;
  channel_count: number;
  manufacturer: string;
  model: string;
  firmware: string;
  created_at: string;
  updated_at: string;
}

export interface GB28181DeviceUpdateRequest {
  sip_id?: string;
  sip_domain?: string;
  sip_password?: string;
  heartbeat_interval?: number;
}

export interface CatalogTaskResponse {
  task_id: string;
}

export interface CatalogTaskStatusResponse {
  task_id: string;
  status: 'pending' | 'completed' | 'failed';
  channel_count: number;
  error?: string;
}

export interface GB28181DeviceListParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  status?: string;
}

/** GB28181 设备列表 */
export function listGB28181Devices(params: GB28181DeviceListParams) {
  const sp = new URLSearchParams();
  if (params.page) sp.set('page', String(params.page));
  if (params.page_size) sp.set('page_size', String(params.page_size));
  if (params.keyword) sp.set('keyword', params.keyword);
  if (params.status) sp.set('status', params.status);
  const qs = sp.toString() ? `?${sp}` : '';
  return request<{ list: GB28181Device[]; total: number; page: number; page_size: number }>(
    `/api/v1/gb28181/devices${qs}`,
  );
}

/** GB28181 设备详情 */
export function getGB28181Device(id: string) {
  return request<GB28181Device>(`/api/v1/gb28181/devices/${id}`);
}

/** 更新 GB28181 设备 */
export function updateGB28181Device(id: string, data: GB28181DeviceUpdateRequest) {
  return request<GB28181Device>(`/api/v1/gb28181/devices/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

/** 删除 GB28181 设备 */
export function deleteGB28181Device(id: string) {
  return request<null>(`/api/v1/gb28181/devices/${id}`, { method: 'DELETE' });
}

/** 触发目录查询 */
export function triggerCatalog(id: string) {
  return request<CatalogTaskResponse>(`/api/v1/gb28181/devices/${id}/catalog`, { method: 'POST' });
}

/** 查询目录任务状态 */
export function getCatalogTaskStatus(taskId: string) {
  return request<CatalogTaskStatusResponse>(`/api/v1/gb28181/catalog-tasks/${taskId}`);
}

/** 设备通道列表 */
export function getGB28181Channels(id: string) {
  return request<any[]>(`/api/v1/gb28181/devices/${id}/channels`);
}

// ============= GB28181 媒体 =============

export interface PlayResponse {
  url: string;
  protocol: string;
  stream_id: string;
}

/** 启动实时预览 */
export function startGB28181Live(deviceId: string) {
  return request<PlayResponse>('/api/v1/media/gb28181/live/start', {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId }),
  });
}

/** 停止实时预览 */
export function stopGB28181Live(deviceId: string, streamId: string) {
  return request<null>('/api/v1/media/gb28181/live/stop', {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId, stream_id: streamId }),
  });
}

/** 启动回放 */
export function startGB28181Playback(deviceId: string, startTime: string, endTime: string) {
  return request<PlayResponse>('/api/v1/media/gb28181/playback/start', {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId, start_time: startTime, end_time: endTime }),
  });
}

/** 回放控制 */
export function controlGB28181Playback(streamId: string, action: string, speed?: number, stamp?: number) {
  return request<null>('/api/v1/media/gb28181/playback/control', {
    method: 'POST',
    body: JSON.stringify({ stream_id: streamId, action, speed, stamp }),
  });
}

/** 停止回放 */
export function stopGB28181Playback(deviceId: string, streamId: string) {
  return request<null>('/api/v1/media/gb28181/playback/stop', {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId, stream_id: streamId }),
  });
}

// ============= GB28181 配置 =============

export function getGB28181Config() {
  return request<Record<string, any>>('/api/v1/system/gb28181/config');
}
