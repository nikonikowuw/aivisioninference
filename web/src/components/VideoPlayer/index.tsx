import React, { useEffect, useRef, useState, useCallback } from 'react';
import { Box, Spinner, Center, Text, Badge } from '@chakra-ui/react';
import { streamManager, Protocol } from './StreamManager';

interface FallbackConfig {
  fallbackOrder: Protocol[];
  webrtcTimeout: number;
  showProtocol: boolean;
}

const DEFAULT_FALLBACK_CONFIG: FallbackConfig = {
  fallbackOrder: ['webrtc', 'hls', 'flv'],
  webrtcTimeout: 10000,
  showProtocol: false,
};

interface VideoPlayerProps {
  url: string;
  protocol?: Protocol;
  poster?: string;
  onError?: (error: string) => void;
  onProtocolChange?: (protocol: Protocol) => void;
  fallbackConfig?: Partial<FallbackConfig>;
}

// ==================== URL 工具 ====================

function detectProtocol(url: string): Protocol {
  if (url.startsWith('webrtc://')) return 'webrtc';
  if (url.includes('.m3u8')) return 'hls';
  if (url.includes('.flv')) return 'flv';
  return 'hls';
}

function isWebRTCSupported(): boolean {
  return !!(window.RTCPeerConnection);
}

/** 将 webrtc:// 转换为 HLS/FLV URL */
function convertUrl(url: string, targetProtocol: Protocol): string | null {
  const currentProtocol = detectProtocol(url);
  if (currentProtocol === targetProtocol) return url;
  if (!url.startsWith('webrtc://')) return null;

  try {
    const urlObj = new URL(url.replace('webrtc://', 'http://'));
    const pathParts = urlObj.pathname.split('/').filter(Boolean);
    if (pathParts.length < 2) return null;
    const [app, stream] = pathParts;
    const token = urlObj.searchParams.get('token');
    const query = token ? `?token=${token}` : '';
    const host = urlObj.host.replace(':8000', ':80');

    switch (targetProtocol) {
      case 'hls': return `http://${host}/${app}/${stream}/hls.m3u8${query}`;
      case 'flv': return `http://${host}/${app}/${stream}.flv${query}`;
      default: return null;
    }
  } catch { return null; }
}

// ==================== 主组件 ====================

const VideoPlayer: React.FC<VideoPlayerProps> = ({
  url,
  protocol: forcedProtocol,
  poster,
  onError,
  onProtocolChange,
  fallbackConfig,
}) => {
  const config = { ...DEFAULT_FALLBACK_CONFIG, ...fallbackConfig };
  const videoRef = useRef<HTMLVideoElement>(null);
  const cleanupRef = useRef<(() => void) | null>(null);
  const mountedRef = useRef(true);

  const [currentProtocol, setCurrentProtocol] = useState<Protocol | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [fallbackInfo, setFallbackInfo] = useState<string | null>(null);

  const cleanup = useCallback(() => {
    if (cleanupRef.current) {
      cleanupRef.current();
      cleanupRef.current = null;
    }
  }, []);

  const tryPlay = useCallback(
    async (targetProtocol: Protocol, originalUrl: string, attemptIndex: number) => {
      const video = videoRef.current;
      if (!video || !mountedRef.current) return;

      cleanup();

      const targetUrl = convertUrl(originalUrl, targetProtocol);
      if (!targetUrl) {
        tryNextProtocol(attemptIndex + 1, originalUrl);
        return;
      }

      console.log(`[VideoPlayer] 尝试: ${targetProtocol}, url: ${targetUrl}`);
      setCurrentProtocol(targetProtocol);
      setError(null);
      setFallbackInfo(attemptIndex > 0 ? `降级自 ${config.fallbackOrder[attemptIndex - 1]}` : null);
      onProtocolChange?.(targetProtocol);

      if (targetProtocol === 'flv') {
        // FLV 不支持共享，独立播放
        let destroyed = false;
        import('flv.js').then(({ default: Flv }) => {
          if (destroyed || !mountedRef.current) return;
          if (Flv.isSupported()) {
            const flv = Flv.createPlayer({ type: 'flv', url: targetUrl });
            flv.attachMediaElement(video);
            flv.load();
            video.onplaying = () => {
              if (mountedRef.current) setLoading(false);
            };
            flv.play();
            cleanupRef.current = () => {
              destroyed = true;
              flv.pause();
              flv.unload();
              flv.detachMediaElement();
              flv.destroy();
            };
          }
        });
        return;
      }

      // WebRTC 和 HLS 都走 StreamManager 共享
      if (targetProtocol === 'webrtc' && !isWebRTCSupported()) {
        tryNextProtocol(attemptIndex + 1, originalUrl);
        return;
      }

      try {
        const result = await streamManager.subscribe(targetUrl, targetProtocol, video, {
          webrtcTimeout: config.webrtcTimeout,
        });

        if (!mountedRef.current) {
          result.destroy();
          return;
        }

        setLoading(false);
        cleanupRef.current = result.destroy;
      } catch (err) {
        console.warn(`[VideoPlayer] ${targetProtocol} 失败:`, err);
        tryNextProtocol(attemptIndex + 1, originalUrl);
      }
    },
    [cleanup, config, onProtocolChange],
  );

  const tryNextProtocol = useCallback(
    (attemptIndex: number, originalUrl: string) => {
      if (attemptIndex >= config.fallbackOrder.length) {
        const finalError = `所有协议均失败: ${config.fallbackOrder.join(' → ')}`;
        setError(finalError);
        setLoading(false);
        onError?.(finalError);
        return;
      }
      tryPlay(config.fallbackOrder[attemptIndex], originalUrl, attemptIndex);
    },
    [config, onError, tryPlay],
  );

  useEffect(() => {
    if (!url) return;

    mountedRef.current = true;
    setLoading(true);
    setError(null);
    setFallbackInfo(null);

    const startProtocol = forcedProtocol || detectProtocol(url);
    const startIndex = config.fallbackOrder.indexOf(startProtocol);
    const effectiveIndex = startIndex >= 0 ? startIndex : 0;

    tryPlay(config.fallbackOrder[effectiveIndex], url, effectiveIndex);

    return () => {
      mountedRef.current = false;
      cleanup();
    };
  }, [url, forcedProtocol]);

  return (
    <Box position="relative" w="100%" h="100%" bg="black" borderRadius="md" overflow="hidden">
      <video
        ref={videoRef}
        style={{ width: '100%', height: '100%', objectFit: 'contain' }}
        controls
        poster={poster}
        autoPlay
        playsInline
        muted
      />
      {loading && !error && (
        <Center position="absolute" top="0" left="0" w="100%" h="100%">
          <Spinner color="white" size="xl" />
        </Center>
      )}
      {error && (
        <Center position="absolute" top="0" left="0" w="100%" h="100%">
          <Text color="red.300" fontSize="sm" textAlign="center" px={4}>{error}</Text>
        </Center>
      )}
      {config.showProtocol && currentProtocol && !error && (
        <Badge position="absolute" top={2} right={2}
          colorScheme={currentProtocol === 'webrtc' ? 'green' : currentProtocol === 'hls' ? 'blue' : 'orange'}
          fontSize="xs">{currentProtocol.toUpperCase()}</Badge>
      )}
      {fallbackInfo && !error && (
        <Badge position="absolute" top={2} left={2} colorScheme="yellow" fontSize="xs">{fallbackInfo}</Badge>
      )}
    </Box>
  );
};

export default VideoPlayer;
