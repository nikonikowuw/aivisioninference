import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ChakraProvider } from '@chakra-ui/react';
import React from 'react';
import SmartRecords from './index';

const mockUser = vi.hoisted(() => ({
  permission_codes: ['records:recognition:list', 'records:alarm:list', 'records:capture:list', 'records:export'],
}));

const listMock = vi.hoisted(() => vi.fn());
const exportCsvMock = vi.hoisted(() => vi.fn());
const updateAlarmStatusMock = vi.hoisted(() => vi.fn());
const setSearchParamsMock = vi.hoisted(() => vi.fn());
let searchParams = new URLSearchParams();

vi.mock('react-router-dom', () => ({
  useSearchParams: () => [searchParams, setSearchParamsMock],
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) => String(options?.defaultValue || key),
    i18n: { language: 'zh-CN' },
  }),
}));

vi.mock('contexts/AuthContext', () => ({
  useAuth: () => ({ user: mockUser }),
}));

vi.mock('hooks/useDateFormat', () => ({
  useDateFormat: () => ({ formatDateTime: (value: string) => value }),
}));

vi.mock('services/api', () => ({
  smartRecordsApi: {
    list: listMock,
    exportCsv: exportCsvMock,
    exportSelected: vi.fn(),
    updateAlarmStatus: updateAlarmStatusMock,
    batchDelete: vi.fn(),
    listCategoryCodes: vi.fn().mockResolvedValue([]),
  },
  devicesApi: { list: vi.fn().mockResolvedValue({ list: [] }) },
  deviceGroupsApi: { list: vi.fn().mockResolvedValue({ list: [] }) },
  tasksApi: { list: vi.fn().mockResolvedValue({ list: [] }) },
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <ChakraProvider>{children}</ChakraProvider>
);

describe('SmartRecords', () => {
  beforeEach(() => {
    searchParams = new URLSearchParams();
    mockUser.permission_codes = ['records:recognition:list', 'records:alarm:list', 'records:capture:list', 'records:export'];
    listMock.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 20 });
    exportCsvMock.mockResolvedValue(undefined);
    updateAlarmStatusMock.mockResolvedValue(undefined);
    setSearchParamsMock.mockClear();
    listMock.mockClear();
    exportCsvMock.mockClear();
    updateAlarmStatusMock.mockClear();
  });

  it('拥有三类记录权限时显示三个 Tab 并默认回退到告警记录', async () => {
    render(<SmartRecords />, { wrapper });

    expect(await screen.findByText('tabs.recognition')).toBeInTheDocument();
    expect(screen.getByText('tabs.alarm')).toBeInTheDocument();
    expect(screen.getByText('tabs.capture')).toBeInTheDocument();

    await waitFor(() => expect(setSearchParamsMock).toHaveBeenCalled());
    const nextParams = setSearchParamsMock.mock.calls[0][0] as URLSearchParams;
    expect(nextParams.get('type')).toBe('alarm');
  });

  it('仅有告警记录权限时只显示告警 Tab', async () => {
    mockUser.permission_codes = ['records:alarm:list'];
    searchParams = new URLSearchParams('type=recognition');

    render(<SmartRecords />, { wrapper });

    expect(await screen.findByText('tabs.alarm')).toBeInTheDocument();
    expect(screen.queryByText('tabs.recognition')).not.toBeInTheDocument();
    expect(screen.queryByText('tabs.capture')).not.toBeInTheDocument();
  });

  it('导出按当前 Tab 与筛选条件执行', async () => {
    searchParams = new URLSearchParams('type=capture');
    render(<SmartRecords />, { wrapper });

    await userEvent.click(await screen.findByRole('button', { name: 'button.export' }));

    await waitFor(() => expect(exportCsvMock).toHaveBeenCalledWith(expect.objectContaining({ type: 'capture' })));
  });

  it('有告警处理权限时可标记未处理告警为已处理', async () => {
    searchParams = new URLSearchParams('type=alarm');
    listMock.mockResolvedValue({
      list: [{ record_id: 'alarm-1', record_type: 'alarm', capture_time: '2026-06-09T00:00:00Z', alarm_status: 'unhandled' }],
      total: 1,
      page: 1,
      page_size: 20,
    });
    mockUser.permission_codes = ['records:alarm:list', 'records:alarm:update_status'];

    render(<SmartRecords />, { wrapper });

    await userEvent.click(await screen.findByRole('button', { name: 'actions.markHandled' }));

    await waitFor(() => expect(updateAlarmStatusMock).toHaveBeenCalledWith('alarm-1', 'handled'));
  });
});
