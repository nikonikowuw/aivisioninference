import { format, parseISO } from 'date-fns';
import i18n from '../i18n';

export function parseOptionalNumber(value: string | undefined): number | undefined {
  if (value === undefined || value === '') return undefined;
  const n = Number(value);
  return Number.isNaN(n) ? undefined : n;
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
