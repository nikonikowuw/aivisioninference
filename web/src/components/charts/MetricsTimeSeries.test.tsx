import { ChakraProvider } from '@chakra-ui/react';
import { fireEvent, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import MetricsTimeSeries from './MetricsTimeSeries';

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => ({
      'button.refresh': '刷新',
      'empty.title': '暂无数据',
      'message.loadFailed': '加载失败',
      'status.loading': '加载中...',
    })[key] || key,
  }),
}));

const wrapper = ({ children }: { children: ReactNode }) => (
  <ChakraProvider>{children}</ChakraProvider>
);

describe('MetricsTimeSeries', () => {
  it('保留标题并显示本地化空状态', () => {
    render(<MetricsTimeSeries title="加速器使用率" data={[]} />, { wrapper });

    expect(screen.getByText('加速器使用率')).toBeInTheDocument();
    expect(screen.getByText('暂无数据')).toBeInTheDocument();
    expect(screen.queryByText('noData')).not.toBeInTheDocument();
  });

  it('区分加载失败并支持重试', () => {
    const onRetry = vi.fn();
    render(
      <MetricsTimeSeries title="活跃视频流数" data={[]} error onRetry={onRetry} />,
      { wrapper },
    );

    expect(screen.getByText('活跃视频流数')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent('加载失败');
    fireEvent.click(screen.getByRole('button', { name: '刷新' }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('加载时仍保留标题', () => {
    render(<MetricsTimeSeries title="CPU 使用率" data={[]} loading />, { wrapper });

    expect(screen.getByText('CPU 使用率')).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveTextContent('加载中...');
  });
});
