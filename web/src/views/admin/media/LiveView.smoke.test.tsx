import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ChakraProvider } from '@chakra-ui/react';
import React from 'react';
import LiveView from './LiveView';

// Mock api service
vi.mock('services/api', () => ({
  request: vi.fn().mockResolvedValue({ url: 'http://localhost/live/test/hls.m3u8', protocol: 'hls', expires: Date.now() }),
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <ChakraProvider>{children}</ChakraProvider>
);

describe('LiveView 冒烟测试', () => {
  it('页面渲染不崩溃', () => {
    const { container } = render(<LiveView />, { wrapper });
    // 验证布局按钮存在
    expect(screen.getByText('1路')).toBeInTheDocument();
    expect(screen.getByText('4路')).toBeInTheDocument();
    expect(container.querySelector('video')).not.toBeInTheDocument();
  });
});
