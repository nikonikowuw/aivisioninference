import { request } from './api';

export interface SmartRecord {
  record_id: string;
  record_type: string;
  device_id?: string;
  device_name?: string;
  task_name?: string;
  alarm_type?: string;
  alarm_level?: string;
  snapshot_image_url?: string;
  confidence?: number;
  raw_result?: any;
  created_at: string;
}

export interface SmartRecordListParams {
  page?: number;
  page_size?: number;
  type?: string;
  device_id?: string;
  alarm_type?: string;
  alarm_level?: string;
  start_time?: string;
  end_time?: string;
}

/** 智能记录列表 */
export function listSmartRecords(params: SmartRecordListParams) {
  const sp = new URLSearchParams();
  if (params.page) sp.set('page', String(params.page));
  if (params.page_size) sp.set('page_size', String(params.page_size));
  if (params.type) sp.set('type', params.type);
  if (params.device_id) sp.set('device_id', params.device_id);
  if (params.alarm_type) sp.set('alarm_type', params.alarm_type);
  if (params.alarm_level) sp.set('alarm_level', params.alarm_level);
  if (params.start_time) sp.set('start_time', params.start_time);
  if (params.end_time) sp.set('end_time', params.end_time);
  const qs = sp.toString() ? `?${sp}` : '';
  return request<{ list: SmartRecord[]; total: number; page: number; page_size: number }>(
    `/api/v1/smart-records${qs}`,
  );
}

/** 导出智能记录 CSV */
export function getSmartRecordsExportUrl(params: SmartRecordListParams): string {
  const sp = new URLSearchParams();
  if (params.type) sp.set('type', params.type);
  if (params.device_id) sp.set('device_id', params.device_id);
  if (params.alarm_type) sp.set('alarm_type', params.alarm_type);
  if (params.alarm_level) sp.set('alarm_level', params.alarm_level);
  if (params.start_time) sp.set('start_time', params.start_time);
  if (params.end_time) sp.set('end_time', params.end_time);
  const qs = sp.toString() ? `?${sp}` : '';
  return `/api/v1/smart-records/export${qs}`;
}
