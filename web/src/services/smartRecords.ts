import { request } from './api';
import { buildQuery } from '../utils/query';

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
  raw_result?: Record<string, unknown>;
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

type SmartRecordPage = { list: SmartRecord[]; total: number; page: number; page_size: number };

function buildSmartRecordQuery(params: SmartRecordListParams, includePagination: boolean) {
  return buildQuery({
    ...(includePagination && { page: params.page, page_size: params.page_size }),
    type: params.type,
    device_id: params.device_id,
    alarm_type: params.alarm_type,
    alarm_level: params.alarm_level,
    start_time: params.start_time,
    end_time: params.end_time,
  });
}

/** 智能记录列表 */
export function listSmartRecords(params: SmartRecordListParams) {
  return request<SmartRecordPage>(`/smart-records${buildSmartRecordQuery(params, true)}`);
}

/** 导出智能记录 CSV */
export function getSmartRecordsExportUrl(params: SmartRecordListParams): string {
  return `/api/v1/smart-records/export${buildSmartRecordQuery(params, false)}`;
}
