import { format, parseISO } from 'date-fns';
import i18n from '../i18n';

export function parseOptionalNumber(value: string | undefined): number | undefined {
  if (value === undefined || value === '') return undefined;
  const n = Number(value);
  return Number.isNaN(n) ? undefined : n;
}

export function formatUptime(seconds: number): string {
  if (!seconds && seconds !== 0) return '-';
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const parts: string[] = [];
  if (d > 0) parts.push(`${d}d`);
  if (h > 0) parts.push(`${h}h`);
  parts.push(`${m}m`);
  return parts.join(' ');
}

export function formatDateTime(date: Date | string | number | undefined, pattern?: string): string {
  if (!date) return '-';
  try {
    const d = typeof date === 'string' ? parseISO(date) : new Date(date);
    const formatPattern = pattern || i18n.t('common:date.format.datetime') || 'yyyy-MM-dd HH:mm:ss';
    return format(d, formatPattern);
  } catch (e) {
    return String(date);
  }
}
