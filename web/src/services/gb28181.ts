import { request } from './api';
import { buildQuery } from '../utils/query';

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

export interface GB28181DeviceCreateRequest {
  device_code: string;
  sip_id?: string;
  sip_domain: string;
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

type PageResult<T> = { list: T[]; total: number; page: number; page_size: number };

/** GB28181 设备列表 */
export function listGB28181Devices(params: GB28181DeviceListParams) {
  return request<PageResult<GB28181Device>>(`/gb28181/devices${buildQuery({ page: params.page, page_size: params.page_size, keyword: params.keyword, status: params.status })}`);
}

/** 创建 GB28181 设备 */
export function createGB28181Device(data: GB28181DeviceCreateRequest) {
  return request<GB28181Device>('/gb28181/devices', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

/** GB28181 设备详情 */
export function getGB28181Device(id: string) {
  return request<GB28181Device>(`/gb28181/devices/${id}`);
}

/** 更新 GB28181 设备 */
export function updateGB28181Device(id: string, data: GB28181DeviceUpdateRequest) {
  return request<GB28181Device>(`/gb28181/devices/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

/** 删除 GB28181 设备 */
export function deleteGB28181Device(id: string) {
  return request<null>(`/gb28181/devices/${id}`, { method: 'DELETE' });
}

/** 批量删除 GB28181 设备 */
export function batchDeleteGB28181Devices(ids: string[]) {
  return request<{ total: number; success: number; failed: number }>('/gb28181/devices/batch-delete', {
    method: 'POST',
    body: JSON.stringify({ ids }),
  });
}

/** 触发目录查询 */
export function triggerCatalog(id: string) {
  return request<CatalogTaskResponse>(`/gb28181/devices/${id}/catalog`, { method: 'POST' });
}

/** 查询目录任务状态 */
export function getCatalogTaskStatus(taskId: string) {
  return request<CatalogTaskStatusResponse>(`/gb28181/catalog-tasks/${taskId}`);
}

/** 设备通道列表 */
export interface GB28181Channel {
  id: string;
  gb28181_device_id: string;
  device_name: string;
  status: string;
  manufacturer: string;
  model: string;
  channel_id: string;
  parental_id: string;
}

export function getGB28181Channels(id: string) {
  return request<GB28181Channel[]>(`/gb28181/devices/${id}/channels`);
}

// ============= GB28181 媒体 =============

export interface PlayResponse {
  url: string;
  protocol: string;
  stream_id: string;
}

/** 启动实时预览 */
export function startGB28181Live(deviceId: string) {
  return request<PlayResponse>('/media/gb28181/live/start', {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId }),
  });
}

/** 停止实时预览 */
export function stopGB28181Live(deviceId: string, streamId: string) {
  return request<null>('/media/gb28181/live/stop', {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId, stream_id: streamId }),
  });
}

/** 启动回放 */
export function startGB28181Playback(deviceId: string, startTime: string, endTime: string) {
  return request<PlayResponse>('/media/gb28181/playback/start', {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId, start_time: startTime, end_time: endTime }),
  });
}

/** 回放控制 */
export function controlGB28181Playback(streamId: string, action: string, speed?: number, stamp?: number) {
  return request<null>('/media/gb28181/playback/control', {
    method: 'POST',
    body: JSON.stringify({ stream_id: streamId, action, speed, stamp }),
  });
}

/** 停止回放 */
export function stopGB28181Playback(deviceId: string, streamId: string) {
  return request<null>('/media/gb28181/playback/stop', {
    method: 'POST',
    body: JSON.stringify({ device_id: deviceId, stream_id: streamId }),
  });
}

// ============= GB28181 配置 =============

// ============= GB28181 NVR（统一 Device 表） =============

/** GB28181 NVR 设备列表（从 Device 表查询） */
export function listGB28181NVRs(params: GB28181DeviceListParams) {
  return request<PageResult<GB28181Device>>(`/gb28181/nvrs${buildQuery({ page: params.page, page_size: params.page_size, keyword: params.keyword, status: params.status })}`);
}

/** 查询 NVR 下的所有通道 */
export function getGB28181NVRChannels(nvrId: string) {
  return request<GB28181Channel[]>(`/gb28181/nvrs/${nvrId}/channels`);
}

export function getGB28181Config() {
  return request<Record<string, string | number>>('/system/gb28181/config');
}
