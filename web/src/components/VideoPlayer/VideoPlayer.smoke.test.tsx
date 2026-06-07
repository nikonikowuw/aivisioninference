import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { ChakraProvider } from '@chakra-ui/react';
import React from 'react';
import VideoPlayer from './index';

// 简单的包装器，提供 Chakra UI 上下文
const wrapper = ({ children }: { children: React.ReactNode }) => (
  <ChakraProvider>{children}</ChakraProvider>
);

describe('VideoPlayer 冒烟测试', () => {
  it('HLS URL 渲染但不崩溃', () => {
    const { container } = render(
      <VideoPlayer url="http://localhost/live/test/hls.m3u8" />,
      { wrapper },
    );
    // 验证 video 元素存在
    const video = container.querySelector('video');
    expect(video).toBeInTheDocument();
  });

  it('WebRTC URL 渲染但不崩溃', () => {
    const { container } = render(
      <VideoPlayer url="webrtc://localhost:8000/live/test?token=xxx" protocol="webrtc" />,
      { wrapper },
    );
    const video = container.querySelector('video');
    expect(video).toBeInTheDocument();
  });

  it('FLV URL 渲染但不崩溃', () => {
    const { container } = render(
      <VideoPlayer url="http://localhost/live/test.flv" />,
      { wrapper },
    );
    const video = container.querySelector('video');
    expect(video).toBeInTheDocument();
  });

  it('空 URL 渲染但不崩溃', () => {
    const { container } = render(
      <VideoPlayer url="" />,
      { wrapper },
    );
    const video = container.querySelector('video');
    expect(video).toBeInTheDocument();
  });
});
